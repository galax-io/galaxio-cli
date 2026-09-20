package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type legacyRequestCounts struct {
	Total int `json:"total"`
	OK    int `json:"ok"`
	KO    int `json:"ko"`
}

type legacyStatsFile struct {
	Name             string              `json:"name"`
	NumberOfRequests legacyRequestCounts `json:"numberOfRequests"`
}

func readJSONFile[T any](t *testing.T, path string) T {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}

	return value
}

func TestReportExportCorpus(t *testing.T) {
	tests := []struct {
		version       string
		total, ok, ko int
	}{
		{version: "3.11.5", total: 36, ok: 18, ko: 18},
		{version: "3.12.0", total: 36, ok: 18, ko: 18},
		{version: "3.13.1", total: 102, ok: 84, ko: 18},
		{version: "3.14.9", total: 102, ok: 84, ko: 18},
		{version: "3.15.1", total: 102, ok: 84, ko: 18},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			dir := writeRun(t, corpusLog(t, tt.version))
			code, stdout, stderr := runCLI("report", "gatling", dir, "-o", "global_stats")
			if code != exitOK || stdout != "" || stderr != "" {
				t.Fatalf("export = %d, stdout %q, stderr %q", code, stdout, stderr)
			}

			global := readJSONFile[legacyStatsFile](t, filepath.Join(dir, "js", "global_stats.json"))
			wantCounts := legacyRequestCounts{Total: tt.total, OK: tt.ok, KO: tt.ko}
			if global.Name != "All Requests" || global.NumberOfRequests != wantCounts {
				t.Fatalf("global stats = %+v, want name All Requests and %+v", global, wantCounts)
			}
		})
	}
}

func TestReportExportValidation(t *testing.T) {
	dir := writeRun(t, corpusLog(t, "3.15.1"))
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "unknown", args: []string{"-o", "json"}, want: "unknown report format"},
		{name: "reserved stats", args: []string{"-o", "stats"}, want: "not available yet"},
		{name: "reserved yml", args: []string{"-o", "yml"}, want: "not available yet"},
		{name: "wrong percentile count", args: []string{"-o", "global_stats", "--percentiles", "50,95"}, want: "four distinct percentile ranks"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"report", "gatling", dir}, tt.args...)
			code, stdout, stderr := runCLI(args...)
			if code != exitUsage || stdout != "" || !strings.Contains(stderr, tt.want) {
				t.Fatalf("validation = %d, %q, %q; want %q", code, stdout, stderr, tt.want)
			}
		})
	}
}
