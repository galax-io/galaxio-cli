package report

import (
	"context"
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

// copyResultsRoot copies the three lastrun corpus runs into a fresh root and
// pins their modification times, so that "newest" is deterministic and is not
// the newest by name.
func copyResultsRoot(t *testing.T) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), "results")
	src := filepath.Join(corpusDir, "lastrun", "results")

	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}

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
	}

	return root
}

// writeLog writes a synthetic log into a fresh run directory and locates it, so
// that a test can exercise parsec's version gate without a recording for every
// version.
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
			name:          "a results root without a marker yields the most recently modified run",
			path:          func(t *testing.T) string { return copyResultsRoot(t) },
			expectedDir:   func(path string) string { return filepath.Join(path, "corpussimulation-20260909022556930") },
			expectedFound: run.FoundByNewest,
		},
		{
			name: "a marker naming a deleted run is ignored",
			path: func(t *testing.T) string {
				root := copyResultsRoot(t)
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

	t.Run("a path that exists but is not a directory", func(t *testing.T) {
		t.Parallel()

		// parsec folds this into its absence of runs, and "no run under X" for
		// a regular file tells the reader to look for run directories inside a
		// file. It is the same typo as a missing path, landed on something.
		path := filepath.Join(t.TempDir(), "results.zip")
		if err := os.WriteFile(path, []byte("not a log"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		_, err := Locate(path)
		if err == nil {
			t.Fatal("a file that is not a simulation.log was accepted as a run")
		}

		if !strings.HasPrefix(err.Error(), path+" is not a Gatling run") {
			t.Errorf("error %q does not say the path is not a run", err)
		}

		var notFound *run.NotFoundError
		if errors.As(err, &notFound) {
			t.Errorf("a file must not be reported as a directory holding no run")
		}
	})

	t.Run("a path that needs cleaning", func(t *testing.T) {
		t.Parallel()

		// run.Find cleans lexically before it searches, so this names the empty
		// root and parsec reads it perfectly. A stat of the spelling the caller
		// typed asks the kernel to walk "absent", which fails — and used to
		// report a directory that is fine as one that cannot be read.
		root := t.TempDir()
		path := filepath.Join(root, "absent", "..")

		_, err := Locate(path)

		var notFound *run.NotFoundError
		if !errors.As(err, &notFound) {
			t.Fatalf("error %v is not an absence of runs; the directory is readable and empty", err)
		}

		if errors.Is(err, fs.ErrNotExist) {
			t.Errorf("error %q reports a readable directory as missing", err)
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

		if !errors.Is(err, fs.ErrPermission) {
			t.Errorf("error %v does not wrap fs.ErrPermission", err)
		}

		// parsec's own message names the directory it was reading. Rewriting it
		// here would name a simulation.log whose existence was never
		// established, so the message is passed through.
		if !strings.Contains(err.Error(), dir) {
			t.Errorf("error %q does not name the directory it could not read", err)
		}

		var notFound *run.NotFoundError
		if errors.As(err, &notFound) {
			t.Errorf("an unreadable directory must not be reported as an absence of runs")
		}
	})
}

func TestLocateDanglingSymlink(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege this test does not assume")
	}

	link := filepath.Join(t.TempDir(), "results")
	if err := os.Symlink(filepath.Join(t.TempDir(), "never-created"), link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	_, err := Locate(link)

	// os.Lstat would call the link itself present and let parsec's "no run
	// under" through, sending the reader to look for a load test instead of a
	// broken mount.
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error %v does not wrap fs.ErrNotExist", err)
	}

	if !strings.HasPrefix(err.Error(), "cannot read "+link+": ") {
		t.Errorf("error %q does not read as a read failure on the link", err)
	}

	var notFound *run.NotFoundError
	if errors.As(err, &notFound) {
		t.Errorf("a dangling symlink must not be reported as an absence of runs")
	}
}

// TestLocateDefaultRootMissing pins the one case that is not a read failure:
// the caller never typed the default root, so an absent one is an absence of
// runs rather than a path they got wrong.
func TestLocateDefaultRootMissing(t *testing.T) {
	t.Chdir(t.TempDir())

	_, err := Locate("")

	var notFound *run.NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Locate(\"\") in an empty project = %v, want *run.NotFoundError", err)
	}

	if strings.Contains(err.Error(), "cannot read") {
		t.Errorf("error %q reads as a read failure for a path the caller never gave", err)
	}
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

			defer func() { _ = src.Close() }()

			if src.Format != tt.format {
				t.Errorf("Format = %v, want %v", src.Format, tt.format)
			}

			description := src.Reader.Run()
			if description.ToolVersion != tt.version {
				t.Errorf("ToolVersion = %q, want %q", description.ToolVersion, tt.version)
			}

			if len(description.Warnings) != 0 {
				t.Errorf("a recorded version must raise no warning, got %v", description.Warnings)
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
				_ = src.Close()
				t.Fatalf("Open succeeded, want a refusal")
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

			defer func() { _ = src.Close() }()

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

	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error %v does not wrap fs.ErrNotExist", err)
	}

	if !strings.Contains(err.Error(), "cannot read ") {
		t.Errorf("error %q does not read as a read failure", err)
	}
}

// TestSourceScanRefusesASecondWalk pins the trap that makes a spent reader look
// like an empty run: both codecs latch their end and keep answering with it, so
// a second walk returns a summary of nothing and no error at all.
func TestSourceScanRefusesASecondWalk(t *testing.T) {
	t.Parallel()

	loc, err := Locate(filepath.Join(corpusDir, "3.12.0"))
	if err != nil {
		t.Fatalf("locate: %v", err)
	}

	src, err := Open(loc)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	defer func() { _ = src.Close() }()

	first, err := src.Scan(context.Background(), DefaultOptions(), nil)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if first.Tally.Requests == 0 {
		t.Fatalf("the corpus run recorded no requests: %+v", first.Tally)
	}

	second, err := src.Scan(context.Background(), DefaultOptions(), nil)
	if !errors.Is(err, ErrSpent) {
		t.Errorf("a second walk returned %+v, %v; want ErrSpent", second.Tally, err)
	}

	if second.Tally != (Tally{}) || second.Options.Percentiles != nil {
		t.Errorf("a refused walk returned a summary: %+v", second)
	}

	// Closing spends it too: the reader over a closed file answers from its
	// latched end rather than failing, so the walk would succeed and report
	// nothing.
	_ = src.Close()

	if _, err := src.Scan(context.Background(), DefaultOptions(), nil); !errors.Is(err, ErrSpent) {
		t.Errorf("a walk after Close returned %v, want ErrSpent", err)
	}
}

// TestSourceRefusesOptionsBeforeItIsSpent holds what a caller needs to correct
// a refusal: options no summary can be computed at are refused before the walk
// is spent, so the same source reads the run once the caller has fixed them.
func TestSourceRefusesOptionsBeforeItIsSpent(t *testing.T) {
	t.Parallel()

	loc, err := Locate(filepath.Join(corpusDir, "3.13.1"))
	if err != nil {
		t.Fatalf("locate: %v", err)
	}

	src, err := Open(loc)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	defer func() { _ = src.Close() }()

	if _, err := src.Scan(context.Background(), Options{}, nil); err == nil || errors.Is(err, ErrSpent) {
		t.Fatalf("a walk at no options returned %v, want the options refused", err)
	}

	summary, err := src.Scan(context.Background(), DefaultOptions(), nil)
	if err != nil {
		t.Fatalf("the walk after a refusal: %v", err)
	}

	if summary.Tally.Requests == 0 {
		t.Errorf("the walk after a refusal read nothing: %+v", summary.Tally)
	}
}

// TestZeroSourceMeasuresNothing holds a Source nothing was opened for to the
// answers the progress block reads from one: it has read nothing and knows no
// size, rather than panicking on the first draw.
func TestZeroSourceMeasuresNothing(t *testing.T) {
	t.Parallel()

	var s Source

	if n := s.BytesRead(); n != 0 {
		t.Errorf("BytesRead = %d, want 0", n)
	}

	if size, known := s.Size(); known || size != 0 {
		t.Errorf("Size = %d, %v; want 0, false", size, known)
	}
}

// TestErrNoPathWrapsTheLibrary keeps the two refusals of the same thing tied
// together: a caller that checks parsec's documented sentinel must see this
// package's refusal as the same condition, not as an unrelated error.
func TestErrNoPathWrapsTheLibrary(t *testing.T) {
	t.Parallel()

	if !errors.Is(ErrNoPath, run.ErrNoPath) {
		t.Errorf("ErrNoPath does not wrap run.ErrNoPath, so errors.Is against the library's sentinel fails")
	}

	// The wording stays this command's: it names the argument the caller can
	// omit, which a library has no argument to name.
	if !strings.Contains(ErrNoPath.Error(), run.DefaultResultsRoot) {
		t.Errorf("ErrNoPath = %q, want it to name %s", ErrNoPath, run.DefaultResultsRoot)
	}
}

// TestSourceCountsTheBytesRead holds the measure a progress display reads: the
// bytes taken never pass the size the log had when it was opened, and reach it
// when the walk ends, for every corpus run.
func TestSourceCountsTheBytesRead(t *testing.T) {
	t.Parallel()

	for _, version := range []string{"3.11.5", "3.12.0", "3.13.1", "3.14.9", "3.15.1"} {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join(corpusDir, version)

			src, err := Open(run.Location{Dir: dir, Log: filepath.Join(dir, "simulation.log")})
			if err != nil {
				t.Fatalf("Open: %v", err)
			}

			defer func() { _ = src.Close() }()

			size, known := src.Size()

			info, err := os.Stat(filepath.Join(dir, "simulation.log"))
			if err != nil {
				t.Fatalf("stat: %v", err)
			}

			if !known || size != info.Size() {
				t.Fatalf("Size() = %d, %v; want %d, true", size, known, info.Size())
			}

			tick := func(Summary) {
				if read := src.BytesRead(); read > size {
					t.Errorf("read %d bytes of a %d-byte log", read, size)
				}
			}

			if _, err := src.Scan(context.Background(), DefaultOptions(), tick); err != nil {
				t.Fatalf("Scan: %v", err)
			}

			if read := src.BytesRead(); read != size {
				t.Errorf("the walk ended having read %d bytes of %d", read, size)
			}
		})
	}
}

// fileInfo is a FileInfo of a chosen mode and size.
type fileInfo struct {
	fs.FileInfo
	mode fs.FileMode
	size int64
}

func (f fileInfo) Mode() fs.FileMode { return f.mode }
func (f fileInfo) Size() int64       { return f.size }

// TestKnownSize holds a size a display cannot go by to unknown: a device, a
// pipe, and a file that was empty.
func TestKnownSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		info     fileInfo
		expected int64
	}{
		{name: "a regular file", info: fileInfo{mode: 0o644, size: 4096}, expected: 4096},
		{name: "an empty file", info: fileInfo{mode: 0o644}, expected: 0},
		{name: "a pipe", info: fileInfo{mode: fs.ModeNamedPipe | 0o600, size: 65536}, expected: 0},
		{name: "a device", info: fileInfo{mode: fs.ModeDevice | fs.ModeCharDevice | 0o666, size: 1}, expected: 0},
	}

	for _, tt := range tests {
		if got := knownSize(tt.info); got != tt.expected {
			t.Errorf("%s: knownSize = %d, want %d", tt.name, got, tt.expected)
		}
	}

	var s Source
	if _, known := s.Size(); known {
		t.Errorf("a source with no size says it knows one")
	}
}
