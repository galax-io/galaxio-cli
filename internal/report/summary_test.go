package report

import (
	"context"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
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

		summary, err := Scan(context.Background(), reporttest.Items(model.Run{}, nil), opts, nil)
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

	t.Run("the caller normalizes before it opens a run", func(t *testing.T) {
		t.Parallel()

		if _, err := (Options{Percentiles: []float64{50}, Bands: Bands{Lower: 800, Upper: 1200}}).Normalize(); err != nil {
			t.Errorf("Normalize refused options a summary can be computed at: %v", err)
		}

		if _, err := (Options{}).Normalize(); err == nil {
			t.Errorf("Normalize accepted options naming no rank")
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

			var handed int

			rd := reporttest.Items(model.Run{}, &handed, model.Item{Kind: model.ItemSample, Sample: model.Sample{
				Name: "r", Start: time.UnixMilli(1000).UTC(), Duration: model.Some(10 * time.Millisecond), Outcome: model.OutcomeSuccess,
			}})

			summary, err := Scan(context.Background(), rd, tt.opts, nil)
			if err == nil || !strings.Contains(err.Error(), tt.expected) {
				t.Fatalf("Scan = %v, want an error containing %q", err, tt.expected)
			}

			if handed != 0 || summary.Tally != (Tally{}) {
				t.Errorf("Scan read the run although its options were refused: %+v, %d items taken", summary.Tally, handed)
			}
		})
	}
}

// TestSummaryBands holds the bands to Gatling's: lower-inclusive boundaries,
// successful responses only, and a failure counted as failed whatever its time.
func TestSummaryBands(t *testing.T) {
	t.Parallel()

	sample := func(ms int64, outcome model.Outcome) model.Item {
		return model.Item{Kind: model.ItemSample, Sample: model.Sample{
			Name: "r", Start: time.UnixMilli(1000).UTC(), Duration: model.Some(time.Duration(ms) * time.Millisecond), Outcome: outcome,
		}}
	}

	untimed := model.Item{Kind: model.ItemSample, Sample: model.Sample{Name: "r", Start: time.UnixMilli(1000).UTC(), Outcome: model.OutcomeSuccess}}

	tests := []struct {
		name                   string
		bands                  Bands
		items                  []model.Item
		under, between, over   int
		expectedFailed, timeOK int
	}{
		{
			name:  "gatling's boundaries include their lower end",
			bands: Bands{Lower: 800, Upper: 1200},
			items: []model.Item{
				sample(799, model.OutcomeSuccess), sample(800, model.OutcomeSuccess),
				sample(1199, model.OutcomeSuccess), sample(1200, model.OutcomeSuccess),
			},
			under: 1, between: 2, over: 1, timeOK: 4,
		},
		{
			name:  "a slow failure is failed and in no timing band",
			bands: Bands{Lower: 800, Upper: 1200},
			items: []model.Item{sample(5000, model.OutcomeFailure), sample(10, model.OutcomeSuccess)},
			under: 1, expectedFailed: 1, timeOK: 1,
		},
		{
			name:    "a success with no recorded end is in no band",
			bands:   Bands{Lower: 800, Upper: 1200},
			items:   []model.Item{untimed, sample(900, model.OutcomeSuccess)},
			between: 1, timeOK: 1,
		},
		{
			name:  "the caller's own boundaries",
			bands: Bands{Lower: 5, Upper: 1000},
			items: []model.Item{
				sample(4, model.OutcomeSuccess), sample(5, model.OutcomeSuccess),
				sample(999, model.OutcomeSuccess), sample(1000, model.OutcomeSuccess),
			},
			under: 1, between: 2, over: 1, timeOK: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := DefaultOptions()
			opts.Bands = tt.bands

			summary, err := Scan(context.Background(), reporttest.Items(model.Run{}, nil, tt.items...), opts, nil)
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			got := [4]int{summary.Under, summary.Between, summary.Over, summary.Failed.Count()}
			if expected := [4]int{tt.under, tt.between, tt.over, tt.expectedFailed}; got != expected {
				t.Errorf("under/between/over/failed = %v, want %v", got, expected)
			}

			if sum, timed := summary.Under+summary.Between+summary.Over, summary.OK.Count()-summary.Untimed(); sum != tt.timeOK || sum != timed {
				t.Errorf("the timing bands hold %d, want %d, the successful requests with a recorded end", sum, timed)
			}
		})
	}
}

// TestSummaryAll holds the figures of all requests to both outcomes, and each
// outcome to its own requests: a failure reaches every figure of all requests
// and no figure of the successful ones.
func TestSummaryAll(t *testing.T) {
	t.Parallel()

	s := Summary{OK: figuresOf(10, 20), Failed: figuresOf(1000)}

	if got, expected := readingOf(t, s.All()), readingOf(t, figuresOf(10, 20, 1000)); got != expected {
		t.Errorf("All = %+v, want %+v", got, expected)
	}

	if got, expected := readingOf(t, s.OK), (reading{count: 2, minimum: 10, maximum: 20, mean: 15, stdDev: 5, timed: true}); got != expected {
		t.Errorf("OK = %+v, want %+v: a failure reached a figure of successful requests", got, expected)
	}
}

// TestSummaryShare divides before it multiplies, as Gatling does, and has no
// share of a run that holds no request.
func TestSummaryShare(t *testing.T) {
	t.Parallel()

	eighteen := make([]int64, 18)
	s := Summary{OK: figuresOf(eighteen...), Failed: figuresOf(eighteen...)}

	// 12 of 36, as 3.11.5 and 3.12.0 recorded it. The other order of
	// operations gives 33.333333333333336.
	if share, ok := s.Share(12); !ok || share != 33.33333333333333 {
		t.Errorf("Share(12) of 36 = %v (%v), want 33.33333333333333", share, ok)
	}

	if share, ok := (Summary{}).Share(0); ok {
		t.Errorf("Share of a run with no request = %v, want absent", share)
	}
}

// TestSummaryRate holds the rate to Gatling's one divisor: the run's span in
// whole seconds, rounded up, shared by every rate.
func TestSummaryRate(t *testing.T) {
	t.Parallel()

	at := func(ms int64) time.Time { return time.UnixMilli(1700000000000 + ms).UTC() }

	userEvent := func(kind model.UserEventKind, ms int64) model.Item {
		return model.Item{Kind: model.ItemUser, User: model.UserEvent{Kind: kind, At: at(ms)}}
	}

	tests := []struct {
		name         string
		items        []model.Item
		count        int
		expectedSpan time.Duration
		spanKnown    bool
		expectedRate float64
		rateKnown    bool
	}{
		{
			// 3.13.1 spans 3226 ms: four seconds, and 102 requests make 25.5.
			name:         "a part second counts as a whole one",
			items:        []model.Item{userEvent(model.UserStart, 0), userEvent(model.UserEnd, 3226)},
			count:        102,
			expectedSpan: 3226 * time.Millisecond, spanKnown: true,
			expectedRate: 25.5, rateKnown: true,
		},
		{
			name:         "whole seconds are not rounded up",
			items:        []model.Item{userEvent(model.UserStart, 0), userEvent(model.UserEnd, 3000)},
			count:        9,
			expectedSpan: 3 * time.Second, spanKnown: true,
			expectedRate: 3, rateKnown: true,
		},
		{
			name:         "one millisecond is one second",
			items:        []model.Item{userEvent(model.UserStart, 0), userEvent(model.UserEnd, 1)},
			count:        1,
			expectedSpan: time.Millisecond, spanKnown: true,
			expectedRate: 1, rateKnown: true,
		},
		{
			// A damaged timestamp leaves the span at the largest a duration
			// holds, about 292 years; rounding it up before dividing wrapped it
			// negative and made every rate of the run -0.
			name:         "a span the clock saturates has a rate, not a negative one",
			items:        []model.Item{userEvent(model.UserStart, 0), userEvent(model.UserEnd, math.MaxInt64/int64(time.Millisecond))},
			count:        2,
			expectedSpan: time.Duration(math.MaxInt64/int64(time.Millisecond)) * time.Millisecond, spanKnown: true,
			expectedRate: 2 / math.Ceil(float64(math.MaxInt64/int64(time.Millisecond))/1000), rateKnown: true,
		},
		{
			name:         "a count of zero over a known span is a rate of 0",
			items:        []model.Item{userEvent(model.UserStart, 0), userEvent(model.UserEnd, 3226)},
			count:        0,
			expectedSpan: 3226 * time.Millisecond, spanKnown: true,
			expectedRate: 0, rateKnown: true,
		},
		{
			name:         "a run at one instant spans zero and has no rate",
			items:        []model.Item{userEvent(model.UserStart, 0), userEvent(model.UserEnd, 0)},
			count:        1,
			expectedSpan: 0, spanKnown: true,
			rateKnown: false,
		},
		{
			name: "a run the bounds cannot place has no span and no rate",
			items: []model.Item{
				userEvent(model.UserStart, 0),
				{Kind: model.ItemSample, Sample: model.Sample{Name: "unplaced", Duration: model.Some(time.Millisecond), Outcome: model.OutcomeSuccess}},
				userEvent(model.UserEnd, 3226),
			},
			count:     1,
			spanKnown: false,
			rateKnown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			summary, err := Scan(context.Background(), reporttest.Items(model.Run{}, nil, tt.items...), DefaultOptions(), nil)
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			span, spanKnown := summary.Span()
			if span != tt.expectedSpan || spanKnown != tt.spanKnown {
				t.Errorf("Span = %v, %v; want %v, %v", span, spanKnown, tt.expectedSpan, tt.spanKnown)
			}

			rate, rateKnown := summary.Rate(tt.count)
			if rate != tt.expectedRate || rateKnown != tt.rateKnown {
				t.Errorf("Rate(%d) = %v, %v; want %v, %v", tt.count, rate, rateKnown, tt.expectedRate, tt.rateKnown)
			}
		})
	}
}

// TestSummaryMatchesGatling holds every non-percentile whole-run figure of every
// corpus run to what Gatling itself recorded for that run: its global_stats.json
// where it wrote one, and the Global Information block of its console where it
// did not. Both are read by reporttest, which the live recordings are compared
// through as well, so there is one reader of Gatling's figures and one rule for
// what equal means.
func TestSummaryMatchesGatling(t *testing.T) {
	t.Parallel()

	for _, version := range reporttest.Versions {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			log := corpusLog(t, version)

			summary, err := Scan(context.Background(), openBytes(t, log), DefaultOptions(), nil)
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			recorded := corpusRecorded(t, filepath.Join(corpusDir, version))
			report(t, reporttest.Compare(observed(summary), recorded, outcomeDurations(t, log)), nil)
		})
	}
}
