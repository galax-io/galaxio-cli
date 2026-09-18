package report

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
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
// They are held to Gatling 3.11's by TestPercentilesEqualGatling311; this table
// is what they are, so that a change of the digest, its parameters or how it is
// read moves a number here on purpose rather than in silence.
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
		// caio/go-tdigest#42 proposes a read by rank that returns 1502, which
		// would part from Gatling 3.11 and so from Principle II: taking it is
		// its own change, and moves this line with the rule.
		{version: "3.13.1", all: [4]int64{1, 1, 1427, 1502}, ok: [4]int64{1, 1, 1502, 1502}, failed: [4]int64{1, 2, 4, 4}},
		{version: "3.14.9", all: [4]int64{1, 1, 1427, 1502}, ok: [4]int64{1, 1, 1502, 1502}, failed: [4]int64{1, 2, 12, 12}},
		{version: "3.15.1", all: [4]int64{0, 1, 1427, 1502}, ok: [4]int64{0, 1, 1502, 1502}, failed: [4]int64{2, 2, 3, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()

			log := corpusLog(t, tt.version)

			first, err := Scan(context.Background(), openBytes(t, log), DefaultOptions(), nil)
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

			second, err := Scan(context.Background(), openBytes(t, log), DefaultOptions(), nil)
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

// TestPercentilesAllRequestsAsOneDigest holds the percentiles of all requests
// of every corpus run to a digest fed every request of the run directly, in log
// order. TestSyntheticRunsEqualGatling311 holds them to t-digest 3.1 itself on
// runs long enough for a digest to compress, where merging the two outcomes'
// digests instead gave other numbers.
func TestPercentilesAllRequestsAsOneDigest(t *testing.T) {
	t.Parallel()

	ranks := []float64{1, 25, 50, 75, 90, 95, 99, 99.9, 100}

	for _, version := range reporttest.Versions {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			log := corpusLog(t, version)

			summary, err := Scan(context.Background(), openBytes(t, log), DefaultOptions(), nil)
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			// Fed through the one reading of what a run holds, so that this
			// digest and the walk's own cannot be given different requests.
			direct := newDigest()

			if err := reporttest.Samples(openBytes(t, log), func(_ model.Outcome, ms int64) {
				_ = direct.Add(float64(ms))
			}); err != nil {
				t.Fatalf("Samples: %v", err)
			}

			all := summary.All()

			for _, rank := range ranks {
				got, _ := all.Percentile(rank)
				if expected := roundHalfUp(quantile(direct, rank/100)); got != expected {
					t.Errorf("p%v of all requests = %d, a digest fed every request gives %d", rank, got, expected)
				}
			}
		})
	}
}

// TestRoundHalfUp holds the rounding to Math.round's, which Gatling applies to
// its digest's estimate: to the nearest, up from exactly a half, and down from
// one bit below it.
func TestRoundHalfUp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value    float64
		expected int64
	}{
		{value: 0, expected: 0},
		{value: 0.49999999999999994, expected: 0},
		{value: 0.5, expected: 1},
		{value: 1427.25, expected: 1427},
		{value: 2.8299999999999983, expected: 3},
		{value: 398.5, expected: 399},
		{value: 398.5000000000001, expected: 399},
		// One bit below a half is below it. What t-digest 3.1 gives for the 95th
		// percentile of [1, 51] is this shape, 48.49999999999999, and Gatling
		// prints 48.
		{value: 398.49999999999994, expected: 398},
		{value: 48.49999999999999, expected: 48},
		{value: 398.4999, expected: 398},
		{value: 1585.5613333333329, expected: 1586},
		{value: math.Nextafter(1e7+0.5, 0), expected: 1e7},
		{value: 1e7 + 0.5, expected: 1e7 + 1},
		{value: 1e7 + 0.4999, expected: 1e7},
	}

	for _, tt := range tests {
		if got := roundHalfUp(tt.value); got != tt.expected {
			t.Errorf("roundHalfUp(%v) = %d, want %d", tt.value, got, tt.expected)
		}
	}
}

// TestPercentileEqualsTDigest31 holds the percentiles of small sets to what
// AVLTreeDigest(100) of t-digest 3.1 gives for them, read as Math.round(quantile),
// which the etalon was run over (testdata/etalon/Etalon.java, every seed alike).
// A small set is where a percentile lies at a half, or one bit from it, and so
// where the order of the floating-point operations decides the millisecond: a
// read that fuses a product into the addition after it, as arm64 compiles the
// library's own, gives 52 for [43, 53], and a rounding with slack below the half
// gave 49 for [1, 51].
func TestPercentileEqualsTDigest31(t *testing.T) {
	t.Parallel()

	tests := []struct {
		values   []int64
		expected [4]int64
	}{
		{values: []int64{1, 51}, expected: [4]int64{26, 39, 48, 51}},
		{values: []int64{2, 12}, expected: [4]int64{7, 10, 11, 12}},
		{values: []int64{43, 53}, expected: [4]int64{48, 51, 53, 53}},
		{values: []int64{14, 24}, expected: [4]int64{19, 22, 23, 24}},
		{values: []int64{1, 2}, expected: [4]int64{2, 2, 2, 2}},
		{values: []int64{1, 2, 7}, expected: [4]int64{2, 5, 6, 7}},
		{values: []int64{3, 8, 20, 21}, expected: [4]int64{14, 20, 21, 21}},
		{values: []int64{5, 5, 5, 6, 6, 6}, expected: [4]int64{6, 6, 6, 6}},
	}

	for _, tt := range tests {
		if got := percentilesOf(figuresOf(tt.values...)); got != tt.expected {
			t.Errorf("p50/p75/p95/p99 of %v = %v, t-digest 3.1 gives %v", tt.values, got, tt.expected)
		}
	}
}

// TestQuantileReadsAsTheLibrary holds this package's read of a digest to the
// library's own at every thousandth, the two ends included, on digests that
// have compressed. They are one arithmetic, so they differ by the last bits an
// architecture that fuses operations moves, and by nothing a millisecond sees.
func TestQuantileReadsAsTheLibrary(t *testing.T) {
	t.Parallel()

	for seed := range uint64(8) {
		rng := rand.New(rand.NewPCG(seed, 99))
		d := newDigest()

		for range 5000 + int(seed)*3000 {
			ms := 1 + rng.IntN(3000)
			if rng.IntN(100) == 0 {
				ms = 60000
			}

			_ = d.Add(float64(ms))
		}

		for i := range 1001 {
			q := float64(i) / 1000

			got, expected := quantile(d, q), d.Quantile(q)
			if math.Abs(got-expected) > 1e-9*math.Max(1, math.Abs(expected)) {
				t.Fatalf("seed %d: quantile(%v) = %v, the library reads %v", seed, q, got, expected)
			}
		}
	}

	if got := quantile(newDigest(), 0.5); !math.IsNaN(got) {
		t.Errorf("quantile of an empty digest = %v, want NaN", got)
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

// TestPercentilesReadingMidWalkChangesNothing guards what the progress block
// relies on: reading the figures in the middle of a walk changes nothing that
// follows. A read that cloned a digest would — the library's Clone draws from
// the original's generator, and the generator decides which centroid a later
// insertion merges into. The walk is long enough for the digests to merge
// centroids, which is when the generator is consulted at all.
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
		figures func(Summary) Figures
	}{
		{name: "all", figures: Summary.All},
		{name: "ok", figures: func(s Summary) Figures { return s.OK }},
		{name: "failed", figures: func(s Summary) Figures { return s.Failed }},
	}

	t.Run("the corpus runs, below 200 requests", func(t *testing.T) {
		t.Parallel()

		for _, version := range reporttest.Versions {
			log := corpusLog(t, version)

			summary, err := Scan(context.Background(), openBytes(t, log), DefaultOptions(), nil)
			if err != nil {
				t.Fatalf("%s: Scan: %v", version, err)
			}

			durations := outcomeDurations(t, log)

			for i, outcome := range outcomes {
				sorted := durations[i]

				if len(sorted) > 200 {
					t.Fatalf("%s %s holds %d requests, past the interpolation rule", version, outcome.name, len(sorted))
				}

				figures := outcome.figures(summary)

				for _, rank := range ranks {
					got, _ := figures.Percentile(rank)
					interpolated, lower, upper := reporttest.Interpolated(sorted, rank)
					atRank := reporttest.AtRank(sorted, rank)
					where := fmt.Sprintf("%s %s p%v = %d (neighbours %d and %d, request at the rank %d)", version, outcome.name, rank, got, lower, upper, atRank)

					if want := roundHalfUp(interpolated); got != want {
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

				var sorted [3][]int64

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

					sorted[0] = append(sorted[0], ms)

					if outcome == model.OutcomeSuccess {
						sorted[1] = append(sorted[1], ms)
					} else {
						sorted[2] = append(sorted[2], ms)
					}
				}

				for i, outcome := range outcomes {
					values := sorted[i]
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

// TestPercentilesEqualGatling311 holds every percentile of every corpus run equal
// to Gatling 3.11's (constitution Principle II): a value the AVLTreeDigest(100)
// of t-digest 3.1 gives for the log, per the etalon.tsv beside it, and for
// 3.11.5 and 3.12.0 the value they wrote in global_stats.json. What 3.13.1,
// 3.14.9 and 3.15.1 printed is described, not held: for the 3.13.1 run Gatling
// printed 1072 ms as the 95th percentile of all requests, where Gatling 3.11's
// digest and this tool give 1427, because t-digest 3.3's AVLTreeDigest
// miscounts (tdunning/t-digest#230).
func TestPercentilesEqualGatling311(t *testing.T) {
	t.Parallel()

	for _, version := range reporttest.Versions {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			log := corpusLog(t, version)

			summary, err := Scan(context.Background(), openBytes(t, log), DefaultOptions(), nil)
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			dir := filepath.Join(corpusDir, version)
			reference := reporttest.IsReference(version)

			differences, notes := reporttest.ComparePercentiles(observed(summary), readEtalon(t, dir), corpusRecorded(t, dir), reference, outcomeDurations(t, log))
			report(t, differences, notes)
		})
	}
}

// corpusRecorded reads what Gatling recorded for a corpus run: the
// global_stats.json it wrote, in the run directory or its js directory, or else
// the Global Information block of its console.
func corpusRecorded(t *testing.T, dir string) reporttest.Recorded {
	t.Helper()

	for _, name := range []string{"global_stats.json", filepath.Join("js", "global_stats.json")} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}

		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		recorded, err := reporttest.ReadGlobalStats(data)
		if err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}

		return recorded
	}

	console, err := os.ReadFile(filepath.Join(dir, "console.txt"))
	if err != nil {
		t.Fatalf("read console.txt: %v", err)
	}

	recorded, err := reporttest.ParseConsole(string(console))
	if err != nil {
		t.Fatalf("parse console.txt: %v", err)
	}

	return recorded
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}

	return v
}
