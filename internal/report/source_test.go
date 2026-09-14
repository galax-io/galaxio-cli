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
