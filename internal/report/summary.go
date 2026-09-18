package report

import (
	"errors"
	"fmt"
	"slices"
	"time"

	tdigest "github.com/caio/go-tdigest/v5"
)

// Options is what a run is summarised at: the percentile ranks to report and
// the two boundaries of the response-time bands. A caller chooses them before
// the walk, and the walk never changes them.
type Options struct {
	// Percentiles are the ranks to report, each above 0 and at most 100. The
	// summary keeps each once, in increasing order, whatever order they came in.
	Percentiles []float64
	// Bands are the boundaries of the response-time bands.
	Bands Bands
}

// Bands are the two boundaries of the response-time bands, in whole
// milliseconds. A successful response falls under Lower, from Lower to under
// Upper, or at or above Upper; a failed one counts as failed whatever its time.
type Bands struct {
	Lower int64
	Upper int64
}

// DefaultOptions returns Gatling's own settings, unchanged from 3.11.5 to
// 3.15.1: the 50th, 75th, 95th and 99th percentile, and bands at 800 ms and
// 1200 ms. A run whose author changed them in gatling.conf records that nowhere
// in its log, so these are what a summary is compared against by default.
func DefaultOptions() Options {
	return Options{
		Percentiles: []float64{50, 75, 95, 99},
		Bands:       Bands{Lower: 800, Upper: 1200},
	}
}

// ValidRank reports whether rank can be a percentile rank: a number above 0
// and at most 100. NaN cannot.
func ValidRank(rank float64) bool {
	return rank > 0 && rank <= 100
}

// Valid reports whether b can divide response times into bands: neither
// boundary is negative, and the upper one is greater than the lower.
func (b Bands) Valid() bool {
	return b.Lower >= 0 && b.Upper > b.Lower
}

// Normalize returns o with its ranks sorted and each kept once, or an error
// naming the rank or the boundaries no summary can be computed at. The ranks
// are copied, so the caller's slice is never reordered under it.
//
// A caller that reads a run normalizes its options before it opens one: the
// refusal is the caller's fault and not the log's, and a command turns it into
// a usage error. [Scan] normalizes again, so a caller that does not is still
// refused rather than answered with a summary at options nobody chose.
func (o Options) Normalize() (Options, error) {
	if len(o.Percentiles) == 0 {
		return Options{}, errors.New("no percentile rank given")
	}

	for _, rank := range o.Percentiles {
		if !ValidRank(rank) {
			return Options{}, fmt.Errorf("percentile rank %v is not above 0 and at most 100", rank)
		}
	}

	if !o.Bands.Valid() {
		return Options{}, fmt.Errorf("band boundaries %d and %d are not two non-negative numbers of milliseconds, the second greater than the first", o.Bands.Lower, o.Bands.Upper)
	}

	ranks := slices.Clone(o.Percentiles)
	slices.Sort(ranks)

	return Options{Percentiles: slices.Compact(ranks), Bands: o.Bands}, nil
}

// Summary is what one pass over a run produced: the tally of what the log
// holds, the options it was summarised at, and the figures of the run as a
// whole.
//
// A request counts toward the figures of the outcome the source recorded for
// it, and toward nothing else: a failure never reaches a figure of successful
// requests, a request whose outcome the source lost reaches no figure at all,
// and a group traversal is not a request.
type Summary struct {
	// Tally counts what the log holds, as milestone v0.13.0 counted it.
	Tally Tally
	// Options is what the run was summarised at, its ranks sorted and each
	// kept once.
	Options Options
	// OK and Failed are the figures of the requests the source recorded as
	// successful and as failed.
	OK     Figures
	Failed Figures
	// Under, Between and Over are the response-time bands: how many successful
	// responses took less than the lower boundary, from the lower boundary to
	// less than the upper one, and the upper boundary or more. Failed requests
	// are a band of their own, Failed.Count(), whatever their time, and a
	// successful request with no recorded end is in no band, so the three add
	// up to the successful requests that have one.
	Under, Between, Over int

	// all estimates the percentiles of all requests: every recorded response
	// time of a successful or failed request, fed in the order the log holds
	// them, which is how Gatling 3.11 feeds the digest of its own. Merging the
	// two outcomes' digests when they are read is not that digest: on a run of
	// 12 000 requests of which 5 % fail slowly it gave a 95th percentile of
	// 358 ms where this one, and Gatling 3.11's, give 366, and on other such
	// runs it was hundreds of milliseconds away.
	all *tdigest.TDigest
}

// band counts a successful response of ms milliseconds into its band. The
// boundaries are lower-inclusive, as Gatling's are.
func (s *Summary) band(ms int64) {
	switch {
	case ms < s.Options.Bands.Lower:
		s.Under++
	case ms < s.Options.Bands.Upper:
		s.Between++
	default:
		s.Over++
	}
}

// All returns the figures of every request of the run, successful and failed
// together: the exact figures of the two outcomes combined, and the percentiles
// of the digest the walk fed with both.
func (s Summary) All() Figures {
	all := merge(s.OK, s.Failed)
	all.digest = s.all

	return all
}

// Share returns count as a percentage of all requests of the run, and false
// when the run holds none. It divides before it multiplies, as Gatling does:
// 12 of 36 is 33.33333333333333, where 100·12/36 would be 33.333333333333336.
func (s Summary) Share(count int) (float64, bool) {
	total := s.OK.count + s.Failed.count
	if total == 0 {
		return 0, false
	}

	return float64(count) / float64(total) * 100, true
}

// Untimed returns how many requests have no recorded end. They are counted, and
// take part in no timing figure.
func (s Summary) Untimed() int {
	return s.OK.count - s.OK.timed + s.Failed.count - s.Failed.timed
}

// Span returns how long the run lasted, from where parsec's bounds say it began
// to where they say it ended, and false when the bounds cannot place the run in
// time. A run that began and ended at one instant spans zero.
func (s Summary) Span() (time.Duration, bool) {
	start, ok := s.Tally.Bounds.Start()
	if !ok {
		return 0, false
	}

	end, ok := s.Tally.Bounds.End()
	if !ok {
		return 0, false
	}

	return end.Sub(start), true
}

// Rate returns count requests per second of the run: count divided by the
// run's span in whole seconds, rounded up, which is the one divisor Gatling
// uses for every rate of a run. It returns false when the run cannot be
// timed or spans no time, so that a rate is never shown as infinite and a
// span is never replaced by a second nobody measured. A count of zero over a
// known span is a rate of 0.
func (s Summary) Rate(count int) (float64, bool) {
	span, ok := s.Span()
	if !ok || span <= 0 {
		return 0, false
	}

	// Rounded up by dividing first: a damaged timestamp can leave a span at the
	// largest a duration holds, where adding a second before the division would
	// wrap it negative and turn every rate of the run into -0.
	seconds := span / time.Second
	if span%time.Second != 0 {
		seconds++
	}

	return float64(count) / float64(seconds), true
}
