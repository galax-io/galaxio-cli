package report

import (
	"bufio"
	"errors"
	"io/fs"
	"os"

	"github.com/galax-io/parsec/gatling"
	"github.com/galax-io/parsec/gatling/run"
	"github.com/galax-io/parsec/gatling/simlog"
)

// readBufferSize is the read-ahead over the log file. A multi-gigabyte log is
// read once, sequentially; a large buffer keeps the syscall count low without
// growing with the log.
const readBufferSize = 256 << 10

// Locate finds the run to read. An empty path means the results root Maven
// and sbt write to, resolved against the working directory; anything else is
// handed to parsec unchanged: a run directory or its simulation.log is the
// run, and any other directory is a results root searched by lastRun.txt,
// then by the most recently modified run.
//
// A path that does not exist is a *ReadError naming it, not an absence of
// runs; so is a directory that cannot be read.
func Locate(path string) (run.Location, error) {
	if path == "" {
		path = run.DefaultResultsRoot
	}

	loc, err := run.Find(path)
	if err == nil {
		return loc, nil
	}

	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return run.Location{}, &ReadError{Path: pathErr.Path, Err: pathErr.Err}
	}

	// parsec reports a directory that is not there as a directory holding no
	// run. The stat is what tells a mistyped path from an empty root.
	var notFound *run.NotFoundError
	if errors.As(err, &notFound) {
		if _, statErr := os.Lstat(path); statErr != nil && errors.As(statErr, &pathErr) {
			return run.Location{}, &ReadError{Path: pathErr.Path, Err: pathErr.Err}
		}
	}

	return run.Location{}, err
}

// Source is an open run: where it was found, which log format it holds, and
// the reader that yields its items. Close it when the read is over.
type Source struct {
	Location run.Location
	Format   gatling.Format
	Reader   simlog.RunReader

	file *os.File
}

// Open opens the run at loc. parsec identifies the log format from its
// leading bytes and applies its version gate; a refusal comes back as an
// *OpenError naming the log, and a log that cannot be opened at all as a
// *ReadError.
//
// The version range parsec accepts is its own; nothing here narrows or widens
// it, so the error for an unsupported version quotes the range that is
// actually in force.
func Open(loc run.Location) (*Source, error) {
	f, err := os.Open(loc.Log)
	if err != nil {
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			return nil, &ReadError{Path: pathErr.Path, Err: pathErr.Err}
		}
		return nil, err
	}

	br := bufio.NewReaderSize(f, readBufferSize)

	// The format is read for the caller's diagnostic only; the reader below
	// identifies it again for itself and is the one that refuses a stream
	// that is not a Gatling log, so a failure here is left for it to name.
	head, _ := br.Peek(gatling.DetectSize)
	format, _ := gatling.Detect(head)

	rd, err := simlog.NewRunReader(br)
	if err != nil {
		_ = f.Close()
		return nil, &OpenError{Path: loc.Log, Err: err}
	}

	return &Source{Location: loc, Format: format, Reader: rd, file: f}, nil
}

// Close releases the log file.
func (s *Source) Close() error {
	return s.file.Close()
}
