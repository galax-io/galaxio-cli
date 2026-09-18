package report

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	tdigest "github.com/caio/go-tdigest/v5"
	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/model"
)

// percentilesOf reads the default ranks of one outcome, 0 for an absent one.
func percentilesOf(f Figures) [4]int64 {
	var p [4]int64

	for i, rank := range DefaultOptions().Percentiles {
		p[i], _ = f.Percentile(rank)
	}

	return p
}

// TestPercentilesCorpus pins the percentiles this tool prints for every corpus
// run, for all, successful and failed requests, at ranks 50, 75, 95 and 99.
// They are this tool's estimates and are compared with nothing Gatling
// recorded. A change of the digest, its parameters or how it is read must
// change this table, on purpose.
func TestPercentilesCorpus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version         string
		all, ok, failed [4]int64
	}{
		{version: "3.11.5", all: [4]int64{2, 7, 1503, 1504}, ok: [4]int64{7, 1503, 1504, 1504}, failed: [4]int64{2, 2, 4, 4}},
		{version: "3.12.0", all: [4]int64{2, 5, 1503, 1504}, ok: [4]int64{5, 1502, 1504, 1504}, failed: [4]int64{1, 2, 2, 3}},
		// The known divergence. 96 requests of this run took at most 7 ms and
		// six took 1502–1503 ms, so the 97th of the 102, the request at the
		// 95th percentile's rank, took 1502 ms. The library's default quantile
		// interpolates between the 96th request and the 97th and gives 1427.25.
		// caio/go-tdigest#42 proposes a read by rank that returns 1502: taking
		// it changes this line, and must.
		{version: "3.13.1", all: [4]int64{1, 1, 1427, 1502}, ok: [4]int64{1, 1, 1502, 1502}, failed: [4]int64{1, 2, 4, 4}},
		{version: "3.14.9", all: [4]int64{1, 1, 1427, 1502}, ok: [4]int64{1, 1, 1502, 1502}, failed: [4]int64{1, 2, 12, 12}},
		{version: "3.15.1", all: [4]int64{0, 1, 1427, 1502}, ok: [4]int64{0, 1, 1502, 1502}, failed: [4]int64{2, 2, 3, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()

			log := corpusLog(t, tt.version)

			first, err := Scan(context.Background(), openBytes(t, log), DefaultOptions())
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			columns := [3]string{"all", "ok", "failed"}
			expected := [3][4]int64{tt.all, tt.ok, tt.failed}

			for i, figures := range [3]Figures{first.All(), first.OK, first.Failed} {
				if got := percentilesOf(figures); got != expected[i] {
					t.Errorf("%s p50/p75/p95/p99 = %v, want %v", columns[i], got, expected[i])
				}
			}

			second, err := Scan(context.Background(), openBytes(t, log), DefaultOptions())
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			for i, pair := range [3][2]Figures{{first.All(), second.All()}, {first.OK, second.OK}, {first.Failed, second.Failed}} {
				if a, b := percentilesOf(pair[0]), percentilesOf(pair[1]); a != b {
					t.Errorf("%s: two reads of one log gave %v and %v", columns[i], a, b)
				}
			}
		})
	}
}

// TestPercentilesAllRequestsAsOneDigest holds the figures of all requests,
// merged from the two outcomes when they are read, to a digest fed every
// request of the run directly.
func TestPercentilesAllRequestsAsOneDigest(t *testing.T) {
	t.Parallel()

	ranks := []float64{1, 25, 50, 75, 90, 95, 99, 99.9, 100}

	for _, version := range []string{"3.11.5", "3.12.0", "3.13.1", "3.14.9", "3.15.1"} {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			log := corpusLog(t, version)

			summary, err := Scan(context.Background(), openBytes(t, log), DefaultOptions())
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			direct := newDigest()
			rd := openBytes(t, log)

			for {
				item, err := rd.Next()
				if errors.Is(err, io.EOF) {
					break
				}

				if err != nil {
					t.Fatalf("Next: %v", err)
				}

				outcome := item.Sample.Outcome
				if item.Kind != model.ItemSample || (outcome != model.OutcomeSuccess && outcome != model.OutcomeFailure) {
					continue
				}

				if d, ok := item.Sample.Duration.Get(); ok {
					_ = direct.Add(float64(d.Milliseconds()))
				}
			}

			all := summary.All()

			for _, rank := range ranks {
				got, _ := all.Percentile(rank)
				if expected := int64(math.Floor(direct.Quantile(rank/100) + 0.5)); got != expected {
					t.Errorf("p%v of all requests = %d, a digest fed every request gives %d", rank, got, expected)
				}
			}
		})
	}
}

func TestPercentile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		figures  Figures
		rank     float64
		expected int64
		present  bool
	}{
		{name: "one request is every percentile", figures: figuresOf(42), rank: 1, expected: 42, present: true},
		{name: "one request at the top rank", figures: figuresOf(42), rank: 100, expected: 42, present: true},
		{name: "equal requests", figures: figuresOf(7, 7, 7, 7), rank: 99.9, expected: 7, present: true},
		{name: "no request", figures: Figures{}, rank: 50, present: false},
		{name: "a request with no recorded end", figures: func() Figures {
			var f Figures

			f.add(model.Opt[time.Duration]{})

			return f
		}(), rank: 50, present: false},
		{name: "a rank of zero is no percentile", figures: figuresOf(42), rank: 0, present: false},
		{name: "a rank above one hundred is no percentile", figures: figuresOf(42), rank: 101, present: false},
		{name: "NaN is no percentile", figures: figuresOf(42), rank: math.NaN(), present: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, present := tt.figures.Percentile(tt.rank)
			if got != tt.expected || present != tt.present {
				t.Errorf("Percentile(%v) = %d, %v; want %d, %v", tt.rank, got, present, tt.expected, tt.present)
			}
		})
	}
}

// TestPercentilesReadingMidWalkChangesNothing guards the reason the figures of
// all requests are merged into a fresh digest rather than into a clone of the
// successful one: the library's Clone draws from the original's generator, so
// a read in the middle of a walk would change every later insertion. The walk
// is long enough for the digests to merge centroids, which is when the
// generator is consulted at all.
func TestPercentilesReadingMidWalkChangesNothing(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(1, 2))

	var read, unread Summary

	read.Options, unread.Options = DefaultOptions(), DefaultOptions()

	for i := range 20000 {
		ms := int64(1 + rng.IntN(2000))
		if rng.IntN(200) == 0 {
			ms = 60000
		}

		outcome := model.OutcomeSuccess
		if rng.IntN(10) == 0 {
			outcome = model.OutcomeFailure
		}

		item := model.Item{Kind: model.ItemSample, Sample: model.Sample{
			Name: "r", Start: time.UnixMilli(1000).UTC(), Duration: model.Some(time.Duration(ms) * time.Millisecond), Outcome: outcome,
		}}

		read.add(&item)
		unread.add(&item)

		if i%100 == 0 {
			_, _ = read.All().Percentile(95)
		}
	}

	for _, pair := range [3][2]*tdigest.TDigest{
		{read.OK.digest, unread.OK.digest},
		{read.Failed.digest, unread.Failed.digest},
		{read.All().digest, unread.All().digest},
	} {
		for q := 0.0; q <= 1; q += 0.001 {
			if a, b := pair[0].Quantile(q), pair[1].Quantile(q); a != b {
				t.Fatalf("quantile %.3f is %v after reads in the middle of the walk and %v without them", q, a, b)
			}
		}
	}
}

// TestPercentilesRule holds every percentile this tool prints to the rule
// research.md §2 states for how one may differ from the response times the run
// recorded, so that the difference is proved rather than described:
//
//   - While the figures hold at most 200 requests, a percentile is the
//     interpolation between the two recorded response times around its
//     position, rounded half up. It lies between them, and differs from the
//     request at the percentile's rank by at most their gap: 1427 against 1502
//     for the 95th percentile of the 3.13.1 run, whose neighbours are 7 and 1502.
//   - At any size, it misplaces its rank among the recorded requests by at most
//     4·q·(1−q)/100 of them, plus one request.
func TestPercentilesRule(t *testing.T) {
	t.Parallel()

	ranks := []float64{1, 5, 25, 50, 75, 90, 95, 99, 99.9, 100}

	outcomes := []struct {
		name    string
		keep    func(model.Outcome) bool
		figures func(Summary) Figures
	}{
		{name: "all", keep: func(o model.Outcome) bool { return o == model.OutcomeSuccess || o == model.OutcomeFailure }, figures: Summary.All},
		{name: "ok", keep: func(o model.Outcome) bool { return o == model.OutcomeSuccess }, figures: func(s Summary) Figures { return s.OK }},
		{name: "failed", keep: func(o model.Outcome) bool { return o == model.OutcomeFailure }, figures: func(s Summary) Figures { return s.Failed }},
	}

	t.Run("the corpus runs, below 200 requests", func(t *testing.T) {
		t.Parallel()

		for _, version := range []string{"3.11.5", "3.12.0", "3.13.1", "3.14.9", "3.15.1"} {
			log := corpusLog(t, version)

			summary, err := Scan(context.Background(), openBytes(t, log), DefaultOptions())
			if err != nil {
				t.Fatalf("%s: Scan: %v", version, err)
			}

			for _, outcome := range outcomes {
				sorted, err := reporttest.Durations(openBytes(t, log), outcome.keep)
				if err != nil {
					t.Fatalf("%s: Durations: %v", version, err)
				}

				if len(sorted) > 200 {
					t.Fatalf("%s %s holds %d requests, past the interpolation rule", version, outcome.name, len(sorted))
				}

				figures := outcome.figures(summary)

				for _, rank := range ranks {
					got, _ := figures.Percentile(rank)
					interpolated, lower, upper := reporttest.Interpolated(sorted, rank)
					atRank := reporttest.AtRank(sorted, rank)
					where := fmt.Sprintf("%s %s p%v = %d (neighbours %d and %d, request at the rank %d)", version, outcome.name, rank, got, lower, upper, atRank)

					if want := int64(math.Floor(interpolated + 0.5)); got != want {
						t.Errorf("%s: want the interpolation %v rounded half up, %d", where, interpolated, want)
					}

					if got < lower || got > upper || abs(got-atRank) > upper-lower {
						t.Errorf("%s: outside its neighbours, or further from the request at the rank than their gap", where)
					}

					if misplaced, tolerance := reporttest.RankMisplacement(sorted, got, rank), reporttest.RankTolerance(rank, len(sorted)); misplaced > tolerance {
						t.Errorf("%s: misplaces the rank by %.5f, over %.5f", where, misplaced, tolerance)
					}
				}
			}
		}
	})

	t.Run("synthetic runs, at any size", func(t *testing.T) {
		t.Parallel()

		distributions := []struct {
			name string
			ms   func(*rand.Rand) int64
		}{
			{name: "log-normal around 40 ms with 0.5 % at 60 s", ms: func(r *rand.Rand) int64 {
				if r.IntN(200) == 0 {
					return 60000
				}

				return int64(math.Round(math.Exp(math.Log(40) + 0.6*r.NormFloat64())))
			}},
			{name: "95 % at most 10 ms and 5 % near 1500 ms", ms: func(r *rand.Rand) int64 {
				if r.IntN(20) == 0 {
					return 1500 + int64(r.IntN(5))
				}

				return int64(r.IntN(11))
			}},
			{name: "six distinct values", ms: func(r *rand.Rand) int64 { return int64(r.IntN(6)) }},
			{name: "uniform up to 5 s", ms: func(r *rand.Rand) int64 { return int64(r.IntN(5001)) }},
			{name: "two modes, 50 ms and 800 ms", ms: func(r *rand.Rand) int64 {
				if r.IntN(2) == 0 {
					return max(0, int64(math.Round(50+5*r.NormFloat64())))
				}

				return max(0, int64(math.Round(800+50*r.NormFloat64())))
			}},
		}

		for d, distribution := range distributions {
			for _, size := range []int{201, 12000, 100000} {
				rng := rand.New(rand.NewPCG(uint64(d+1), uint64(size)))

				var s Summary

				s.Options = DefaultOptions()

				sorted := map[string][]int64{}

				for range size {
					ms := distribution.ms(rng)

					outcome := model.OutcomeSuccess
					if rng.IntN(20) == 0 {
						outcome = model.OutcomeFailure
					}

					item := model.Item{Kind: model.ItemSample, Sample: model.Sample{
						Name: "r", Start: time.UnixMilli(1000).UTC(), Duration: model.Some(time.Duration(ms) * time.Millisecond), Outcome: outcome,
					}}
					s.add(&item)

					for _, outcomeRow := range outcomes {
						if outcomeRow.keep(outcome) {
							sorted[outcomeRow.name] = append(sorted[outcomeRow.name], ms)
						}
					}
				}

				for _, outcome := range outcomes {
					values := sorted[outcome.name]
					if len(values) == 0 {
						continue
					}

					slices.Sort(values)
					figures := outcome.figures(s)

					for _, rank := range ranks {
						got, _ := figures.Percentile(rank)

						if misplaced, tolerance := reporttest.RankMisplacement(values, got, rank), reporttest.RankTolerance(rank, len(values)); misplaced > tolerance {
							t.Errorf("%s, %d requests, %s p%v = %d: misplaces the rank by %.5f, over %.5f", distribution.name, size, outcome.name, rank, got, misplaced, tolerance)
						}
					}
				}
			}
		}
	})
}

// TestGatlingPercentilesAsReference holds the percentiles Gatling itself
// recorded to the same rank rule, for the versions whose digest has no known
// defect: 3.11.5 and 3.12.0, which use com.tdunning:t-digest 3.1. They are a
// reference, not a target: each is held to the response times the run recorded
// and never to this tool's percentile, and nothing asserts the two equal.
//
// From 3.13.0 Gatling uses t-digest 3.3, whose AVLTreeDigest miscounts
// (tdunning/t-digest#230). For the recorded 3.13.1 run it printed 1072 ms as
// the 95th percentile of all requests, where the request at that rank took
// 1502 ms, and rebuilding its digest from the same requests gives 916 or 1061
// as well; 3.14.9 and 3.15.1 printed 1060 and 1090. Those numbers describe the
// defect, not the run, so those versions are left out on purpose.
func TestGatlingPercentilesAsReference(t *testing.T) {
	t.Parallel()

	keep := [3]func(model.Outcome) bool{
		func(o model.Outcome) bool { return o == model.OutcomeSuccess || o == model.OutcomeFailure },
		func(o model.Outcome) bool { return o == model.OutcomeSuccess },
		func(o model.Outcome) bool { return o == model.OutcomeFailure },
	}
	columns := [3]string{"total", "ok", "ko"}

	for _, version := range []string{"3.11.5", "3.12.0"} {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(filepath.Join(corpusDir, version, "global_stats.json"))
			if err != nil {
				t.Fatalf("read global_stats.json: %v", err)
			}

			recorded, err := reporttest.ReadGlobalStats(data)
			if err != nil {
				t.Fatalf("decode global_stats.json: %v", err)
			}

			log := corpusLog(t, version)

			for i := range columns {
				sorted, err := reporttest.Durations(openBytes(t, log), keep[i])
				if err != nil {
					t.Fatalf("Durations: %v", err)
				}

				for j, rank := range reporttest.GatlingRanks {
					value := recorded.Percentiles[j][i].N

					if misplaced, tolerance := reporttest.RankMisplacement(sorted, value, rank), reporttest.RankTolerance(rank, len(sorted)); misplaced > tolerance {
						t.Errorf("Gatling %s %s p%v = %d misplaces the rank by %.5f, over %.5f", version, columns[i], rank, value, misplaced, tolerance)
					}
				}
			}
		})
	}
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}

	return v
}
