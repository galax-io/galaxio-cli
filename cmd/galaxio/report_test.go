package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

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
