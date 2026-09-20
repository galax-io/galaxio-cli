package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
	MinResponseTime  legacyRequestCounts `json:"minResponseTime"`
	MaxResponseTime  legacyRequestCounts `json:"maxResponseTime"`
}

type legacyTreeFile struct {
	Type     string                     `json:"type"`
	Name     string                     `json:"name"`
	Stats    json.RawMessage            `json:"stats"`
	Contents map[string]*legacyTreeFile `json:"contents"`
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
			code, stdout, stderr := runCLI("report", "gatling", dir, "-o", "stats,global_stats")
			if code != exitOK || stdout != "" || stderr != "" {
				t.Fatalf("export = %d, stdout %q, stderr %q", code, stdout, stderr)
			}

			globalPath := filepath.Join(dir, "js", "global_stats.json")
			global := readJSONFile[legacyStatsFile](t, globalPath)
			wantCounts := legacyRequestCounts{Total: tt.total, OK: tt.ok, KO: tt.ko}
			if global.Name != "All Requests" || global.NumberOfRequests != wantCounts {
				t.Fatalf("global stats = %+v, want name All Requests and %+v", global, wantCounts)
			}

			root := readJSONFile[legacyTreeFile](t, filepath.Join(dir, "js", "stats.json"))
			if root.Type != "GROUP" || root.Name != "All Requests" || len(root.Contents) == 0 {
				t.Fatalf("tree root = %+v", root)
			}

			var rootStats, globalStats any
			if err := json.Unmarshal(root.Stats, &rootStats); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(globalPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &globalStats); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(rootStats, globalStats) {
				t.Fatal("stats.json root statistics differ from global_stats.json")
			}
		})
	}
}

func TestReportExportSelectionAndOverwrite(t *testing.T) {
	dir := writeRun(t, corpusLog(t, "3.15.1"))
	path := filepath.Join(dir, "js", "global_stats.json")

	if code, stdout, stderr := runCLI("report", "gatling", dir, "-o", "global_stats"); code != exitOK || stdout != "" || stderr != "" {
		t.Fatalf("first export = %d, %q, %q", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "js", "stats.json")); !os.IsNotExist(err) {
		t.Fatalf("unselected stats.json exists: %v", err)
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := runCLI("report", "gatling", dir, "-o", "global_stats"); code != exitRuntime || stdout != "" || !strings.Contains(stderr, path) {
		t.Fatalf("collision = %d, %q, %q", code, stdout, stderr)
	}
	after, err := os.ReadFile(path)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("collision changed the existing file")
	}

	if code, stdout, stderr := runCLI("report", "gatling", dir, "-o", "global_stats", "--overwrite"); code != exitOK || stdout != "" || stderr != "" {
		t.Fatalf("overwrite = %d, %q, %q", code, stdout, stderr)
	}

	statsPath := filepath.Join(dir, "js", "stats.json")
	sentinel := []byte("unselected")
	if err := os.WriteFile(statsPath, sentinel, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := runCLI("report", "gatling", dir, "-o", "global_stats", "--overwrite"); code != exitOK || stdout != "" || stderr != "" {
		t.Fatalf("selected-only overwrite = %d, %q, %q", code, stdout, stderr)
	}
	got, err := os.ReadFile(statsPath)
	if err != nil || !reflect.DeepEqual(got, sentinel) {
		t.Fatalf("unselected stats.json = %q, %v", got, err)
	}

	preflightDir := writeRun(t, corpusLog(t, "3.15.1"))
	preflightJS := filepath.Join(preflightDir, "js")
	if err := os.Mkdir(preflightJS, 0o755); err != nil {
		t.Fatal(err)
	}
	preflightGlobal := filepath.Join(preflightJS, "global_stats.json")
	if err := os.WriteFile(preflightGlobal, sentinel, 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI("report", "gatling", preflightDir, "-o", "stats,global_stats")
	if code != exitRuntime || stdout != "" || !strings.Contains(stderr, preflightGlobal) {
		t.Fatalf("combined preflight = %d, %q, %q", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(preflightJS, "stats.json")); !os.IsNotExist(err) {
		t.Fatalf("combined preflight created stats.json: %v", err)
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
		{name: "reserved yml", args: []string{"-o", "yml"}, want: "not available yet"},
		{name: "overwrite without output", args: []string{"--overwrite"}, want: "requires -o"},
		{name: "wrong percentile count", args: []string{"-o", "stats", "--percentiles", "50,95"}, want: "four distinct percentile ranks"},
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

func TestReportExportFailurePublishesNothing(t *testing.T) {
	tests := []struct {
		name string
		log  []byte
	}{
		{name: "damaged", log: damagedTextLog(t)},
		{name: "truncated", log: corpusLog(t, "3.15.1")[:2000]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeRun(t, tt.log)
			code, stdout, stderr := runCLI("report", "gatling", dir, "-o", "stats,global_stats")
			if code != exitRuntime || stdout != "" || stderr == "" {
				t.Fatalf("failed export = %d, %q, %q", code, stdout, stderr)
			}
			for _, name := range []string{"stats.json", "global_stats.json"} {
				if _, err := os.Stat(filepath.Join(dir, "js", name)); !os.IsNotExist(err) {
					t.Fatalf("failed export published %s: %v", name, err)
				}
			}
		})
	}
}

func TestReportExportPreservesDecodedNames(t *testing.T) {
	name := `say "hi" \\ 雪`
	log := []byte("RUN\tsim\tsim\t1000\t \t3.12.0\n" +
		"REQUEST\t\t" + name + "\t1000\t1010\tOK\t \n")
	dir := writeRun(t, log)
	code, stdout, stderr := runCLI("report", "gatling", dir, "-o", "stats")
	if code != exitOK || stdout != "" || stderr != "" {
		t.Fatalf("export = %d, %q, %q", code, stdout, stderr)
	}

	root := readJSONFile[legacyTreeFile](t, filepath.Join(dir, "js", "stats.json"))
	found := false
	var visit func(*legacyTreeFile)
	visit = func(node *legacyTreeFile) {
		if node.Name == name {
			found = true
		}
		for _, child := range node.Contents {
			visit(child)
		}
	}
	visit(&root)
	if !found {
		t.Fatalf("decoded request name %q was not preserved", name)
	}
}

func TestReportExportReproducible(t *testing.T) {
	dir := writeRun(t, corpusLog(t, "3.15.1"))
	args := []string{
		"report", "gatling", dir, "-o", "stats,global_stats",
		"--percentiles", "25,50,75,99", "--bounds", "5,1000",
	}
	code, stdout, stderr := runCLI(args...)
	if code != exitOK || stdout != "" || stderr != "" {
		t.Fatalf("first export = %d, %q, %q", code, stdout, stderr)
	}

	jsDir := filepath.Join(dir, "js")
	first := make(map[string][]byte, 2)
	for _, name := range []string{"stats.json", "global_stats.json"} {
		data, err := os.ReadFile(filepath.Join(jsDir, name))
		if err != nil {
			t.Fatal(err)
		}
		first[name] = data
	}

	args = []string{
		"report", "gatling", dir, "-o", "global_stats,stats,global_stats", "--overwrite",
		"--percentiles", "25,50,75,99", "--bounds", "5,1000",
	}
	code, stdout, stderr = runCLI(args...)
	if code != exitOK || stdout != "" || stderr != "" {
		t.Fatalf("repeated export = %d, %q, %q", code, stdout, stderr)
	}
	for name, want := range first {
		got, err := os.ReadFile(filepath.Join(jsDir, name))
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("repeated %s differs: %v", name, err)
		}
	}
	entries, err := os.ReadDir(jsDir)
	if err != nil || len(entries) != 2 {
		t.Fatalf("js inventory = %v, %v; want two selected files", entries, err)
	}

	statsOnlyDir := writeRun(t, corpusLog(t, "3.12.0"))
	code, stdout, stderr = runCLI("report", "gatling", statsOnlyDir, "-o", "stats")
	if code != exitOK || stdout != "" || stderr != "" {
		t.Fatalf("stats-only export = %d, %q, %q", code, stdout, stderr)
	}
	entries, err = os.ReadDir(filepath.Join(statsOnlyDir, "js"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "stats.json" {
		t.Fatalf("stats-only inventory = %v, %v", entries, err)
	}
}

func TestReportExportEmptyOutcomeColumns(t *testing.T) {
	tests := []struct {
		name    string
		outcome string
		ok, ko  int
	}{
		{name: "no failures", outcome: "OK", ok: 1},
		{name: "no successes", outcome: "KO", ko: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := []byte("RUN\tsim\tsim\t1000\t \t3.12.0\n" +
				"REQUEST\t\trequest\t1000\t1010\t" + tt.outcome + "\t \n")
			dir := writeRun(t, log)
			code, stdout, stderr := runCLI("report", "gatling", dir, "-o", "global_stats")
			if code != exitOK || stdout != "" || stderr != "" {
				t.Fatalf("export = %d, %q, %q", code, stdout, stderr)
			}

			global := readJSONFile[legacyStatsFile](t, filepath.Join(dir, "js", "global_stats.json"))
			if global.NumberOfRequests != (legacyRequestCounts{Total: 1, OK: tt.ok, KO: tt.ko}) {
				t.Fatalf("counts = %+v", global.NumberOfRequests)
			}
			if tt.ok == 0 && (global.MinResponseTime.OK != 0 || global.MaxResponseTime.OK != 0) {
				t.Fatalf("empty OK timings = %+v / %+v", global.MinResponseTime, global.MaxResponseTime)
			}
			if tt.ko == 0 && (global.MinResponseTime.KO != 0 || global.MaxResponseTime.KO != 0) {
				t.Fatalf("empty KO timings = %+v / %+v", global.MinResponseTime, global.MaxResponseTime)
			}
		})
	}
}
