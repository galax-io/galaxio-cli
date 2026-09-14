package report

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/gatling"
	"github.com/galax-io/parsec/gatling/simlog"
	"github.com/galax-io/parsec/model"
)

var update = flag.Bool("update", false, "rewrite the golden files under testdata/golden from the current output")

// openBytes opens an in-memory log through the same reader Open uses.
func openBytes(t *testing.T, data []byte) simlog.RunReader {
	t.Helper()

	rd, err := simlog.NewRunReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewRunReader: %v", err)
	}
	return rd
}

// splitCorpus splits the 3.12.0 corpus log into its header and body bytes.
func splitCorpus(t *testing.T) (header, body []byte) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(corpusDir, "3.12.0", "simulation.log"))
	if err != nil {
		t.Fatalf("read corpus log: %v", err)
	}
	header, body, err = reporttest.Split(data)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	return header, body
}

// textLog is the 3.12.0 corpus log as lines: the header through the RUN
// record, and the body, so that a test can build a log with the body it
// needs and a header parsec accepts.
func textLog(t *testing.T) (header, body []string) {
	t.Helper()

	h, b := splitCorpus(t)
	return strings.Split(strings.TrimSuffix(string(h), "\n"), "\n"), strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

func joinLines(lines []string) []byte {
	return []byte(strings.Join(lines, "\n") + "\n")
}

// repeatLog is the 3.12.0 log with its body repeated n times: a larger log
// with the same grammar and no file on disk.
func repeatLog(t *testing.T, n int) []byte {
	t.Helper()

	header, body := splitCorpus(t)
	data, err := io.ReadAll(reporttest.Replay(header, body, n))
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	return data
}

// parseLines checks that every line of out is a standalone JSON object with
// a kind, and returns the kinds in order.
func parseLines(t *testing.T, out []byte) []string {
	t.Helper()

	if len(out) == 0 || out[len(out)-1] != '\n' {
		t.Fatalf("output does not end with a newline")
	}
	lines := bytes.Split(bytes.TrimSuffix(out, []byte("\n")), []byte("\n"))
	kinds := make([]string, 0, len(lines))
	for i, line := range lines {
		var rec struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(line, &rec); err != nil {
			t.Fatalf("line %d is not a JSON object: %v: %s", i+1, err, line)
		}
		if rec.Kind == "" {
			t.Fatalf("line %d carries no kind: %s", i+1, line)
		}
		kinds = append(kinds, rec.Kind)
	}
	return kinds
}

func countKinds(kinds []string) map[string]int {
	counts := map[string]int{}
	for _, k := range kinds {
		counts[k]++
	}
	return counts
}

func TestWriteGolden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version  string
		requests int
	}{
		{version: "3.11.5", requests: 36},
		{version: "3.12.0", requests: 36},
		{version: "3.13.1", requests: 102},
		{version: "3.14.9", requests: 102},
		{version: "3.15.1", requests: 102},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()

			loc, err := Locate(filepath.Join(corpusDir, tt.version))
			if err != nil {
				t.Fatalf("Locate: %v", err)
			}
			src, err := Open(loc)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			defer src.Close()

			var out bytes.Buffer
			sum, err := Write(context.Background(), src.Reader, &out)
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
			expected := Summary{Requests: tt.requests, Groups: 12, Users: 12, Errors: 6}
			if sum != expected {
				t.Errorf("Summary = %+v, want %+v", sum, expected)
			}

			kinds := parseLines(t, out.Bytes())
			if kinds[0] != string(KindRun) {
				t.Errorf("first line kind = %q, want %q", kinds[0], KindRun)
			}
			counts := countKinds(kinds)
			if counts["request"] != tt.requests || counts["group"] != 12 || counts["user"] != 12 || counts["error"] != 6 || counts["run"] != 1 {
				t.Errorf("record counts = %v, want %d requests, 12 groups, 12 users, 6 errors, 1 run", counts, tt.requests)
			}

			golden := filepath.Join("testdata", "golden", tt.version+".jsonl")
			if *update {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				if err := os.WriteFile(golden, out.Bytes(), 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden (run with -update to create it): %v", err)
			}
			if !bytes.Equal(out.Bytes(), want) {
				t.Errorf("output differs from %s; diff the two or rerun with -update after reading the change", golden)
			}
		})
	}
}

func TestWriteIsDeterministic(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join(corpusDir, "3.15.1", "simulation.log"))
	if err != nil {
		t.Fatalf("read corpus log: %v", err)
	}
	var first, second bytes.Buffer
	if _, err := Write(context.Background(), openBytes(t, data), &first); err != nil {
		t.Fatalf("first Write: %v", err)
	}
	if _, err := Write(context.Background(), openBytes(t, data), &second); err != nil {
		t.Fatalf("second Write: %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatalf("two writes of the same log differ")
	}
}

func TestWriteTruncated(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join(corpusDir, "3.15.1", "simulation.log"))
	if err != nil {
		t.Fatalf("read corpus log: %v", err)
	}
	var out bytes.Buffer
	sum, err := Write(context.Background(), openBytes(t, data[:2000]), &out)

	var truncated *TruncatedError
	if !errors.As(err, &truncated) {
		t.Fatalf("Write(cut log) = %v, want *TruncatedError", err)
	}
	var parsecErr *gatling.TruncationError
	if !errors.As(err, &parsecErr) {
		t.Fatalf("error %v does not wrap parsec's truncation", err)
	}
	if sum.Truncated == nil {
		t.Errorf("Summary.Truncated is nil")
	}
	kinds := parseLines(t, out.Bytes())
	if written := len(kinds) - 1; written != sum.Records() || truncated.Written != written {
		t.Errorf("written %d lines after the header, Summary says %d, error says %d", written, sum.Records(), truncated.Written)
	}
	if sum.Records() == 0 {
		t.Errorf("no record was written before the cut")
	}
	for _, want := range []string{"cut short", "already written are what the run recorded"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not say %q", err, want)
		}
	}
}

func TestWriteIncomplete(t *testing.T) {
	t.Parallel()

	header, body := textLog(t)
	body[20] = "BOGUS\tnot a record"
	var out bytes.Buffer
	sum, err := Write(context.Background(), openBytes(t, joinLines(append(header, body...))), &out)

	var incomplete *IncompleteError
	if !errors.As(err, &incomplete) {
		t.Fatalf("Write(damaged log) = %v, want *IncompleteError", err)
	}
	var syntaxErr *gatling.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("error %v does not wrap parsec's syntax error", err)
	}
	if sum.Truncated != nil {
		t.Errorf("a damaged log must not be reported as truncated")
	}
	kinds := parseLines(t, out.Bytes())
	if written := len(kinds) - 1; written != 20 || incomplete.Written != 20 {
		t.Errorf("written %d records before the damage, error says %d, want 20", written, incomplete.Written)
	}
	if !strings.Contains(err.Error(), "do not form a complete run") {
		t.Errorf("error %q does not say the records are not a complete run", err)
	}
}

func TestWriteRunWithoutRequests(t *testing.T) {
	t.Parallel()

	header, body := textLog(t)
	var lines []string
	users, errs := 0, 0
	for _, line := range body {
		switch {
		case strings.HasPrefix(line, "USER\t") && users < 2:
			lines = append(lines, line)
			users++
		case strings.HasPrefix(line, "ERROR\t") && errs < 1:
			lines = append(lines, line)
			errs++
		}
	}
	var out bytes.Buffer
	sum, err := Write(context.Background(), openBytes(t, joinLines(append(header, lines...))), &out)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if expected := (Summary{Users: 2, Errors: 1}); sum != expected {
		t.Errorf("Summary = %+v, want %+v", sum, expected)
	}
	if kinds := parseLines(t, out.Bytes()); len(kinds) != 4 || kinds[0] != "run" {
		t.Errorf("kinds = %v, want run then three records", kinds)
	}
}

// failingWriter fails on its failAt-th Write and records how many it saw.
type failingWriter struct {
	failAt int
	calls  int
}

var errWriterClosed = errors.New("reader went away")

func (w *failingWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls >= w.failAt {
		return 0, errWriterClosed
	}
	return len(p), nil
}

func TestWriteStopsAtFirstWriteFailure(t *testing.T) {
	t.Parallel()

	w := &failingWriter{failAt: 3}
	_, err := Write(context.Background(), openBytes(t, repeatLog(t, 40)), w)

	var writeErr *WriteError
	if !errors.As(err, &writeErr) {
		t.Fatalf("Write(failing writer) = %v, want *WriteError", err)
	}
	if !errors.Is(err, errWriterClosed) {
		t.Errorf("error %v does not wrap the writer's error", err)
	}
	if w.calls != 3 {
		t.Errorf("writer saw %d writes after failing on the third, want exactly 3", w.calls)
	}
}

func TestWriteCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	sum, err := Write(ctx, openBytes(t, repeatLog(t, 1)), &out)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Write(cancelled) = %v, want context.Canceled", err)
	}
	if sum.Records() != 0 {
		t.Errorf("wrote %d records after cancellation", sum.Records())
	}
	if kinds := parseLines(t, out.Bytes()); len(kinds) != 1 || kinds[0] != "run" {
		t.Errorf("kinds = %v, want the flushed header only", kinds)
	}
}

// stubReader yields fixed items. It stands in for what no Gatling log can
// produce: an assertion among the events, and an item of unknown kind.
type stubReader struct {
	items []model.Item
}

func (s *stubReader) Run() model.Run {
	return model.Run{ID: "stub", Name: "stub", Tool: "stub", ToolVersion: "0", Start: time.UnixMilli(1700000000000).UTC()}
}

func (s *stubReader) Next() (model.Item, error) {
	if len(s.items) == 0 {
		return model.Item{}, io.EOF
	}
	item := s.items[0]
	s.items = s.items[1:]
	return item, nil
}

func TestWriteAssertionAmongEvents(t *testing.T) {
	t.Parallel()

	rd := &stubReader{items: []model.Item{
		{Kind: model.ItemUser, User: model.UserEvent{Scenario: "s", Kind: model.UserStart}},
		{Kind: model.ItemAssertion, Assertion: "\x00\x01"},
	}}
	var out bytes.Buffer
	sum, err := Write(context.Background(), rd, &out)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if sum.Assertions != 1 || sum.Users != 1 {
		t.Errorf("Summary = %+v, want one user event and one assertion", sum)
	}
	if !strings.Contains(out.String(), `{"kind":"assertion","payload":"AAE="}`) {
		t.Errorf("output lacks the assertion record:\n%s", out.String())
	}
}

func TestWriteUnknownItemKind(t *testing.T) {
	t.Parallel()

	rd := &stubReader{items: []model.Item{
		{Kind: model.ItemUser, User: model.UserEvent{Scenario: "s", Kind: model.UserStart}},
		{Kind: model.ItemKind(200)},
	}}
	var out bytes.Buffer
	_, err := Write(context.Background(), rd, &out)

	var incomplete *IncompleteError
	if !errors.As(err, &incomplete) {
		t.Fatalf("Write(unknown kind) = %v, want *IncompleteError", err)
	}
	if incomplete.Written != 1 {
		t.Errorf("Written = %d, want 1", incomplete.Written)
	}
	if kinds := parseLines(t, out.Bytes()); len(kinds) != 2 {
		t.Errorf("kinds = %v, want the header and one record flushed", kinds)
	}
}

// maxAllocsPerRecord is the per-record allocation goal from the plan. The
// count includes parsec's own reader, so it bounds the whole path a record
// takes, not only this package's share.
const maxAllocsPerRecord = 4

func TestWriteAllocations(t *testing.T) {
	header, body := splitCorpus(t)
	const repeats = 200

	var records int
	allocs := testing.AllocsPerRun(1, func() {
		rd, err := simlog.NewRunReader(reporttest.Replay(header, body, repeats))
		if err != nil {
			t.Fatalf("NewRunReader: %v", err)
		}
		sum, err := Write(context.Background(), rd, io.Discard)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
		records = sum.Records()
	})

	perRecord := allocs / float64(records)
	t.Logf("%.0f allocations over %d records: %.2f per record", allocs, records, perRecord)
	if perRecord > maxAllocsPerRecord {
		t.Fatalf("%.2f allocations per record exceeds the goal of %d", perRecord, maxAllocsPerRecord)
	}
}
