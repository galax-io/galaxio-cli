//go:build integration

package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
)

// buildGalaxio compiles the binary under test into a temporary directory.
func buildGalaxio(t *testing.T) string {
	t.Helper()

	bin := filepath.Join(t.TempDir(), "galaxio")
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

// writeLargeRun replays the 3.12.0 recording into a run directory large
// enough that its output overflows any pipe buffer many times over.
func writeLargeRun(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(reportCorpus, "3.12.0", "simulation.log"))
	if err != nil {
		t.Fatalf("read corpus log: %v", err)
	}
	header, body, err := reporttest.Split(data)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "simulation.log"))
	if err != nil {
		t.Fatalf("create log: %v", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, reporttest.Replay(header, body, 2000)); err != nil {
		t.Fatalf("write log: %v", err)
	}
	return dir
}

func TestReportIntegrationClosedPipe(t *testing.T) {
	for _, tool := range []string{"sh", "head"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not available: %v", tool, err)
		}
	}
	bin := buildGalaxio(t)
	dir := writeLargeRun(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var stdout, stderr bytes.Buffer
	pipeline := exec.CommandContext(ctx, "sh", "-c", `"$1" report gatling "$2" | head -n 1`, "sh", bin, dir)
	pipeline.Stdout = &stdout
	pipeline.Stderr = &stderr
	err := pipeline.Run()

	if ctx.Err() != nil {
		t.Fatalf("the pipeline did not end within 5 s after head closed the pipe")
	}
	if err != nil {
		t.Fatalf("pipeline: %v; stderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"kind":"run"`) || strings.Count(stdout.String(), "\n") != 1 {
		t.Errorf("head -n 1 did not receive the header line, got %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "Error:") {
		t.Errorf("a closed pipe must not produce a diagnostic, got %q", stderr.String())
	}
}

func TestReportIntegrationDeterministic(t *testing.T) {
	bin := buildGalaxio(t)
	dir := filepath.Join(reportCorpus, "3.15.1")

	run := func() []byte {
		t.Helper()
		out, err := exec.Command(bin, "report", "gatling", dir, "--quiet").Output()
		if err != nil {
			t.Fatalf("galaxio report: %v", err)
		}
		return out
	}
	first, second := run(), run()
	if !bytes.Equal(first, second) {
		t.Fatalf("two runs of the binary on the same log differ")
	}
	if strings.Count(string(first), "\n") != 1+102+12+12+6 {
		t.Errorf("expected header plus 132 records, got %d lines", strings.Count(string(first), "\n"))
	}
}

// maxFirstLineLatency is SC-003's bound on when the first record reaches a
// reader, measured here from process start on a 14 MB run.
const maxFirstLineLatency = time.Second

func TestReportIntegrationFirstLineLatency(t *testing.T) {
	bin := buildGalaxio(t)
	dir := writeLargeRun(t)

	// The first execution of a freshly built binary is the operating
	// system's, not the command's: on macOS it costs about a second while the
	// new executable is scanned, and every later run starts in milliseconds.
	// One throwaway run keeps the measurement about the command.
	if err := exec.Command(bin, "version").Run(); err != nil {
		t.Fatalf("warm-up run: %v", err)
	}

	cmd := exec.Command(bin, "report", "gatling", dir, "--quiet")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	started := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	first, err := bufio.NewReader(stdout).ReadString('\n')
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("reading the first line: %v", err)
	}
	if !strings.Contains(first, `"kind":"run"`) {
		t.Errorf("first line is not the header: %q", first)
	}
	if elapsed > maxFirstLineLatency {
		t.Errorf("first line arrived after %v, want under %v", elapsed, maxFirstLineLatency)
	}
	t.Logf("first line after %v", elapsed)
}
