package reporttest

import (
	"errors"
	"io"
	"math"
	"slices"
	"sort"

	"github.com/galax-io/parsec/gatling/simlog"
	"github.com/galax-io/parsec/model"
)

// Samples walks the run rd yields and calls yield with the outcome and the
// response time, in whole milliseconds, of every request that counts towards a
// figure: one the source recorded an outcome and an end for. It is the one
// reading of what a run holds, so that the etalon, the rank rule and a test's
// own digest are never fed different sets of requests.
func Samples(rd simlog.RunReader, yield func(outcome model.Outcome, ms int64)) error {
	for {
		item, err := rd.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return err
		}

		if item.Kind != model.ItemSample {
			continue
		}

		if item.Sample.Outcome != model.OutcomeSuccess && item.Sample.Outcome != model.OutcomeFailure {
			continue
		}

		if d, ok := item.Sample.Duration.Get(); ok && d >= 0 {
			yield(item.Sample.Outcome, d.Milliseconds())
		}
	}
}

// Durations walks the run rd yields and returns the response times, in whole
// milliseconds, of all, successful and failed requests, each sorted. These are
// the exact values a percentile estimate is held to.
func Durations(rd simlog.RunReader) ([3][]int64, error) {
	var durations [3][]int64

	err := Samples(rd, func(outcome model.Outcome, ms int64) {
		durations[0] = append(durations[0], ms)

		if outcome == model.OutcomeSuccess {
			durations[1] = append(durations[1], ms)
		} else {
			durations[2] = append(durations[2], ms)
		}
	})
	if err != nil {
		return [3][]int64{}, err
	}

	for i := range durations {
		slices.Sort(durations[i])
	}

	return durations, nil
}

// Interpolated returns the value at rank, a percentile rank above 0 and at most
// 100, by linear interpolation between the two sorted response times around
// the position rank/100·(n−1), and those two neighbours. It is how a t-digest
// that has merged no centroid reads a quantile, computed the same way.
func Interpolated(sorted []int64, rank float64) (value float64, lower, upper int64) {
	index := rank / 100 * float64(len(sorted)-1)
	i := int(math.Floor(index))

	if i+1 >= len(sorted) {
		return float64(sorted[i]), sorted[i], sorted[i]
	}

	lowerWeight := float64(i+1) - index
	upperWeight := index - float64(i)

	return float64(sorted[i])*lowerWeight + float64(sorted[i+1])*upperWeight, sorted[i], sorted[i+1]
}

// AtRank returns the request standing at the percentile's rank: the
// ⌈rank/100·n⌉-th smallest response time.
func AtRank(sorted []int64, rank float64) int64 {
	position := int(math.Ceil(rank / 100 * float64(len(sorted))))

	return sorted[max(position, 1)-1]
}

// RankTolerance is how far the rule lets an estimate misplace its rank among n
// requests: 4·q·(1−q)/100 of them, the most a t-digest at compression 100
// merges around the quantile q, plus one request for the step between two
// recorded values.
func RankTolerance(rank float64, n int) float64 {
	q := rank / 100

	return 4*q*(1-q)/100 + 1/float64(n)
}

// RankMisplacement returns how far the quantile rank/100 lies outside the range
// value occupies among the sorted response times — from the share strictly
// below it to the share at or below it — and zero when it lies inside.
func RankMisplacement(sorted []int64, value int64, rank float64) float64 {
	n := float64(len(sorted))
	below := float64(sort.Search(len(sorted), func(i int) bool { return sorted[i] >= value })) / n
	atOrBelow := float64(sort.Search(len(sorted), func(i int) bool { return sorted[i] > value })) / n

	switch q := rank / 100; {
	case q < below:
		return below - q
	case q > atOrBelow:
		return q - atOrBelow
	default:
		return 0
	}
}

// Items returns a reader over items, as a run reader yields them, for the run
// shapes no Gatling log produces. Handed will hold how many it has yielded when
// it is not nil, which a test that reads the walk midway asks for.
func Items(run model.Run, handed *int, items ...model.Item) simlog.RunReader {
	return &itemsReader{run: run, items: items, handed: handed}
}

type itemsReader struct {
	run    model.Run
	items  []model.Item
	handed *int
}

func (r *itemsReader) Run() model.Run { return r.run }

func (r *itemsReader) Next() (model.Item, error) {
	if len(r.items) == 0 {
		return model.Item{}, io.EOF
	}

	item := r.items[0]
	r.items = r.items[1:]

	if r.handed != nil {
		*r.handed++
	}

	return item, nil
}
