package report

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/gatling"
	"github.com/galax-io/parsec/gatling/simlog"
	"github.com/galax-io/parsec/model"
)

// corpusLog reads one corpus recording, for a test or a benchmark.
func corpusLog(tb testing.TB, version string) []byte {
	tb.Helper()

	data, err := os.ReadFile(filepath.Join(corpusDir, version, "simulation.log"))
	if err != nil {
		tb.Fatalf("read corpus log: %v", err)
	}

	return data
}

// openBytes opens an in-memory log through the same reader Open uses.
func openBytes(t *testing.T, data []byte) simlog.RunReader {
	t.Helper()

	rd, err := simlog.NewRunReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewRunReader: %v", err)
	}

	return rd
}

// TestScanCorpus holds the tally to each recording's own Gatling console
// summary, which PROVENANCE.md records beside the logs.
func TestScanCorpus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version  string
		expected Tally
	}{
		{version: "3.11.5", expected: Tally{Requests: 36, Successes: 18, Failures: 18, Groups: 12, Users: 12, Errors: 6}},
		{version: "3.12.0", expected: Tally{Requests: 36, Successes: 18, Failures: 18, Groups: 12, Users: 12, Errors: 6}},
		{version: "3.13.1", expected: Tally{Requests: 102, Successes: 84, Failures: 18, Groups: 12, Users: 12, Errors: 6}},
		{version: "3.14.9", expected: Tally{Requests: 102, Successes: 84, Failures: 18, Groups: 12, Users: 12, Errors: 6}},
		{version: "3.15.1", expected: Tally{Requests: 102, Successes: 84, Failures: 18, Groups: 12, Users: 12, Errors: 6}},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()

			got, err := Scan(context.Background(), openBytes(t, corpusLog(t, tt.version)))
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			if got.Requests != tt.expected.Requests || got.Successes != tt.expected.Successes || got.Failures != tt.expected.Failures {
				t.Errorf("requests = %d (%d ok, %d ko), want %d (%d ok, %d ko)",
					got.Requests, got.Successes, got.Failures,
					tt.expected.Requests, tt.expected.Successes, tt.expected.Failures)
			}

			if got.Groups != tt.expected.Groups || got.Users != tt.expected.Users || got.Errors != tt.expected.Errors {
				t.Errorf("groups/users/errors = %d/%d/%d, want %d/%d/%d",
					got.Groups, got.Users, got.Errors,
					tt.expected.Groups, tt.expected.Users, tt.expected.Errors)
			}

			if got.Unknown != 0 {
				t.Errorf("Unknown = %d, want 0: an adapter never loses an outcome", got.Unknown)
			}

			if sum := got.Successes + got.Failures + got.Unknown; sum != got.Requests {
				t.Errorf("successes + failures + unknown = %d, want %d", sum, got.Requests)
			}

			start, startOK := got.Bounds.Start()
			end, endOK := got.Bounds.End()

			if !startOK || !endOK {
				t.Fatalf("Bounds report no span for a complete run")
			}

			if !end.After(start) {
				t.Errorf("span ends at %s, which is not after its start %s", end, start)
			}
		})
	}
}

// stubReader yields fixed items over a fixed run description, for the shapes no
// Gatling log produces.
type stubReader struct {
	run   model.Run
	items []model.Item
}

func (s *stubReader) Run() model.Run { return s.run }

func (s *stubReader) Next() (model.Item, error) {
	if len(s.items) == 0 {
		return model.Item{}, io.EOF
	}

	item := s.items[0]
	s.items = s.items[1:]

	return item, nil
}

func TestScanCountsEveryKind(t *testing.T) {
	t.Parallel()

	at := func(ms int64) time.Time { return time.UnixMilli(ms).UTC() }

	rd := &stubReader{items: []model.Item{
		{Kind: model.ItemUser, User: model.UserEvent{Scenario: "s", Kind: model.UserStart, At: at(1000)}},
		{Kind: model.ItemSample, Sample: model.Sample{Name: "ok", Start: at(1100), Duration: model.Some(10 * time.Millisecond), Outcome: model.OutcomeSuccess}},
		{Kind: model.ItemSample, Sample: model.Sample{Name: "ko", Start: at(1200), Duration: model.Some(20 * time.Millisecond), Outcome: model.OutcomeFailure}},
		{Kind: model.ItemSample, Sample: model.Sample{Name: "lost", Start: at(1300), Duration: model.Some(30 * time.Millisecond)}},
		{Kind: model.ItemGroup, Group: model.GroupSample{Groups: []string{"g"}, Start: at(1000), Duration: model.Some(400 * time.Millisecond), Outcome: model.OutcomeSuccess}},
		{Kind: model.ItemError, Error: model.RunError{Message: "boom", At: at(1350)}},
		{Kind: model.ItemAssertion, Assertion: "payload"},
		{Kind: model.ItemUser, User: model.UserEvent{Scenario: "s", Kind: model.UserEnd, At: at(1400)}},
	}}

	got, err := Scan(context.Background(), rd)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	expected := Tally{Requests: 3, Successes: 1, Failures: 1, Unknown: 1, Groups: 1, Users: 2, Errors: 1}
	if got.Requests != expected.Requests || got.Successes != expected.Successes || got.Failures != expected.Failures || got.Unknown != expected.Unknown {
		t.Errorf("requests = %+v, want %+v", got, expected)
	}

	if got.Groups != expected.Groups || got.Users != expected.Users || got.Errors != expected.Errors {
		t.Errorf("groups/users/errors = %d/%d/%d, want %d/%d/%d", got.Groups, got.Users, got.Errors, expected.Groups, expected.Users, expected.Errors)
	}

	start, ok := got.Bounds.Start()
	if !ok || !start.Equal(at(1000)) {
		t.Errorf("Bounds.Start = %v (%v), want %v", start, ok, at(1000))
	}
}

func TestScanRunWithoutRequests(t *testing.T) {
	t.Parallel()

	got, err := Scan(context.Background(), &stubReader{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if (got != Tally{}) {
		t.Errorf("Tally = %+v, want the zero tally", got)
	}

	if _, ok := got.Bounds.Start(); ok {
		t.Errorf("a run with no items must report no span")
	}
}

func TestScanTruncated(t *testing.T) {
	t.Parallel()

	got, err := Scan(context.Background(), openBytes(t, corpusLog(t, "3.15.1")[:2000]))

	var cutShort *gatling.TruncationError
	if !errors.As(err, &cutShort) {
		t.Fatalf("Scan(cut log) = %v, want a wrapped *gatling.TruncationError", err)
	}

	if !strings.Contains(err.Error(), "the run was not read to the end") {
		t.Errorf("error %q does not say the run was not read to the end", err)
	}

	if got.Requests == 0 {
		t.Errorf("the tally of what the log did hold is empty")
	}

	if sum := got.Successes + got.Failures + got.Unknown; sum != got.Requests {
		t.Errorf("successes + failures + unknown = %d, want %d", sum, got.Requests)
	}
}

// splitCorpus splits one recording into its header and body.
func splitCorpus(tb testing.TB, version string) (header, body []byte) {
	tb.Helper()

	header, body, err := reporttest.Split(corpusLog(tb, version))
	if err != nil {
		tb.Fatalf("Split: %v", err)
	}

	return header, body
}

func TestScanDamaged(t *testing.T) {
	t.Parallel()

	header, body := splitCorpus(t, "3.12.0")

	lines := strings.Split(string(body), "\n")
	lines[20] = "BOGUS\tnot a record"
	damaged := append(append([]byte(nil), header...), []byte(strings.Join(lines, "\n"))...)

	got, scanErr := Scan(context.Background(), openBytes(t, damaged))

	var syntaxErr *gatling.SyntaxError
	if !errors.As(scanErr, &syntaxErr) {
		t.Fatalf("Scan(damaged log) = %v, want a wrapped *gatling.SyntaxError", scanErr)
	}

	if !strings.Contains(scanErr.Error(), "the run was not read completely") {
		t.Errorf("error %q does not say the run was not read completely", scanErr)
	}

	if got.Requests+got.Groups+got.Users+got.Errors != 20 {
		t.Errorf("walked %+v, want the 20 records before the damage", got)
	}
}

func TestScanCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := Scan(ctx, openBytes(t, corpusLog(t, "3.12.0")))

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Scan(cancelled) = %v, want context.Canceled", err)
	}

	if (got != Tally{}) {
		t.Errorf("Tally = %+v, want nothing walked", got)
	}
}

// maxHeapGoal is the peak-memory goal the plan states: heap in use stays under
// it for a log of any size.
const maxHeapGoal = 32 << 20

// TestScanMemoryDoesNotGrowWithTheLog is the gate the plan's memory goal needs.
// It runs in the ordinary suite, which CI runs, rather than in a benchmark,
// which CI does not; and it measures the difference a scan makes to the heap
// rather than the whole process's, so the number is the read path's own.
func TestScanMemoryDoesNotGrowWithTheLog(t *testing.T) {
	if testing.Short() {
		t.Skip("replays a quarter of a gigabyte")
	}

	header, body := splitCorpus(t, "3.12.0")

	heapFor := func(size int64) uint64 {
		repeats := replayCount(t, size, body)

		runtime.GC()

		var before, after runtime.MemStats

		runtime.ReadMemStats(&before)
		discardScan(t, header, body, repeats)
		runtime.ReadMemStats(&after)

		if after.HeapInuse < before.HeapInuse {
			return 0
		}

		return after.HeapInuse - before.HeapInuse
	}

	const (
		small = 16 << 20
		large = 256 << 20
	)

	smallHeap, largeHeap := heapFor(small), heapFor(large)
	t.Logf("heap in use: %.2f MiB over %d MiB, %.2f MiB over %d MiB",
		float64(smallHeap)/(1<<20), small>>20, float64(largeHeap)/(1<<20), large>>20)

	for _, m := range []struct {
		name string
		heap uint64
	}{{"16 MiB log", smallHeap}, {"256 MiB log", largeHeap}} {
		if m.heap > maxHeapGoal {
			t.Errorf("%s: heap in use grew by %.1f MiB, over the %d MiB goal", m.name, float64(m.heap)/(1<<20), maxHeapGoal>>20)
		}
	}

	// Sixteen times the log must not cost meaningfully more heap. The slack
	// covers the allocator's own noise and the race detector's overhead; a read
	// path that retained records would blow past it by orders of magnitude.
	const slack = 8 << 20
	if largeHeap > smallHeap+slack {
		t.Errorf("heap grew with the log: %.1f MiB for 16 MiB against %.1f MiB for 256 MiB", float64(smallHeap)/(1<<20), float64(largeHeap)/(1<<20))
	}
}
