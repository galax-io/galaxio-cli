package report

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/galax-io/parsec/gatling"
	"github.com/galax-io/parsec/gatling/run"
)

var corpusDir = filepath.Join("testdata", "corpus", "gatling")

// copyLog copies one corpus simulation.log into dir, creating dir.
func copyLog(t *testing.T, src, dir string) {
	t.Helper()

	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "simulation.log"), data, 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
}

// copyResultsRoot copies the three lastrun corpus runs into a fresh root,
// pins their modification times so "newest" is deterministic, and returns the
// root together with the run the times make newest.
func copyResultsRoot(t *testing.T) (root, newest string) {
	t.Helper()

	root = filepath.Join(t.TempDir(), "results")
	src := filepath.Join(corpusDir, "lastrun", "results")
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}

	// Oldest id first by name; the modification times below invert that order
	// on purpose, so a reader that picks the newest by time cannot pass by
	// picking the newest by name.
	stamp := time.Now().Add(-3 * time.Hour)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		copyLog(t, filepath.Join(src, e.Name(), "simulation.log"), dir)
		stamp = stamp.Add(-time.Hour)
		for _, p := range []string{filepath.Join(dir, "simulation.log"), dir} {
			if err := os.Chtimes(p, stamp, stamp); err != nil {
				t.Fatalf("chtimes %s: %v", p, err)
			}
		}
		if newest == "" {
			newest = e.Name()
		}
	}
	return root, newest
}

func TestLocate(t *testing.T) {
	t.Parallel()

	runDir := filepath.Join(corpusDir, "3.15.1")

	tests := []struct {
		name          string
		path          func(t *testing.T) string
		expectedDir   func(path string) string
		expectedFound run.FoundBy
	}{
		{
			name:          "a run directory is the run",
			path:          func(*testing.T) string { return runDir },
			expectedDir:   func(path string) string { return path },
			expectedFound: run.FoundByPath,
		},
		{
			name:          "the log itself is the run",
			path:          func(*testing.T) string { return filepath.Join(runDir, "simulation.log") },
			expectedDir:   func(string) string { return runDir },
			expectedFound: run.FoundByPath,
		},
		{
			name: "a directory named simulation.log is examined for a log inside it",
			path: func(t *testing.T) string {
				dir := filepath.Join(t.TempDir(), "simulation.log")
				copyLog(t, filepath.Join(runDir, "simulation.log"), dir)
				return dir
			},
			expectedDir:   func(path string) string { return path },
			expectedFound: run.FoundByPath,
		},
		{
			name: "a run directory holding run directories beneath it is a run, not a root",
			path: func(t *testing.T) string {
				dir := filepath.Join(t.TempDir(), "run")
				copyLog(t, filepath.Join(runDir, "simulation.log"), dir)
				copyLog(t, filepath.Join(runDir, "simulation.log"), filepath.Join(dir, "nested"))
				return dir
			},
			expectedDir:   func(path string) string { return path },
			expectedFound: run.FoundByPath,
		},
		{
			name:          "a results root with lastRun.txt yields the run the marker names",
			path:          func(*testing.T) string { return filepath.Join(corpusDir, "lastrun", "results") },
			expectedDir:   func(path string) string { return filepath.Join(path, "corpussimulation-20260909022708912") },
			expectedFound: run.FoundByLastRun,
		},
		{
			name: "a results root without a marker yields the most recently modified run",
			path: func(t *testing.T) string {
				root, _ := copyResultsRoot(t)
				return root
			},
			expectedDir:   func(path string) string { return filepath.Join(path, "corpussimulation-20260909022556930") },
			expectedFound: run.FoundByNewest,
		},
		{
			name: "a marker naming a deleted run is ignored",
			path: func(t *testing.T) string {
				root, _ := copyResultsRoot(t)
				if err := os.WriteFile(filepath.Join(root, "lastRun.txt"), []byte("corpussimulation-20991231235959999\n"), 0o644); err != nil {
					t.Fatalf("write lastRun.txt: %v", err)
				}
				return root
			},
			expectedDir:   func(path string) string { return filepath.Join(path, "corpussimulation-20260909022556930") },
			expectedFound: run.FoundByNewest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := tt.path(t)
			loc, err := Locate(path)
			if err != nil {
				t.Fatalf("Locate(%q): %v", path, err)
			}
			if expected := tt.expectedDir(path); loc.Dir != expected {
				t.Errorf("Dir = %q, want %q", loc.Dir, expected)
			}
			if expected := filepath.Join(loc.Dir, "simulation.log"); loc.Log != expected {
				t.Errorf("Log = %q, want %q", loc.Log, expected)
			}
			if loc.Found != tt.expectedFound {
				t.Errorf("Found = %v, want %v", loc.Found, tt.expectedFound)
			}
		})
	}
}

func TestLocateNotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := Locate(dir)

	var notFound *run.NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Locate(empty dir) = %v, want *run.NotFoundError", err)
	}
	if notFound.Dir != dir {
		t.Fatalf("NotFoundError.Dir = %q, want %q", notFound.Dir, dir)
	}
	if !strings.Contains(err.Error(), dir) {
		t.Fatalf("error %q does not name the searched directory", err)
	}
}

func TestLocateReadError(t *testing.T) {
	t.Parallel()

	t.Run("a path that does not exist", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "nonexistent")
		_, err := Locate(path)

		var readErr *ReadError
		if !errors.As(err, &readErr) {
			t.Fatalf("Locate(missing) = %v, want *ReadError", err)
		}
		if readErr.Path != path {
			t.Errorf("ReadError.Path = %q, want %q", readErr.Path, path)
		}
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("error %v does not wrap fs.ErrNotExist", err)
		}
		if !strings.HasPrefix(err.Error(), "cannot read "+path+": ") {
			t.Errorf("error %q does not read as a read failure on the path", err)
		}
		var notFound *run.NotFoundError
		if errors.As(err, &notFound) {
			t.Errorf("a missing path must not be reported as an absence of runs")
		}
	})

	t.Run("a directory that cannot be read", func(t *testing.T) {
		t.Parallel()

		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("permission bits are not enforced for this user")
		}
		dir := filepath.Join(t.TempDir(), "locked")
		copyLog(t, filepath.Join(corpusDir, "3.15.1", "simulation.log"), filepath.Join(dir, "run"))
		if err := os.Chmod(dir, 0); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

		_, err := Locate(dir)

		var readErr *ReadError
		if !errors.As(err, &readErr) {
			t.Fatalf("Locate(unreadable) = %v, want *ReadError", err)
		}
		if !errors.Is(err, fs.ErrPermission) {
			t.Errorf("error %v does not wrap fs.ErrPermission", err)
		}
		var notFound *run.NotFoundError
		if errors.As(err, &notFound) {
			t.Errorf("an unreadable directory must not be reported as an absence of runs")
		}
	})
}

func TestLocateDefaultRoot(t *testing.T) {
	project := t.TempDir()
	copyLog(t, filepath.Join(corpusDir, "3.12.0", "simulation.log"), filepath.Join(project, run.DefaultResultsRoot, "corpussimulation-20260903000000000"))
	t.Chdir(project)

	loc, err := Locate("")
	if err != nil {
		t.Fatalf("Locate(\"\"): %v", err)
	}
	if expected := filepath.Join(run.DefaultResultsRoot, "corpussimulation-20260903000000000"); loc.Dir != expected {
		t.Errorf("Dir = %q, want %q", loc.Dir, expected)
	}
	if loc.Found != run.FoundByNewest {
		t.Errorf("Found = %v, want %v", loc.Found, run.FoundByNewest)
	}
}

// writeLog writes a synthetic log into a fresh directory and returns its
// location, so that a test can exercise parsec's version gate without a
// recording for every version.
func writeLog(t *testing.T, data []byte) run.Location {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "simulation.log"), data, 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	loc, err := Locate(dir)
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	return loc
}

// textHeader is the RUN line of a text log claiming the given version, with
// nothing after it.
func textHeader(version string) []byte {
	return []byte("RUN\tio.x.Sim\tsim\t1700000000000\t \t" + version + "\n")
}

// patchedBinaryLog returns the 3.13.1 corpus log with the six version bytes at
// offset 5 replaced, which is exactly what parsec's gate reads first.
func patchedBinaryLog(t *testing.T, version string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(corpusDir, "3.13.1", "simulation.log"))
	if err != nil {
		t.Fatalf("read corpus log: %v", err)
	}
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

func TestOpen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		format  gatling.Format
	}{
		{version: "3.11.5", format: gatling.FormatText},
		{version: "3.12.0", format: gatling.FormatText},
		{version: "3.13.1", format: gatling.FormatBinary},
		{version: "3.14.9", format: gatling.FormatBinary},
		{version: "3.15.1", format: gatling.FormatBinary},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()

			loc, err := Locate(filepath.Join(corpusDir, tt.version))
			if err != nil {
				t.Fatalf("Locate: %v", err)
			}
			src, err := Open(loc)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			defer src.Close()

			if src.Location != loc {
				t.Errorf("Location = %+v, want %+v", src.Location, loc)
			}
			if src.Format != tt.format {
				t.Errorf("Format = %v, want %v", src.Format, tt.format)
			}
			run := src.Reader.Run()
			if run.ToolVersion != tt.version {
				t.Errorf("ToolVersion = %q, want %q", run.ToolVersion, tt.version)
			}
			if len(run.Warnings) != 0 {
				t.Errorf("a recorded version must raise no warning, got %v", run.Warnings)
			}
		})
	}
}

func TestOpenRefuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		log      func(t *testing.T) []byte
		wraps    func(error) bool
		contains []string
	}{
		{
			name:     "not a Gatling log",
			log:      func(*testing.T) []byte { return []byte("<!DOCTYPE html>\n<html><body>report</body></html>\n") },
			wraps:    func(err error) bool { var e *gatling.FormatError; return errors.As(err, &e) },
			contains: []string{"not a Gatling simulation.log"},
		},
		{
			name:     "a text version below the supported range",
			log:      func(*testing.T) []byte { return textHeader("3.10.0") },
			wraps:    func(err error) bool { var e *gatling.VersionError; return errors.As(err, &e) },
			contains: []string{"3.10.0", "3.11.5 through 3.12.0"},
		},
		{
			name:     "3.13.0 is refused although the format is readable",
			log:      func(t *testing.T) []byte { return patchedBinaryLog(t, "3.13.0") },
			wraps:    func(err error) bool { var e *gatling.VersionError; return errors.As(err, &e) },
			contains: []string{"3.13.0", "3.13.1"},
		},
		{
			name:     "a header cut short",
			log:      func(*testing.T) []byte { return []byte("RUN\tio.x.Sim\tsim") },
			wraps:    func(err error) bool { var e *gatling.TruncationError; return errors.As(err, &e) },
			contains: []string{"cut short"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loc := writeLog(t, tt.log(t))
			src, err := Open(loc)
			if err == nil {
				src.Close()
				t.Fatalf("Open succeeded, want a refusal")
			}

			var openErr *OpenError
			if !errors.As(err, &openErr) {
				t.Fatalf("Open = %v (%T), want *OpenError", err, err)
			}
			if openErr.Path != loc.Log {
				t.Errorf("OpenError.Path = %q, want %q", openErr.Path, loc.Log)
			}
			if !tt.wraps(err) {
				t.Errorf("error %v does not wrap the parsec error the case expects", err)
			}
			for _, want := range append(tt.contains, loc.Log) {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %q", err, want)
				}
			}
		})
	}
}

func TestOpenWarnsAboveRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		log  func(t *testing.T) []byte
	}{
		{name: "text 3.99.0", log: func(*testing.T) []byte { return textHeader("3.99.0") }},
		{name: "binary 3.99.9", log: func(t *testing.T) []byte { return patchedBinaryLog(t, "3.99.9") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src, err := Open(writeLog(t, tt.log(t)))
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			defer src.Close()

			warnings := src.Reader.Run().Warnings
			if len(warnings) != 1 {
				t.Fatalf("Warnings = %v, want exactly one", warnings)
			}
			if !strings.HasPrefix(warnings[0].Version, "3.99.") {
				t.Errorf("Warning.Version = %q, want the unverified version", warnings[0].Version)
			}
			if warnings[0].Reason == "" {
				t.Errorf("Warning.Reason is empty")
			}
		})
	}
}

func TestOpenMissingLog(t *testing.T) {
	t.Parallel()

	loc := writeLog(t, textHeader("3.12.0"))
	if err := os.Remove(loc.Log); err != nil {
		t.Fatalf("remove: %v", err)
	}

	_, err := Open(loc)
	var readErr *ReadError
	if !errors.As(err, &readErr) {
		t.Fatalf("Open(missing log) = %v, want *ReadError", err)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error %v does not wrap fs.ErrNotExist", err)
	}
}
