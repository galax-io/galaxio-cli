package report

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/galax-io/parsec/gatling"
	"github.com/galax-io/parsec/gatling/run"
	"github.com/galax-io/parsec/gatling/simlog"
)

// ErrNoPath refuses an argument that was given but is empty. The zero value of
// every configuration field and every unset shell variable is the empty string,
// and guessing a results root for one would answer confidently about a run
// nobody asked for.
//
// It wraps parsec's refusal of the same thing rather than restating it, so that
// errors.Is against run.ErrNoPath holds for what this package returns. The
// wording is this command's, because it can say the one thing a library cannot:
// that omitting the argument is how the default root is asked for.
var ErrNoPath error = noPath{}

type noPath struct{}

func (noPath) Error() string {
	return "no path given: pass a run directory, a simulation.log or a results root, or omit the argument to search " + run.DefaultResultsRoot
}

func (noPath) Unwrap() error { return run.ErrNoPath }

// Locate finds the run to read. An empty path means the results root Maven and
// sbt write to, resolved against the working directory; anything else is handed
// to parsec unchanged: a run directory or its simulation.log is the run, and any
// other directory is a results root searched by lastRun.txt, then by the most
// recently modified run.
//
// A path the caller named that does not exist, or that exists and is not a
// directory, is reported as what it is, never as an absence of runs: those are
// different things and the wrong one sends a caller to debug a place rather
// than a spelling. The default root is not treated that way: the caller never
// typed it, so "no run under target/gatling" is the honest answer for a project
// that has not run a load test.
func Locate(path string) (run.Location, error) {
	defaulted := path == ""
	if defaulted {
		path = run.DefaultResultsRoot
	}

	loc, err := run.Find(path)
	if err == nil {
		return loc, nil
	}

	// parsec reports three things as one absence: a path that is not there, a
	// path that is there but is not a directory, and a directory it read that
	// held no run. Only the third is an absence of runs; the other two are
	// about the path the caller named, and answering them with "no run under X"
	// sends that caller to debug the wrong thing.
	//
	// The stat is parsec's own call, on parsec's own path: run.Find cleans
	// lexically before it searches, and "a/nope/../b" names the directory "a/b"
	// whether or not "a/nope" exists. Stat'ing the spelling the caller typed
	// asks the kernel to walk it, which fails on a directory parsec read
	// perfectly.
	var notFound *run.NotFoundError
	if !defaulted && errors.As(err, &notFound) {
		cleaned := filepath.Clean(path)

		info, statErr := os.Stat(cleaned)

		switch {
		case statErr != nil:
			var pathErr *fs.PathError
			if errors.As(statErr, &pathErr) {
				return run.Location{}, fmt.Errorf("cannot read %s: %w", cleaned, pathErr.Err)
			}

			return run.Location{}, fmt.Errorf("cannot read %s: %w", cleaned, statErr)
		case !info.IsDir():
			// parsec takes a file as the run only when it is named
			// simulation.log, so a file reaching here is some other file the
			// caller named — an archive, a rotated log, a typo that landed.
			return run.Location{}, fmt.Errorf("%s is not a Gatling run: pass a simulation.log, a run directory, or a results root", cleaned)
		}
	}

	// Anything else is parsec's own, including a directory it could not read:
	// its message names the directory it was reading, which a rewrite here
	// would replace with a simulation.log whose existence was never established.
	return run.Location{}, err
}

// ErrSpent refuses a second walk over a run that has already been read.
var ErrSpent = errors.New("the run has already been read: a reader yields its items once")

// Source is an open run: the log format it holds and the reader that yields its
// items. Close it when the read is over.
type Source struct {
	Format gatling.Format
	Reader simlog.RunReader

	file *os.File
	// read counts the bytes the reader has taken from the file, and size is
	// what the file held when it was opened, or 0 when that is unknown.
	read *countingReader
	size int64
	// spent says the reader has nothing left to yield, either because it was
	// walked or because the file under it was closed.
	spent bool
}

// Scan walks the run once and summarises it at opts, and refuses to walk it
// again. Options no summary can be computed at are refused before the walk is
// spent, so that a caller can correct them and try again.
//
// A reader yields its items once: both codecs latch their end and answer every
// later call with it, so a second pass returns a summary of nothing and no error
// — which is exactly what a run that recorded nothing returns, and no caller
// could tell the two apart. A closed source is the same: the latched end means
// the file is never touched, so reading one succeeds too.
//
// This is why the summary of galaxio-cli#51 and the stats.json writer of
// galaxio-cli#52 fold over one pass rather than one each: the first caller to
// fold twice would have shipped a report of zero requests for a full run and
// exited 0.
func (s *Source) Scan(ctx context.Context, opts Options, tick func(Summary)) (Summary, error) {
	return s.scan(ctx, opts, scanHooks{tick: tick})
}

// ScanTree collects request and group populations alongside the root in one walk.
func (s *Source) ScanTree(ctx context.Context, opts Options) (*Tree, error) {
	if s.spent {
		return nil, ErrSpent
	}
	tree, err := newTree(opts)
	if err != nil {
		return nil, err
	}
	tree.Summary, err = s.scan(ctx, tree.options, scanHooks{collect: tree.collect})
	return tree, err
}

func (s *Source) scan(ctx context.Context, opts Options, hooks scanHooks) (Summary, error) {
	if s.spent {
		return Summary{}, ErrSpent
	}

	opts, err := opts.Normalize()
	if err != nil {
		return Summary{}, err
	}

	s.spent = true

	return scan(ctx, s.Reader, opts, hooks)
}

// BytesRead returns how many bytes of the log the reader has taken so far. The
// reader buffers, so it runs ahead of the items walked by at most its buffer.
// A source no file was opened for has read nothing, as it has no size.
func (s *Source) BytesRead() int64 {
	if s.read == nil {
		return 0
	}

	return s.read.n
}

// Size returns the size the log had when it was opened, and false when that is
// unknown: a file that is not a regular one, or one that was empty then.
func (s *Source) Size() (int64, bool) {
	return s.size, s.size > 0
}

// countingReader counts the bytes read through it. The walk and whoever reads
// the count share one goroutine, so the count needs no lock.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)

	return n, err
}

// knownSize returns the size of a file as it was opened, or 0 when it has none
// to go by: a device, a pipe or an empty file.
func knownSize(info fs.FileInfo) int64 {
	if !info.Mode().IsRegular() {
		return 0
	}

	return info.Size()
}

// Open opens the run at loc. The format is identified from the log's leading
// bytes, never from its name, and parsec applies its own version gate; a
// refusal comes back naming the log.
//
// The accepted version range is parsec's own. Nothing here narrows or widens
// it, so the error for an unsupported version quotes the range in force.
func Open(loc run.Location) (*Source, error) {
	f, err := os.Open(loc.Log)
	if err != nil {
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			return nil, fmt.Errorf("cannot read %s: %w", loc.Log, pathErr.Err)
		}

		return nil, err
	}

	format, err := detect(f)
	if err != nil {
		_ = f.Close()

		// A read that failed already names the log, the way os.Open's does;
		// only a refusal from the format itself needs the log added to it.
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			return nil, fmt.Errorf("cannot read %s: %w", loc.Log, pathErr.Err)
		}

		return nil, fmt.Errorf("%s: %w", loc.Log, err)
	}

	var size int64
	if info, err := f.Stat(); err == nil {
		size = knownSize(info)
	}

	// The count starts after detection, which reads with ReadAt and moves no
	// offset, so it counts every byte the reader takes and nothing else.
	read := &countingReader{r: f}

	rd, err := simlog.NewRunReader(read)
	if err != nil {
		_ = f.Close()

		return nil, fmt.Errorf("%s: %w", loc.Log, err)
	}

	return &Source{Format: format, Reader: rd, file: f, read: read, size: size}, nil
}

// detect reads the log's leading bytes and names its format, leaving the file
// offset where it found it so that parsec sees the log from its first byte.
//
// It reads with ReadAt rather than Read: a single Read may return fewer bytes
// than asked for on any source slower than a local disk, and a short head makes
// gatling.Detect answer "unknown" for a log parsec then reads perfectly. A file
// shorter than the detection window is not an error here — parsec refuses it
// with the message that names what it found.
func detect(f *os.File) (gatling.Format, error) {
	head := make([]byte, gatling.DetectSize)

	n, err := f.ReadAt(head, 0)
	if err != nil && !errors.Is(err, io.EOF) {
		return gatling.FormatUnknown, err
	}

	format, err := gatling.Detect(head[:n])
	if err != nil {
		return gatling.FormatUnknown, err
	}

	return format, nil
}

// Close releases the log file, and spends the source: the reader over a closed
// file answers from its latched end rather than failing, so a walk afterwards
// would look like a run that recorded nothing.
func (s *Source) Close() error {
	s.spent = true

	return s.file.Close()
}
