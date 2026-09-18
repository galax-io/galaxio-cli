package reporttest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var corpusDir = filepath.Join("..", "testdata", "corpus", "gatling")

func present(ns ...int64) [3]Value {
	var v [3]Value

	for i, n := range ns {
		v[i] = Value{N: n, Present: true}
	}

	return v
}

// TestParseConsole reads the Global Information block of both console layouts
// the corpus holds: parenthesised up to 3.13.x, a table from 3.14.0.
func TestParseConsole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version  string
		expected Recorded
	}{
		{
			version: "3.13.1",
			expected: Recorded{
				Count: [3]int64{102, 84, 18},
				Min:   present(0, 0, 0), Max: present(1503, 1503, 4), Mean: present(89, 108, 1), StdDev: present(353, 387, 1),
				Percentiles: [4][3]Value{present(1, 1, 1), present(1, 1, 2), present(1072, 1214, 4), present(1503, 1503, 4)},
				Rate:        [3]float64{25.5, 21, 4.5},
				Bands:       [4]int64{78, 0, 6, 18},
				Shares:      [4]float64{76.47, 0, 5.88, 17.65},
				Rounded:     true,
			},
		},
		{
			version: "3.14.9",
			expected: Recorded{
				Count: [3]int64{102, 84, 18},
				Min:   present(0, 0, 0), Max: present(1502, 1502, 12), Mean: present(90, 108, 3), StdDev: present(353, 387, 4),
				Percentiles: [4][3]Value{present(1, 1, 1), present(1, 1, 2), present(1060, 1213, 12), present(1502, 1502, 12)},
				Rate:        [3]float64{25.5, 21, 4.5},
				Bands:       [4]int64{78, 0, 6, 18},
				Shares:      [4]float64{76.47, 0, 5.88, 17.65},
				Rounded:     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()

			console, err := os.ReadFile(filepath.Join(corpusDir, tt.version, "console.txt"))
			if err != nil {
				t.Fatalf("read console.txt: %v", err)
			}

			got, err := ParseConsole(string(console))
			if err != nil {
				t.Fatalf("ParseConsole: %v", err)
			}

			if got != tt.expected {
				t.Errorf("ParseConsole =\n%+v\nwant\n%+v", got, tt.expected)
			}
		})
	}

	t.Run("an empty outcome and a missing block", func(t *testing.T) {
		t.Parallel()

		console := strings.Join([]string{
			"---- Global Information --------------------------------------------------------",
			"> request count                                       1000 (OK=0      KO=1000  )",
			"> min response time                                      0 (OK=-      KO=0     )",
			"> max response time                                     31 (OK=-      KO=31    )",
			"> mean response time                                     2 (OK=-      KO=2     )",
			"> std deviation                                          2 (OK=-      KO=2     )",
			"> response time 50th percentile                          2 (OK=-      KO=2     )",
			"> response time 75th percentile                          3 (OK=-      KO=3     )",
			"> response time 95th percentile                          4 (OK=-      KO=4     )",
			"> response time 99th percentile                          7 (OK=-      KO=7     )",
			"> mean requests/sec                                     40 (OK=-      KO=40    )",
			"---- Response Time Distribution ------------------------------------------------",
			"> t < 800 ms                                             0 (     0%)",
			"> 800 ms <= t < 1200 ms                                  0 (     0%)",
			"> t >= 1200 ms                                           0 (     0%)",
			"> failed                                              1000 (   100%)",
			"================================================================================",
		}, "\n")

		got, err := ParseConsole(console)
		if err != nil {
			t.Fatalf("ParseConsole: %v", err)
		}

		if got.Mean[1].Present || !got.Mean[2].Present || got.Rate[1] != 0 || got.Rate[2] != 40 || got.Shares[3] != 100 {
			t.Errorf("an outcome no request reached was read as %+v", got)
		}

		if _, err := ParseConsole("no summary here"); err == nil {
			t.Errorf("a console without a Global Information block was read")
		}
	})
}

// TestGlobalInformation holds the cut a live recording makes of a console: the
// last Global Information block, rules included, reading as the whole console
// reads, with none of the build tool's lines around it.
func TestGlobalInformation(t *testing.T) {
	t.Parallel()

	for _, version := range []string{"3.13.1", "3.14.9"} {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			console, err := os.ReadFile(filepath.Join(corpusDir, version, "console.txt"))
			if err != nil {
				t.Fatalf("read console.txt: %v", err)
			}

			block, err := GlobalInformation(string(console))
			if err != nil {
				t.Fatalf("GlobalInformation: %v", err)
			}

			lines := strings.Split(strings.TrimSuffix(block, "\n"), "\n")
			if !strings.HasPrefix(lines[0], "====") || !strings.HasPrefix(lines[1], "---- Global Information") || !strings.HasPrefix(lines[len(lines)-1], "====") {
				t.Errorf("the block does not run from rule to rule:\n%s", block)
			}

			if strings.Contains(block, "[info]") || strings.Contains(block, "Reports generated") {
				t.Errorf("the block holds lines from around it:\n%s", block)
			}

			whole, err := ParseConsole(string(console))
			if err != nil {
				t.Fatalf("ParseConsole of the console: %v", err)
			}

			if cut, err := ParseConsole(block); err != nil || cut != whole {
				t.Errorf("ParseConsole of the block = %+v, %v; want %+v", cut, err, whole)
			}
		})
	}

	t.Run("a console without a closed block", func(t *testing.T) {
		t.Parallel()

		for _, console := range []string{
			"no summary here",
			"---- Global Information ----\n> request count 1 (OK=1 KO=0 )\n",
			"====\n---- Global Information ----\n> request count 1 (OK=1 KO=0 )\n",
		} {
			if block, err := GlobalInformation(console); err == nil {
				t.Errorf("GlobalInformation(%q) = %q, want an error", console, block)
			}
		}
	})
}

func TestReadGlobalStats(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join(corpusDir, "3.12.0", "global_stats.json"))
	if err != nil {
		t.Fatalf("read global_stats.json: %v", err)
	}

	got, err := ReadGlobalStats(data)
	if err != nil {
		t.Fatalf("ReadGlobalStats: %v", err)
	}

	expected := Recorded{
		Count: [3]int64{36, 18, 18},
		Min:   present(0, 0, 0), Max: present(1504, 1504, 3), Mean: present(252, 503, 1), StdDev: present(559, 707, 1),
		Percentiles: [4][3]Value{present(2, 5, 1), present(5, 1502, 2), present(1503, 1504, 2), present(1504, 1504, 3)},
		Rate:        [3]float64{9, 4.5, 4.5},
		Bands:       [4]int64{12, 0, 6, 18},
		Shares:      [4]float64{33.33333333333333, 0, 16.666666666666664, 50},
	}

	if got != expected {
		t.Errorf("ReadGlobalStats =\n%+v\nwant\n%+v", got, expected)
	}
}

// fakeFigures reads fixed figures, as a summary would.
type fakeFigures struct {
	count                  int
	min, max, mean, stdDev int64
	percentile             func(rank float64) int64
}

func (f fakeFigures) Count() int            { return f.count }
func (f fakeFigures) Min() (int64, bool)    { return f.min, f.count > 0 }
func (f fakeFigures) Max() (int64, bool)    { return f.max, f.count > 0 }
func (f fakeFigures) Mean() (int64, bool)   { return f.mean, f.count > 0 }
func (f fakeFigures) StdDev() (int64, bool) { return f.stdDev, f.count > 0 }
func (f fakeFigures) Percentile(rank float64) (int64, bool) {
	if f.count == 0 {
		return 0, false
	}

	return f.percentile(rank), true
}

// TestCompare holds the comparison to what it is for: silent on a summary that
// matches, and naming each figure that does not.
func TestCompare(t *testing.T) {
	t.Parallel()

	// Ten requests of 1 to 10 ms, all successful, over a run of one second.
	durations := [3][]int64{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, {1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, nil}
	exact := func(rank float64) int64 { return AtRank(durations[0], rank) }
	outcome := fakeFigures{count: 10, min: 1, max: 10, mean: 6, stdDev: 3, percentile: exact}

	observed := func() Observed {
		return Observed{
			Outcomes: [3]Figures{outcome, outcome, fakeFigures{percentile: exact}},
			Rate:     func(count int) (float64, bool) { return float64(count), true },
			Share:    func(count int) (float64, bool) { return float64(count) * 10, true },
			Bands:    [4]int{10, 0, 0, 0},
		}
	}

	recorded := func() Recorded {
		return Recorded{
			Count: [3]int64{10, 10, 0},
			Min:   [3]Value{{1, true}, {1, true}, {}}, Max: [3]Value{{10, true}, {10, true}, {}},
			Mean: [3]Value{{6, true}, {6, true}, {}}, StdDev: [3]Value{{3, true}, {3, true}, {}},
			Percentiles: [4][3]Value{{{5, true}, {5, true}, {}}, {{8, true}, {8, true}, {}}, {{10, true}, {10, true}, {}}, {{10, true}, {10, true}, {}}},
			Rate:        [3]float64{10, 10, 0},
			Bands:       [4]int64{10, 0, 0, 0},
			Shares:      [4]float64{100, 0, 0, 0},
		}
	}

	if differences := Compare(observed(), recorded(), durations); len(differences) != 0 {
		t.Fatalf("a matching summary was reported: %v", differences)
	}

	tests := []struct {
		name     string
		change   func(*Observed, *Recorded)
		expected string
	}{
		{name: "a mean", change: func(_ *Observed, r *Recorded) { r.Mean[0].N = 7 }, expected: "all mean = 6"},
		{name: "an absent figure", change: func(_ *Observed, r *Recorded) { r.Max[1].Present = false }, expected: "ok max = 10 (present true)"},
		{name: "a rate", change: func(_ *Observed, r *Recorded) { r.Rate[0] = 10.5 }, expected: "all rate = 10"},
		{name: "a band", change: func(_ *Observed, r *Recorded) { r.Bands[1] = 1 }, expected: "band 2 = 0"},
		{name: "a percentile off its rank", change: func(o *Observed, _ *Recorded) {
			o.Outcomes[0] = fakeFigures{count: 10, min: 1, max: 10, mean: 6, stdDev: 3, percentile: func(float64) int64 { return 1 }}
		}, expected: "all p95 = 1 misplaces the rank"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			o, r := observed(), recorded()
			tt.change(&o, &r)

			differences := Compare(o, r, durations)
			if !strings.Contains(strings.Join(differences, "\n"), tt.expected) {
				t.Errorf("differences %q do not name %q", differences, tt.expected)
			}
		})
	}

	t.Run("gatling's own percentiles are not read", func(t *testing.T) {
		t.Parallel()

		r := recorded()
		r.Percentiles[2][0].N = 2

		if differences := Compare(observed(), r, durations); len(differences) != 0 {
			t.Errorf("differences %q; ComparePercentiles reads Gatling's percentiles, not Compare", differences)
		}
	})
}
