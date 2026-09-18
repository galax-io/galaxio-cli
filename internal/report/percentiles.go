package report

import (
	"math"

	tdigest "github.com/caio/go-tdigest/v5"
)

// Percentile returns the response time at rank, a percentile rank above 0 and
// at most 100, rounded half up to a whole millisecond, and false when no request
// of this outcome has a recorded end or rank is not a percentile rank.
//
// Unlike every other figure, a percentile is an estimate: it is read from a
// t-digest, github.com/caio/go-tdigest at its defaults. It is the number Gatling
// 3.11 and 3.12 print for the same log, whose digest — AVLTreeDigest(100) of
// t-digest 3.1, read as Math.round(quantile) — groups and interpolates the same
// way; the tests hold every percentile of every recorded run to that digest
// (constitution Principle II). The quantile interpolates between neighbouring
// recorded values, so where response times have a gap it can be a value no
// request had — 1427 ms for the 95th percentile of the recorded 3.13.1 run,
// whose request at that rank took 1502 ms, and 1427 is what Gatling 3.11's
// digest gives for it too. caio/go-tdigest#42 proposes a read by rank that
// returns the recorded value, which would part from Gatling 3.11's numbers. The
// same requests in the same order always give the same percentiles.
//
// How far it may differ is a rule the tests hold every percentile to. While an
// outcome holds at most 200 requests, the digest has merged nothing and the
// percentile is the interpolation between the two recorded response times
// around the position rank/100·(n−1): it lies between them. At any size it
// misplaces its rank among the recorded requests by at most 4·q·(1−q)/100 of
// them plus one request, q being rank/100.
func (f Figures) Percentile(rank float64) (int64, bool) {
	if f.digest == nil || !ValidRank(rank) {
		return 0, false
	}

	return roundHalfUp(quantile(f.digest, rank/100)), true
}

// quantile reads d at q, from 0 to 1, as AVLTreeDigest.quantile of t-digest 3.1
// does, operation for operation: the position q·(n−1) among the requests, the
// two centroids whose centres lie around it, and the line between them, drawn
// past the first centre and the last one where the position lies outside them.
// It returns NaN for a digest that holds nothing.
//
// The library's own Quantile is that arithmetic too, and on amd64 it gives
// t-digest 3.1's numbers bit for bit. On arm64 the compiler fuses each of its
// products into the addition that follows, which moves the last bit of the
// result, and a percentile that lies at a half then rounds the other way: for
// the requests [43, 53] the 95th percentile is 52.5 in Java, and 53 by
// Math.round, where the fused read gave 52.49999999999999 and 52. Every product
// below is therefore converted to float64 before it is used, which is what
// keeps an operation from being fused, so that every architecture computes what
// Java computes.
func quantile(d *tdigest.TDigest, q float64) float64 {
	var means, counts []float64

	d.ForEachCentroid(func(mean float64, count uint64) bool {
		means, counts = append(means, mean), append(counts, float64(count))

		return true
	})

	switch len(means) {
	case 0:
		return math.NaN()
	case 1:
		return means[0]
	}

	last := float64(d.Count() - 1)
	index := float64(q * last)

	// next is the last centroid that begins at or before the position, and
	// total the requests before it.
	next, total := 0, 0.0

	for next+1 < len(means) && total+counts[next] <= index {
		total += counts[next]
		next++
	}

	previousMean, previousIndex := math.NaN(), 0.0
	if next > 0 {
		previousMean, previousIndex = means[next-1], total-(counts[next-1]+1)/2
	}

	for {
		nextIndex := total + (counts[next]-1)/2

		switch {
		case nextIndex >= index:
			if math.IsNaN(previousMean) {
				// The position lies before the centre of the first centroid.
				if nextIndex == previousIndex {
					return means[next]
				}

				nextIndex2 := total + counts[next] + (counts[next+1]-1)/2
				previousMean = (float64(nextIndex2*means[next]) - float64(nextIndex*means[next+1])) / (nextIndex2 - nextIndex)
			}

			return between(index, previousIndex, nextIndex, previousMean, means[next])
		case next+1 == len(means):
			// The position lies after the centre of the last centroid.
			afterMean := (float64(means[next]*(last-previousIndex)) - float64(previousMean*(last-nextIndex))) / (nextIndex - previousIndex)

			return between(index, nextIndex, last, means[next], afterMean)
		}

		total += counts[next]
		previousMean, previousIndex = means[next], nextIndex
		next++
	}
}

// between returns the value at index on the line from previousMean at
// previousIndex to nextMean at nextIndex, with the weights divided before they
// are multiplied and each product rounded on its own, as t-digest 3.1 has it.
func between(index, previousIndex, nextIndex, previousMean, nextMean float64) float64 {
	delta := nextIndex - previousIndex
	previousWeight := (nextIndex - index) / delta
	nextWeight := (index - previousIndex) / delta

	return float64(previousMean*previousWeight) + float64(nextMean*nextWeight)
}

// roundHalfUp rounds a non-negative estimate to a whole millisecond as
// Math.round does: to the nearest, and up from exactly a half. The fraction of
// a float64 is exact, so a value one bit below a half stays below it, where
// floor(v + 0.5) would carry 0.49999999999999994 up to 1.
func roundHalfUp(v float64) int64 {
	whole := math.Floor(v)
	if v-whole >= 0.5 {
		whole++
	}

	return int64(whole)
}

// estimate feeds one recorded response time to the outcome's digest, creating
// the digest on the first.
func (f *Figures) estimate(ms int64) {
	if f.digest == nil {
		f.digest = newDigest()
	}

	// Add fails only for NaN, and a whole number of milliseconds never is.
	_ = f.digest.Add(float64(ms))
}

// newDigest returns a t-digest at the library's defaults: compression 100, and
// its own generator seeded with a constant, so the same requests in the same
// order always make the same digest. New fails only when an option does, and
// none is given.
func newDigest() *tdigest.TDigest {
	d, _ := tdigest.New()

	return d
}
