package report

import (
	"context"
	"errors"
	"io"
	"math"
	"math/rand/v2"
	"testing"
	"time"

	tdigest "github.com/caio/go-tdigest/v5"
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
