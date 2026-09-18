package reporttest

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/galax-io/parsec/model"
)

// TestEtalonSamples holds what the etalon is fed to the requests a run recorded,
// in log order: every other item, a request with no end and one whose outcome
// was lost are left out.
func TestEtalonSamples(t *testing.T) {
	t.Parallel()

	sample := func(ms int64, outcome model.Outcome) model.Item {
		return model.Item{Kind: model.ItemSample, Sample: model.Sample{
			Name: "r", Start: time.UnixMilli(1000).UTC(), Duration: model.Some(time.Duration(ms) * time.Millisecond), Outcome: outcome,
		}}
	}

	rd := Items(model.Run{}, nil,
		sample(12, model.OutcomeSuccess),
		model.Item{Kind: model.ItemUser, User: model.UserEvent{Kind: model.UserStart, At: time.UnixMilli(1000).UTC()}},
		sample(1502, model.OutcomeFailure),
		model.Item{Kind: model.ItemSample, Sample: model.Sample{Name: "r", Start: time.UnixMilli(1000).UTC(), Outcome: model.OutcomeSuccess}},
		sample(7, model.OutcomeUnknown),
		sample(0, model.OutcomeSuccess),
	)

	var out bytes.Buffer
	if err := EtalonSamples(&out, rd); err != nil {
		t.Fatalf("EtalonSamples: %v", err)
	}

	if expected := "ok\t12\nfailed\t1502\nok\t0\n"; out.String() != expected {
		t.Errorf("EtalonSamples wrote %q, want %q", out.String(), expected)
	}
}

const etalonOutput = `# t-digest 3.1 AVLTreeDigest(100) over seeds 1-200 and t-digest 3.3 MergingDigest(100), each read as Math.round(quantile(rank / 100))
# samples 12250 sha256 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
column	rank	avl-3.1	merging-3.3
all	50	27	27
all	75	38	38
all	95	814	690
all	99	1576	1584
ok	50	27	27
ok	75	38	38
ok	95	831	716
ok	99	1585,1586	1592
failed	50	-	-
failed	75	-	-
failed	95	-	-
failed	99	-	-
`

func TestReadEtalon(t *testing.T) {
	t.Parallel()

	e, err := ReadEtalon([]byte(etalonOutput))
	if err != nil {
		t.Fatalf("ReadEtalon: %v", err)
	}

	if got := e.AVL[1][3]; !slices.Equal(got, []int64{1585, 1586}) {
		t.Errorf("ok p99 = %v, want both values the digest gave", got)
	}

	if got := e.Merging[0][2]; got != (Value{N: 690, Present: true}) {
		t.Errorf("MergingDigest all p95 = %+v, want 690", got)
	}

	if e.Samples != 12250 || e.SHA256 != "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08" {
		t.Errorf("the etalon was run over %d samples of SHA-256 %q, want the 12250 it names", e.Samples, e.SHA256)
	}

	if e.AVL[2][0] != nil || e.Merging[2][0].Present {
		t.Errorf("an empty column was read as %v, %+v", e.AVL[2][0], e.Merging[2][0])
	}

	for _, tt := range []struct {
		column   int
		rank     float64
		value    int64
		expected bool
	}{
		{column: 1, rank: 99, value: 1585, expected: true},
		{column: 1, rank: 99, value: 1586, expected: true},
		{column: 1, rank: 99, value: 1587, expected: false},
		{column: 0, rank: 95, value: 690, expected: false},
		{column: 0, rank: 99.9, value: 1576, expected: false},
		{column: 2, rank: 50, value: 0, expected: false},
	} {
		if got := e.Gives(tt.column, tt.rank, tt.value); got != tt.expected {
			t.Errorf("Gives(%d, %v, %d) = %v, want %v", tt.column, tt.rank, tt.value, got, tt.expected)
		}
	}

	for name, damaged := range map[string]string{
		"a missing line":   strings.Replace(etalonOutput, "all\t75\t38\t38\n", "", 1),
		"a rank of 90":     strings.Replace(etalonOutput, "all\t75", "all\t90", 1),
		"a column unknown": strings.Replace(etalonOutput, "all\t75", "total\t75", 1),
		"a value unread":   strings.Replace(etalonOutput, "814", "81x", 1),
		"a field short":    strings.Replace(etalonOutput, "\t690", "", 1),
		"a line twice":     strings.Replace(etalonOutput, "all\t75\t38\t38\n", "all\t75\t38\t38\nall\t75\t39\t38\n", 1),
		"no samples named": strings.Replace(etalonOutput, "# samples 12250 sha256 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08\n", "", 1),
		"a digest short":   strings.Replace(etalonOutput, "sha256 9f86", "sha256 86", 1),
	} {
		if _, err := ReadEtalon([]byte(damaged)); err == nil {
			t.Errorf("%s: ReadEtalon read a damaged etalon", name)
		}
	}
}

// TestComparePercentiles holds the equality with Gatling 3.11 to what it is for:
// silent where every percentile is one the digest gives and Gatling printed it,
// naming each that is not, and only describing a later Gatling and MergingDigest.
func TestComparePercentiles(t *testing.T) {
	t.Parallel()

	e, err := ReadEtalon([]byte(etalonOutput))
	if err != nil {
		t.Fatalf("ReadEtalon: %v", err)
	}

	sorted := make([]int64, 0, 100)
	for i := int64(1); i <= 100; i++ {
		sorted = append(sorted, i*20)
	}

	durations := [3][]int64{sorted, sorted, nil}

	percentiles := func(values map[float64]int64) Figures {
		return fakeFigures{count: 10, percentile: func(rank float64) int64 { return values[rank] }}
	}

	all := map[float64]int64{50: 27, 75: 38, 95: 814, 99: 1576}
	ok := map[float64]int64{50: 27, 75: 38, 95: 831, 99: 1586}

	observed := func() Observed {
		return Observed{Outcomes: [3]Figures{percentiles(all), percentiles(ok), fakeFigures{}}}
	}

	printed := func() Recorded {
		var r Recorded

		for j, rank := range GatlingRanks {
			r.Percentiles[j] = [3]Value{{N: all[rank], Present: true}, {N: ok[rank], Present: true}, {}}
		}

		return r
	}

	t.Run("gatling 3.11's numbers", func(t *testing.T) {
		t.Parallel()

		differences, notes := ComparePercentiles(observed(), e, printed(), true, durations)
		if len(differences) != 0 {
			t.Errorf("differences %q for a summary with Gatling 3.11's numbers", differences)
		}

		// MergingDigest differs at p95 and p99 of both columns, and says so.
		if joined := strings.Join(notes, "\n"); strings.Count(joined, "MergingDigest gives") != 4 || !strings.Contains(joined, "MergingDigest gives all p95 = 690") {
			t.Errorf("notes %q do not describe the four MergingDigest values that differ", notes)
		}
	})

	t.Run("a percentile the digest does not give", func(t *testing.T) {
		t.Parallel()

		o := observed()
		o.Outcomes[0] = percentiles(map[float64]int64{50: 27, 75: 38, 95: 813, 99: 1576})

		r := printed()
		r.Percentiles[2][0].N = 813

		differences, _ := ComparePercentiles(o, e, r, true, durations)
		if len(differences) != 1 || !strings.Contains(differences[0], "all p95 = 813, where Gatling 3.11's digest gives [814]") {
			t.Errorf("differences %q, want the one percentile the digest does not give", differences)
		}
	})

	t.Run("gatling printed another value", func(t *testing.T) {
		t.Parallel()

		r := printed()
		r.Percentiles[2][0].N = 815

		differences, _ := ComparePercentiles(observed(), e, r, true, durations)
		if len(differences) != 1 || !strings.Contains(differences[0], "Gatling printed all p95 = 815, this tool 814") {
			t.Errorf("differences %q, want the printed value named", differences)
		}
	})

	t.Run("gatling's generator drew the other value", func(t *testing.T) {
		t.Parallel()

		r := printed()
		r.Percentiles[3][1].N = 1585

		differences, notes := ComparePercentiles(observed(), e, r, true, durations)
		if len(differences) != 0 || !strings.Contains(strings.Join(notes, "\n"), "Gatling printed ok p99 = 1585 and this tool 1586") {
			t.Errorf("differences %q, notes %q; want the draw described and nothing failed", differences, notes)
		}
	})

	t.Run("a later gatling is described", func(t *testing.T) {
		t.Parallel()

		r := printed()
		r.Percentiles[2][0].N = 723

		differences, notes := ComparePercentiles(observed(), e, r, false, durations)
		if len(differences) != 0 || !strings.Contains(strings.Join(notes, "\n"), "Gatling printed all p95 = 723") {
			t.Errorf("differences %q, notes %q; want the defect described and nothing failed", differences, notes)
		}
	})
}
