package report

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/gatling/simlog"
)

// maxPeakHeap is the peak-memory goal from the plan: heap in use must stay
// under it for a log of any size.
const maxPeakHeap = 32 << 20

// sampleHeap reports the largest HeapInuse seen until stop is closed.
func sampleHeap(stop <-chan struct{}) <-chan uint64 {
	peak := make(chan uint64, 1)
	go func() {
		var stats runtime.MemStats
		var max uint64
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			runtime.ReadMemStats(&stats)
			if stats.HeapInuse > max {
				max = stats.HeapInuse
			}
			select {
			case <-stop:
				peak <- max
				return
			case <-ticker.C:
			}
		}
	}()
	return peak
}

// BenchmarkWrite replays the 3.12.0 recording into logs of 64 MiB and 2 GiB
// and writes them to io.Discard: records/s and MB/s are the throughput, and
// peakHeapMiB is the memory goal, which the 2 GiB case must meet for SC-003.
// Run with -benchtime=1x; the 2 GiB case is skipped under -short.
func BenchmarkWrite(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("testdata", "corpus", "gatling", "3.12.0", "simulation.log"))
	if err != nil {
		b.Fatalf("read corpus log: %v", err)
	}
	header, body, err := reporttest.Split(data)
	if err != nil {
		b.Fatalf("Split: %v", err)
	}

	for _, size := range []int64{64 << 20, 2 << 30} {
		repeats := int(size / int64(len(body)))
		logSize := int64(len(header)) + int64(repeats)*int64(len(body))

		b.Run(fmt.Sprintf("size=%dMiB", size>>20), func(b *testing.B) {
			if size >= 1<<30 && testing.Short() {
				b.Skip("the 2 GiB replay takes tens of seconds; run without -short for SC-003")
			}
			b.ReportAllocs()
			b.SetBytes(logSize)

			var peak uint64
			var records int
			for b.Loop() {
				stop := make(chan struct{})
				sampled := sampleHeap(stop)

				rd, err := simlog.NewRunReader(reporttest.Replay(header, body, repeats))
				if err != nil {
					b.Fatalf("NewRunReader: %v", err)
				}
				sum, err := Write(context.Background(), rd, io.Discard)
				if err != nil {
					b.Fatalf("Write: %v", err)
				}

				close(stop)
				if p := <-sampled; p > peak {
					peak = p
				}
				records += sum.Records()
			}

			b.ReportMetric(float64(peak)/(1<<20), "peakHeapMiB")
			b.ReportMetric(float64(records)/b.Elapsed().Seconds(), "records/s")
			if peak > maxPeakHeap {
				b.Fatalf("peak heap in use %d MiB exceeds the %d MiB goal", peak>>20, maxPeakHeap>>20)
			}
		})
	}
}
