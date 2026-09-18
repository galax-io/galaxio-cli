package report

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/galax-io/parsec/model"
)

// figuresOf accumulates one outcome's figures from response times in whole
// milliseconds, the way a walk feeds them.
func figuresOf(ms ...int64) Figures {
	var f Figures

	for _, v := range ms {
		f.add(model.Some(time.Duration(v) * time.Millisecond))
	}

	return f
}

// reading is every figure a caller can read, in a form a test can compare.
type reading struct {
	count            int
	minimum, maximum int64
	mean, stdDev     int64
	timed            bool
}

func readingOf(tb testing.TB, f Figures) reading {
	tb.Helper()

	r := reading{count: f.Count()}

	var ok [4]bool

	r.minimum, ok[0] = f.Min()
	r.maximum, ok[1] = f.Max()
	r.mean, ok[2] = f.Mean()
	r.stdDev, ok[3] = f.StdDev()

	r.timed = ok[0]
	if ok != [4]bool{r.timed, r.timed, r.timed, r.timed} {
		tb.Fatalf("some timing figures are present and others absent: %v", ok)
	}

	return r
}

// TestFiguresArithmetic holds each rule to the case that tells it from the
// alternative Gatling's own recorded figures rejected.
func TestFiguresArithmetic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ms       []int64
		expected reading
	}{
		{
			// 3.13.1's "outer / inner, with comma / GET /fail": an exact mean of
			// 0.5, which Gatling recorded as 1, and a deviation of exactly 0.5.
			name:     "three of 0 ms and three of 1 ms round half up",
			ms:       []int64{0, 0, 0, 1, 1, 1},
			expected: reading{count: 6, minimum: 0, maximum: 1, mean: 1, stdDev: 1, timed: true},
		},
		{
			name:     "a mean of exactly 2.5 rounds up, not to even",
			ms:       []int64{2, 3},
			expected: reading{count: 2, minimum: 2, maximum: 3, mean: 3, stdDev: 1, timed: true},
		},
		{
			name:     "a deviation of exactly 1.5 rounds up",
			ms:       []int64{0, 3},
			expected: reading{count: 2, minimum: 0, maximum: 3, mean: 2, stdDev: 2, timed: true},
		},
		{
			// The sample deviation of {0, 4} is 2.83, which rounds to 3.
			name:     "the population deviation, not the sample one",
			ms:       []int64{0, 4},
			expected: reading{count: 2, minimum: 0, maximum: 4, mean: 2, stdDev: 2, timed: true},
		},
		{
			// About the unrounded mean of ⅓ the deviation is 0.47; about the
			// rounded mean of 0 it would be 0.58, which rounds to 1.
			name:     "the deviation is taken about the unrounded mean",
			ms:       []int64{0, 0, 1},
			expected: reading{count: 3, minimum: 0, maximum: 1, mean: 0, stdDev: 0, timed: true},
		},
		{
			name:     "one request is every figure",
			ms:       []int64{1503},
			expected: reading{count: 1, minimum: 1503, maximum: 1503, mean: 1503, stdDev: 0, timed: true},
		},
		{
			name:     "equal requests deviate by nothing",
			ms:       []int64{7, 7, 7, 7},
			expected: reading{count: 4, minimum: 7, maximum: 7, mean: 7, stdDev: 0, timed: true},
		},
		{
			name:     "no request has no timing figure",
			ms:       nil,
			expected: reading{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := readingOf(t, figuresOf(tt.ms...)); got != tt.expected {
				t.Errorf("figures of %v = %+v, want %+v", tt.ms, got, tt.expected)
			}
		})
	}
}

// TestFiguresWithoutARecordedEnd counts a request whose end the source did not
// record, and keeps it out of every timing figure rather than timing it at 0.
func TestFiguresWithoutARecordedEnd(t *testing.T) {
	t.Parallel()

	var f Figures

	f.add(model.Opt[time.Duration]{})
	f.add(model.Some(-time.Millisecond))

	if got, expected := readingOf(t, f), (reading{count: 2}); got != expected {
		t.Fatalf("figures of two requests with no usable end = %+v, want %+v", got, expected)
	}

	f.add(model.Some(40 * time.Millisecond))

	if got, expected := readingOf(t, f), (reading{count: 3, minimum: 40, maximum: 40, mean: 40, timed: true}); got != expected {
		t.Errorf("figures once a timed request arrives = %+v, want %+v", got, expected)
	}
}

// TestFiguresSumsCarry drives both sums across a word boundary, which no corpus
// run comes near, and checks the words against the arithmetic they encode.
func TestFiguresSumsCarry(t *testing.T) {
	t.Parallel()

	t.Run("each sum carries into its next word", func(t *testing.T) {
		t.Parallel()

		// 2⁴² + 1 ms, about 139 years, is within what a duration can hold. Its
		// square is 2⁸⁴ + 2⁴³ + 1, whose low word is 2⁴³ + 1 and whose high word
		// is 2²⁰: adding it to a full low word carries into the middle word, and
		// that one into the high word. A square of 2⁸⁴ alone would not — its low
		// word is zero, which is why this reaches further than it looks.
		const x = int64(1)<<42 + 1

		f := Figures{
			count: 1, timed: 1, minimum: x, maximum: x,
			sum:     [2]uint64{math.MaxUint64, 0},
			squares: [3]uint64{math.MaxUint64, math.MaxUint64, 0},
		}

		f.add(model.Some(time.Duration(x) * time.Millisecond))

		if expected := [2]uint64{1 << 42, 1}; f.sum != expected {
			t.Errorf("sum words = %#x, want %#x", f.sum, expected)
		}

		if expected := [3]uint64{1 << 43, 1 << 20, 1}; f.squares != expected {
			t.Errorf("sum of squares words = %#x, want %#x", f.squares, expected)
		}
	})

	t.Run("figures read across the words", func(t *testing.T) {
		t.Parallel()

		// 2²³ requests, half at 2⁴² ms and half at 2⁴² − 2 ms: a sum past 64
		// bits whose mean is 2⁴² − 1 and whose deviation is exactly 1. The
		// accumulators are set to what feeding them would leave, rather than
		// fed eight million times.
		const n = 1 << 23

		big2 := func(exp uint) *big.Int { return new(big.Int).Lsh(big.NewInt(1), exp) }

		sum := new(big.Int).Sub(big2(65), big2(23))

		squares := new(big.Int).Sub(big2(107), big2(66))
		squares.Add(squares, big2(24))

		f := Figures{count: n, timed: n, minimum: 1<<42 - 2, maximum: 1 << 42}
		copy(f.sum[:], toWords(sum, 2))
		copy(f.squares[:], toWords(squares, 3))

		if f.sum[1] == 0 || f.squares[1] == 0 {
			t.Fatalf("the fixture does not reach a second word: sum %#x, squares %#x", f.sum, f.squares)
		}

		if mean, _ := f.Mean(); mean != 1<<42-1 {
			t.Errorf("Mean = %d, want %d", mean, int64(1<<42-1))
		}

		if stdDev, _ := f.StdDev(); stdDev != 1 {
			t.Errorf("StdDev = %d, want 1", stdDev)
		}
	})
}

// toWords splits a non-negative z into n 64-bit words, low word first.
func toWords(z *big.Int, n int) []uint64 {
	words := make([]uint64, n)
	mask := new(big.Int).SetUint64(math.MaxUint64)
	rest := new(big.Int).Set(z)

	for i := range words {
		words[i] = new(big.Int).And(rest, mask).Uint64()
		rest.Rsh(rest, 64)
	}

	return words
}

// TestMerge holds the figures of two outcomes together to the figures of their
// requests fed as one.
func TestMerge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b []int64
	}{
		{name: "both timed", a: []int64{10, 20}, b: []int64{1000}},
		{name: "the second empty", a: []int64{5, 9}, b: nil},
		{name: "the first empty", a: nil, b: []int64{7, 3}},
		{name: "both empty", a: nil, b: nil},
		{name: "extremes from different sides", a: []int64{50, 60}, b: []int64{1, 900}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := readingOf(t, merge(figuresOf(tt.a...), figuresOf(tt.b...)))
			expected := readingOf(t, figuresOf(append(append([]int64(nil), tt.a...), tt.b...)...))

			if got != expected {
				t.Errorf("merge(%v, %v) = %+v, want %+v", tt.a, tt.b, got, expected)
			}
		})
	}

	t.Run("an outcome whose requests have no recorded end", func(t *testing.T) {
		t.Parallel()

		// Its count is not zero and its extremes are the zero value, so a merge
		// that asked whether the outcome held requests, rather than whether any
		// of them was timed, would report a minimum of 0 ms for the run.
		var untimed Figures

		untimed.add(model.Opt[time.Duration]{})
		untimed.add(model.Opt[time.Duration]{})

		timed := figuresOf(7, 900)

		for _, m := range [2]Figures{merge(untimed, timed), merge(timed, untimed)} {
			if m.count != 4 || m.timed != 2 {
				t.Errorf("merged counts = %d of %d, want 2 of 4", m.timed, m.count)
			}

			if minimum, _ := m.Min(); minimum != 7 {
				t.Errorf("merged minimum = %d, want the fastest request that has a time", minimum)
			}

			if maximum, _ := m.Max(); maximum != 900 {
				t.Errorf("merged maximum = %d, want the slowest request that has a time", maximum)
			}
		}
	})

	t.Run("both sums carry", func(t *testing.T) {
		t.Parallel()

		// Two halves whose words add past 2⁶⁴, so that every carry of the merge
		// is used. The accumulators are set to what feeding them would leave.
		half := Figures{
			count: 1, timed: 1, minimum: 1, maximum: 1,
			sum:     [2]uint64{math.MaxUint64, 3},
			squares: [3]uint64{math.MaxUint64, math.MaxUint64, 5},
		}

		m := merge(half, half)

		if expected := [2]uint64{math.MaxUint64 - 1, 7}; m.sum != expected {
			t.Errorf("merged sum words = %#x, want %#x", m.sum, expected)
		}

		if expected := [3]uint64{math.MaxUint64 - 1, math.MaxUint64, 11}; m.squares != expected {
			t.Errorf("merged sum of squares words = %#x, want %#x", m.squares, expected)
		}
	})
}
