package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/gatling/run"
)

// reportCorpus is the frozen Gatling corpus next to the package that converts
// it; the command tests reach it rather than duplicating recordings.
var reportCorpus = filepath.Join("..", "..", "internal", "report", "testdata", "corpus", "gatling")

// recordKinds checks that stdout is JSON Lines with a kind on every line and
// returns the kinds in order.
func recordKinds(t *testing.T, stdout string) []string {
	t.Helper()

	if !strings.HasSuffix(stdout, "\n") {
		t.Fatalf("stdout does not end with a newline")
	}
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	kinds := make([]string, 0, len(lines))
	for i, line := range lines {
		var rec struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("stdout line %d is not a JSON object: %v: %s", i+1, err, line)
		}
		kinds = append(kinds, rec.Kind)
	}
	return kinds
}

func countKind(kinds []string, kind string) int {
	n := 0
	for _, k := range kinds {
		if k == kind {
			n++
		}
	}
	return n
}

func TestReportWritesRecords(t *testing.T) {
	tests := []struct {
		version  string
		requests int
	}{
		{version: "3.12.0", requests: 36},
		{version: "3.15.1", requests: 102},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			code, stdout, stderr := runCLI("report", "gatling", filepath.Join(reportCorpus, tt.version))

			if code != exitOK {
				t.Fatalf("expected exit code %d, got %d; stderr: %s", exitOK, code, stderr)
			}
			if strings.Contains(stderr, "Error:") {
				t.Fatalf("unexpected error on stderr: %s", stderr)
			}
			kinds := recordKinds(t, stdout)
			if kinds[0] != "run" {
				t.Errorf("first line kind = %q, want run", kinds[0])
			}
			if got := countKind(kinds, "request"); got != tt.requests {
				t.Errorf("request records = %d, want %d (Gatling's console summary)", got, tt.requests)
			}
		})
	}
}

func TestReportLogPathEqualsDirectory(t *testing.T) {
	dir := filepath.Join(reportCorpus, "3.15.1")
	code, fromDir, _ := runCLI("report", "gatling", dir)
	if code != exitOK {
		t.Fatalf("directory: exit code %d", code)
	}
	code, fromLog, _ := runCLI("report", "gatling", filepath.Join(dir, "simulation.log"))
	if code != exitOK {
		t.Fatalf("log path: exit code %d", code)
	}
	if fromDir != fromLog {
		t.Fatalf("the log path and its directory produced different output")
	}
}

func TestReportRequiresTool(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "a path where the tool should be", args: []string{"report", filepath.Join(reportCorpus, "3.15.1")}},
		{name: "an unsupported tool", args: []string{"report", "jmeter", filepath.Join(reportCorpus, "3.15.1")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runCLI(tt.args...)

			if code != exitUsage {
				t.Fatalf("expected exit code %d, got %d", exitUsage, code)
			}
			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			for _, want := range []string{"unsupported tool", "accepted tools: gatling"} {
				if !strings.Contains(stderr, want) {
					t.Errorf("expected stderr to contain %q, got %q", want, stderr)
				}
			}
		})
	}
}

func TestRunReportOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	dir := filepath.Join(reportCorpus, "3.15.1")

	out, err := runReport(context.Background(), reportOptions{Tool: "gatling", Path: dir, Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		t.Fatalf("runReport: %v", err)
	}
	if out.Dir != dir {
		t.Errorf("Dir = %q, want %q", out.Dir, dir)
	}
	if out.Log != filepath.Join(dir, "simulation.log") {
		t.Errorf("Log = %q, want the simulation.log inside the run", out.Log)
	}
	if out.Found != run.FoundByPath {
		t.Errorf("Found = %v, want %v", out.Found, run.FoundByPath)
	}
	if out.Summary.Requests != 102 || out.Summary.Truncated != nil {
		t.Errorf("Summary = %+v, want 102 requests and no truncation", out.Summary)
	}
}

// writeRun writes a synthetic simulation.log into a fresh run directory and
// returns that directory.
func writeRun(t *testing.T, log []byte) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "simulation.log"), log, 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	return dir
}

func TestReportWarnsAboutUnverifiedVersion(t *testing.T) {
	dir := writeRun(t, []byte("RUN\tio.x.Sim\tsim\t1700000000000\t \t3.99.0\n"))

	code, stdout, stderr := runCLI("report", "gatling", dir, "--quiet")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", exitOK, code, stderr)
	}
	if !strings.Contains(stderr, "report: warning: 3.99.0: ") {
		t.Errorf("expected the warning on stderr even under --quiet, got %q", stderr)
	}
	kinds := recordKinds(t, stdout)
	if len(kinds) != 1 || kinds[0] != "run" {
		t.Errorf("kinds = %v, want the header only", kinds)
	}
	if !strings.Contains(stdout, `"warnings":[{"version":"3.99.0","reason":`) {
		t.Errorf("header lacks the warning: %s", stdout)
	}
}

// copyRun copies one corpus run's simulation.log under root/name.
func copyRun(t *testing.T, root, src, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(src, "simulation.log"))
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "simulation.log"), data, 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	return dir
}

func TestReportNamesTheRunRead(t *testing.T) {
	lastrun := filepath.Join(reportCorpus, "lastrun", "results")

	tests := []struct {
		name     string
		args     func(t *testing.T) []string
		expected string
	}{
		{
			name:     "a results root with lastRun.txt",
			args:     func(*testing.T) []string { return []string{"report", "gatling", lastrun} },
			expected: "report: reading " + filepath.Join(lastrun, "corpussimulation-20260909022708912") + " (found by lastRun.txt)\n",
		},
		{
			name: "a results root without a marker",
			args: func(t *testing.T) []string {
				root := filepath.Join(t.TempDir(), "results")
				copyRun(t, root, filepath.Join(lastrun, "corpussimulation-20260909022556930"), "corpussimulation-20260909022556930")
				return []string{"report", "gatling", root}
			},
			expected: "(found by newest)\n",
		},
		{
			name:     "a run named explicitly",
			args:     func(*testing.T) []string { return []string{"report", "gatling", filepath.Join(reportCorpus, "3.15.1")} },
			expected: "report: reading " + filepath.Join(reportCorpus, "3.15.1") + " (found by path)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, stderr := runCLI(tt.args(t)...)

			if code != exitOK {
				t.Fatalf("expected exit code %d, got %d; stderr: %s", exitOK, code, stderr)
			}
			if !strings.Contains(stderr, tt.expected) {
				t.Errorf("expected stderr to contain %q, got %q", tt.expected, stderr)
			}
		})
	}
}

func TestReportQuietSilencesTheDiagnostic(t *testing.T) {
	code, stdout, stderr := runCLI("report", "gatling", filepath.Join(reportCorpus, "3.15.1"), "--quiet")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr under --quiet, got %q", stderr)
	}
	if kinds := recordKinds(t, stdout); countKind(kinds, "request") != 102 {
		t.Errorf("--quiet must not change stdout")
	}
}

func TestReportVerboseSummary(t *testing.T) {
	code, _, stderr := runCLI("report", "gatling", filepath.Join(reportCorpus, "3.15.1"), "--verbose")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	expected := "report: binary log, Gatling 3.15.1: 102 requests, 12 groups, 12 user events, 6 errors\n"
	if !strings.Contains(stderr, expected) {
		t.Errorf("expected stderr to contain %q, got %q", expected, stderr)
	}
}

func TestReportDefaultRoot(t *testing.T) {
	project := t.TempDir()
	corpus, err := filepath.Abs(filepath.Join(reportCorpus, "3.12.0"))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	copyRun(t, filepath.Join(project, "target", "gatling"), corpus, "corpussimulation-20260903000000000")
	t.Chdir(project)

	code, stdout, stderr := runCLI("report", "gatling")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", exitOK, code, stderr)
	}
	expected := "report: reading " + filepath.Join("target", "gatling", "corpussimulation-20260903000000000") + " (found by newest)\n"
	if !strings.Contains(stderr, expected) {
		t.Errorf("expected stderr to contain %q, got %q", expected, stderr)
	}
	if kinds := recordKinds(t, stdout); countKind(kinds, "request") != 36 {
		t.Errorf("expected the 3.12.0 run's 36 requests, got kinds %v", kinds)
	}
}

func TestReportNoRunFound(t *testing.T) {
	dir := t.TempDir()

	code, stdout, stderr := runCLI("report", "gatling", dir)

	if code != exitRuntime {
		t.Fatalf("expected exit code %d, got %d", exitRuntime, code)
	}
	if stdout != "" {
		t.Errorf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "no Gatling run under "+dir) {
		t.Errorf("expected stderr to name the searched directory, got %q", stderr)
	}
}

func TestReportRejectsReportFormats(t *testing.T) {
	dir := filepath.Join(reportCorpus, "3.15.1")

	tests := []struct {
		output   string
		expected string
	}{
		{output: "stats", expected: `report format "stats" is not available yet: it arrives with milestone v0.15.0 Legacy stats.json`},
		{output: "stats,global_stats", expected: `report format "stats" is not available yet: it arrives with milestone v0.15.0 Legacy stats.json`},
		{output: "global_stats", expected: `report format "global_stats" is not available yet: it arrives with milestone v0.15.0 Legacy stats.json`},
		{output: "yml", expected: `report format "yml" is not available yet: the OpenNFR YAML report is postponed`},
		{output: "json", expected: `unknown report format "json": known formats: stats, global_stats, yml`},
		{output: "text", expected: `unknown report format "text": known formats: stats, global_stats, yml`},
	}

	for _, tt := range tests {
		t.Run(tt.output, func(t *testing.T) {
			code, stdout, stderr := runCLI("report", "gatling", dir, "-o", tt.output)

			if code != exitUsage {
				t.Fatalf("expected exit code %d, got %d", exitUsage, code)
			}
			if stdout != "" {
				t.Errorf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, tt.expected) {
				t.Errorf("expected stderr to contain %q, got %q", tt.expected, stderr)
			}
		})
	}
}

// corpusLog reads one corpus recording.
func corpusLog(t *testing.T, version string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(reportCorpus, version, "simulation.log"))
	if err != nil {
		t.Fatalf("read corpus log: %v", err)
	}
	return data
}

// patchedCorpusLog is the 3.13.1 recording with its six version bytes at
// offset 5 replaced — the bytes parsec's gate reads first.
func patchedCorpusLog(t *testing.T, version string) []byte {
	t.Helper()

	data := corpusLog(t, "3.13.1")
	if got := string(data[5:11]); got != "3.13.1" {
		t.Fatalf("corpus log carries version %q at offset 5, want 3.13.1", got)
	}
	patched := append([]byte(nil), data...)
	copy(patched[5:11], version)
	return patched
}

// damagedTextLog is the 3.12.0 recording with one event line replaced by
// something no codec can decode.
func damagedTextLog(t *testing.T) []byte {
	t.Helper()

	header, body, err := reporttest.Split(corpusLog(t, "3.12.0"))
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	lines := strings.Split(string(body), "\n")
	lines[20] = "BOGUS\tnot a record"
	return append(append([]byte(nil), header...), []byte(strings.Join(lines, "\n"))...)
}

func TestReportFailures(t *testing.T) {
	tests := []struct {
		name        string
		args        func(t *testing.T) []string
		code        int
		stdoutLines int // -1: not checked
		stderrHas   []string
		stderrLacks []string
	}{
		{
			name: "a version below the supported range",
			args: func(t *testing.T) []string {
				return []string{"report", "gatling", writeRun(t, []byte("RUN\tio.x.Sim\tsim\t1700000000000\t \t3.10.0\n"))}
			},
			code:        exitRuntime,
			stdoutLines: 0,
			stderrHas:   []string{"version 3.10.0 is below the supported range 3.11.5 through 3.12.0"},
		},
		{
			name: "3.13.0 is refused",
			args: func(t *testing.T) []string {
				return []string{"report", "gatling", writeRun(t, patchedCorpusLog(t, "3.13.0"))}
			},
			code:        exitRuntime,
			stdoutLines: 0,
			stderrHas:   []string{"3.13.0", "3.13.1"},
		},
		{
			name: "a log cut short",
			args: func(t *testing.T) []string {
				return []string{"report", "gatling", writeRun(t, corpusLog(t, "3.15.1")[:2000])}
			},
			code:        exitRuntime,
			stdoutLines: 63,
			stderrHas:   []string{"the log is cut short", "the 62 records already written are what the run recorded"},
		},
		{
			name: "a path that does not exist",
			args: func(t *testing.T) []string {
				return []string{"report", "gatling", filepath.Join(t.TempDir(), "nonexistent")}
			},
			code:        exitRuntime,
			stdoutLines: 0,
			stderrHas:   []string{"cannot read ", "nonexistent"},
			stderrLacks: []string{"no Gatling run"},
		},
		{
			name: "not a Gatling log",
			args: func(t *testing.T) []string {
				return []string{"report", "gatling", writeRun(t, []byte("<!DOCTYPE html>\n<html></html>\n"))}
			},
			code:        exitRuntime,
			stdoutLines: 0,
			stderrHas:   []string{"not a Gatling simulation.log", "simulation.log"},
		},
		{
			name:        "a damaged log",
			args:        func(t *testing.T) []string { return []string{"report", "gatling", writeRun(t, damagedTextLog(t))} },
			code:        exitRuntime,
			stdoutLines: 21,
			stderrHas:   []string{"the 20 records already written do not form a complete run"},
		},
		{
			name:        "three positional arguments",
			args:        func(*testing.T) []string { return []string{"report", "gatling", "a", "b"} },
			code:        exitUsage,
			stdoutLines: 0,
			stderrHas:   []string{"accepts at most 2 arg(s), received 3"},
		},
		{
			name: "an unknown flag",
			args: func(*testing.T) []string {
				return []string{"report", "gatling", filepath.Join(reportCorpus, "3.15.1"), "--bogus"}
			},
			code:        exitUsage,
			stdoutLines: 0,
			stderrHas:   []string{"unknown flag: --bogus"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runCLI(tt.args(t)...)

			if code != tt.code {
				t.Fatalf("expected exit code %d, got %d; stderr: %s", tt.code, code, stderr)
			}
			if tt.stdoutLines == 0 && stdout != "" {
				t.Errorf("expected empty stdout, got %q", stdout)
			}
			if tt.stdoutLines > 0 {
				if kinds := recordKinds(t, stdout); len(kinds) != tt.stdoutLines {
					t.Errorf("stdout holds %d complete lines, want %d", len(kinds), tt.stdoutLines)
				}
			}
			for _, want := range tt.stderrHas {
				if !strings.Contains(stderr, want) {
					t.Errorf("expected stderr to contain %q, got %q", want, stderr)
				}
			}
			for _, unwanted := range tt.stderrLacks {
				if strings.Contains(stderr, unwanted) {
					t.Errorf("expected stderr not to contain %q, got %q", unwanted, stderr)
				}
			}
		})
	}
}

// failAfterWriter accepts n writes and then fails every one, like a reader
// that went away.
type failAfterWriter struct {
	remaining int
	calls     int
}

var errPipeClosed = errors.New("pipe closed")

func (w *failAfterWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.remaining == 0 {
		return 0, errPipeClosed
	}
	w.remaining--
	return len(p), nil
}

func TestReportStopsAtFirstWriteFailure(t *testing.T) {
	header, body, err := reporttest.Split(corpusLog(t, "3.12.0"))
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	large, err := io.ReadAll(reporttest.Replay(header, body, 40))
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	dir := writeRun(t, large)

	var stderr bytes.Buffer
	stdout := &failAfterWriter{remaining: 2}
	code := execute([]string{"report", "gatling", dir, "--quiet"}, stdout, &stderr)

	if code != exitRuntime {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", exitRuntime, code, stderr.String())
	}
	if stdout.calls != 3 {
		t.Errorf("stdout saw %d writes after failing on the third, want exactly 3", stdout.calls)
	}
	if got := strings.Count(stderr.String(), "Error:"); got != 1 {
		t.Errorf("expected exactly one Error line on stderr, got %d: %s", got, stderr.String())
	}
	if !strings.Contains(stderr.String(), "writing output: pipe closed") {
		t.Errorf("expected the write failure to be named, got %q", stderr.String())
	}
}
