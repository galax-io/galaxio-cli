package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/galax-io/galaxio-cli/internal/report"
	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/gatling"
	"github.com/galax-io/parsec/gatling/run"
	"github.com/galax-io/parsec/model"
)

// reportCorpus is the frozen Gatling corpus next to the package that reads it;
// the command tests reach it rather than duplicating recordings.
var reportCorpus = filepath.Join("..", "..", "internal", "report", "testdata", "corpus", "gatling")

// reportFields parses the command's aligned key-value block, keeping the order
// the keys appeared in: the contract pins that order, so a test that only had
// the map could not see it change.
func reportFields(t *testing.T, stdout string) (map[string]string, []string) {
	t.Helper()

	fields := map[string]string{}
	order := []string{}

	for _, line := range strings.Split(strings.TrimSuffix(stdout, "\n"), "\n") {
		if line == "" {
			continue
		}

		if len(line) <= reportKeyWidth {
			t.Fatalf("report line %q is shorter than the key column", line)
		}

		key := strings.TrimSpace(line[:reportKeyWidth])
		fields[key] = strings.TrimSpace(line[reportKeyWidth:])
		order = append(order, key)
	}

	return fields, order
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

// writeRun writes a log into a fresh run directory and returns that directory.
func writeRun(t *testing.T, log []byte) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "simulation.log"), log, 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	return dir
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

// hostileRunLog returns a text recording whose identity fields carry bytes a
// terminal obeys: an escape sequence, a bell, a carriage return that erases the
// line already written, and a pair that is not UTF-8 at all.
func hostileRunLog(t *testing.T) []byte {
	t.Helper()

	header, body, err := reporttest.Split(corpusLog(t, "3.12.0"))
	if err != nil {
		t.Fatalf("split corpus log: %v", err)
	}

	lines := bytes.Split(header, []byte("\n"))
	patched := false

	for i, l := range lines {
		if !bytes.HasPrefix(l, []byte("RUN\t")) {
			continue
		}

		fields := bytes.Split(l, []byte("\t"))
		if len(fields) < 3 {
			t.Fatalf("the RUN record has %d fields, want the name and the id", len(fields))
		}

		fields[1] = []byte("\x1b[31mPWNED\x1b[0m\x07")
		fields[2] = []byte("io.x.Sim\rOVERWRITTEN\xff\xfe")
		lines[i] = bytes.Join(fields, []byte("\t"))
		patched = true
	}

	if !patched {
		t.Fatal("the corpus header has no RUN record to patch")
	}

	return append(bytes.Join(lines, []byte("\n")), body...)
}

// TestReportQuotesWhatALogWrote pins the report against a log the reader did not
// write. The identity fields are free text a tool put in the file, and a report
// that hands them to a terminal unchanged lets that file colour the output, ring
// the bell, or erase the line it is on and show a name the log does not contain.
func TestReportQuotesWhatALogWrote(t *testing.T) {
	t.Parallel()

	code, stdout, stderr := runCLI("report", "gatling", writeRun(t, hostileRunLog(t)))
	if code != exitOK {
		t.Fatalf("a hostile log exited %d, want %d: %s", code, exitOK, stderr)
	}

	for _, forbidden := range []string{"\x1b", "\r", "\x07", "\xff"} {
		if strings.Contains(stdout, forbidden) {
			t.Errorf("the report passed %q through from the log:\n%q", forbidden, stdout)
		}
	}

	if !utf8.ValidString(stdout) {
		t.Errorf("the report is not valid UTF-8:\n%q", stdout)
	}

	// Quoted, the bytes are still reported — the point is to render them, not
	// to drop what the log said.
	for _, want := range []string{`\x1b[31mPWNED`, `io.x.Sim\rOVERWRITTEN`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the report lost %q instead of quoting it:\n%s", want, stdout)
		}
	}

	// A log with nothing to escape is printed as it stands, so the ordinary
	// report never acquires quotes.
	_, plain, _ := runCLI("report", "gatling", filepath.Join(reportCorpus, "3.12.0"))
	if !strings.Contains(plain, "run         io.galaxio.parsec.corpus.CorpusSimulation\n") {
		t.Errorf("an ordinary run was quoted:\n%s", plain)
	}
}

// TestReportRejectsReportFormatsBeforeHelp pins -o against the invocation that
// used to slip past it. cobra answers --help before it reaches RunE, so a check
// stated there accepted the flag and printed help, exiting 0 for a format the
// command cannot produce — the success the reservation exists to prevent.
func TestReportRejectsReportFormatsBeforeHelp(t *testing.T) {
	t.Parallel()

	tests := [][]string{
		{"report", "--help", "-o", "stats"},
		{"report", "-h", "-o", "stats"},
		{"report", "--help", "-o", "bogus"},
		{"report", "gatling", filepath.Join(reportCorpus, "3.15.1"), "--help", "-o", "stats"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()

			code, stdout, stderr := runCLI(args...)
			if code != exitUsage {
				t.Fatalf("%v exited %d, want %d", args, code, exitUsage)
			}

			if strings.Contains(stdout, "Usage:") {
				t.Errorf("%v printed help for a format it cannot produce:\n%s", args, stdout)
			}

			if !strings.Contains(stderr, "output") {
				t.Errorf("%v reported %q, want it to name the flag", args, stderr)
			}
		})
	}
}

// TestReportRefusesAnEmptyPath drives the rule through runReport as well as the
// command, because the distinction it rests on — an argument that is absent
// against one that is present and empty — is one the empty string alone cannot
// make, and a rule only the cobra layer could state is one no seam can pin.
func TestReportRefusesAnEmptyPath(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"", "   "} {
		t.Run("through the command "+path, func(t *testing.T) {
			t.Parallel()

			code, stdout, stderr := runCLI("report", "gatling", path)
			if code != exitUsage {
				t.Fatalf("an empty path exited %d, want %d: %s", code, exitUsage, stderr)
			}

			if stdout != "" {
				t.Errorf("an empty path printed a report:\n%s", stdout)
			}

			if !strings.Contains(stderr, report.ErrNoPath.Error()) {
				t.Errorf("an empty path reported %q, want %q", stderr, report.ErrNoPath.Error())
			}
		})
	}

	t.Run("through the seam", func(t *testing.T) {
		t.Parallel()

		var stdout, stderr bytes.Buffer

		_, err := runReport(context.Background(), reportOptions{
			Tool: "gatling", Path: "", PathSet: true, Stdout: &stdout, Stderr: &stderr,
		})
		if !errors.Is(err, report.ErrNoPath) {
			t.Fatalf("a path given as empty returned %v, want %v", err, report.ErrNoPath)
		}

		var usage UsageError
		if !errors.As(err, &usage) {
			t.Errorf("a path given as empty returned %T, want a usage error", err)
		}
	})

	t.Run("an absent path still searches the default root", func(t *testing.T) {
		t.Parallel()

		// The same empty string, with PathSet false, must take the other
		// branch: this is the pair the distinction exists for. The package
		// directory holds no results root, so the search reports the root it
		// looked in — which is the evidence that it looked.
		var stdout, stderr bytes.Buffer

		_, err := runReport(context.Background(), reportOptions{
			Tool: "gatling", Path: "", Stdout: &stdout, Stderr: &stderr,
		})
		if errors.Is(err, report.ErrNoPath) {
			t.Fatalf("an absent path was refused as an empty one: %v", err)
		}

		if err == nil || !strings.Contains(err.Error(), run.DefaultResultsRoot) {
			t.Errorf("an absent path reported %v, want it to name %s", err, run.DefaultResultsRoot)
		}
	})
}

// TestReportStopsWhenTheContextIsCancelled drives the cancellation through the
// CLI, not through runReport, because the defect it guards was the wiring: the
// read polled a context nothing in the shipped binary could ever cancel.
func TestReportStopsWhenTheContextIsCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stdout, stderr bytes.Buffer

	code := executeContext(ctx, []string{"report", "gatling", filepath.Join(reportCorpus, "3.15.1")}, &stdout, &stderr)
	if code != exitRuntime {
		t.Fatalf("a cancelled read exited %d, want %d: %s", code, exitRuntime, stderr.String())
	}

	if stdout.String() != "" {
		t.Errorf("a cancelled read printed a report:\n%s", stdout.String())
	}

	for _, want := range []string{"simulation.log", context.Canceled.Error()} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("a cancelled read reported %q, want it to name %q", stderr.String(), want)
		}
	}
}

func TestReportReadsRun(t *testing.T) {
	tests := []struct {
		version  string
		format   string
		requests string
	}{
		{version: "3.12.0", format: "text", requests: "36 (18 ok, 18 ko)"},
		{version: "3.15.1", format: "binary", requests: "102 (84 ok, 18 ko)"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			dir := filepath.Join(reportCorpus, tt.version)

			code, stdout, stderr := runCLI("report", "gatling", dir)

			if code != exitOK {
				t.Fatalf("expected exit code %d, got %d; stderr: %s", exitOK, code, stderr)
			}

			if strings.Contains(stderr, "Error:") {
				t.Fatalf("unexpected error on stderr: %s", stderr)
			}

			fields, order := reportFields(t, stdout)

			if got, want := fields["tool"], "gatling "+tt.version; got != want {
				t.Errorf("tool = %q, want %q", got, want)
			}

			if got, want := fields["log"], tt.format+", "+filepath.Join(dir, "simulation.log"); got != want {
				t.Errorf("log = %q, want %q", got, want)
			}

			if got := fields["requests"]; got != tt.requests {
				t.Errorf("requests = %q, want %q (Gatling's own console summary)", got, tt.requests)
			}

			for key, want := range map[string]string{
				"groups":   "12 traversals",
				"users":    "12 events",
				"errors":   "6",
				"found by": "path",
			} {
				if got := fields[key]; got != want {
					t.Errorf("%s = %q, want %q", key, got, want)
				}
			}

			expectedOrder := []string{"run", "id", "started", "tool", "log", "found by", "span", "requests", "groups", "users", "errors", "assertions"}
			if !slices.Equal(order, expectedOrder) {
				t.Errorf("line order = %v, want %v — contracts/cli.md pins it", order, expectedOrder)
			}

			if fields["span"] == "" || fields["started"] == "" || fields["run"] == "" || fields["id"] == "" {
				t.Errorf("report omits a fact the run recorded: %q", stdout)
			}

			if _, ok := fields["absent"]; ok {
				t.Errorf("absent is a --verbose fact and must not appear by default")
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
		t.Fatalf("the log path and its directory produced different reports")
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

	if out.Location.Dir != dir || out.Location.Found != run.FoundByPath {
		t.Errorf("Location = %+v, want the named run at %q", out.Location, dir)
	}

	if out.Run.ToolVersion != "3.15.1" {
		t.Errorf("Run.ToolVersion = %q, want 3.15.1", out.Run.ToolVersion)
	}

	if tally := out.Summary.Tally; tally.Requests != 102 || tally.Successes != 84 || tally.Failures != 18 {
		t.Errorf("Tally = %+v, want 102 requests, 84 ok, 18 ko", tally)
	}
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

	if stdout != "" {
		t.Errorf("--quiet must suppress the report, got %q", stdout)
	}
}

func TestReportFindsTheLatestRun(t *testing.T) {
	lastrun := filepath.Join(reportCorpus, "lastrun", "results")

	tests := []struct {
		name          string
		args          func(t *testing.T) []string
		expectedFound string
	}{
		{
			name:          "a results root with lastRun.txt",
			args:          func(*testing.T) []string { return []string{"report", "gatling", lastrun} },
			expectedFound: "lastRun.txt",
		},
		{
			name: "a results root without a marker",
			args: func(t *testing.T) []string {
				root := filepath.Join(t.TempDir(), "results")
				copyRun(t, root, filepath.Join(lastrun, "corpussimulation-20260909022556930"), "corpussimulation-20260909022556930")

				return []string{"report", "gatling", root}
			},
			expectedFound: "newest",
		},
		{
			name:          "a run named explicitly",
			args:          func(*testing.T) []string { return []string{"report", "gatling", filepath.Join(reportCorpus, "3.15.1")} },
			expectedFound: "path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runCLI(tt.args(t)...)

			if code != exitOK {
				t.Fatalf("expected exit code %d, got %d; stderr: %s", exitOK, code, stderr)
			}

			fields, _ := reportFields(t, stdout)
			if got := fields["found by"]; got != tt.expectedFound {
				t.Errorf("found by = %q, want %q", got, tt.expectedFound)
			}
		})
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

	fields, _ := reportFields(t, stdout)
	if fields["found by"] != "newest" {
		t.Errorf("found by = %q, want newest", fields["found by"])
	}

	if fields["requests"] != "36 (18 ok, 18 ko)" {
		t.Errorf("requests = %q, want the 3.12.0 run's counts", fields["requests"])
	}
}

func TestReportVerboseNamesWhatTheSourceCannotRecord(t *testing.T) {
	code, stdout, _ := runCLI("report", "gatling", filepath.Join(reportCorpus, "3.15.1"), "--verbose")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}

	fields, _ := reportFields(t, stdout)
	absent := fields["absent"]
	for _, want := range []string{"sample response code", "sample scenario"} {
		if !strings.Contains(absent, want) {
			t.Errorf("absent = %q, want it to name %q", absent, want)
		}
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
		// An unknown name anywhere in the list outranks a reserved one.
		{output: "stats,bogus", expected: `unknown report format "bogus": known formats: stats, global_stats, yml`},
		// An explicitly empty -o names no format, and must not take a different
		// branch from -o " ".
		{output: "", expected: `unknown report format "": known formats: stats, global_stats, yml`},
		{output: " ", expected: `unknown report format "": known formats: stats, global_stats, yml`},
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

// patchedCorpusLog is the 3.13.1 recording with its six version bytes at offset
// 5 replaced, which is what parsec's gate reads first.
func patchedCorpusLog(t *testing.T, version string) []byte {
	t.Helper()

	data := corpusLog(t, "3.13.1")
	if got := string(data[5:11]); got != "3.13.1" {
		t.Fatalf("corpus log carries version %q at offset 5, want 3.13.1", got)
	}

	if len(version) != 6 {
		t.Fatalf("patch version %q must be six bytes", version)
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
		name          string
		args          func(t *testing.T) []string
		code          int
		wantReport    bool
		stderrHas     []string
		stderrHasPath bool
		stderrLacks   []string
	}{
		{
			name: "a version below the supported range",
			args: func(t *testing.T) []string {
				return []string{"report", "gatling", writeRun(t, []byte("RUN\tio.x.Sim\tsim\t1700000000000\t \t3.10.0\n"))}
			},
			code:      exitRuntime,
			stderrHas: []string{"version 3.10.0 is below the supported range 3.11.5 through 3.12.0"},
		},
		{
			name: "3.13.0 is refused",
			args: func(t *testing.T) []string {
				return []string{"report", "gatling", writeRun(t, patchedCorpusLog(t, "3.13.0"))}
			},
			code:      exitRuntime,
			stderrHas: []string{"3.13.0", "3.13.1"},
		},
		{
			name: "a log cut short still reports what it held",
			args: func(t *testing.T) []string {
				return []string{"report", "gatling", writeRun(t, corpusLog(t, "3.15.1")[:2000])}
			},
			code:       exitRuntime,
			wantReport: true,
			stderrHas:  []string{"the log is cut short", "the run was not read to the end"},
		},
		{
			name: "a path that does not exist",
			args: func(t *testing.T) []string {
				return []string{"report", "gatling", filepath.Join(t.TempDir(), "nonexistent")}
			},
			code:        exitRuntime,
			stderrHas:   []string{"cannot read ", "nonexistent"},
			stderrLacks: []string{"no Gatling run"},
		},
		{
			name:      "no run under the directory",
			args:      func(t *testing.T) []string { return []string{"report", "gatling", t.TempDir()} },
			code:      exitRuntime,
			stderrHas: []string{"no Gatling run under "},
		},
		{
			name: "not a Gatling log",
			args: func(t *testing.T) []string {
				return []string{"report", "gatling", writeRun(t, []byte("<!DOCTYPE html>\n<html></html>\n"))}
			},
			code:          exitRuntime,
			stderrHas:     []string{"not a Gatling simulation.log"},
			stderrHasPath: true,
		},
		{
			// Unlike a truncation, the records before a damaged one are not a
			// result: parsec says no total may be derived from them, so the
			// command reports nothing rather than a count it cannot stand behind.
			name:      "a damaged log reports nothing",
			args:      func(t *testing.T) []string { return []string{"report", "gatling", writeRun(t, damagedTextLog(t))} },
			code:      exitRuntime,
			stderrHas: []string{"the run was not read completely"},
		},
		{
			name:      "three positional arguments",
			args:      func(*testing.T) []string { return []string{"report", "gatling", "a", "b"} },
			code:      exitUsage,
			stderrHas: []string{"accepts at most 2 arg(s), received 3"},
		},
		{
			name: "an unknown flag",
			args: func(*testing.T) []string {
				return []string{"report", "gatling", filepath.Join(reportCorpus, "3.15.1"), "--bogus"}
			},
			code:      exitUsage,
			stderrHas: []string{"unknown flag: --bogus"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args(t)

			code, stdout, stderr := runCLI(args...)

			if code != tt.code {
				t.Fatalf("expected exit code %d, got %d; stderr: %s", tt.code, code, stderr)
			}

			// FR-021: the message names what was read. The run directory alone
			// is not enough — without this the log-path wrap can be deleted and
			// every case still passes.
			if tt.stderrHasPath {
				log := filepath.Join(args[len(args)-1], "simulation.log")
				if !strings.Contains(stderr, log) {
					t.Errorf("expected stderr to name the log it read, %q, got %q", log, stderr)
				}
			}

			if tt.wantReport {
				if fields, _ := reportFields(t, stdout); fields["requests"] == "" {
					t.Errorf("expected the counts of what was read, got %q", stdout)
				}
			} else if stdout != "" {
				t.Errorf("expected empty stdout, got %q", stdout)
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

// failingWriter fails every write, like a reader that went away.
type failingWriter struct{ calls int }

var errPipeClosed = errors.New("pipe closed")

func (w *failingWriter) Write([]byte) (int, error) {
	w.calls++

	return 0, errPipeClosed
}

func TestReportReportsAWriteFailureOnce(t *testing.T) {
	var stderr bytes.Buffer

	stdout := &failingWriter{}

	code := execute([]string{"report", "gatling", filepath.Join(reportCorpus, "3.15.1")}, stdout, &stderr)

	if code != exitRuntime {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", exitRuntime, code, stderr.String())
	}

	if stdout.calls != 1 {
		t.Errorf("stdout saw %d writes, want exactly one", stdout.calls)
	}

	if got := strings.Count(stderr.String(), "Error:"); got != 1 {
		t.Errorf("expected exactly one Error line on stderr, got %d: %s", got, stderr.String())
	}

	if !strings.Contains(stderr.String(), "writing report: pipe closed") {
		t.Errorf("expected the write failure to be named, got %q", stderr.String())
	}
}

// TestReportNamesTheLogWhenTheWriteAlsoFails covers the combination the test
// above cannot: an intact run leaves nothing to join, so the scan's own error
// never travelled beside the write's. When it does, it must still name the log
// it came from — the same log the failure names when it travels alone.
func TestReportNamesTheLogWhenTheWriteAlsoFails(t *testing.T) {
	t.Parallel()

	cutShort := writeRun(t, corpusLog(t, "3.15.1")[:2000])

	var alone bytes.Buffer

	if code := execute([]string{"report", "gatling", cutShort}, &bytes.Buffer{}, &alone); code != exitRuntime {
		t.Fatalf("a truncated run exited %d, want %d", code, exitRuntime)
	}

	var together bytes.Buffer

	stdout := &failingWriter{}
	if code := execute([]string{"report", "gatling", cutShort}, stdout, &together); code != exitRuntime {
		t.Fatalf("a truncated run with a failing stdout exited %d, want %d", code, exitRuntime)
	}

	for _, want := range []string{cutShort, "the log is cut short", "writing report: pipe closed"} {
		if !strings.Contains(together.String(), want) {
			t.Errorf("the joined failure reported %q, want it to contain %q", together.String(), want)
		}
	}

	// The truncation reads the same whichever way it travelled: the write
	// failure is added to it, never substituted for the log's name.
	if !strings.Contains(alone.String(), cutShort) {
		t.Fatalf("the lone failure does not name the log: %q", alone.String())
	}
}

func TestFormatReportOmitsWhatTheSourceDidNotRecord(t *testing.T) {
	t.Parallel()

	// A run the source could not place in time, with no description and an
	// outcome it lost: every one of those is left out or named, never shown as
	// an empty string or a zero.
	out := reportOutput{
		Location: run.Location{Log: "/x/simulation.log", Found: run.FoundByPath},
		Run:      model.Run{ID: "sim", Name: "io.x.Sim", Tool: "gatling", ToolVersion: "3.12.0"},
		Format:   gatling.FormatText,
		Summary:  report.Summary{Tally: report.Tally{Requests: 3, Successes: 1, Failures: 1, Unknown: 1}},
	}

	got := formatReport(out, false)

	for _, unwanted := range []string{"started", "span", "description"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("report names %q for a run that recorded none:\n%s", unwanted, got)
		}
	}

	// The identity lines obey the same rule. A source that recorded no name or
	// no id must not be shown as one that recorded an empty string — which is
	// what a bare key and a trailing space say.
	t.Run("a run with no identity", func(t *testing.T) {
		t.Parallel()

		anonymous := out
		anonymous.Run.ID, anonymous.Run.Name = "", ""

		bare := formatReport(anonymous, false)

		_, keys := reportFields(t, bare)
		for _, key := range []string{"run", "id"} {
			if slices.Contains(keys, key) {
				t.Errorf("report shows %q empty for a source that did not record it:\n%q", key, bare)
			}
		}

		if !strings.Contains(bare, "tool        gatling 3.12.0") {
			t.Errorf("report lost the facts the source did record:\n%s", bare)
		}
	})

	if !strings.Contains(got, "unknown     1 request whose outcome the source lost") {
		t.Errorf("report does not name the request whose outcome the source lost:\n%s", got)
	}

	// And with the instants present, both lines appear.
	out.Run.Start = time.UnixMilli(1700000000000).UTC()
	out.Summary.Tally.Bounds.Extend(&model.Item{Kind: model.ItemUser, User: model.UserEvent{Kind: model.UserStart, At: time.UnixMilli(1700000000000).UTC()}})
	out.Summary.Tally.Bounds.Extend(&model.Item{Kind: model.ItemUser, User: model.UserEvent{Kind: model.UserEnd, At: time.UnixMilli(1700000004000).UTC()}})

	got = formatReport(out, false)
	for _, want := range []string{"started     2023-11-14T22:13:20.000Z", "span        2023-11-14T22:13:20.000Z", "(4s)"} {
		if !strings.Contains(got, want) {
			t.Errorf("report omits %q although the run recorded it:\n%s", want, got)
		}
	}
}

// TestReportLineWidth pins the block's alignment to what the report actually
// renders, not to a second copy of its keys.
//
// The earlier form re-typed the key list and compared its longest entry to
// reportKeyWidth, so the drift it named — a key added to formatReport and not
// to the test — passed green while the block misaligned and reportFields sliced
// mid-key. Render every line the report can print and assert the property that
// matters: every value begins in the same column, the one reportFields reads.
func TestReportLineWidth(t *testing.T) {
	t.Parallel()

	out := reportOutput{
		Location: run.Location{Log: "/x/simulation.log", Found: run.FoundByPath},
		Run: model.Run{
			ID:          "sim",
			Name:        "io.x.Sim",
			Tool:        "gatling",
			ToolVersion: "3.12.0",
			Start:       time.UnixMilli(1700000000000).UTC(),
			Assertions:  []string{"payload"},
		},
		Format: gatling.FormatText,
		Summary: report.Summary{Tally: report.Tally{
			Requests: 4, Successes: 1, Failures: 1, Unknown: 1,
			Groups: 1, Users: 2, Errors: 1, Assertions: 1, Other: 1,
		}},
	}
	out.Summary.Tally.Bounds.Extend(&model.Item{Kind: model.ItemUser, User: model.UserEvent{Kind: model.UserStart, At: time.UnixMilli(1700000000000).UTC()}})
	out.Summary.Tally.Bounds.Extend(&model.Item{Kind: model.ItemUser, User: model.UserEvent{Kind: model.UserEnd, At: time.UnixMilli(1700000004000).UTC()}})

	got := formatReport(out, true)

	_, keys := reportFields(t, got)
	if len(keys) < 14 {
		t.Fatalf("the fixture renders %d lines (%v); it is meant to turn on every one", len(keys), keys)
	}

	for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		if len(line) <= reportKeyWidth+1 {
			t.Errorf("line %q is too short to hold a key and a value", line)

			continue
		}

		// %-*s pads a short key and does not truncate a long one, so a key as
		// wide as the column still aligns and a wider one pushes its value out.
		if line[reportKeyWidth] != ' ' || line[reportKeyWidth+1] == ' ' {
			t.Errorf("line %q does not start its value at column %d: a key is wider than reportKeyWidth = %d", line, reportKeyWidth+1, reportKeyWidth)
		}
	}
}
