package report

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/gatling/simlog"
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

// outcomeDurations returns a recorded log's response times for all, successful
// and failed requests, sorted.
func outcomeDurations(t *testing.T, log []byte) [3][]int64 {
	t.Helper()

	durations, err := reporttest.Durations(openBytes(t, log))
	if err != nil {
		t.Fatalf("Durations: %v", err)
	}

	return durations
}

// readEtalon reads what the real t-digest libraries give for the run recorded
// in dir, the etalon.tsv TestEtalonRecordings keeps beside it.
func readEtalon(t *testing.T, dir string) reporttest.Etalon {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, "etalon.tsv"))
	if err != nil {
		t.Fatalf("read etalon.tsv: %v", err)
	}

	etalon, err := reporttest.ReadEtalon(data)
	if err != nil {
		t.Fatalf("ReadEtalon: %v", err)
	}

	return etalon
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

// reportRecording is report for a committed recording. None of them needs the
// one difference from Gatling 3.11's digest that Principle II admits, the order
// of equal centroids, so a note describing it fails t as a difference does: a
// change that makes a recording need it has to be looked at.
func reportRecording(t *testing.T, differences, notes []string) {
	t.Helper()

	for _, note := range notes {
		if strings.Contains(note, reporttest.OrderOfEqualCentroids) {
			differences = append(differences, note)
		}
	}

	report(t, differences, notes)
}

// TestSummaryMatchesLiveGatlingRuns holds the summary of every live Gatling run
// — about 12 000 requests at 50 rps each, rendered from galaxio's own
// gatling/scala-sbt template and served the same responses whatever the
// version (RECORDING.md) — to what Gatling printed on its console and, up to
// 3.13.x, wrote in global_stats.json: every non-percentile whole-run figure
// equal, and every percentile within the rank rule over the run's own log and
// equal to Gatling 3.11's — a value its digest gives for the log, per the
// etalon.tsv beside it, and for 3.11.5 and 3.12.0 the value they printed. What
// later versions printed is described, not held, because of
// tdunning/t-digest#230.
func TestSummaryMatchesLiveGatlingRuns(t *testing.T) {
	t.Parallel()

	for _, version := range liveVersions(t) {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			log := liveLog(t, version)

			summary, err := Scan(context.Background(), openBytes(t, log), DefaultOptions(), nil)
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			durations := outcomeDurations(t, log)
			reference := reporttest.IsReference(version)

			console, err := os.ReadFile(filepath.Join(liveDir(), version, "console.txt"))
			if err != nil {
				t.Fatalf("read console.txt: %v", err)
			}

			printed, err := reporttest.ParseConsole(string(console))
			if err != nil {
				t.Fatalf("parse console.txt: %v", err)
			}

			etalon := readEtalon(t, filepath.Join(liveDir(), version))

			t.Run("console", func(t *testing.T) {
				report(t, reporttest.Compare(observed(summary), printed, durations), nil)

				differences, notes := reporttest.ComparePercentiles(observed(summary), etalon, printed, reference, durations)
				reportRecording(t, differences, notes)
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
				report(t, reporttest.Compare(observed(summary), written, durations), nil)

				differences, notes := reporttest.ComparePercentiles(observed(summary), etalon, written, reference, durations)
				reportRecording(t, differences, notes)
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
		summary, err := Scan(context.Background(), openBytes(t, liveLog(t, version)), DefaultOptions(), nil)
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

// TestEtalonsAreOfTheirRuns holds every committed etalon.tsv to the run beside
// it: the etalon names how many requests it was fed and the SHA-256 of what it
// read, and both are recomputed here from the log, or from the synthetic run's
// own requests, as reporttest.EtalonSamples writes them. The etalon itself runs
// only where a JDK and the jars are (TestEtalonRecordings); this is what keeps a
// file from standing for another run's numbers everywhere else.
func TestEtalonsAreOfTheirRuns(t *testing.T) {
	t.Parallel()

	runs := map[string]func(t *testing.T) simlog.RunReader{}

	for _, version := range reporttest.Versions {
		runs[filepath.Join(corpusDir, version)] = func(t *testing.T) simlog.RunReader { return openBytes(t, corpusLog(t, version)) }
	}

	for _, version := range liveVersions(t) {
		runs[filepath.Join(liveDir(), version)] = func(t *testing.T) simlog.RunReader { return openBytes(t, liveLog(t, version)) }
	}

	for _, synthetic := range syntheticRuns {
		runs[syntheticDir(synthetic.name)] = func(*testing.T) simlog.RunReader { return reporttest.Items(model.Run{}, nil, synthetic.items()...) }
	}

	if len(runs) < 14 {
		t.Fatalf("%d runs with an etalon, want the five of the corpus, the five live ones and the four synthetic", len(runs))
	}

	for dir, run := range runs {
		t.Run(dir, func(t *testing.T) {
			t.Parallel()

			var samples bytes.Buffer
			if err := reporttest.EtalonSamples(&samples, run(t)); err != nil {
				t.Fatalf("EtalonSamples: %v", err)
			}

			etalon := readEtalon(t, dir)
			sum := sha256.Sum256(samples.Bytes())

			if got := bytes.Count(samples.Bytes(), []byte("\n")); got != etalon.Samples {
				t.Errorf("the run holds %d requests with a recorded end, the etalon was fed %d", got, etalon.Samples)
			}

			if got := hex.EncodeToString(sum[:]); got != etalon.SHA256 {
				t.Errorf("the run's samples have SHA-256 %s, the etalon was fed %s", got, etalon.SHA256)
			}
		})
	}
}
