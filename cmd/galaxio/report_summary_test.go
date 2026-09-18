package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/galax-io/galaxio-cli/internal/report"
	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/model"
	"github.com/spf13/cobra"
)

var corpusVersions = []string{"3.11.5", "3.12.0", "3.13.1", "3.14.9", "3.15.1"}

// summaryOf returns the summary part of what the command printed: everything
// after the blank line that ends the run description.
func summaryOf(t *testing.T, stdout string) string {
	t.Helper()

	i := strings.Index(stdout, "\n\n")
	if i < 0 {
		t.Fatalf("no blank line between the description and the summary in %q", stdout)
	}

	return stdout[i+2:]
}

// readGolden reads one golden file of testdata/report.
func readGolden(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "report", name))
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}

	return string(data)
}

// runSummary runs the command in process over one corpus run and returns what it
// printed.
func runSummary(t *testing.T, dir string, color bool) string {
	t.Helper()

	var stdout, stderr bytes.Buffer

	if _, err := runReport(context.Background(), reportOptions{Tool: "gatling", Path: dir, Color: color, Stdout: &stdout, Stderr: &stderr}); err != nil {
		t.Fatalf("runReport: %v; stderr: %s", err, stderr.String())
	}

	return stdout.String()
}

// TestReportSummaryGolden pins the summary of every corpus run, plain and, for
// 3.13.1, in colour. The 3.13.1 file is the block contracts/cli.md shows.
func TestReportSummaryGolden(t *testing.T) {
	t.Parallel()

	for _, version := range corpusVersions {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			code, stdout, stderr := runCLI("report", "gatling", filepath.Join(reportCorpus, version))
			if code != exitOK {
				t.Fatalf("exit code %d; stderr: %s", code, stderr)
			}

			if got, want := summaryOf(t, stdout), readGolden(t, version+".summary.golden"); got != want {
				t.Errorf("summary =\n%s\nwant\n%s", got, want)
			}
		})
	}

	t.Run("3.13.1 in colour", func(t *testing.T) {
		t.Parallel()

		got := summaryOf(t, runSummary(t, filepath.Join(reportCorpus, "3.13.1"), true))
		if want := readGolden(t, "3.13.1.summary.color.golden"); got != want {
			t.Errorf("summary =\n%q\nwant\n%q", got, want)
		}
	})
}

// summaryFigures is what a test reads back from a printed summary: the headline
// segments, the table's cells by row label, and the four band lines' shares and
// counts.
type summaryFigures struct {
	headline []string
	columns  []string
	rows     map[string][]string
	bands    [][2]string
}

func parseSummary(t *testing.T, summary string) summaryFigures {
	t.Helper()

	parts := strings.Split(strings.TrimSuffix(summary, "\n"), "\n\n")
	if len(parts) != 4 {
		t.Fatalf("summary has %d parts, want headline, table, bands and closing line:\n%s", len(parts), summary)
	}

	f := summaryFigures{rows: map[string][]string{}}

	for _, line := range strings.Split(parts[0], "\n") {
		f.headline = append(f.headline, strings.Split(line, "      ")...)
	}

	table := strings.Split(parts[1], "\n")
	f.columns = strings.Fields(string([]rune(table[0])[summaryLabelWidth:]))

	for _, line := range table[1:] {
		runes := []rune(line)
		f.rows[strings.TrimSpace(string(runes[:summaryLabelWidth]))] = strings.Fields(string(runes[summaryLabelWidth:]))
	}

	for _, line := range strings.Split(parts[2], "\n") {
		fields := strings.Fields(string([]rune(line)[barCells:]))
		f.bands = append(f.bands, [2]string{fields[0], fields[2]})
	}

	return f
}

// cell returns a row's figure under a column.
func (f summaryFigures) cell(t *testing.T, row, column string) string {
	t.Helper()

	i := slices.Index(f.columns, column)
	if i < 0 || len(f.rows[row]) != len(f.columns) {
		t.Fatalf("no %s under %s in %v %v", row, column, f.columns, f.rows)
	}

	return f.rows[row][i]
}

// TestReportSummaryAcceptance walks the acceptance scenarios of user story 1
// through the command.
func TestReportSummaryAcceptance(t *testing.T) {
	t.Run("the 3.13.1 run prints Gatling's figures", func(t *testing.T) {
		t.Parallel()

		dir := filepath.Join(reportCorpus, "3.13.1")

		code, stdout, stderr := runCLI("report", "gatling", dir)
		if code != exitOK {
			t.Fatalf("exit code %d; stderr: %s", code, stderr)
		}

		if fields, _ := reportFields(t, stdout); fields["requests"] != "102 (84 ok, 18 failed)" {
			t.Errorf("the run description is not printed above the summary: %q", stdout)
		}

		f := parseSummary(t, summaryOf(t, stdout))

		if expected := []string{"102 requests · 25.5 req/s", "✓ 84 ok · 82.35 % · 21/s", "✗ 18 failed · 17.65 % · 4.5/s"}; !slices.Equal(f.headline, expected) {
			t.Errorf("headline = %q, want %q", f.headline, expected)
		}

		for _, tt := range []struct {
			row      string
			expected map[string]string
		}{
			{"all", map[string]string{"min": "0", "max": "1503", "mean": "89", "std": "353"}},
			{"✓ ok", map[string]string{"min": "0", "max": "1503", "mean": "108", "std": "387"}},
			{"✗ failed", map[string]string{"min": "0", "max": "4", "mean": "1", "std": "1"}},
		} {
			for column, expected := range tt.expected {
				if got := f.cell(t, tt.row, column); got != expected {
					t.Errorf("%s %s = %s, want %s", tt.row, column, got, expected)
				}
			}
		}

		if expected := [][2]string{{"76.47", "78"}, {"0", "0"}, {"5.88", "6"}, {"17.65", "18"}}; !slices.Equal(f.bands, expected) {
			t.Errorf("bands = %v, want %v", f.bands, expected)
		}
	})

	t.Run("a text run prints what global_stats.json holds", func(t *testing.T) {
		t.Parallel()

		for _, version := range []string{"3.11.5", "3.12.0"} {
			data, err := os.ReadFile(filepath.Join(reportCorpus, version, "global_stats.json"))
			if err != nil {
				t.Fatalf("read global_stats.json: %v", err)
			}

			recorded, err := reporttest.ReadGlobalStats(data)
			if err != nil {
				t.Fatalf("ReadGlobalStats: %v", err)
			}

			f := parseSummary(t, summaryOf(t, runSummary(t, filepath.Join(reportCorpus, version), false)))

			for i, row := range []string{"all", "✓ ok", "✗ failed"} {
				for column, value := range map[string]reporttest.Value{"min": recorded.Min[i], "max": recorded.Max[i], "mean": recorded.Mean[i], "std": recorded.StdDev[i]} {
					if got := f.cell(t, row, column); got != strconv.FormatInt(value.N, 10) {
						t.Errorf("%s: %s %s = %s, Gatling wrote %d", version, row, column, got, value.N)
					}
				}
			}

			counts := [3]int64{recorded.Count[0], recorded.Count[1], recorded.Count[2]}
			expected := []string{
				strconv.FormatInt(counts[0], 10) + " requests · " + decimal(recorded.Rate[0]) + " req/s",
				"✓ " + strconv.FormatInt(counts[1], 10) + " ok · " + decimal(float64(counts[1])/float64(counts[0])*100) + " % · " + decimal(recorded.Rate[1]) + "/s",
				"✗ " + strconv.FormatInt(counts[2], 10) + " failed · " + decimal(float64(counts[2])/float64(counts[0])*100) + " % · " + decimal(recorded.Rate[2]) + "/s",
			}

			if !slices.Equal(f.headline, expected) {
				t.Errorf("%s: headline = %q, Gatling wrote %q", version, f.headline, expected)
			}

			for i, band := range f.bands {
				if want := [2]string{decimal(recorded.Shares[i]), strconv.FormatInt(recorded.Bands[i], 10)}; band != want {
					t.Errorf("%s: band %d = %v, Gatling wrote %v", version, i+1, band, want)
				}
			}
		}
	})

	t.Run("a directory holding only the log prints what the full directory prints", func(t *testing.T) {
		t.Parallel()

		full := filepath.Join(reportCorpus, "3.13.1")
		bare := writeRun(t, corpusLog(t, "3.13.1"))

		if got, want := summaryOf(t, runSummary(t, bare, false)), summaryOf(t, runSummary(t, full, false)); got != want {
			t.Errorf("the bare directory printed\n%s\nthe full one\n%s", got, want)
		}
	})

	t.Run("two runs of one log are identical", func(t *testing.T) {
		t.Parallel()

		dir := filepath.Join(reportCorpus, "3.15.1")
		if first, second := runSummary(t, dir, false), runSummary(t, dir, false); first != second {
			t.Errorf("two runs differ:\n%s\n%s", first, second)
		}
	})

	t.Run("quiet prints nothing", func(t *testing.T) {
		t.Parallel()

		code, stdout, stderr := runCLI("report", "gatling", filepath.Join(reportCorpus, "3.13.1"), "--quiet")
		if code != exitOK || stdout != "" || stderr != "" {
			t.Errorf("--quiet: exit %d, stdout %q, stderr %q", code, stdout, stderr)
		}
	})

	t.Run("each outcome is a row and each band a bar, and no request or group has one", func(t *testing.T) {
		t.Parallel()

		stdout := runSummary(t, filepath.Join(reportCorpus, "3.12.0"), false)
		summary := summaryOf(t, stdout)
		f := parseSummary(t, summary)

		if len(f.rows) != 3 || f.rows["all"] == nil || f.rows["✓ ok"] == nil || f.rows["✗ failed"] == nil || len(f.bands) != 4 {
			t.Errorf("rows %v and %d bands, want all, ok and failed and four bands", f.rows, len(f.bands))
		}

		for _, name := range []string{"GET /", "unknown host", "connect refused", "outer", "inner"} {
			if strings.Contains(stdout, name) {
				t.Errorf("the output names %q, a request or group of the run", name)
			}
		}
	})

	t.Run("colour only when asked", func(t *testing.T) {
		t.Parallel()

		dir := filepath.Join(reportCorpus, "3.13.1")

		coloured := runSummary(t, dir, true)
		for _, want := range []string{"\x1b[32m✓ 84 ok\x1b[0m", "\x1b[31m✗ 18 failed\x1b[0m", "\x1b[32m✓ ok\x1b[0m", "\x1b[2mtimes in ms"} {
			if !strings.Contains(coloured, want) {
				t.Errorf("the coloured summary lacks %q", want)
			}
		}

		if _, stdout, _ := runCLI("report", "gatling", dir); strings.Contains(stdout, "\x1b") {
			t.Errorf("the command wrote an escape sequence to a buffer: %q", stdout)
		}
	})
}

// TestReportChangesNoFile holds the command to writing no file: the run
// directory and the working directory are the same before and after.
func TestReportChangesNoFile(t *testing.T) {
	run := t.TempDir()
	copyTree(t, filepath.Join(reportCorpus, "3.13.1"), run)

	work := t.TempDir()
	t.Chdir(work)

	before := [2]map[string]string{snapshot(t, run), snapshot(t, work)}

	if code, _, stderr := runCLI("report", "gatling", run); code != exitOK {
		t.Fatalf("exit code %d; stderr: %s", code, stderr)
	}

	after := [2]map[string]string{snapshot(t, run), snapshot(t, work)}

	for i, dir := range []string{"the run directory", "the working directory"} {
		if !mapsEqual(before[i], after[i]) {
			t.Errorf("%s changed:\nbefore %v\nafter  %v", dir, before[i], after[i])
		}
	}
}

// copyTree copies every regular file under src into dst.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()

	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(filepath.Join(dst, rel), data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy %s: %v", src, err)
	}
}

// snapshot records every entry under root: its kind, mode, size, modification
// time and, for a file, the digest of its content.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()

	entries := map[string]string{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		entry := info.Mode().String() + " " + strconv.FormatInt(info.Size(), 10) + " " + info.ModTime().UTC().Format(time.RFC3339Nano)

		if info.Mode().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			sum := sha256.Sum256(data)
			entry += " " + hex.EncodeToString(sum[:])
		}

		entries[path] = entry

		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}

	return entries
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if b[k] != v {
			return false
		}
	}

	return true
}

// itemsReader yields fixed items, as a run reader would.
type itemsReader struct{ items []model.Item }

func (r *itemsReader) Run() model.Run { return model.Run{} }

func (r *itemsReader) Next() (model.Item, error) {
	if len(r.items) == 0 {
		return model.Item{}, io.EOF
	}

	item := r.items[0]
	r.items = r.items[1:]

	return item, nil
}

// summarise builds a summary from hand-made items at the default options.
func summarise(t *testing.T, items ...model.Item) report.Summary {
	t.Helper()

	s, err := report.Scan(context.Background(), &itemsReader{items: items}, report.DefaultOptions())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	return s
}

func request(at, ms int64, outcome model.Outcome) model.Item {
	start := time.Time{}
	if at >= 0 {
		start = time.UnixMilli(at).UTC()
	}

	return model.Item{Kind: model.ItemSample, Sample: model.Sample{
		Name: "r", Start: start, Duration: model.Some(time.Duration(ms) * time.Millisecond), Outcome: outcome,
	}}
}

// TestFormatSummaryAbsence holds the summary's rendering of what does not exist:
// never a zero where there is nothing to count.
func TestFormatSummaryAbsence(t *testing.T) {
	t.Parallel()

	t.Run("a run with no request", func(t *testing.T) {
		t.Parallel()

		f := parseSummary(t, formatSummary(summarise(t), false))

		if expected := []string{"0 requests · - req/s", "✓ 0 ok · - % · -/s", "✗ 0 failed · - % · -/s"}; !slices.Equal(f.headline, expected) {
			t.Errorf("headline = %q, want %q", f.headline, expected)
		}

		for _, band := range f.bands {
			if band != [2]string{"-", "0"} {
				t.Errorf("band %v, want a share of - and a count of 0", band)
			}
		}

		for row, cells := range f.rows {
			for _, cell := range cells {
				if cell != "-" {
					t.Errorf("%s holds %q, want - in every column", row, cell)
				}
			}
		}
	})

	t.Run("nothing failed", func(t *testing.T) {
		t.Parallel()

		s := summarise(t, request(1000, 10, model.OutcomeSuccess), request(2500, 900, model.OutcomeSuccess))

		f := parseSummary(t, formatSummary(s, false))
		if f.headline[2] != "✗ 0 failed · 0 % · 0/s" || f.bands[3] != [2]string{"0", "0"} {
			t.Errorf("headline %q and failed band %v, want 0, 0 %% and 0/s", f.headline, f.bands[3])
		}

		for _, cell := range f.rows["✗ failed"] {
			if cell != "-" {
				t.Errorf("failed row holds %q, want -", cell)
			}
		}

		if coloured := formatSummary(s, true); strings.Contains(coloured, "\x1b[31m") {
			t.Errorf("nothing failed and the summary is red: %q", coloured)
		}
	})

	t.Run("an untimed run", func(t *testing.T) {
		t.Parallel()

		f := parseSummary(t, formatSummary(summarise(t, request(-1, 10, model.OutcomeSuccess), request(-1, 20, model.OutcomeFailure)), false))

		if expected := []string{"2 requests · - req/s", "✓ 1 ok · 50 % · -/s", "✗ 1 failed · 50 % · -/s"}; !slices.Equal(f.headline, expected) {
			t.Errorf("headline = %q, want %q", f.headline, expected)
		}
	})

	t.Run("a value wider than its column", func(t *testing.T) {
		t.Parallel()

		summary := formatSummary(summarise(t, request(1000, 12345678, model.OutcomeSuccess)), false)
		table := strings.Split(strings.Split(summary, "\n\n")[1], "\n")

		for _, line := range table {
			if n := utf8.RuneCountInString(line); n != utf8.RuneCountInString(table[0]) {
				t.Errorf("line %q is %d wide, the heading %d", line, n, utf8.RuneCountInString(table[0]))
			}
		}

		// Every column holding the eight-digit value grows to ten; std, which
		// holds 0, keeps its seven.
		if !strings.HasSuffix(table[0], "p99       max") || !strings.Contains(table[0], "   std       p50") || !strings.HasSuffix(table[1], "12345678  12345678") {
			t.Errorf("the max column does not keep two spaces before 12345678:\n%s", strings.Join(table, "\n"))
		}
	})
}

// TestJoinHeadline holds the headline to one line while it fits 100 columns and
// one segment a line beyond that.
func TestJoinHeadline(t *testing.T) {
	t.Parallel()

	segments := func(total int) []segment {
		first := total - 2*utf8.RuneCountInString(headlineGap) - 2*10

		return []segment{{drawn: strings.Repeat("a", first), width: first}, {drawn: "bbbbbbbbbb", width: 10}, {drawn: "cccccccccc", width: 10}}
	}

	if got := joinHeadline(segments(100)); strings.Contains(got, "\n") || utf8.RuneCountInString(got) != 100 {
		t.Errorf("a headline of 100 columns = %q, want one line", got)
	}

	if got := joinHeadline(segments(101)); strings.Count(got, "\n") != 2 {
		t.Errorf("a headline of 101 columns = %q, want three lines", got)
	}
}

// TestBarLength holds the bar to its rule: the share rounded half up to a
// twentieth, at least one cell for a band that holds a request, and never full
// for one that does not hold them all.
func TestBarLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		count, total int
		expected     int
	}{
		{name: "nothing", count: 0, total: 102, expected: 0},
		{name: "no request at all", count: 0, total: 0, expected: 0},
		{name: "a sliver", count: 1, total: 1000, expected: 1},
		{name: "half a cell rounds up", count: 1, total: 40, expected: 1},
		{name: "just under half a cell", count: 12, total: 1000, expected: 1},
		{name: "76.47 %", count: 78, total: 102, expected: 15},
		{name: "19.6 of 20", count: 98, total: 100, expected: 19},
		{name: "all", count: 102, total: 102, expected: 20},
	}

	for _, tt := range tests {
		if got := barLength(tt.count, tt.total); got != tt.expected {
			t.Errorf("%s: barLength(%d, %d) = %d, want %d", tt.name, tt.count, tt.total, got, tt.expected)
		}
	}
}

// TestAnsiTerminal holds colour to a character device whose TERM says it
// understands escape sequences.
func TestAnsiTerminal(t *testing.T) {
	device, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}

	defer func() { _ = device.Close() }()

	file, err := os.Create(filepath.Join(t.TempDir(), "out"))
	if err != nil {
		t.Fatalf("create a file: %v", err)
	}

	defer func() { _ = file.Close() }()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	defer func() { _ = reader.Close(); _ = writer.Close() }()

	// Nothing here is a terminal, whatever TERM says. /dev/null is the case
	// worth naming: it is a character device, which is what this used to take
	// for a terminal, so `galaxio report gatling big 2>/dev/null` redrew a block
	// five times a second into it.
	for _, tt := range []struct {
		name string
		w    io.Writer
	}{
		{name: "a device that is no terminal", w: device},
		{name: "a file", w: file},
		{name: "a pipe", w: writer},
		{name: "a buffer", w: &bytes.Buffer{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, term := range []string{"xterm-256color", "dumb", ""} {
				t.Setenv("TERM", term)

				if ansiTerminal(tt.w) {
					t.Errorf("%s is an ANSI terminal at TERM=%q", tt.name, term)
				}
			}
		})
	}

	t.Run("a terminal", func(t *testing.T) {
		tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
		if err != nil {
			t.Skipf("no controlling terminal to test against: %v", err)
		}

		defer func() { _ = tty.Close() }()

		for term, expected := range map[string]bool{"xterm-256color": true, "dumb": false, "": false} {
			t.Setenv("TERM", term)

			if got := ansiTerminal(tty); got != expected {
				t.Errorf("a terminal at TERM=%q is %v, want %v", term, got, expected)
			}
		}
	})
}

func TestIsNoColor(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		env      string
		expected bool
	}{
		{name: "by default", args: []string{"probe"}, expected: false},
		{name: "--no-color", args: []string{"--no-color", "probe"}, expected: true},
		{name: "NO_COLOR", args: []string{"probe"}, env: "1", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", tt.env)

			var got bool

			root := newRootCommand()
			root.AddCommand(&cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, _ []string) error {
				got = isNoColor(cmd)

				return nil
			}})
			root.SetArgs(tt.args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)

			if err := root.ExecuteContext(context.Background()); err != nil {
				t.Fatalf("execute: %v", err)
			}

			if got != tt.expected {
				t.Errorf("isNoColor = %v, want %v", got, tt.expected)
			}
		})
	}
}
