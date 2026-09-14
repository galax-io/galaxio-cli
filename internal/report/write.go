package report

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/galax-io/parsec/gatling"
	"github.com/galax-io/parsec/gatling/simlog"
	"github.com/galax-io/parsec/model"
)

// writeBufferSize is the buffer between the encoder and the caller's writer:
// large enough to make a syscall per few hundred records, fixed so that
// memory does not grow with the log.
const writeBufferSize = 64 << 10

// cancelCheckInterval is how many items are written between looks at the
// context. Checking every item would cost more than it is worth; checking
// every thousand keeps cancellation prompt on a log of any size.
const cancelCheckInterval = 1024

// Summary is what Write tallied: how many records of each kind it wrote and,
// for a log cut short, where. The counts are tallies of items seen, not
// statistics, and are meant for a diagnostic line.
type Summary struct {
	Requests   int
	Groups     int
	Users      int
	Errors     int
	Assertions int

	// Truncated is set when the log ended inside a record. The records
	// written are what the run recorded up to that point.
	Truncated *gatling.TruncationError
}

// Records returns how many records were written after the header.
func (s Summary) Records() int {
	return s.Requests + s.Groups + s.Users + s.Errors + s.Assertions
}

// Write streams the run rd yields to w as JSON Lines: the header first, then
// one record per item in the order the log recorded them. Each item is
// encoded before the next is read, so memory does not grow with the log and
// parsec's reused group slice is never aliased.
//
// The endings: io.EOF from rd is a clean end and returns nil; a
// *gatling.TruncationError returns a *TruncatedError after every record the
// log held has been written; any other read failure returns an
// *IncompleteError; a failure to write returns a *WriteError at once; a
// cancelled ctx returns its error. Whatever was written is flushed before
// any error is returned, so a consumer never sees a partial line unless the
// write itself failed.
func Write(ctx context.Context, rd simlog.RunReader, w io.Writer) (Summary, error) {
	bw := bufio.NewWriterSize(w, writeBufferSize)
	enc := json.NewEncoder(bw)
	enc.SetEscapeHTML(false)

	var sum Summary

	header := headerFrom(rd.Run())
	if err := enc.Encode(&header); err != nil {
		return sum, &WriteError{Err: err}
	}

	// One value per kind, reused for every item, and one scratch for their
	// optional fields: encoding a pointer to a reused value allocates nothing
	// per record.
	var (
		sc  scratch
		req RequestRecord
		grp GroupRecord
		usr UserRecord
		rer ErrorRecord
		asr AssertionRecord
	)

	for n := 0; ; n++ {
		if n%cancelCheckInterval == 0 {
			if err := ctx.Err(); err != nil {
				return sum, flushThen(bw, err)
			}
		}

		item, err := rd.Next()
		if err != nil {
			return sum, ended(bw, &sum, err)
		}

		var rec any
		switch item.Kind {
		case model.ItemSample:
			req = requestFrom(item.Sample, &sc)
			rec = &req
		case model.ItemGroup:
			grp = groupFrom(item.Group, &sc)
			rec = &grp
		case model.ItemUser:
			usr = userFrom(item.User)
			rec = &usr
		case model.ItemError:
			rer = errorFrom(item.Error)
			rec = &rer
		case model.ItemAssertion:
			asr = assertionFrom(item.Assertion)
			rec = &asr
		default:
			return sum, flushThen(bw, &IncompleteError{Err: fmt.Errorf("report: unknown item kind %v", item.Kind), Written: sum.Records()})
		}

		if err := enc.Encode(rec); err != nil {
			return sum, &WriteError{Err: err}
		}
		sum.count(item.Kind)
	}
}

func (s *Summary) count(kind model.ItemKind) {
	switch kind {
	case model.ItemSample:
		s.Requests++
	case model.ItemGroup:
		s.Groups++
	case model.ItemUser:
		s.Users++
	case model.ItemError:
		s.Errors++
	case model.ItemAssertion:
		s.Assertions++
	}
}

// ended turns the error that stopped Next into Write's result, flushing what
// was written first. A flush failure outranks the read's ending: the output
// is broken, and that is what the caller must hear about.
func ended(bw *bufio.Writer, sum *Summary, err error) error {
	if flushErr := bw.Flush(); flushErr != nil {
		return &WriteError{Err: flushErr}
	}
	if errors.Is(err, io.EOF) {
		return nil
	}
	var truncated *gatling.TruncationError
	if errors.As(err, &truncated) {
		sum.Truncated = truncated
		return &TruncatedError{Err: truncated, Written: sum.Records()}
	}
	return &IncompleteError{Err: err, Written: sum.Records()}
}

// flushThen flushes and returns err, unless the flush itself failed.
func flushThen(bw *bufio.Writer, err error) error {
	if flushErr := bw.Flush(); flushErr != nil {
		return &WriteError{Err: flushErr}
	}
	return err
}
