package report

import (
	"math/big"
	"math/bits"
	"time"

	"github.com/galax-io/parsec/model"
)

// Figures are the response-time figures of one outcome of a run — its
// successful requests or its failed ones — accumulated in one pass without
// keeping a single request.
//
// Every figure here is exact. Durations are whole milliseconds; the sum is kept
// in 128 bits and the sum of squares in 192, and the mean and the deviation are
// divided and rooted in integers only when they are read. A duration is at most
// 2⁶³−1 ns, under 2⁴³ ms, and a count is at most 2⁶³, so the sum stays under
// 2¹⁰⁶ and the sum of squares under 2¹⁴⁹: neither can wrap, whatever a log holds.
//
// A figure that does not exist — any timing figure of an outcome no request
// with a recorded end reached — is reported as absent, never as zero, because
// zero is a real response time.
type Figures struct {
	count int
	timed int

	minimum, maximum int64

	// sum and squares hold the sum of the recorded durations in milliseconds
	// and the sum of their squares, as 64-bit words, low word first.
	sum     [2]uint64
	squares [3]uint64
}

// add counts one request of this outcome. A request whose end the source did
// not record is counted and takes part in no timing figure; so is a negative
// duration, which parsec promises never to yield and its own bounds already
// treat as no end.
func (f *Figures) add(duration model.Opt[time.Duration]) {
	f.count++

	d, ok := duration.Get()
	if !ok || d < 0 {
		return
	}

	ms := d.Milliseconds()

	if f.timed == 0 {
		f.minimum, f.maximum = ms, ms
	} else {
		f.minimum, f.maximum = min(f.minimum, ms), max(f.maximum, ms)
	}

	f.timed++

	x := uint64(ms)

	var carry uint64

	f.sum[0], carry = bits.Add64(f.sum[0], x, 0)
	f.sum[1] += carry

	high, low := bits.Mul64(x, x)
	f.squares[0], carry = bits.Add64(f.squares[0], low, 0)
	f.squares[1], carry = bits.Add64(f.squares[1], high, carry)
	f.squares[2] += carry
}

// merge returns the figures of a and b together, exactly as if every request
// of both had been added to one. The bound above holds for the two together,
// so no word carries out of its sum.
func merge(a, b Figures) Figures {
	m := Figures{count: a.count + b.count, timed: a.timed + b.timed}

	switch {
	case a.timed == 0:
		m.minimum, m.maximum = b.minimum, b.maximum
	case b.timed == 0:
		m.minimum, m.maximum = a.minimum, a.maximum
	default:
		m.minimum, m.maximum = min(a.minimum, b.minimum), max(a.maximum, b.maximum)
	}

	var carry uint64

	m.sum[0], carry = bits.Add64(a.sum[0], b.sum[0], 0)
	m.sum[1] = a.sum[1] + b.sum[1] + carry

	m.squares[0], carry = bits.Add64(a.squares[0], b.squares[0], 0)
	m.squares[1], carry = bits.Add64(a.squares[1], b.squares[1], carry)
	m.squares[2] = a.squares[2] + b.squares[2] + carry

	return m
}

// Count returns how many requests of this outcome the run holds, with or
// without a recorded end.
func (f Figures) Count() int {
	return f.count
}

// Min returns the fastest recorded response time in milliseconds, and false
// when no request of this outcome has a recorded end.
func (f Figures) Min() (int64, bool) {
	return f.minimum, f.timed > 0
}

// Max returns the slowest recorded response time in milliseconds, and false
// when no request of this outcome has a recorded end.
func (f Figures) Max() (int64, bool) {
	return f.maximum, f.timed > 0
}

// Mean returns the arithmetic mean of the recorded response times, rounded half
// up to a whole millisecond as Gatling rounds it — 0.5 ms is 1 — and false when
// no request of this outcome has a recorded end.
func (f Figures) Mean() (int64, bool) {
	if f.timed == 0 {
		return 0, false
	}

	n := big.NewInt(int64(f.timed))

	// floor((2·sum + n) / (2·n)) is floor(sum/n + 1/2), in integers.
	numerator := fromWords(f.sum[:])
	numerator.Lsh(numerator, 1).Add(numerator, n)

	denominator := new(big.Int).Lsh(n, 1)

	return numerator.Quo(numerator, denominator).Int64(), true
}

// StdDev returns the population standard deviation of the recorded response
// times, taken about their unrounded mean and rounded half up to a whole
// millisecond, and false when no request of this outcome has a recorded end.
// These are the definitions Gatling's recorded figures reproduce: a sample
// deviation, or one about the rounded mean, does not.
func (f Figures) StdDev() (int64, bool) {
	if f.timed == 0 {
		return 0, false
	}

	n := big.NewInt(int64(f.timed))
	sum := fromWords(f.sum[:])

	// The variance is numerator / denominator, with the numerator
	// n·Σx² − (Σx)², which is never negative, and the denominator n².
	numerator := fromWords(f.squares[:])
	numerator.Mul(numerator, n).Sub(numerator, new(big.Int).Mul(sum, sum))

	denominator := new(big.Int).Mul(n, n)

	// k is the deviation rounded down: floor(√(numerator / denominator)) is
	// the integer square root of the floored quotient.
	k := new(big.Int).Quo(numerator, denominator)
	k.Sqrt(k)

	// Rounding half up raises k by one when the deviation reaches k + ½, that
	// is when 4·numerator ≥ (2k + 1)²·denominator.
	odd := new(big.Int).Lsh(k, 1)
	odd.Add(odd, big.NewInt(1))

	threshold := new(big.Int).Mul(odd, odd)
	threshold.Mul(threshold, denominator)

	if new(big.Int).Lsh(numerator, 2).Cmp(threshold) >= 0 {
		k.Add(k, big.NewInt(1))
	}

	return k.Int64(), true
}

// fromWords returns the unsigned integer whose 64-bit words, low word first,
// are words. It shifts rather than handing the words to big.Int directly,
// because a big.Word is 32 bits wide on a 32-bit platform.
func fromWords(words []uint64) *big.Int {
	z := new(big.Int)

	for i := len(words) - 1; i >= 0; i-- {
		z.Lsh(z, 64)
		z.Or(z, new(big.Int).SetUint64(words[i]))
	}

	return z
}
