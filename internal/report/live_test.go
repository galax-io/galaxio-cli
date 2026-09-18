package report

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/model"
)

// liveDir holds the recordings of the live Gatling runs: GALAXIO_LIVE_RECORDINGS
// names a set just recorded, and the committed set is read otherwise.
func liveDir() string {
	if dir := os.Getenv("GALAXIO_LIVE_RECORDINGS"); dir != "" {
		return dir
	}

	return filepath.Join("testdata", "live", "gatling")
}

// liveVersions lists the Gatling versions a recording exists for.
func liveVersions(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(liveDir())
	if err != nil {
		t.Fatalf("read the live recordings: %v", err)
	}

	var versions []string

	for _, entry := range entries {
		if entry.IsDir() {
			versions = append(versions, entry.Name())
		}
	}

	if len(versions) == 0 {
		t.Fatalf("no live recording under %s", liveDir())
	}

	return versions
}

// liveLog reads one recording's simulation.log, kept compressed.
func liveLog(t *testing.T, version string) []byte {
	t.Helper()

	f, err := os.Open(filepath.Join(liveDir(), version, "simulation.log.gz"))
	if err != nil {
		t.Fatalf("open the live log: %v", err)
	}

	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("read the live log: %v", err)
	}

	var log bytes.Buffer
	if _, err := io.Copy(&log, gz); err != nil {
		t.Fatalf("decompress the live log: %v", err)
	}

	return log.Bytes()
}

// liveDurations returns a recording's response times for all, successful and
// failed requests, sorted.
func liveDurations(t *testing.T, log []byte) [3][]int64 {
	t.Helper()

	keeps := [3]func(model.Outcome) bool{
		func(o model.Outcome) bool { return o == model.OutcomeSuccess || o == model.OutcomeFailure },
		func(o model.Outcome) bool { return o == model.OutcomeSuccess },
		func(o model.Outcome) bool { return o == model.OutcomeFailure },
	}

	var durations [3][]int64

	for i, keep := range keeps {
		sorted, err := reporttest.Durations(openBytes(t, log), keep)
		if err != nil {
			t.Fatalf("Durations: %v", err)
		}

		durations[i] = sorted
	}

	return durations
}

// observed puts a summary in the shape a comparison with Gatling reads.
func observed(s Summary) reporttest.Observed {
	return reporttest.Observed{
		Outcomes: [3]reporttest.Figures{s.All(), s.OK, s.Failed},
		Rate:     s.Rate,
		Share:    s.Share,
		Bands:    [4]int{s.Under, s.Between, s.Over, s.Failed.Count()},
	}
}

// report fails t on every difference and logs every note of a comparison.
func report(t *testing.T, differences, notes []string) {
	t.Helper()

	for _, note := range notes {
		t.Log(note)
	}

	for _, difference := range differences {
		t.Error(difference)
	}
}

// TestSummaryMatchesLiveGatlingRuns holds the summary of every live Gatling run
// — about 12 000 requests at 50 rps each, rendered from galaxio's own
// gatling/scala-sbt template and served the same responses whatever the
// version (RECORDING.md) — to what Gatling printed on its console and, up to
// 3.13.x, wrote in global_stats.json: every non-percentile whole-run figure
// equal, and every percentile within the rank rule over the run's own log. The
// percentiles Gatling 3.11.x and 3.12.x printed are the reference and are held
// to the same rule; later versions' are reported, not held, because of
// tdunning/t-digest#230.
func TestSummaryMatchesLiveGatlingRuns(t *testing.T) {
	t.Parallel()

	for _, version := range liveVersions(t) {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			log := liveLog(t, version)

			summary, err := Scan(context.Background(), openBytes(t, log), DefaultOptions())
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			durations := liveDurations(t, log)
			reference := strings.HasPrefix(version, "3.11.") || strings.HasPrefix(version, "3.12.")

			console, err := os.ReadFile(filepath.Join(liveDir(), version, "console.txt"))
			if err != nil {
				t.Fatalf("read console.txt: %v", err)
			}

			printed, err := reporttest.ParseConsole(string(console))
			if err != nil {
				t.Fatalf("parse console.txt: %v", err)
			}

			t.Run("console", func(t *testing.T) {
				differences, notes := reporttest.Compare(observed(summary), printed, durations, reference)
				report(t, differences, notes)
			})

			stats, err := os.ReadFile(filepath.Join(liveDir(), version, "js", "global_stats.json"))
			if errors.Is(err, fs.ErrNotExist) {
				return
			}

			if err != nil {
				t.Fatalf("read global_stats.json: %v", err)
			}

			written, err := reporttest.ReadGlobalStats(stats)
			if err != nil {
				t.Fatalf("decode global_stats.json: %v", err)
			}

			t.Run("global_stats.json", func(t *testing.T) {
				differences, notes := reporttest.Compare(observed(summary), written, durations, reference)
				report(t, differences, notes)
			})
		})
	}
}

// TestLiveGatlingRunsWereServedTheSameResponses holds the recordings to what makes
// them comparable across versions: the stub answers request i the same way
// whichever version sends it, so every run holds the same number of requests
// and fails the same ones. Response times differ by the few milliseconds each
// JVM and scheduler adds; the log shows those as they are.
func TestLiveGatlingRunsWereServedTheSameResponses(t *testing.T) {
	t.Parallel()

	type counts struct{ all, ok, failed int }

	var first counts

	versions := liveVersions(t)

	for i, version := range versions {
		summary, err := Scan(context.Background(), openBytes(t, liveLog(t, version)), DefaultOptions())
		if err != nil {
			t.Fatalf("%s: Scan: %v", version, err)
		}

		got := counts{summary.All().Count(), summary.OK.Count(), summary.Failed.Count()}

		p50, _ := summary.All().Percentile(50)
		p95, _ := summary.All().Percentile(95)
		p99, _ := summary.All().Percentile(99)
		t.Logf("%s: %d requests (%d ok, %d failed), p50 %d, p95 %d, p99 %d", version, got.all, got.ok, got.failed, p50, p95, p99)

		if i == 0 {
			first = got

			continue
		}

		if got != first {
			t.Errorf("%s holds %+v, %s holds %+v: the runs were not served the same responses", version, got, versions[0], first)
		}
	}
}
