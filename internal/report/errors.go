package report

import (
	"fmt"

	"github.com/galax-io/parsec/gatling"
)

// ReadError is a filesystem failure on a path: it does not exist, or cannot
// be read. It is deliberately not an absence of runs — a broken mount or a
// permission problem reported as "no run here" sends a user to debug the
// wrong thing.
type ReadError struct {
	Path string
	Err  error
}

func (e *ReadError) Error() string {
	return "cannot read " + e.Path + ": " + e.Err.Error()
}

func (e *ReadError) Unwrap() error {
	return e.Err
}

// OpenError is parsec's refusal to read a log, with the path it refused.
// parsec's own message names the version and range, the bytes it found, or
// where the log was cut; this type only adds the path.
type OpenError struct {
	Path string
	Err  error
}

func (e *OpenError) Error() string {
	return e.Path + ": " + e.Err.Error()
}

func (e *OpenError) Unwrap() error {
	return e.Err
}

// TruncatedError says the log ended inside a record — a run killed
// mid-flight — after Written records were written. Those records are what
// the run recorded; whether a partial run is usable is the caller's decision,
// which is why this is still an error.
type TruncatedError struct {
	Err     *gatling.TruncationError
	Written int
}

func (e *TruncatedError) Error() string {
	return fmt.Sprintf("%v; the %d records already written are what the run recorded", e.Err, e.Written)
}

func (e *TruncatedError) Unwrap() error {
	return e.Err
}

// IncompleteError says the read failed after Written records were written.
// Unlike a truncation, nothing delivered before the failure is a result.
type IncompleteError struct {
	Err     error
	Written int
}

func (e *IncompleteError) Error() string {
	return fmt.Sprintf("%v; the %d records already written do not form a complete run", e.Err, e.Written)
}

func (e *IncompleteError) Unwrap() error {
	return e.Err
}

// WriteError is a failure to write the stream. It is reported once, and
// nothing is written after it.
type WriteError struct {
	Err error
}

func (e *WriteError) Error() string {
	return "writing output: " + e.Err.Error()
}

func (e *WriteError) Unwrap() error {
	return e.Err
}
