//go:build integration

package main

import (
	"bytes"
	"context"
	"errors"
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

// writeLargeRun replays the 3.12.0 recording into a run directory far larger
// than any recording, so that the binary is exercised on a log it cannot hold.
func writeLargeRun(t *testing.T, repeats int) string {
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

	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, reporttest.Replay(header, body, repeats)); err != nil {
		t.Fatalf("write log: %v", err)
	}

	return dir
}

func TestReportIntegrationReadsALargeRun(t *testing.T) {
	const repeats = 20000

	bin := buildGalaxio(t)
	dir := writeLargeRun(t, repeats)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, bin, "report", "gatling", dir)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("galaxio report: %v; stderr: %s", err, stderr.String())
	}

	// The 3.12.0 recording holds 36 requests, 18 of each outcome, per replay.
	for _, want := range []string{
		"requests    720000 (360000 ok, 360000 ko)",
		"groups      240000 traversals",
		"users       240000 events",
		"tool        gatling 3.12.0",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("expected the report to contain %q, got:\n%s", want, stdout.String())
		}
	}

	if strings.Contains(stderr.String(), "Error:") {
		t.Errorf("unexpected error on stderr: %s", stderr.String())
	}
}

func TestReportIntegrationIsDeterministic(t *testing.T) {
	bin := buildGalaxio(t)
	dir := filepath.Join(reportCorpus, "3.15.1")

	read := func() []byte {
		t.Helper()

		out, err := exec.Command(bin, "report", "gatling", dir).Output()
		if err != nil {
			t.Fatalf("galaxio report: %v", err)
		}

		return out
	}

	if first, second := read(), read(); !bytes.Equal(first, second) {
		t.Fatalf("two reads of the same run differ:\n%s\nversus\n%s", first, second)
	}
}

func TestReportIntegrationExitCodes(t *testing.T) {
	bin := buildGalaxio(t)

	tests := []struct {
		name string
		args []string
		code int
	}{
		{name: "a run that reads", args: []string{"report", "gatling", filepath.Join(reportCorpus, "3.15.1")}, code: 0},
		{name: "no run under the directory", args: []string{"report", "gatling", t.TempDir()}, code: 1},
		{name: "an unsupported tool", args: []string{"report", "jmeter"}, code: 2},
		{name: "a reserved report format", args: []string{"report", "gatling", filepath.Join(reportCorpus, "3.15.1"), "-o", "stats"}, code: 2},
		{name: "no arguments prints help", args: []string{"report"}, code: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exec.Command(bin, tt.args...).Run()

			code := 0

			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				code = exitErr.ExitCode()
			} else if err != nil {
				t.Fatalf("running the binary: %v", err)
			}

			if code != tt.code {
				t.Errorf("exit code = %d, want %d", code, tt.code)
			}
		})
	}
}
