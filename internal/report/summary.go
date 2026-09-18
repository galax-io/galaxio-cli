package report

import (
	"errors"
	"fmt"
	"slices"
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

// normalize returns o with its ranks sorted and each kept once, or an error
// naming the rank or the boundaries no summary can be computed at. The ranks
// are copied, so the caller's slice is never reordered under it.
func (o Options) normalize() (Options, error) {
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
// holds and the options it was summarised at.
type Summary struct {
	// Tally counts what the log holds, as milestone v0.13.0 counted it.
	Tally Tally
	// Options is what the run was summarised at, its ranks sorted and each
	// kept once.
	Options Options
}
