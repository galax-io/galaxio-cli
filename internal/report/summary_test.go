package report

import (
	"context"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/galax-io/parsec/model"
)

// TestDefaultOptions pins Gatling's own settings, which a summary is compared
// against by default: a changed default is a changed report.
func TestDefaultOptions(t *testing.T) {
	t.Parallel()

	opts := DefaultOptions()

	if expected := []float64{50, 75, 95, 99}; !slices.Equal(opts.Percentiles, expected) {
		t.Errorf("Percentiles = %v, want %v", opts.Percentiles, expected)
	}

	if expected := (Bands{Lower: 800, Upper: 1200}); opts.Bands != expected {
		t.Errorf("Bands = %+v, want %+v", opts.Bands, expected)
	}

	// Each call hands out its own ranks, so a caller that edits them cannot
	// change what the next caller is given.
	opts.Percentiles[0] = 1
	if DefaultOptions().Percentiles[0] != 50 {
		t.Errorf("editing one DefaultOptions result changed the next one")
	}
}

func TestValidRank(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rank     float64
		expected bool
	}{
		{name: "zero", rank: 0, expected: false},
		{name: "negative", rank: -1, expected: false},
		{name: "just above zero", rank: 0.0001, expected: true},
		{name: "median", rank: 50, expected: true},
		{name: "fraction", rank: 99.9, expected: true},
		{name: "one hundred", rank: 100, expected: true},
		{name: "just above one hundred", rank: 100.0001, expected: false},
		{name: "not a number", rank: math.NaN(), expected: false},
		{name: "infinity", rank: math.Inf(1), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ValidRank(tt.rank); got != tt.expected {
				t.Errorf("ValidRank(%v) = %v, want %v", tt.rank, got, tt.expected)
			}
		})
	}
}

func TestBandsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bands    Bands
		expected bool
	}{
		{name: "gatling's defaults", bands: Bands{Lower: 800, Upper: 1200}, expected: true},
		{name: "a lower boundary of zero", bands: Bands{Lower: 0, Upper: 1}, expected: true},
		{name: "equal boundaries", bands: Bands{Lower: 800, Upper: 800}, expected: false},
		{name: "decreasing boundaries", bands: Bands{Lower: 1200, Upper: 800}, expected: false},
		{name: "a negative lower boundary", bands: Bands{Lower: -1, Upper: 5}, expected: false},
		{name: "both zero", bands: Bands{}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.bands.Valid(); got != tt.expected {
				t.Errorf("%+v.Valid() = %v, want %v", tt.bands, got, tt.expected)
			}
		})
	}
}

// TestScanOptions holds Scan to the options it is given: ranks come back sorted
// and each once, and options no summary can be computed at are refused before
// a single item is read.
func TestScanOptions(t *testing.T) {
	t.Parallel()

	t.Run("ranks are sorted and each kept once", func(t *testing.T) {
		t.Parallel()

		given := []float64{99, 50, 99.9, 50, 90}
		opts := Options{Percentiles: given, Bands: Bands{Lower: 5, Upper: 1000}}

		summary, err := Scan(context.Background(), &stubReader{}, opts)
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}

		if expected := []float64{50, 90, 99, 99.9}; !slices.Equal(summary.Options.Percentiles, expected) {
			t.Errorf("Options.Percentiles = %v, want %v", summary.Options.Percentiles, expected)
		}

		if summary.Options.Bands != opts.Bands {
			t.Errorf("Options.Bands = %+v, want %+v", summary.Options.Bands, opts.Bands)
		}

		if expected := []float64{99, 50, 99.9, 50, 90}; !slices.Equal(given, expected) {
			t.Errorf("Scan reordered the caller's ranks: %v", given)
		}
	})

	refusals := []struct {
		name     string
		opts     Options
		expected string
	}{
		{name: "no rank", opts: Options{Bands: Bands{Lower: 800, Upper: 1200}}, expected: "no percentile rank given"},
		{name: "a rank of zero", opts: Options{Percentiles: []float64{50, 0}, Bands: Bands{Lower: 800, Upper: 1200}}, expected: "percentile rank 0 is not above 0 and at most 100"},
		{name: "a rank above one hundred", opts: Options{Percentiles: []float64{101}, Bands: Bands{Lower: 800, Upper: 1200}}, expected: "percentile rank 101 is not above 0 and at most 100"},
		{name: "boundaries that do not increase", opts: Options{Percentiles: []float64{50}, Bands: Bands{Lower: 1200, Upper: 800}}, expected: "band boundaries 1200 and 800"},
		{name: "a negative boundary", opts: Options{Percentiles: []float64{50}, Bands: Bands{Lower: -1, Upper: 5}}, expected: "band boundaries -1 and 5"},
	}

	for _, tt := range refusals {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rd := &stubReader{items: []model.Item{{Kind: model.ItemSample, Sample: model.Sample{
				Name: "r", Start: time.UnixMilli(1000).UTC(), Duration: model.Some(10 * time.Millisecond), Outcome: model.OutcomeSuccess,
			}}}}

			summary, err := Scan(context.Background(), rd, tt.opts)
			if err == nil || !strings.Contains(err.Error(), tt.expected) {
				t.Fatalf("Scan = %v, want an error containing %q", err, tt.expected)
			}

			if len(rd.items) != 1 || summary.Tally != (Tally{}) {
				t.Errorf("Scan read the run although its options were refused: %+v, %d items left", summary.Tally, len(rd.items))
			}
		})
	}
}
