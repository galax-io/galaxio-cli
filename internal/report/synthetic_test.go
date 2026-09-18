package report

import (
	"context"
	"math/rand/v2"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/model"
)

// syntheticRun is a run of a shape no recording has. Every recorded run failed
// under 2 % of its requests, and as fast as it answered the rest, and that
// shape is the one on which two merged digests happen to agree with one fed in
// log order. These runs fail slowly, which is what a timeout looks like.
//
// The requests are drawn in whole numbers only, so that every architecture
// draws the same run, and the etalon.tsv beside each is what the real t-digest
// 3.1 gives for it (TestEtalonRecordings writes and checks it).
type syntheticRun struct {
	name  string
	items func() []model.Item
}

var syntheticRuns = []syntheticRun{
	{name: "every-20th-fails-slowly", items: func() []model.Item {
		return drawRun(1, 12000, func(_ *rand.Rand, i int) bool { return i%20 == 19 }, slowFailure)
	}},
	{name: "5-percent-fail-slowly", items: func() []model.Item {
		return drawRun(2, 12000, func(rng *rand.Rand, _ int) bool { return rng.IntN(20) == 0 }, slowFailure)
	}},
	{name: "30-percent-fail-slowly", items: func() []model.Item {
		return drawRun(3, 12000, func(rng *rand.Rand, _ int) bool { return rng.IntN(10) < 3 }, slowFailure)
	}},
	{name: "5-percent-fail-as-fast", items: func() []model.Item {
		return drawRun(4, 12000, func(rng *rand.Rand, _ int) bool { return rng.IntN(20) == 0 }, response)
	}},
}

// response draws a successful response time: 20 to 60 ms, one in ten up to
// 300 ms more, one in two hundred up to two seconds more.
func response(rng *rand.Rand) int64 {
	ms := int64(20 + rng.IntN(41))

	if rng.IntN(10) == 0 {
		ms += int64(rng.IntN(300))
	}

	if rng.IntN(200) == 0 {
		ms += int64(rng.IntN(2000))
	}

	return ms
}

// slowFailure draws the response time of a request that failed slowly: one to
// five seconds.
func slowFailure(rng *rand.Rand) int64 {
	return int64(1000 + rng.IntN(4001))
}

// drawRun draws n requests, one every 10 ms, from a generator seeded with
// seed: request i fails when fails says so and takes what failure draws, and
// takes what response draws otherwise.
func drawRun(seed uint64, n int, fails func(*rand.Rand, int) bool, failure func(*rand.Rand) int64) []model.Item {
	rng := rand.New(rand.NewPCG(seed, uint64(n)))
	items := make([]model.Item, 0, n)

	for i := range n {
		outcome, ms := model.OutcomeSuccess, int64(0)

		if fails(rng, i) {
			outcome, ms = model.OutcomeFailure, failure(rng)
		} else {
			ms = response(rng)
		}

		items = append(items, model.Item{Kind: model.ItemSample, Sample: model.Sample{
			Name:     "request",
			Start:    time.UnixMilli(int64(1_000_000 + 10*i)).UTC(),
			Duration: model.Some(time.Duration(ms) * time.Millisecond),
			Outcome:  outcome,
		}})
	}

	return items
}

// syntheticDir is where a synthetic run's etalon.tsv is kept.
func syntheticDir(name string) string {
	return filepath.Join("testdata", "synthetic", name)
}

// itemDurations returns the response times of all, successful and failed
// requests among items, sorted.
func itemDurations(items []model.Item) [3][]int64 {
	var durations [3][]int64

	for _, item := range items {
		d, _ := item.Sample.Duration.Get()
		ms := d.Milliseconds()
		durations[0] = append(durations[0], ms)

		if item.Sample.Outcome == model.OutcomeSuccess {
			durations[1] = append(durations[1], ms)
		} else {
			durations[2] = append(durations[2], ms)
		}
	}

	for i := range durations {
		slices.Sort(durations[i])
	}

	return durations
}

// TestSyntheticRunsEqualGatling311 holds every percentile of every synthetic
// run — all, successful and failed requests — to a value the AVLTreeDigest(100)
// of t-digest 3.1 gives for the same requests in the same order, with no
// exception: the percentiles of all requests come from a digest fed in log
// order, as Gatling 3.11 feeds its own. Merging the digests of the two outcomes
// instead gave 358 ms for the 95th percentile of all requests of the second of
// these runs, where Gatling 3.11 gives 366, and failed three of the four.
func TestSyntheticRunsEqualGatling311(t *testing.T) {
	t.Parallel()

	columns := [3]string{"all", "ok", "failed"}

	for _, run := range syntheticRuns {
		t.Run(run.name, func(t *testing.T) {
			t.Parallel()

			summary, err := Scan(context.Background(), reporttest.Items(model.Run{}, nil, run.items()...), DefaultOptions())
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			etalon := readEtalon(t, syntheticDir(run.name))

			for i, figures := range [3]Figures{summary.All(), summary.OK, summary.Failed} {
				for j, rank := range reporttest.GatlingRanks {
					got, present := figures.Percentile(rank)
					if !present {
						t.Fatalf("%s p%v is absent", columns[i], rank)
					}

					if !etalon.Gives(i, rank, got) {
						t.Errorf("%s p%v = %d, where Gatling 3.11's digest gives %v", columns[i], rank, got, etalon.AVL[i][j])
					}
				}
			}
		})
	}
}
