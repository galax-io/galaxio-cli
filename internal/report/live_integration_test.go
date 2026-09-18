//go:build integration

package report

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
)

// liveSeed seeds the stub's plan, so that the responses are the same for every
// recording ever made with this harness.
const liveSeed = 20260917

// livePlan returns how long request i waits and which status it answers with.
// It depends on i alone, so every Gatling version is served the same responses
// in the same order: 85 % within 5–40 ms, 8 % 80–400 ms, 3 % 800–1199 ms, 2 %
// 1200–2000 ms, and 2 % a 500 within 5 ms, which the template's check fails.
func livePlan(i uint64) (time.Duration, int) {
	r := rand.New(rand.NewPCG(liveSeed, i))
	ms := func(lo, hi int) time.Duration { return time.Duration(lo+r.IntN(hi-lo+1)) * time.Millisecond }

	switch x := r.IntN(100); {
	case x < 2:
		return ms(1, 5), http.StatusInternalServerError
	case x < 87:
		return ms(5, 40), http.StatusOK
	case x < 95:
		return ms(80, 400), http.StatusOK
	case x < 98:
		return ms(800, 1199), http.StatusOK
	default:
		return ms(1200, 2000), http.StatusOK
	}
}

// liveStub answers every request by livePlan, counting from zero.
func liveStub() http.Handler {
	var next atomic.Uint64

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		wait, status := livePlan(next.Add(1) - 1)
		time.Sleep(wait)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte("{}"))
	})
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
// template with the galaxio binary, runs the template's Stability simulation at
// 3000 rpm against a stub that serves every version the same responses, and
// compares the summary with the console and global_stats.json as
// TestSummaryMatchesLiveGatlingRuns compares the committed recordings, which
// GALAXIO_LIVE_RECORD writes.
//
// It runs only with GALAXIO_LIVE_GATLING=1, because it needs a JDK, sbt and the
// network, and takes about five minutes a version. GALAXIO_LIVE_VERSIONS lists
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
			stub := httptest.NewServer(liveStub())
			defer stub.Close()

			work := t.TempDir()
			project := filepath.Join(work, "project")

			render := []string{
				"template", "init", "gatling/scala-sbt", "--destination", project,
				"--set", "Name=live", "--set", "NameWord=live", "--set", "GatlingVersion=" + version,
				"--set", "GatlingPicatinnyVersion=1.27.0", "--set", "SbtGatlingVersion=4.19.1",
				"--set", "SbtVersion=1.12.13", "--set", "SbtScalafmtVersion=2.6.1",
				"--set", "BaseUrl=" + stub.URL, "--set", "Intensity=3000 rpm", "--set", "RampDuration=10 seconds",
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

			summary, err := Scan(context.Background(), openBytes(t, log), DefaultOptions())
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			requests := fmt.Sprintf("requests    %d (%d ok, %d failed)", summary.Tally.Requests, summary.OK.Count(), summary.Failed.Count())
			if !strings.Contains(string(reported), requests) {
				t.Errorf("galaxio report printed\n%s\nwithout %q", reported, requests)
			}

			durations := liveDurations(t, log)
			reference := strings.HasPrefix(version, "3.11.") || strings.HasPrefix(version, "3.12.")

			printed, err := reporttest.ParseConsole(string(console))
			if err != nil {
				t.Fatalf("parse the console: %v", err)
			}

			differences, notes := reporttest.Compare(observed(summary), printed, durations, reference)
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

				differences, notes := reporttest.Compare(observed(summary), written, durations, reference)
				report(t, differences, notes)
			}

			if dir := os.Getenv("GALAXIO_LIVE_RECORD"); dir != "" {
				recordLiveRun(t, filepath.Join(dir, version), log, console, stats)
			}
		})
	}
}

// recordLiveRun keeps what a live run left for the ordinary suite: the log,
// compressed, the Global Information block of the console, and
// global_stats.json where Gatling wrote one. The rest of the console — the
// progress blocks, and the build tool's lines naming paths on this machine — is
// not kept.
func recordLiveRun(t *testing.T, dir string, log, console, stats []byte) {
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

	files := map[string][]byte{"simulation.log.gz": compressed.Bytes(), "console.txt": []byte(block)}
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
