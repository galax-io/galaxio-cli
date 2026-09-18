package report

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/galax-io/parsec/gatling"
	"github.com/galax-io/parsec/gatling/simlog"
	"github.com/galax-io/parsec/model"
)

// cancelCheckInterval is how many items are walked between looks at the
// context. Checking every item would cost more than it is worth; checking
// every thousand keeps cancellation prompt on a log of any size.
const cancelCheckInterval = 1024

// Tally is what one pass over a run counted, beside the bounds parsec derives
// from the same pass.
//
// These are counts of records walked, not statistics: nothing here is a mean,
// a percentile, a range or a rate. Successes and failures are kept apart from
// the outcome the source recorded so that the figures of a Summary inherit a
// split that was never conflated.
type Tally struct {
	// Requests is every sample walked; Successes, Failures and Unknown sum to
	// it. Unknown counts a sample whose outcome the source lost, which is
	// neither a success nor a failure and must not be silently made either.
	Requests  int
	Successes int
	Failures  int
	Unknown   int

	// Groups counts group traversals and Users virtual-user events, not
	// distinct groups or virtual users: a run emits one traversal per entry
	// and a start and an end per user.
	Groups int
	Users  int
	Errors int

	// Assertions counts declared-assertion payloads a source yielded among its
	// events rather than ahead of them. Gatling writes its own before the
	// events, where they land on the run description instead, so this is zero
	// for a Gatling log — but a payload that does arrive here must be counted
	// rather than walked past in silence.
	Assertions int

	// Other counts items of a kind this package does not know, which a later
	// parsec release could introduce. Nothing can be said about them except
	// that they were there, and saying nothing at all would under-report the
	// log.
	Other int

	// Bounds is where the run begins and ends, as parsec defines it. It may
	// report nothing when an item could not be placed in time.
	Bounds model.Bounds
}

// Scan walks the run rd yields once and returns its summary at opts: what the
// log holds, counted, with the run's bounds extended as it goes, and the
// figures of its successful and failed requests. Nothing is retained, so
// memory does not grow with the log. Options no summary can be
// computed at are refused before anything is read.
//
// A log cut short — a run killed mid-flight — returns the summary of everything
// it did hold together with an error, because a partial run must not be
// mistaken for a complete one. One shape cannot be detected: a run killed
// exactly on a record boundary leaves a log neither format carries an end
// marker for, so parsec ends it with io.EOF and this reports it as complete.
// The format cannot express the difference and neither can this. Any other
// read failure returns the summary walked so far and an error saying the run
// was not read completely. A cancelled ctx returns its error.
//
// rd must not have been walked already. A reader yields its items once, so a
// second walk returns an empty summary and no error, which no caller can tell
// from a run that recorded nothing. [Source.Scan] holds that precondition for
// the open runs this package hands out; a caller holding a bare reader owns it.
func Scan(ctx context.Context, rd simlog.RunReader, opts Options) (Summary, error) {
	opts, err := opts.normalize()
	if err != nil {
		return Summary{}, err
	}

	s := Summary{Options: opts}

	for n := 0; ; n++ {
		if n%cancelCheckInterval == 0 {
			if err := ctx.Err(); err != nil {
				return s, err
			}
		}

		item, err := rd.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return s, nil
			}

			var cutShort *gatling.TruncationError
			if errors.As(err, &cutShort) {
				return s, fmt.Errorf("%w; the run was not read to the end", err)
			}

			return s, fmt.Errorf("%w; the run was not read completely", err)
		}

		s.add(&item)
	}
}

// add folds one item into the summary: the tally counts every item, and a
// request feeds the figures of the outcome the source recorded for it. A
// request whose outcome the source lost stays in Tally.Unknown and feeds
// nothing, and a group traversal feeds nothing either.
func (s *Summary) add(item *model.Item) {
	s.Tally.count(item)

	if item.Kind != model.ItemSample {
		return
	}

	switch item.Sample.Outcome {
	case model.OutcomeSuccess:
		s.OK.add(item.Sample.Duration)
	case model.OutcomeFailure:
		s.Failed.add(item.Sample.Duration)
	}
}

// count adds one item to the tally and extends the bounds with it. Every kind
// of item may move a bound; only samples carry an outcome.
func (t *Tally) count(item *model.Item) {
	t.Bounds.Extend(item)

	switch item.Kind {
	case model.ItemSample:
		t.Requests++

		switch item.Sample.Outcome {
		case model.OutcomeSuccess:
			t.Successes++
		case model.OutcomeFailure:
			t.Failures++
		default:
			t.Unknown++
		}
	case model.ItemGroup:
		t.Groups++
	case model.ItemUser:
		t.Users++
	case model.ItemError:
		t.Errors++
	case model.ItemAssertion:
		t.Assertions++
	default:
		t.Other++
	}
}
