package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
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
	"github.com/galax-io/parsec/gatling"
	"github.com/galax-io/parsec/gatling/simlog"
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

	if _, err := runReport(context.Background(), reportOptions{Tool: "gatling", Path: dir, outputModes: outputModes{Color: color}, Stdout: &stdout, Stderr: &stderr}); err != nil {
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
// counts and their labels.
type summaryFigures struct {
	headline []string
	columns  []string
	rows     map[string][]string
	bands    [][2]string
	labels   []string
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
		f.labels = append(f.labels, strings.Join(fields[3:], " "))
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
		if !maps.Equal(before[i], after[i]) {
			t.Errorf("%s changed:\nbefore %v\nafter  %v", dir, before[i], after[i])
		}
	}
}

// copyTree copies every file under src into dst.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()

	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
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

// summarise builds a summary from hand-made items at the default options.
func summarise(t *testing.T, items ...model.Item) report.Summary {
	t.Helper()

	s, err := report.Scan(context.Background(), reporttest.Items(model.Run{}, nil, items...), report.DefaultOptions(), nil)
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

// TestOutputModes holds what an invocation may draw to every combination of its
// two streams and its flags. Nothing else holds these: with no terminal in a
// test every mode is off, so drawing the block from standard output's terminal,
// or losing --no-color, was invisible.
func TestOutputModes(t *testing.T) {
	t.Parallel()

	both := streams{color: true, loud: true, stdout: true, stderr: true}

	tests := []struct {
		name     string
		streams  streams
		expected outputModes
	}{
		{name: "two terminals", streams: both, expected: outputModes{Color: true, Block: true, BlockColor: true}},
		{name: "no colour", streams: streams{loud: true, stdout: true, stderr: true}, expected: outputModes{Block: true}},
		{name: "quiet", streams: streams{color: true, stdout: true, stderr: true}, expected: outputModes{Color: true, BlockColor: true}},
		{name: "standard output redirected", streams: streams{color: true, loud: true, stderr: true}, expected: outputModes{Block: true, BlockColor: true}},
		{name: "standard error redirected", streams: streams{color: true, loud: true, stdout: true}, expected: outputModes{Color: true}},
		{name: "neither is a terminal", streams: streams{color: true, loud: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.streams.modes(); got != tt.expected {
				t.Errorf("modes = %+v, want %+v", got, tt.expected)
			}
		})
	}

	if both.modes() == (streams{color: true, loud: true, stdout: true}.modes()) {
		t.Errorf("the block is drawn from standard output's terminal, not standard error's")
	}
}

// TestModesFor holds the wiring: which stream and which flag each mode is read
// from. A test has no terminal, so every mode is off; TestOutputModes holds the
// rule itself.
func TestModesFor(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		env      string
		expected outputModes
	}{
		{name: "no terminal", args: []string{"probe"}},
		{name: "--no-color", args: []string{"--no-color", "probe"}},
		{name: "NO_COLOR", args: []string{"probe"}, env: "1"},
		{name: "--quiet", args: []string{"--quiet", "probe"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", tt.env)

			var got outputModes

			root := newRootCommand()
			root.AddCommand(&cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, _ []string) error {
				got = modesFor(cmd)

				return nil
			}})
			root.SetArgs(tt.args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)

			if err := root.ExecuteContext(context.Background()); err != nil {
				t.Fatalf("execute: %v", err)
			}

			if got != tt.expected {
				t.Errorf("modesFor = %+v, want %+v", got, tt.expected)
			}
		})
	}

	// Each mode reads the stream and the flags it belongs to. Every case here
	// would be reported as a terminal by a check that asked for a character
	// device, which is what /dev/null is.
	device, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}

	defer func() { _ = device.Close() }()

	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_COLOR", "")

	root := newRootCommand()
	root.AddCommand(&cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, _ []string) error {
		if modes := modesFor(cmd); modes != (outputModes{}) {
			t.Errorf("modesFor over %s = %+v, want nothing drawn", os.DevNull, modes)
		}

		return nil
	}})
	root.SetArgs([]string{"probe"})
	root.SetOut(device)
	root.SetErr(device)

	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute: %v", err)
	}
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

// TestReportPercentilesFlag holds --percentiles to its contract: the ranks named,
// once each and ascending, and a value that is not a list of ranks refused while
// the flag is parsed, before help and before any work.
func TestReportPercentilesFlag(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(reportCorpus, "3.13.1")

	tests := []struct {
		name     string
		value    string
		expected []string
	}{
		{name: "the default", value: "", expected: []string{"min", "mean", "std", "p50", "p75", "p95", "p99", "max"}},
		{name: "ranks out of order", value: "99.9,90", expected: []string{"min", "mean", "std", "p90", "p99.9", "max"}},
		{name: "a rank named twice", value: "50,50,99", expected: []string{"min", "mean", "std", "p50", "p99", "max"}},
		{name: "a rank with a trailing zero", value: "99.90", expected: []string{"min", "mean", "std", "p99.9", "max"}},
		{name: "spaces around ranks", value: " 25 , 75 ", expected: []string{"min", "mean", "std", "p25", "p75", "max"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			args := []string{"report", "gatling", dir}
			if tt.value != "" {
				args = append(args, "--percentiles", tt.value)
			}

			code, stdout, stderr := runCLI(args...)
			if code != exitOK {
				t.Fatalf("exit code %d; stderr: %s", code, stderr)
			}

			if f := parseSummary(t, summaryOf(t, stdout)); !slices.Equal(f.columns, tt.expected) {
				t.Errorf("columns = %v, want %v", f.columns, tt.expected)
			}
		})
	}

	t.Run("the percentiles equal what the default ranks print", func(t *testing.T) {
		t.Parallel()

		_, named, _ := runCLI("report", "gatling", dir, "--percentiles", "95,50")
		_, defaults, _ := runCLI("report", "gatling", dir)

		f, d := parseSummary(t, summaryOf(t, named)), parseSummary(t, summaryOf(t, defaults))
		for _, row := range []string{"all", "✓ ok", "✗ failed"} {
			for _, column := range []string{"p50", "p95"} {
				if f.cell(t, row, column) != d.cell(t, row, column) {
					t.Errorf("%s %s = %s with --percentiles, %s by default", row, column, f.cell(t, row, column), d.cell(t, row, column))
				}
			}
		}
	})

	for _, value := range []string{"0", "101", "abc", "50,abc", "", "50,", "NaN", "-5"} {
		for _, help := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q refused, help %v", value, help), func(t *testing.T) {
				t.Parallel()

				args := []string{"report", "gatling", dir, "--percentiles", value}
				if help {
					args = append(args, "--help")
				}

				code, stdout, stderr := runCLI(args...)
				if code != exitUsage || stdout != "" {
					t.Fatalf("exit code %d, stdout %q; want %d and nothing", code, stdout, exitUsage)
				}

				if want := fmt.Sprintf("invalid argument %q for \"--percentiles\" flag: percentile rank ", value); !strings.Contains(stderr, want) || !strings.Contains(stderr, "is not a number above 0 and at most 100") {
					t.Errorf("stderr %q, want it to quote the value", stderr)
				}
			})
		}
	}

	t.Run("help names the default", func(t *testing.T) {
		t.Parallel()

		if _, stdout, _ := runCLI("report", "--help"); !strings.Contains(stdout, "--percentiles ranks") || !strings.Contains(stdout, "(default 50,75,95,99)") {
			t.Errorf("help does not name the flag and its default:\n%s", stdout)
		}
	})
}

// TestReportBoundsFlag holds --bounds to its contract: the bands counted at the
// boundaries named, against the log's own response times, and a value that is
// not two boundaries refused while the flag is parsed.
func TestReportBoundsFlag(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(reportCorpus, "3.13.1")

	t.Run("5,1000", func(t *testing.T) {
		t.Parallel()

		code, stdout, stderr := runCLI("report", "gatling", dir, "--bounds", "5,1000")
		if code != exitOK {
			t.Fatalf("exit code %d; stderr: %s", code, stderr)
		}

		f := parseSummary(t, summaryOf(t, stdout))

		if expected := []string{"ok under 5 ms", "ok 5 to 1000 ms", "ok 1000 ms and over", "failed"}; !slices.Equal(f.labels, expected) {
			t.Errorf("labels = %q, want %q", f.labels, expected)
		}

		durations, err := reporttest.Durations(openRun(t, dir))
		if err != nil {
			t.Fatalf("Durations: %v", err)
		}

		var counted [3]int

		for _, ms := range durations[1] {
			switch {
			case ms < 5:
				counted[0]++
			case ms < 1000:
				counted[1]++
			default:
				counted[2]++
			}
		}

		sum := 0

		for i, band := range f.bands {
			n, err := strconv.Atoi(band[1])
			if err != nil {
				t.Fatalf("band %d count %q: %v", i+1, band[1], err)
			}

			sum += n

			if i < 3 && n != counted[i] {
				t.Errorf("%s = %d, the log holds %d", f.labels[i], n, counted[i])
			}
		}

		if f.bands[3] != [2]string{"17.65", "18"} || sum != 102 {
			t.Errorf("failed band %v and a total of %d, want 18 (17.65 %%) and 102", f.bands[3], sum)
		}
	})

	t.Run("both flags", func(t *testing.T) {
		t.Parallel()

		code, stdout, stderr := runCLI("report", "gatling", dir, "--bounds", "5,1000", "--percentiles", "90")
		if code != exitOK {
			t.Fatalf("exit code %d; stderr: %s", code, stderr)
		}

		f := parseSummary(t, summaryOf(t, stdout))
		if !slices.Contains(f.columns, "p90") || len(f.columns) != 5 || f.labels[0] != "ok under 5 ms" {
			t.Errorf("columns %v and labels %q, want p90 alone and the bands at 5 and 1000", f.columns, f.labels)
		}
	})

	t.Run("the default", func(t *testing.T) {
		t.Parallel()

		_, stdout, _ := runCLI("report", "gatling", dir)
		if f := parseSummary(t, summaryOf(t, stdout)); !slices.Equal(f.labels, []string{"ok under 800 ms", "ok 800 to 1200 ms", "ok 1200 ms and over", "failed"}) {
			t.Errorf("labels = %q, want Gatling's boundaries", f.labels)
		}

		if _, help, _ := runCLI("report", "--help"); !strings.Contains(help, "--bounds low,high") || !strings.Contains(help, "(default 800,1200)") {
			t.Errorf("help does not name the flag and its default:\n%s", help)
		}
	})

	for _, value := range []string{"1200,800", "800,800", "-1,5", "800", "1,2,3", "a,b", "1.5,2", ""} {
		t.Run(fmt.Sprintf("%q refused", value), func(t *testing.T) {
			t.Parallel()

			code, stdout, stderr := runCLI("report", "gatling", dir, "--bounds", value)
			if code != exitUsage || stdout != "" {
				t.Fatalf("exit code %d, stdout %q; want %d and nothing", code, stdout, exitUsage)
			}

			if want := fmt.Sprintf("invalid argument %q for \"--bounds\" flag: boundaries are two whole, non-negative numbers of milliseconds, the second greater than the first", value); !strings.Contains(stderr, want) {
				t.Errorf("stderr %q, want %q", stderr, want)
			}
		})
	}
}

// openRun opens the simulation.log of a run directory as a run reader.
func openRun(t *testing.T, dir string) simlog.RunReader {
	t.Helper()

	f, err := os.Open(filepath.Join(dir, "simulation.log"))
	if err != nil {
		t.Fatalf("open the log: %v", err)
	}

	t.Cleanup(func() { _ = f.Close() })

	rd, err := simlog.NewRunReader(f)
	if err != nil {
		t.Fatalf("NewRunReader: %v", err)
	}

	return rd
}

// untimedRun is a text log of two requests, the second of which has no recorded
// end, which Gatling writes as an end of 0.
var untimedRun = []byte("RUN\tio.galaxio.Sim\tsim\t1788379676999\t \t3.12.0\n" +
	"REQUEST\t\tGET /ok\t1788379677000\t1788379677010\tOK\t \n" +
	"REQUEST\t\tGET /lost\t1788379678000\t0\tOK\t \n")

// TestSummaryFailures holds the failures of a summary that is not complete to
// the contract's words, alone and together, and to nothing for a run that is
// complete or merely cannot be placed in time.
func TestSummaryFailures(t *testing.T) {
	t.Parallel()

	const log = "/runs/x/simulation.log"

	lost := func(at int64) model.Item { return request(at, 10, model.OutcomeUnknown) }

	zeroSpan := "/runs/x/simulation.log: the run spans no time (2023-11-14T22:13:20.000Z .. 2023-11-14T22:13:20.000Z): no request rate can be computed"

	tests := []struct {
		name     string
		items    []model.Item
		expected []string
	}{
		{name: "a complete run", items: []model.Item{request(1_700_000_000_000, 10, model.OutcomeSuccess), request(1_700_000_002_000, 10, model.OutcomeFailure)}},
		{name: "a run no bound can place", items: []model.Item{request(-1, 10, model.OutcomeSuccess)}},
		{name: "a run of one instant that holds no request", items: []model.Item{
			{Kind: model.ItemUser, User: model.UserEvent{Scenario: "s", Kind: model.UserStart, At: time.UnixMilli(1_700_000_000_000).UTC()}},
			{Kind: model.ItemUser, User: model.UserEvent{Scenario: "s", Kind: model.UserEnd, At: time.UnixMilli(1_700_000_000_000).UTC()}},
		}},
		{name: "a span of zero", items: []model.Item{request(1_700_000_000_000, 0, model.OutcomeSuccess)}, expected: []string{zeroSpan}},
		{name: "one lost outcome", items: []model.Item{request(1_700_000_000_000, 10, model.OutcomeSuccess), lost(1_700_000_001_000)}, expected: []string{
			"/runs/x/simulation.log: 1 request has an outcome the source lost: it is neither ok nor failed and the summary does not add up",
		}},
		{name: "several lost outcomes", items: []model.Item{request(1_700_000_000_000, 10, model.OutcomeSuccess), lost(1_700_000_001_000), lost(1_700_000_002_000)}, expected: []string{
			"/runs/x/simulation.log: 2 requests have an outcome the source lost: they are neither ok nor failed and the summary does not add up",
		}},
		{name: "both", items: []model.Item{request(1_700_000_000_000, 0, model.OutcomeSuccess), request(1_700_000_000_000, 0, model.OutcomeUnknown)}, expected: []string{
			zeroSpan,
			"/runs/x/simulation.log: 1 request has an outcome the source lost: it is neither ok nor failed and the summary does not add up",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := summaryFailures(summarise(t, tt.items...), log)

			if len(tt.expected) == 0 {
				if err != nil {
					t.Errorf("summaryFailures = %v, want none", err)
				}

				return
			}

			if err == nil || err.Error() != strings.Join(tt.expected, "\n") {
				t.Errorf("summaryFailures = %v, want\n%s", err, strings.Join(tt.expected, "\n"))
			}
		})
	}

	if got := untimedWarning(1); got != "1 request has no recorded end and takes part in no timing figure" {
		t.Errorf("untimedWarning(1) = %q", got)
	}

	if got := untimedWarning(3); got != "3 requests have no recorded end and take part in no timing figure" {
		t.Errorf("untimedWarning(3) = %q", got)
	}
}

// TestReportFailsOnAnIncompleteSummary holds the command to printing the summary
// of a run that cannot be summarised completely and then exiting 1, naming why.
func TestReportFailsOnAnIncompleteSummary(t *testing.T) {
	t.Parallel()

	t.Run("a run that spans no time", func(t *testing.T) {
		t.Parallel()

		dir := filepath.Join("testdata", "report", "zero-span")

		code, stdout, stderr := runCLI("report", "gatling", dir)
		if code != exitRuntime {
			t.Fatalf("exit code %d, want %d; stderr: %s", code, exitRuntime, stderr)
		}

		f := parseSummary(t, summaryOf(t, stdout))
		if expected := []string{"1 request · - req/s", "✓ 1 ok · 100 % · -/s", "✗ 0 failed · 0 % · -/s"}; !slices.Equal(f.headline, expected) {
			t.Errorf("headline = %q, want every rate -", f.headline)
		}

		want := filepath.Join(dir, "simulation.log") + ": the run spans no time (2026-09-02T20:07:57.529Z .. 2026-09-02T20:07:57.529Z): no request rate can be computed"
		if !strings.Contains(stderr, "Error: "+want) {
			t.Errorf("stderr %q, want %q", stderr, want)
		}
	})

	t.Run("--quiet keeps the exit code and the warning", func(t *testing.T) {
		t.Parallel()

		// --quiet is how a CI job asks for the exit code alone, so a run that
		// is not complete must still fail under it, and the warning that
		// qualifies the figures must still be written.
		zeroSpan := filepath.Join("testdata", "report", "zero-span")

		code, stdout, stderr := runCLI("report", "gatling", zeroSpan, "--quiet")
		if code != exitRuntime {
			t.Errorf("a run that spans no time exits %d under --quiet, want %d", code, exitRuntime)
		}

		if summary := strings.Index(stdout, "response time, ms"); summary >= 0 {
			t.Errorf("--quiet printed the summary: %q", stdout)
		}

		if !strings.Contains(stderr, "the run spans no time") {
			t.Errorf("stderr %q, want the failure named", stderr)
		}

		untimed := writeRun(t, untimedRun)

		code, _, stderr = runCLI("report", "gatling", untimed, "--quiet")
		if code != exitOK || !strings.Contains(stderr, "no recorded end") {
			t.Errorf("exit %d, stderr %q; want 0 and the warning for a request with no recorded end", code, stderr)
		}
	})

	t.Run("a failed request with no recorded end is counted", func(t *testing.T) {
		t.Parallel()

		// Untimed() adds both outcomes; a failure with no end was left out of
		// the warning by any mutation that forgot the failed term.
		items := []model.Item{
			request(1_700_000_000_000, 10, model.OutcomeSuccess),
			{Kind: model.ItemSample, Sample: model.Sample{Name: "r", Start: time.UnixMilli(1_700_000_001_000).UTC(), Outcome: model.OutcomeFailure}},
		}

		if n := summarise(t, items...).Untimed(); n != 1 {
			t.Errorf("Untimed = %d, want the failed request with no recorded end counted", n)
		}
	})

	t.Run("options no summary can be computed at are the invocation's fault", func(t *testing.T) {
		t.Parallel()

		// The flags refuse both as they are parsed, so this holds the seam the
		// products of #52 will fold over: the refusal is a usage error, it does
		// not name the log, and it arrives before a run is looked for — the path
		// here is one no run is under.
		for _, tt := range []struct {
			name     string
			opts     reportOptions
			expected string
		}{
			{name: "a rank of zero", opts: reportOptions{Percentiles: []float64{0}}, expected: "percentile rank 0"},
			{name: "boundaries that do not increase", opts: reportOptions{Bounds: report.Bands{Lower: 5, Upper: 1}}, expected: "band boundaries 5 and 1"},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				opts := tt.opts
				opts.Tool, opts.Path, opts.PathSet = "gatling", filepath.Join(t.TempDir(), "nowhere"), true
				opts.Stdout, opts.Stderr = io.Discard, io.Discard

				_, err := runReport(context.Background(), opts)

				var usage UsageError
				if !errors.As(err, &usage) || !strings.Contains(err.Error(), tt.expected) {
					t.Errorf("runReport = %v, want a UsageError naming %q", err, tt.expected)
				}

				if strings.Contains(err.Error(), "nowhere") {
					t.Errorf("runReport blamed the path for options it was given: %v", err)
				}
			})
		}
	})

	t.Run("a log cut short prints its summary and keeps the truncation", func(t *testing.T) {
		t.Parallel()

		dir := writeRun(t, corpusLog(t, "3.15.1")[:2000])

		var stdout, stderr bytes.Buffer

		_, err := runReport(context.Background(), reportOptions{Tool: "gatling", Path: dir, Stdout: &stdout, Stderr: &stderr})

		var cutShort *gatling.TruncationError
		if !errors.As(err, &cutShort) {
			t.Fatalf("runReport = %v, want the truncation in the chain", err)
		}

		if parseSummary(t, summaryOf(t, stdout.String())).rows["all"] == nil {
			t.Errorf("no summary of what the log held: %q", stdout.String())
		}
	})

	t.Run("-o is refused with the new flags present", func(t *testing.T) {
		t.Parallel()

		code, stdout, stderr := runCLI("report", "gatling", filepath.Join(reportCorpus, "3.15.1"), "--percentiles", "90", "--bounds", "5,1000", "-o", "stats")
		if code != exitUsage || stdout != "" || !strings.Contains(stderr, `report format "stats" is not available yet`) {
			t.Errorf("exit %d, stdout %q, stderr %q; want exit 2 refusing -o", code, stdout, stderr)
		}
	})
}
