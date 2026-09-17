package report

import (
	"context"
	"fmt"
	"testing"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/gatling/simlog"
)

// BenchmarkScan replays the 3.12.0 recording into logs of 64 MiB and 2 GiB and
// reads them, reporting throughput and allocations. The memory goal is not
// measured here: a benchmark is an instrument for comparing, and the goal needs
// a gate CI runs, which is TestScanMemoryDoesNotGrowWithTheLog.
func BenchmarkScan(b *testing.B) {
	header, body := splitCorpus(b, "3.12.0")

	for _, size := range []int64{64 << 20, 2 << 30} {
		repeats := replayCount(b, size, body)
		logSize := int64(len(header)) + int64(repeats)*int64(len(body))

		b.Run(fmt.Sprintf("size=%dMiB", size>>20), func(b *testing.B) {
			if size >= 1<<30 && testing.Short() {
				b.Skip("the 2 GiB replay takes tens of seconds")
			}

			b.ReportAllocs()
			b.SetBytes(logSize)

			var records int

			for b.Loop() {
				rd, err := simlog.NewRunReader(reporttest.Replay(header, body, repeats))
				if err != nil {
					b.Fatalf("NewRunReader: %v", err)
				}

				tally, err := Scan(context.Background(), rd)
				if err != nil {
					b.Fatalf("Scan: %v", err)
				}

				records += tally.Requests + tally.Groups + tally.Users + tally.Errors
			}

			b.ReportMetric(float64(records)/b.Elapsed().Seconds(), "records/s")
		})
	}
}

// replayCount is how many times body must repeat to reach size, refusing a body
// that cannot be repeated at all rather than dividing by its length blind.
func replayCount(tb testing.TB, size int64, body []byte) int {
	tb.Helper()

	if len(body) == 0 {
		tb.Fatalf("the recording has no body to replay")
	}

	return int(size / int64(len(body)))
}

// discardScan reads one replay to the end and returns nothing but its error, so
// that a caller measuring memory holds no record of its own.
func discardScan(tb testing.TB, header, body []byte, repeats int) {
	tb.Helper()

	rd, err := simlog.NewRunReader(reporttest.Replay(header, body, repeats))
	if err != nil {
		tb.Fatalf("NewRunReader: %v", err)
	}

	if _, err := Scan(context.Background(), rd); err != nil {
		tb.Fatalf("Scan: %v", err)
	}
}
