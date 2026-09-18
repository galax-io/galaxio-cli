package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/galax-io/galaxio-cli/internal/report"
	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/gatling"
	"github.com/galax-io/parsec/model"
)

// steppingClock returns a clock that moves on by step every time it is read,
// and calls onRead, when given, with how many times it has been read.
func steppingClock(step time.Duration, onRead func(int)) func() time.Time {
	now := time.UnixMilli(1_700_000_000_000)
	reads := 0

	return func() time.Time {
		reads++
		if onRead != nil {
			onRead(reads)
		}

		now = now.Add(step)

		return now
	}
}

// replayRun writes the 3.12.0 recording repeated into a fresh run directory,
// changed by edit when it is given, and returns the directory.
func replayRun(t testing.TB, repeats int, edit func([]byte) []byte) string {
	t.Helper()

	header, body, err := reporttest.Split(corpusLog(t, "3.12.0"))
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	var log bytes.Buffer
	if _, err := io.Copy(&log, reporttest.Replay(header, body, repeats)); err != nil {
		t.Fatalf("replay: %v", err)
	}

	data := log.Bytes()
	if edit != nil {
		data = edit(data)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "simulation.log"), data, 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	return dir
}

// TestBlockLines holds the six lines of a draw to contracts/cli.md: the first
// three as the contract shows them, and the rows in their columns.
func TestBlockLines(t *testing.T) {
	t.Parallel()

	s := summarise(t,
		request(1_700_000_000_000, 10, model.OutcomeSuccess),
		request(1_700_000_000_100, 20, model.OutcomeSuccess),
		request(1_700_000_000_200, 30, model.OutcomeSuccess),
		request(1_700_000_000_300, 500, model.OutcomeFailure),
	)

	lines := blockLines(2, "simulation.log", 37, 100, 7*time.Second, s, false)

	expected := []string{
		"⠹ reading simulation.log  ━━━━━━━━━━━───────────────────   37 %  0:12 left",
		"figures so far · times in ms · percentiles are t-digest estimates, interpolated",
		"                       count   share    min   mean    p50    p95    p99    max",
	}

	if len(lines) != progressLines {
		t.Fatalf("%d lines, want %d", len(lines), progressLines)
	}

	for i, want := range expected {
		if lines[i] != want {
			t.Errorf("line %d = %q, want %q", i+1, lines[i], want)
		}
	}

	row := func(label string, f report.Figures, share string) string {
		cells := []string{abbreviate(f.Count()), share, figure(f.Min()), figure(f.Mean()), figure(f.Percentile(50)), figure(f.Percentile(95)), figure(f.Percentile(99)), figure(f.Max())}
		widths := []int{progressCountWidth, progressCountWidth, progressFigureWidth, progressFigureWidth, progressFigureWidth, progressFigureWidth, progressFigureWidth, progressFigureWidth}

		text := padRight(label, summaryLabelWidth)
		for i, cell := range cells {
			text += padLeft(cell, widths[i])
		}

		return text
	}

	for i, want := range []string{row("all requests", s.All(), ""), row("  ✓ ok", s.OK, "75.0 %"), row("  ✗ failed", s.Failed, "25.0 %")} {
		if lines[3+i] != want {
			t.Errorf("line %d = %q, want %q", 4+i, lines[3+i], want)
		}

		if n := utf8.RuneCountInString(lines[3+i]); n != 78 {
			t.Errorf("line %d is %d columns, want 78", 4+i, n)
		}
	}

	t.Run("the first draw has no time left", func(t *testing.T) {
		t.Parallel()

		if first := blockLines(0, "simulation.log", 37, 100, 0, s, false)[0]; first != "⠋ reading simulation.log  ━━━━━━━━━━━───────────────────   37 %" {
			t.Errorf("first line = %q", first)
		}
	})

	t.Run("an unknown size", func(t *testing.T) {
		t.Parallel()

		if first := blockLines(3, "simulation.log", 37, 0, time.Second, s, false)[0]; first != "⠸ reading simulation.log" {
			t.Errorf("first line = %q, want the spinner and the name alone", first)
		}
	})

	t.Run("a log that grew", func(t *testing.T) {
		t.Parallel()

		first := blockLines(1, "simulation.log", 150, 100, time.Second, s, false)[0]
		if !strings.Contains(first, strings.Repeat("━", progressBarCells)+"  100 %") || strings.Contains(first, "─") {
			t.Errorf("first line = %q, want a full bar at 100 %%", first)
		}
	})

	t.Run("nothing failed yet", func(t *testing.T) {
		t.Parallel()

		ok := summarise(t, request(1_700_000_000_000, 10, model.OutcomeSuccess))
		if failed := blockLines(1, "simulation.log", 1, 2, time.Second, ok, false)[5]; failed != "  ✗ failed                 0   0.0 %      -      -      -      -      -      -" {
			t.Errorf("failed line = %q", failed)
		}
	})

	t.Run("a long name and a long time are cut at 79 columns", func(t *testing.T) {
		t.Parallel()

		name := cut(strings.Repeat("n", 40)+".log", progressNameWidth)

		first := blockLines(5, name, 1, 100, 10*time.Hour, s, false)[0]
		if n := utf8.RuneCountInString(first); n != progressWidth {
			t.Errorf("first line is %d columns, want %d: %q", n, progressWidth, first)
		}
	})

	t.Run("in colour", func(t *testing.T) {
		t.Parallel()

		coloured := strings.Join(blockLines(2, "simulation.log", 37, 100, 7*time.Second, s, true), "\n")
		for _, want := range []string{"\x1b[2mfigures so far", "\x1b[32m  ✓ ok", "\x1b[31m  ✗ failed", "\x1b[2m───"} {
			if !strings.Contains(coloured, want) {
				t.Errorf("the coloured block lacks %q", want)
			}
		}
	})
}

func TestAbbreviate(t *testing.T) {
	t.Parallel()

	for n, expected := range map[int]string{
		0: "0", 9_999: "9999", 10_000: "10.0k", 12_345: "12.3k", 99_950: "100k", 291_000: "291k",
		999_950: "1.0M", 41_200_000: "41.2M", 999_950_000: "1.0G", 1_000_000_000: "1.0G", 1_234_567_890_123: "1235G",
	} {
		if got := abbreviate(n); got != expected {
			t.Errorf("abbreviate(%d) = %q, want %q", n, got, expected)
		}
	}
}

func TestClockTime(t *testing.T) {
	t.Parallel()

	for d, expected := range map[time.Duration]string{
		0: "0:00", 12 * time.Second: "0:12", 11900 * time.Millisecond: "0:12", 59*time.Minute + 59*time.Second: "59:59",
		time.Hour: "1:00:00", time.Hour + 2*time.Minute + 5*time.Second: "1:02:05",
	} {
		if got := clockTime(d); got != expected {
			t.Errorf("clockTime(%v) = %q, want %q", d, got, expected)
		}
	}
}

// runWithBlock runs the command over dir with standard error able to redraw and
// the clock given, and returns what it wrote and returned.
func runWithBlock(ctx context.Context, dir string, clock func() time.Time, redraw, quiet bool) (stdout, stderr string, err error) {
	var out, errs bytes.Buffer

	_, err = runReport(ctx, reportOptions{Tool: "gatling", Path: dir, Quiet: quiet, outputModes: outputModes{Block: redraw && !quiet}, Clock: clock, Stdout: &out, Stderr: &errs})

	return out.String(), errs.String(), err
}

// TestProgressBlock holds when the block is drawn, how it is redrawn and that it
// is gone before the command writes or returns anything else.
func TestProgressBlock(t *testing.T) {
	t.Parallel()

	const (
		up    = "\x1b[6A"
		erase = "\x1b[6A\x1b[J"
	)

	dir := replayRun(t, 400, nil)

	t.Run("drawn, redrawn and erased", func(t *testing.T) {
		t.Parallel()

		stdout, stderr, err := runWithBlock(context.Background(), dir, steppingClock(150*time.Millisecond, nil), true, false)
		if err != nil {
			t.Fatalf("runReport: %v", err)
		}

		if !strings.HasPrefix(stderr, "\x1b[2K⠋ reading simulation.log  ") {
			t.Errorf("the first draw does not open with the spinner and the name: %q", firstLine(stderr))
		}

		redraws := strings.Count(stderr, up+"\x1b[2K")
		if redraws < 2 || !strings.Contains(stderr, up+"\x1b[2K⠙ reading") {
			t.Errorf("%d redraws, want the block redrawn with the spinner advanced", redraws)
		}

		if !strings.HasSuffix(stderr, erase) || strings.Count(stderr, erase) != 1 {
			t.Errorf("the block is not erased once, at the end: %q", lastBytes(stderr))
		}

		for _, sequence := range []string{"\x1b[?", "\x1b[s", "\x1b[u", "\x1b[H"} {
			if strings.Contains(stderr, sequence) {
				t.Errorf("the block writes %q, which changes the terminal", sequence)
			}
		}

		plain, _, plainErr := runWithBlock(context.Background(), dir, nil, false, false)
		if stdout != plain || plainErr != nil {
			t.Errorf("standard output differs with the block, or the error does: %v", plainErr)
		}
	})

	t.Run("nothing drawn", func(t *testing.T) {
		t.Parallel()

		for name, run := range map[string]func() (string, string, error){
			"a read that ends within 500 ms": func() (string, string, error) {
				return runWithBlock(context.Background(), dir, steppingClock(0, nil), true, false)
			},
			"a standard error that cannot redraw": func() (string, string, error) {
				return runWithBlock(context.Background(), dir, steppingClock(time.Second, nil), false, false)
			},
			"--quiet": func() (string, string, error) {
				return runWithBlock(context.Background(), dir, steppingClock(time.Second, nil), true, true)
			},
		} {
			if _, stderr, err := run(); err != nil || stderr != "" {
				t.Errorf("%s: stderr %q, error %v; want nothing", name, stderr, err)
			}
		}
	})

	t.Run("erased before the warning and the report are written", func(t *testing.T) {
		t.Parallel()

		// Every other case reads a stderr the block is alone on. Here a warning
		// follows it, so the order matters: erased first, or the erase would
		// take six lines of what was written after it off the screen.
		untimed := replayRun(t, 400, func(log []byte) []byte {
			at := bytes.Index(log, []byte("\nREQUEST\t"))
			end := at + 1 + bytes.IndexByte(log[at+1:], '\n')
			fields := bytes.Split(log[at+1:end], []byte("\t"))

			// A request with no recorded end: parsec writes it as an end of 0.
			fields[4] = []byte("0")

			return append(append(append([]byte(nil), log[:at+1]...), bytes.Join(fields, []byte("\t"))...), log[end:]...)
		})

		_, stderr, err := runWithBlock(context.Background(), untimed, steppingClock(150*time.Millisecond, nil), true, false)
		if err != nil {
			t.Fatalf("runReport: %v", err)
		}

		warning := strings.Index(stderr, "report: warning:")
		if warning < 0 || !strings.Contains(stderr, "no recorded end") {
			t.Fatalf("no warning for the request with no recorded end: %q", stderr)
		}

		if erased := strings.Index(stderr, erase); erased < 0 || erased > warning {
			t.Errorf("the block is erased at %d and the warning written at %d; want the block gone first", erased, warning)
		}

		if strings.Contains(stderr[warning:], up) {
			t.Errorf("the block is redrawn after the warning: %q", stderr[warning:])
		}
	})

	t.Run("redrawn no more than five times a second", func(t *testing.T) {
		t.Parallel()

		// The walk ticks every 1024 items, far more often than the block may be
		// drawn; the clock makes each tick 40 ms, so four ticks in five are due
		// nothing.
		_, stderr, err := runWithBlock(context.Background(), dir, steppingClock(40*time.Millisecond, nil), true, false)
		if err != nil {
			t.Fatalf("runReport: %v", err)
		}

		draws := strings.Count(stderr, "\x1b[2K⠋ reading") + strings.Count(stderr, up+"\x1b[2K")
		ticks := strings.Count(stderr, "\x1b[2K") / progressLines

		if draws == 0 || draws != ticks {
			t.Fatalf("%d draws over %d line groups", draws, ticks)
		}

		// The read is replayed from memory, so the walk ticks hundreds of
		// times; at 40 ms a tick the block may be drawn once every five.
		if draws > 40 {
			t.Errorf("%d draws, want the redraw held to one every 200 ms", draws)
		}
	})

	t.Run("erased before the error of a log cut short", func(t *testing.T) {
		t.Parallel()

		cutShort := replayRun(t, 400, func(log []byte) []byte { return log[:len(log)-10] })

		stdout, stderr, err := runWithBlock(context.Background(), cutShort, steppingClock(150*time.Millisecond, nil), true, false)

		var truncated *gatling.TruncationError
		if !errors.As(err, &truncated) || !strings.HasSuffix(stderr, erase) || stdout == "" {
			t.Errorf("error %v, stderr ending %q, %d bytes of report; want the truncation, the block erased and the report", err, lastBytes(stderr), len(stdout))
		}
	})

	t.Run("erased before the error of a damaged log", func(t *testing.T) {
		t.Parallel()

		damaged := replayRun(t, 400, func(log []byte) []byte {
			at := bytes.LastIndex(log, []byte("\nREQUEST\t"))

			return append(append(append([]byte(nil), log[:at]...), []byte("\nBOGUS\tnot a record")...), log[at+len("\nREQUEST\t"):]...)
		})

		stdout, stderr, err := runWithBlock(context.Background(), damaged, steppingClock(150*time.Millisecond, nil), true, false)
		if err == nil || !strings.HasSuffix(stderr, erase) || stdout != "" {
			t.Errorf("error %v, stderr ending %q, stdout %q; want an error, the block erased and no report", err, lastBytes(stderr), stdout)
		}
	})

	t.Run("erased on an interrupt", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		clock := steppingClock(150*time.Millisecond, func(reads int) {
			if reads == 12 {
				cancel()
			}
		})

		_, stderr, err := runWithBlock(ctx, dir, clock, true, false)
		if !errors.Is(err, context.Canceled) || !strings.HasSuffix(stderr, erase) {
			t.Errorf("error %v, stderr ending %q; want the cancellation and the block erased", err, lastBytes(stderr))
		}
	})

	t.Run("standard output and the exit code are the same with and without it", func(t *testing.T) {
		t.Parallel()

		for _, version := range corpusVersions {
			corpus := filepath.Join(reportCorpus, version)

			with, _, withErr := runWithBlock(context.Background(), corpus, steppingClock(time.Second, nil), true, false)
			without, _, withoutErr := runWithBlock(context.Background(), corpus, nil, false, false)

			if with != without || fmt.Sprint(withErr) != fmt.Sprint(withoutErr) {
				t.Errorf("%s: the report or its error differs with the block", version)
			}
		}
	})
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")

	return line
}

func lastBytes(s string) string {
	return s[max(0, len(s)-40):]
}

// BenchmarkRunReport prices the progress block: the same 64 MiB replay read with
// standard error able to redraw and not, for the 5 % of SC-010. With the real
// clock a read that ends within 500 ms draws nothing and pays only for reading
// the clock every 1024 items; every-tick draws at every tick, far more often
// than five times a second, as the cost's upper bound.
func BenchmarkRunReport(b *testing.B) {
	header, body, err := reporttest.Split(corpusLog(b, "3.12.0"))
	if err != nil {
		b.Fatalf("Split: %v", err)
	}

	repeats := (64 << 20) / len(body)
	dir := replayRun(b, repeats, nil)

	for _, variant := range []struct {
		name   string
		redraw bool
		clock  func() func() time.Time
	}{
		{name: "off"},
		{name: "on", redraw: true, clock: func() func() time.Time { return time.Now }},
		{name: "every-tick", redraw: true, clock: func() func() time.Time { return steppingClock(time.Second, nil) }},
	} {
		b.Run("progress="+variant.name, func(b *testing.B) {
			b.SetBytes(int64(len(header)) + int64(repeats)*int64(len(body)))

			for b.Loop() {
				opts := reportOptions{Tool: "gatling", Path: dir, outputModes: outputModes{Block: variant.redraw}, Stdout: io.Discard, Stderr: io.Discard}
				if variant.clock != nil {
					opts.Clock = variant.clock()
				}

				if _, err := runReport(context.Background(), opts); err != nil {
					b.Fatalf("runReport: %v", err)
				}
			}
		})
	}
}
