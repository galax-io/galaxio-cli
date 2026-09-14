package report

import (
	"errors"
	"io/fs"
	"os"

	"github.com/galax-io/parsec/gatling/run"
)

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
