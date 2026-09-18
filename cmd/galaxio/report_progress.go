package main

import (
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/galax-io/galaxio-cli/internal/report"
)

// The progress block of contracts/cli.md, drawn on standard error while a run is
// read.
const (
	// progressLines is how many lines the block has.
	progressLines = 6
	// progressWidth is the widest a line of the block may be; a longer one is
	// cut, never wrapped.
	progressWidth = 79
	// progressNameWidth is the most of the log's name the first line shows.
	progressNameWidth = 24
	// progressBarCells is how many cells the bar of bytes read has.
	progressBarCells = 30
	// progressCountWidth is the width of the count and share columns, and
	// progressFigureWidth that of every response-time column.
	progressCountWidth  = 8
	progressFigureWidth = 7
	// progressFirstDraw is how far into a read the block is first drawn, so
	// that a read that ends sooner draws nothing.
	progressFirstDraw = 500 * time.Millisecond
	// progressRedraw is the least time between two draws.
	progressRedraw = 200 * time.Millisecond
	// progressStatement is the second line: the unit of every time and what
	// the percentiles are.
	progressStatement = "figures so far · times in ms · percentiles are t-digest estimates, interpolated"
)

// progressSpinner is the spinner's frames, one a draw.
var progressSpinner = []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")

// progressRanks are the percentile ranks the block shows while a log is read,
// whatever --percentiles asks of the summary: the block has room for three and
// is not the report (FR-038). The heading and the rows are both written from
// this list, so a column cannot end up under the wrong name.
var progressRanks = []float64{50, 95, 99}

// progress draws the block from the ticks of one walk and erases it. It is used
// from the goroutine that walks, and holds nothing of the run but the time of
// its last draw.
type progress struct {
	w      io.Writer
	name   string
	source *report.Source
	clock  func() time.Time
	paint  painter

	start, last time.Time
	draws       int
}

func newProgress(w io.Writer, log string, source *report.Source, clock func() time.Time, color bool) *progress {
	return &progress{
		w:      w,
		name:   cut(printable(filepath.Base(log)), progressNameWidth),
		source: source,
		clock:  clock,
		paint:  painter(color),
		start:  clock(),
	}
}

// tick draws the block when it is due: progressFirstDraw into the read, then
// once progressRedraw has passed since the last draw.
func (p *progress) tick(s report.Summary) {
	now := p.clock()

	if p.draws == 0 && now.Sub(p.start) < progressFirstDraw || p.draws > 0 && now.Sub(p.last) < progressRedraw {
		return
	}

	var b strings.Builder

	if p.draws > 0 {
		fmt.Fprintf(&b, "\x1b[%dA", progressLines)
	}

	size, _ := p.source.Size()

	for _, line := range blockLines(p.draws, p.name, p.source.BytesRead(), size, now.Sub(p.start), s, p.paint) {
		b.WriteString("\x1b[2K")
		b.WriteString(line)
		b.WriteString("\n")
	}

	// A write that fails is not the read's failure: the report still follows,
	// and it reports its own.
	_, _ = io.WriteString(p.w, b.String())

	p.draws++
	p.last = now
}

// erase removes the block if it is on the screen. It is safe on a nil progress
// and to call more than once.
func (p *progress) erase() {
	if p == nil || p.draws == 0 {
		return
	}

	_, _ = fmt.Fprintf(p.w, "\x1b[%dA\x1b[J", progressLines)
	p.draws = 0
}

// blockLines renders the six lines of a draw: draws before it, the log's name,
// the bytes read of the size seen at opening — 0 where that is unknown, as
// report.Source reports it — the time since the read began, and the summary so
// far.
func blockLines(draws int, name string, read, size int64, elapsed time.Duration, s report.Summary, p painter) []string {
	first := []span{{text: string(progressSpinner[draws%len(progressSpinner)]) + " reading " + name}}

	if size > 0 {
		done := min(read, size)
		filled := int(done * progressBarCells / size)
		percent := done * 100 / size

		first = append(first,
			span{text: "  " + strings.Repeat("━", filled)},
			span{text: strings.Repeat("─", progressBarCells-filled), st: faint},
			span{text: "  " + padLeft(strconv.FormatInt(percent, 10), 3) + " %"},
		)

		if draws > 0 && done > 0 {
			left := time.Duration(float64(elapsed) * float64(size-done) / float64(done))
			first = append(first, span{text: "  " + clockTime(left) + " left"})
		}
	}

	heading := padRight("", summaryLabelWidth) + padLeft("count", progressCountWidth) + padLeft("share", progressCountWidth)
	for _, column := range progressColumns() {
		heading += padLeft(column, progressFigureWidth)
	}

	all := s.All()

	return []string{
		render(first, p),
		render([]span{{text: progressStatement, st: faint}}, p),
		render([]span{{text: heading, st: faint}}, p),
		progressRow(span{text: "all requests"}, all, "", p),
		progressRow(span{text: "  ✓ ok", st: green}, s.OK, progressShare(s.Share(s.OK.Count())), p),
		progressRow(span{text: "  ✗ failed", st: failedStyle(s)}, s.Failed, progressShare(s.Share(s.Failed.Count())), p),
	}
}

// progressColumns names the block's figure columns, the ranks of progressRanks
// among them.
func progressColumns() []string {
	columns := []string{"min", "mean"}
	for _, rank := range progressRanks {
		columns = append(columns, "p"+strconv.FormatFloat(rank, 'f', -1, 64))
	}

	return append(columns, "max")
}

// progressRow renders one outcome's line: its label, its count, its share, and
// its times so far, a column for each of progressColumns.
func progressRow(name span, f report.Figures, share string, p painter) string {
	values := []string{abbreviate(f.Count()), share, figure(f.Min()), figure(f.Mean())}
	for _, rank := range progressRanks {
		values = append(values, figure(f.Percentile(rank)))
	}

	values = append(values, figure(f.Max()))

	text := strings.Repeat(" ", max(0, summaryLabelWidth-utf8.RuneCountInString(name.text)))
	for i, value := range values {
		// The count and the share have a column of their own; every figure
		// after them is a response time and shares one width.
		width := progressFigureWidth
		if i < 2 {
			width = progressCountWidth
		}

		text += padLeft(value, width)
	}

	return render([]span{name, {text: text}}, p)
}

// progressShare renders a share with one decimal, or "-" before any request.
func progressShare(v float64, ok bool) string {
	if !ok {
		return "-"
	}

	return strconv.FormatFloat(v, 'f', 1, 64) + " %"
}

// abbreviate renders a count exactly below 10 000 and above that in thousands,
// millions or billions: with one decimal below 100 of the unit and none above,
// moving to the next unit when rounding reaches 1000 of this one.
func abbreviate(n int) string {
	if n < 10_000 {
		return strconv.Itoa(n)
	}

	const suffixes = "kMG"

	v := float64(n) / 1e3

	for i := 0; ; i++ {
		suffix := suffixes[i : i+1]

		if math.Round(v*10)/10 < 100 {
			return strconv.FormatFloat(v, 'f', 1, 64) + suffix
		}

		if math.Round(v) < 1000 || i == len(suffixes)-1 {
			return strconv.FormatFloat(math.Round(v), 'f', 0, 64) + suffix
		}

		v /= 1e3
	}
}

// clockTime renders a duration as m:ss, or h:mm:ss from an hour.
func clockTime(d time.Duration) string {
	seconds := int64(d.Round(time.Second) / time.Second)

	if seconds >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
	}

	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

// span is a piece of a line and how it is drawn.
type span struct {
	text string
	st   style
}

// render joins the spans of a line, cut at progressWidth columns, painting only
// what is kept, so that an escape sequence is never cut in half.
func render(spans []span, p painter) string {
	var b strings.Builder

	room := progressWidth

	for _, s := range spans {
		if room <= 0 {
			break
		}

		text := cut(s.text, room)
		room -= utf8.RuneCountInString(text)
		b.WriteString(p.paint(text, s.st))
	}

	return b.String()
}

// cut returns text cut to at most width runes.
func cut(text string, width int) string {
	if utf8.RuneCountInString(text) <= width {
		return text
	}

	return string([]rune(text)[:width])
}
