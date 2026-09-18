package reporttest

import (
	"math"
	"testing"
)

func TestInterpolated(t *testing.T) {
	t.Parallel()

	// The 3.13.1 run's shape: 96 requests of at most 7 ms, then six of 1502 ms.
	sorted := make([]int64, 102)
	for i := range sorted {
		sorted[i] = 7
		if i >= 96 {
			sorted[i] = 1502
		}
	}

	tests := []struct {
		name         string
		sorted       []int64
		rank         float64
		value        float64
		lower, upper int64
	}{
		{name: "between a gap's two sides", sorted: sorted, rank: 95, value: 1427.25, lower: 7, upper: 1502},
		{name: "on a recorded value", sorted: []int64{1, 2, 3, 4, 5}, rank: 50, value: 3, lower: 3, upper: 4},
		{name: "the top rank", sorted: []int64{1, 2, 3}, rank: 100, value: 3, lower: 3, upper: 3},
		{name: "one request", sorted: []int64{42}, rank: 1, value: 42, lower: 42, upper: 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			value, lower, upper := Interpolated(tt.sorted, tt.rank)
			if math.Abs(value-tt.value) > 1e-9 || lower != tt.lower || upper != tt.upper {
				t.Errorf("Interpolated(p%v) = %v, %d, %d; want %v, %d, %d", tt.rank, value, lower, upper, tt.value, tt.lower, tt.upper)
			}
		})
	}
}

func TestAtRank(t *testing.T) {
	t.Parallel()

	sorted := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}

	for rank, expected := range map[float64]int64{0.1: 10, 10: 10, 10.5: 20, 50: 50, 95: 100, 100: 100} {
		if got := AtRank(sorted, rank); got != expected {
			t.Errorf("AtRank(p%v) = %d, want %d", rank, got, expected)
		}
	}
}

func TestRankTolerance(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		rank     float64
		n        int
		expected float64
	}{
		{rank: 95, n: 1_000_000, expected: 0.0019 + 1e-6},
		{rank: 50, n: 100, expected: 0.01 + 0.01},
		{rank: 100, n: 10, expected: 0.1},
	} {
		if got := RankTolerance(tt.rank, tt.n); math.Abs(got-tt.expected) > 1e-12 {
			t.Errorf("RankTolerance(p%v, %d) = %v, want %v", tt.rank, tt.n, got, tt.expected)
		}
	}
}

func TestRankMisplacement(t *testing.T) {
	t.Parallel()

	sorted := []int64{1, 2, 2, 2, 3, 4, 5, 6, 7, 8}

	tests := []struct {
		name     string
		value    int64
		rank     float64
		expected float64
	}{
		{name: "a tied value spans its ranks", value: 2, rank: 30, expected: 0},
		{name: "a value below the rank", value: 2, rank: 60, expected: 0.2},
		{name: "a value above the rank", value: 8, rank: 50, expected: 0.4},
		{name: "a recorded value at its own rank", value: 5, rank: 65, expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := RankMisplacement(sorted, tt.value, tt.rank); math.Abs(got-tt.expected) > 1e-9 {
				t.Errorf("RankMisplacement(%d, p%v) = %v, want %v", tt.value, tt.rank, got, tt.expected)
			}
		})
	}
}
