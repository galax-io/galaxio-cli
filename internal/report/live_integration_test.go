//go:build integration

package report

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
)

// liveScenario holds the Scala TestReportLiveGatling writes over the rendered
// project, at the same paths under src/test/scala/org/galaxio/performance/live:
// the requests of reporttest.Mock in cases/, and the scenario that sends them in
// scenarios/. Every user GETs / as the template's own scenario does. Chosen by
// their number, which Gatling gives in the order users start, 7 % of users also
// GET /report and 3 % GET /export: every version sends the same requests, and a
// second of work for each of 50 users a second would take 50 cores
// (specs/006-live-mock/research.md §2). The same files compile on every version
// the test runs.
var liveScenario = filepath.Join("testdata", "live", "scenario")

// liveEndpoints are the requests liveScenario sends, by the names Gatling gives
// them.
var liveEndpoints = []string{"GET /", "GET /report", "GET /export"}

// servedEverything holds the mock to what Gatling printed of the run: no request
// failed, and every endpoint of liveScenario succeeded. The mock is tested by
// the Gatling run it serves and by nothing else; a summary equal to Gatling's
// proves nothing about it, since Gatling records a broken endpoint as failed
// requests the summary counts just as faithfully.
func servedEverything(t *testing.T, console string, printed reporttest.Recorded) {
	t.Helper()

	if failed := printed.Count[2]; failed != 0 {
		t.Errorf("Gatling printed %d failed requests where the mock fails none", failed)
	}

	for _, name := range liveEndpoints {
		// Every progress block prints each request with its counts so far, and
		// the last block holds the whole run: "> GET /  (OK=3250  KO=0 )" up to
		// Gatling 3.13, and "> GET /  |  3,250 |  3,250 |  0" in columns of the
		// total, ok and failed from 3.14.
		line := regexp.MustCompile(`(?m)^> ` + regexp.QuoteMeta(name) + `\s+(?:\(OK=([\d,]+)\s+KO=[\d,]+\s*\)|\|\s*[\d,]+\s*\|\s*([\d,]+)\s*\|\s*[\d,]+)\s*$`)

		lines := line.FindAllStringSubmatch(console, -1)
		if len(lines) == 0 || strings.Trim(lines[len(lines)-1][1]+lines[len(lines)-1][2], "0,") == "" {
			t.Errorf("Gatling printed no successful %s: the mock or the live scenario is broken", name)
		}
	}
}

// writeLiveScenario copies liveScenario into the project rendered at project,
// and refuses a project whose template no longer renders a file it replaces:
// the run would otherwise send requests nobody wrote here.
func writeLiveScenario(t *testing.T, project string) {
	t.Helper()

	dir := filepath.Join(project, "src", "test", "scala", "org", "galaxio", "performance", "live")

	err := filepath.WalkDir(liveScenario, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		name, err := filepath.Rel(liveScenario, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dir, name)
		if _, err := os.Stat(target); err != nil {
			return fmt.Errorf("the template no longer renders %s, which the live scenario replaces: %w", target, err)
		}

		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, source, 0o644)
	})
	if err != nil {
		t.Fatalf("write the live scenario: %v", err)
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}

// TestReportLiveGatling runs Gatling itself and holds this tool's summary of each
// run to what Gatling printed: the live half of the proof research.md §17
// describes. For every version it renders galaxio's own gatling/scala-sbt
// template with the galaxio binary, writes liveScenario into it, runs the
// template's Stability simulation at 3000 rpm against reporttest.Mock, whose
// every response time comes from work it does, and holds the mock to what
// Gatling printed (servedEverything). It then runs the etalon over the fresh
// log and compares the summary with the console, global_stats.json and the
// etalon as TestSummaryMatchesLiveGatlingRuns compares the committed recordings.
// GALAXIO_LIVE_RECORD writes a new set; the committed one was served by the stub
// RECORDING.md names.
//
// It runs only with GALAXIO_LIVE_GATLING=1, because it needs a JDK, sbt, the two
// t-digest jars and the network, and takes about five minutes a version. GALAXIO_LIVE_VERSIONS lists
// the versions and GALAXIO_LIVE_STEADY shortens the steady stage. The template
// comes from the published registry, with every input RECORDING.md lists set as
// the committed recordings set it; GALAXIO_LIVE_TEMPLATES names another pack
// source, such as local:<a templates-gatling checkout>.
func TestReportLiveGatling(t *testing.T) {
	if os.Getenv("GALAXIO_LIVE_GATLING") != "1" {
		t.Skip("set GALAXIO_LIVE_GATLING=1 to run Gatling for about five minutes a version; it needs a JDK, sbt and the network")
	}

	for _, tool := range []string{"java", "sbt"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not on the path: %v", tool, err)
		}
	}

	// The etalon needs both t-digest jars; finding them now skips before a
	// Gatling run rather than after it.
	for _, jar := range tdigestJars {
		tdigestJar(t, jar.version, jar.sha256)
	}

	steady, err := time.ParseDuration(envOr("GALAXIO_LIVE_STEADY", "4m"))
	if err != nil {
		t.Fatalf("GALAXIO_LIVE_STEADY: %v", err)
	}

	versions := strings.Split(envOr("GALAXIO_LIVE_VERSIONS", "3.11.5,3.12.0,3.13.1,3.14.9,3.15.1"), ",")
	tools := t.TempDir()
	bin := filepath.Join(tools, "galaxio")

	if out, err := exec.Command("go", "build", "-o", bin, "github.com/galax-io/galaxio-cli/cmd/galaxio").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	var registry []string

	if templates := os.Getenv("GALAXIO_LIVE_TEMPLATES"); templates != "" {
		manifest := "apiVersion: galaxio.io/v1\nkind: TemplateRegistry\nversion: 1.0.0\npacks:\n  - name: gatling\n    source: " + templates + "\n"
		if err := os.WriteFile(filepath.Join(tools, "galaxio-registry.yaml"), []byte(manifest), 0o644); err != nil {
			t.Fatalf("write the registry: %v", err)
		}

		registry = []string{"--registry", "local:" + tools}
	}

	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			mock := httptest.NewServer(reporttest.Mock())
			defer mock.Close()

			work := t.TempDir()
			project := filepath.Join(work, "project")

			render := []string{
				"template", "init", "gatling/scala-sbt", "--destination", project,
				"--set", "Name=live", "--set", "NameWord=live", "--set", "GatlingVersion=" + version,
				"--set", "GatlingPicatinnyVersion=1.27.0", "--set", "SbtGatlingVersion=4.19.1",
				"--set", "SbtVersion=1.12.13", "--set", "SbtScalafmtVersion=2.6.1",
				"--set", "BaseUrl=" + mock.URL, "--set", "Intensity=3000 rpm", "--set", "RampDuration=10 seconds",
				"--set", fmt.Sprintf("StageDuration=%d seconds", int(steady.Seconds())),
				"--set", fmt.Sprintf("TestDuration=%d seconds", int((steady + time.Minute).Seconds())),
				"--set", "StartupBannerEnabled=false",
			}
			render = append(render, registry...)

			scaffold := exec.Command(bin, render...)
			scaffold.Env = append(os.Environ(), "GALAXIO_CONFIG="+filepath.Join(work, "config.yaml"), "GALAXIO_CACHE_DIR="+filepath.Join(work, "cache"))

			if out, err := scaffold.CombinedOutput(); err != nil {
				t.Fatalf("galaxio template init: %v\n%s", err, out)
			}

			writeLiveScenario(t, project)

			ctx, cancel := context.WithTimeout(context.Background(), steady+20*time.Minute)
			defer cancel()

			sbt := exec.CommandContext(ctx, "sbt", "-batch", "Gatling/testOnly org.galaxio.performance.live.Stability")
			sbt.Dir = project

			console, err := sbt.CombinedOutput()
			if err != nil {
				t.Fatalf("sbt: %v\n%s", err, console)
			}

			logs, err := filepath.Glob(filepath.Join(project, "target", "gatling", "stability-*", "simulation.log"))
			if err != nil || len(logs) != 1 {
				t.Fatalf("found %d runs (%v), want one", len(logs), err)
			}

			runDir := filepath.Dir(logs[0])

			reported, err := exec.Command(bin, "report", "gatling", runDir).CombinedOutput()
			if err != nil {
				t.Fatalf("galaxio report: %v\n%s", err, reported)
			}

			log, err := os.ReadFile(logs[0])
			if err != nil {
				t.Fatalf("read simulation.log: %v", err)
			}

			summary, err := Scan(context.Background(), openBytes(t, log), DefaultOptions(), nil)
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			requests := fmt.Sprintf("requests    %d (%d ok, %d failed)", summary.Tally.Requests, summary.OK.Count(), summary.Failed.Count())
			if !strings.Contains(string(reported), requests) {
				t.Errorf("galaxio report printed\n%s\nwithout %q", reported, requests)
			}

			durations := outcomeDurations(t, log)
			reference := reporttest.IsReference(version)

			printed, err := reporttest.ParseConsole(string(console))
			if err != nil {
				t.Fatalf("parse the console: %v", err)
			}

			servedEverything(t, string(console), printed)

			given := runEtalon(t, openBytes(t, log))

			etalon, err := reporttest.ReadEtalon(given)
			if err != nil {
				t.Fatalf("ReadEtalon: %v", err)
			}

			report(t, reporttest.Compare(observed(summary), printed, durations), nil)

			differences, notes := reporttest.ComparePercentiles(observed(summary), etalon, printed, reference, durations)
			report(t, differences, notes)

			stats, err := os.ReadFile(filepath.Join(runDir, "js", "global_stats.json"))

			switch {
			case errors.Is(err, fs.ErrNotExist):
			case err != nil:
				t.Fatalf("read global_stats.json: %v", err)
			default:
				written, err := reporttest.ReadGlobalStats(stats)
				if err != nil {
					t.Fatalf("decode global_stats.json: %v", err)
				}

				report(t, reporttest.Compare(observed(summary), written, durations), nil)

				differences, notes := reporttest.ComparePercentiles(observed(summary), etalon, written, reference, durations)
				report(t, differences, notes)
			}

			if dir := os.Getenv("GALAXIO_LIVE_RECORD"); dir != "" {
				recordLiveRun(t, filepath.Join(dir, version), log, console, stats, given)
			}
		})
	}
}

// recordLiveRun keeps what a live run left for the ordinary suite: the log,
// compressed, the Global Information block of the console, global_stats.json
// where Gatling wrote one, and what the etalon gave for the log. The rest of
// the console — the progress blocks, and the build tool's lines naming paths on
// this machine — is not kept.
func recordLiveRun(t *testing.T, dir string, log, console, stats, etalon []byte) {
	t.Helper()

	block, err := reporttest.GlobalInformation(string(console))
	if err != nil {
		t.Fatalf("cut the console: %v", err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create %s: %v", dir, err)
	}

	var compressed bytes.Buffer

	gz, err := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}

	if _, err := gz.Write(log); err != nil {
		t.Fatalf("compress the log: %v", err)
	}

	if err := gz.Close(); err != nil {
		t.Fatalf("compress the log: %v", err)
	}

	files := map[string][]byte{"simulation.log.gz": compressed.Bytes(), "console.txt": []byte(block), "etalon.tsv": etalon}
	if stats != nil {
		if err := os.MkdirAll(filepath.Join(dir, "js"), 0o755); err != nil {
			t.Fatalf("create %s: %v", dir, err)
		}

		files[filepath.Join("js", "global_stats.json")] = stats
	}

	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}
