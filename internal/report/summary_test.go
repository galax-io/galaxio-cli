package report

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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

			summary, err := Scan(context.Background(), &stubReader{items: tt.items}, opts)
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

			summary, err := Scan(context.Background(), &stubReader{items: tt.items}, DefaultOptions())
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

// gatlingFigures are the whole-run figures Gatling itself recorded for a run,
// for all, successful and failed requests in that order. No percentile is
// among them: Gatling's are never a target for this tool's.
type gatlingFigures struct {
	count, minimum, maximum, mean, stdDev [3]int64
	rate                                  [3]float64

	// bands counts the successful responses under 800 ms, from 800 ms to under
	// 1200 ms and at 1200 ms or more, then the failed requests; shares are
	// their percentages. A console prints a share to two decimals, a
	// global_stats.json as the double itself, and roundedShares says which.
	bands         [4]int64
	shares        [4]float64
	roundedShares bool
}

// recordedGlobalStats is what a test reads from a global_stats.json Gatling
// wrote, and deliberately nothing more.
type recordedGlobalStats struct {
	NumberOfRequests              recordedTriple[int64]   `json:"numberOfRequests"`
	MinResponseTime               recordedTriple[int64]   `json:"minResponseTime"`
	MaxResponseTime               recordedTriple[int64]   `json:"maxResponseTime"`
	MeanResponseTime              recordedTriple[int64]   `json:"meanResponseTime"`
	StandardDeviation             recordedTriple[int64]   `json:"standardDeviation"`
	MeanNumberOfRequestsPerSecond recordedTriple[float64] `json:"meanNumberOfRequestsPerSecond"`
	Group1                        recordedBand            `json:"group1"`
	Group2                        recordedBand            `json:"group2"`
	Group3                        recordedBand            `json:"group3"`
	Group4                        recordedBand            `json:"group4"`
}

type recordedBand struct {
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

type recordedTriple[T int64 | float64] struct {
	Total T `json:"total"`
	OK    T `json:"ok"`
	KO    T `json:"ko"`
}

func (r recordedTriple[T]) values() [3]T {
	return [3]T{r.Total, r.OK, r.KO}
}

// readGlobalStats reads the figures Gatling wrote beside a corpus run.
func readGlobalStats(t *testing.T, version, file string) gatlingFigures {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(corpusDir, version, file))
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}

	var recorded recordedGlobalStats
	if err := json.Unmarshal(data, &recorded); err != nil {
		t.Fatalf("decode %s: %v", file, err)
	}

	return gatlingFigures{
		count:   recorded.NumberOfRequests.values(),
		minimum: recorded.MinResponseTime.values(),
		maximum: recorded.MaxResponseTime.values(),
		mean:    recorded.MeanResponseTime.values(),
		stdDev:  recorded.StandardDeviation.values(),
		rate:    recorded.MeanNumberOfRequestsPerSecond.values(),
		bands:   [4]int64{recorded.Group1.Count, recorded.Group2.Count, recorded.Group3.Count, recorded.Group4.Count},
		shares:  [4]float64{recorded.Group1.Percentage, recorded.Group2.Percentage, recorded.Group3.Percentage, recorded.Group4.Percentage},
	}
}

// TestSummaryMatchesGatling holds every non-percentile whole-run figure of every
// corpus run to what Gatling itself recorded for that run: its global_stats.json
// where it wrote one, and the Global Information block of its console where it
// did not, or as well.
func TestSummaryMatchesGatling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		expected func(*testing.T) gatlingFigures
	}{
		{
			name: "3.11.5 global_stats.json", version: "3.11.5",
			expected: func(t *testing.T) gatlingFigures { return readGlobalStats(t, "3.11.5", "global_stats.json") },
		},
		{
			name: "3.12.0 global_stats.json", version: "3.12.0",
			expected: func(t *testing.T) gatlingFigures { return readGlobalStats(t, "3.12.0", "global_stats.json") },
		},
		{
			name: "3.13.1 global_stats.json", version: "3.13.1",
			expected: func(t *testing.T) gatlingFigures { return readGlobalStats(t, "3.13.1", "js/global_stats.json") },
		},
		{
			// 3.13.1/console.txt, lines 44–58.
			name: "3.13.1 console", version: "3.13.1",
			expected: func(*testing.T) gatlingFigures {
				return gatlingFigures{
					count: [3]int64{102, 84, 18}, minimum: [3]int64{0, 0, 0}, maximum: [3]int64{1503, 1503, 4},
					mean: [3]int64{89, 108, 1}, stdDev: [3]int64{353, 387, 1}, rate: [3]float64{25.5, 21, 4.5},
					bands: [4]int64{78, 0, 6, 18}, shares: [4]float64{76.47, 0, 5.88, 17.65}, roundedShares: true,
				}
			},
		},
		{
			// 3.14.9/console.txt, lines 41–55.
			name: "3.14.9 console", version: "3.14.9",
			expected: func(*testing.T) gatlingFigures {
				return gatlingFigures{
					count: [3]int64{102, 84, 18}, minimum: [3]int64{0, 0, 0}, maximum: [3]int64{1502, 1502, 12},
					mean: [3]int64{90, 108, 3}, stdDev: [3]int64{353, 387, 4}, rate: [3]float64{25.5, 21, 4.5},
					bands: [4]int64{78, 0, 6, 18}, shares: [4]float64{76.47, 0, 5.88, 17.65}, roundedShares: true,
				}
			},
		},
		{
			// 3.15.1/console.txt, lines 40–54.
			name: "3.15.1 console", version: "3.15.1",
			expected: func(*testing.T) gatlingFigures {
				return gatlingFigures{
					count: [3]int64{102, 84, 18}, minimum: [3]int64{0, 0, 0}, maximum: [3]int64{1502, 1502, 3},
					mean: [3]int64{89, 108, 1}, stdDev: [3]int64{353, 387, 1}, rate: [3]float64{25.5, 21, 4.5},
					bands: [4]int64{78, 0, 6, 18}, shares: [4]float64{76.47, 0, 5.88, 17.65}, roundedShares: true,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expected := tt.expected(t)

			summary, err := Scan(context.Background(), openBytes(t, corpusLog(t, tt.version)), DefaultOptions())
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			columns := [3]string{"all", "ok", "failed"}

			for i, figures := range [3]Figures{summary.All(), summary.OK, summary.Failed} {
				r := readingOf(t, figures)
				if !r.timed {
					t.Fatalf("%s: no timing figure, though Gatling recorded them", columns[i])
				}

				got := [5]int64{int64(r.count), r.minimum, r.maximum, r.mean, r.stdDev}
				want := [5]int64{expected.count[i], expected.minimum[i], expected.maximum[i], expected.mean[i], expected.stdDev[i]}

				if got != want {
					t.Errorf("%s count/min/max/mean/std = %v, Gatling recorded %v", columns[i], got, want)
				}

				rate, ok := summary.Rate(figures.Count())
				if !ok || rate != expected.rate[i] {
					t.Errorf("%s rate = %v (%v), Gatling recorded %v", columns[i], rate, ok, expected.rate[i])
				}
			}

			bandNames := [4]string{"under 800 ms", "800 to 1200 ms", "1200 ms and over", "failed"}

			for i, count := range [4]int{summary.Under, summary.Between, summary.Over, summary.Failed.Count()} {
				if int64(count) != expected.bands[i] {
					t.Errorf("band %s = %d, Gatling recorded %d", bandNames[i], count, expected.bands[i])
				}

				share, ok := summary.Share(count)

				matches := share == expected.shares[i]
				if expected.roundedShares {
					matches = strconv.FormatFloat(share, 'f', 2, 64) == strconv.FormatFloat(expected.shares[i], 'f', 2, 64)
				}

				if !ok || !matches {
					t.Errorf("band %s share = %v (%v), Gatling recorded %v", bandNames[i], share, ok, expected.shares[i])
				}
			}
		})
	}
}
