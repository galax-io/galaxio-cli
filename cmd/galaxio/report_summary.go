package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/galax-io/galaxio-cli/internal/report"
)

// The layout of the summary, as contracts/cli.md pins it.
const (
	// summaryLabelWidth is the width of the response-time table's label column.
	summaryLabelWidth = 20
	// summaryFigureWidth is the least width of a figure column: a column grows
	// to keep two spaces before a longer value.
	summaryFigureWidth = 7
	// summaryShareWidth is the width of a band's share.
	summaryShareWidth = 7
	// summaryCountWidth is the least width of a band's count: it grows, as a
	// figure column does, to keep two spaces before a longer count.
	summaryCountWidth = 6
	// barCells is how many cells a band's bar has.
	barCells = 20
	// headlineWidth is the widest the headline may be on one line before its
	// segments go on lines of their own. It is measured on the text, never on
	// the terminal.
	headlineWidth = 100
	// headlineGap separates the headline's segments on one line.
	headlineGap = "      "
	// closingLine is the unit of every time and the percentile statement.
	closingLine = "times in ms · percentiles are galaxio's t-digest estimates, interpolated"
)

// style is how a piece of the summary is drawn on a terminal that shows colour.
type style string

const (
	plain style = ""
	faint style = "2"
	green style = "32"
	red   style = "31"
)

// painter draws in colour, or writes the text unchanged when colour is off.
type painter bool

func (p painter) paint(text string, s style) string {
	if !bool(p) || s == plain || text == "" {
		return text
	}

	return "\x1b[" + string(s) + "m" + text + "\x1b[0m"
}

// formatSummary renders the summary of a run as contracts/cli.md lays it out: a
// headline, the response-time table, the bands as bars, and the closing line.
// With color set, outcomes are green and red and headings faint; without it the
// text holds not one escape byte.
func formatSummary(s report.Summary, color bool) string {
	p := painter(color)

	// The figures of all requests are merged from the two outcomes' once, for
	// the three parts that read them.
	all := s.All()

	var b strings.Builder

	b.WriteString(headline(s, all, p))
	b.WriteString("\n\n")
	b.WriteString(responseTimeTable(s, all, p))
	b.WriteString("\n")
	b.WriteString(bandLines(s, all.Count(), p))
	b.WriteString("\n")
	b.WriteString(p.paint(closingLine, faint))
	b.WriteString("\n")

	return b.String()
}

// failedStyle is how the failed outcome is drawn: red once the run holds a
// failure, and like any other text while it holds none, so that a run that
// failed nothing is not coloured as though it had.
func failedStyle(s report.Summary) style {
	if s.Failed.Count() > 0 {
		return red
	}

	return plain
}

// label draws a row's label and pads it to the table's label column. The
// padding is written outside the paint, so that no colour runs into the
// figures beside it.
func label(text string, st style, p painter) string {
	return p.paint(text, st) + strings.Repeat(" ", max(0, summaryLabelWidth-utf8.RuneCountInString(text)))
}

// segment is one part of the headline, as drawn and as measured.
type segment struct {
	drawn string
	width int
}

// headline renders the requests and their rate, then ok and failed with their
// share and rate.
func headline(s report.Summary, all report.Figures, p painter) string {
	outcome := func(label string, count int, st style) segment {
		mark := fmt.Sprintf(label, count)
		rest := fmt.Sprintf(" · %s %% · %s/s", rate(s.Share(count)), rate(s.Rate(count)))

		return segment{drawn: p.paint(mark, st) + rest, width: utf8.RuneCountInString(mark + rest)}
	}

	first := fmt.Sprintf("%s · %s req/s", plural(all.Count(), "request"), rate(s.Rate(all.Count())))
	segments := []segment{
		{drawn: first, width: utf8.RuneCountInString(first)},
		outcome("✓ %d ok", s.OK.Count(), green),
		outcome("✗ %d failed", s.Failed.Count(), failedStyle(s)),
	}

	return joinHeadline(segments)
}

// joinHeadline puts the segments on one line, six spaces apart, while that line
// fits headlineWidth columns, and one to a line beyond that.
func joinHeadline(segments []segment) string {
	width := utf8.RuneCountInString(headlineGap) * (len(segments) - 1)

	drawn := make([]string, len(segments))
	for i, seg := range segments {
		width += seg.width
		drawn[i] = seg.drawn
	}

	if width > headlineWidth {
		return strings.Join(drawn, "\n")
	}

	return strings.Join(drawn, headlineGap)
}

// responseTimeTable renders the heading and one row each for all, ok and failed
// requests, a column for each statistic.
func responseTimeTable(s report.Summary, all report.Figures, p painter) string {
	headers := []string{"min", "mean", "std"}
	for _, rank := range s.Options.Percentiles {
		headers = append(headers, "p"+strconv.FormatFloat(rank, 'f', -1, 64))
	}

	headers = append(headers, "max")

	rows := []struct {
		label   string
		st      style
		figures report.Figures
	}{
		{"all", plain, all},
		{"✓ ok", green, s.OK},
		{"✗ failed", failedStyle(s), s.Failed},
	}

	cells := make([][]string, len(rows))
	for i, row := range rows {
		values := []string{figure(row.figures.Min()), figure(row.figures.Mean()), figure(row.figures.StdDev())}
		for _, rank := range s.Options.Percentiles {
			values = append(values, figure(row.figures.Percentile(rank)))
		}

		cells[i] = append(values, figure(row.figures.Max()))
	}

	widths := make([]int, len(headers))
	for j, header := range headers {
		widths[j] = max(summaryFigureWidth, len(header)+2)
		for i := range rows {
			widths[j] = max(widths[j], len(cells[i][j])+2)
		}
	}

	var b strings.Builder

	heading := padRight("response time, ms", summaryLabelWidth)
	for j, header := range headers {
		heading += padLeft(header, widths[j])
	}

	b.WriteString(p.paint(heading, faint))
	b.WriteString("\n")

	for i, row := range rows {
		b.WriteString(label(row.label, row.st, p))

		for j, cell := range cells[i] {
			b.WriteString(padLeft(cell, widths[j]))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// bandLines renders the three response-time bands of successful requests and
// the failed band, each as a bar, a share of the total requests and a count.
func bandLines(s report.Summary, total int, p painter) string {
	bands := s.Options.Bands
	lines := []struct {
		count  int
		label  string
		failed bool
	}{
		{s.Under, fmt.Sprintf("ok under %d ms", bands.Lower), false},
		{s.Between, fmt.Sprintf("ok %d to %d ms", bands.Lower, bands.Upper), false},
		{s.Over, fmt.Sprintf("ok %d ms and over", bands.Upper), false},
		{s.Failed.Count(), "failed", true},
	}

	countWidth := summaryCountWidth
	for _, line := range lines {
		countWidth = max(countWidth, len(strconv.Itoa(line.count))+2)
	}

	var b strings.Builder

	for _, line := range lines {
		filled := barLength(line.count, total)

		filledStyle := plain
		if line.failed && line.count > 0 {
			filledStyle = red
		}

		b.WriteString(p.paint(strings.Repeat("█", filled), filledStyle))
		b.WriteString(p.paint(strings.Repeat("░", barCells-filled), faint))
		b.WriteString(padLeft(rate(s.Share(line.count)), summaryShareWidth))
		b.WriteString(" %")
		b.WriteString(padLeft(strconv.Itoa(line.count), countWidth))
		b.WriteString("  ")
		b.WriteString(line.label)
		b.WriteString("\n")
	}

	return b.String()
}

// barLength returns how many of a bar's cells a band of count requests out of
// total fills: its share rounded half up to a twentieth, at least one for a
// band that holds a request and at most one short of full for a band that does
// not hold them all, so that a small band stays visible and a nearly full one
// honest.
func barLength(count, total int) int {
	switch {
	case total <= 0 || count <= 0:
		return 0
	case count >= total:
		return barCells
	}

	cells := (2*barCells*count + total) / (2 * total)

	return min(max(cells, 1), barCells-1)
}

// figure renders a time in whole milliseconds, or "-" for one that does not
// exist, which is never shown as 0.
func figure(v int64, ok bool) string {
	if !ok {
		return "-"
	}

	return strconv.FormatInt(v, 10)
}

// decimal renders a rate or a share with at most two decimals, trailing zeros
// removed, a point, and no thousands separator.
func decimal(v float64) string {
	return strings.TrimSuffix(strings.TrimRight(strconv.FormatFloat(v, 'f', 2, 64), "0"), ".")
}

// rate renders a rate or a share, or "-" for a figure that does not exist — a
// run that cannot be timed, or one that holds no request — which is never shown
// as 0.
func rate(v float64, ok bool) string {
	if !ok {
		return "-"
	}

	return decimal(v)
}

func padLeft(text string, width int) string {
	return strings.Repeat(" ", max(0, width-utf8.RuneCountInString(text))) + text
}

func padRight(text string, width int) string {
	return text + strings.Repeat(" ", max(0, width-utf8.RuneCountInString(text)))
}
