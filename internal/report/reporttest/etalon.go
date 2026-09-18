package reporttest

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/galax-io/parsec/gatling/simlog"
	"github.com/galax-io/parsec/model"
)

// EtalonSamples writes what the etalon, testdata/etalon/Etalon.java, reads from
// the run rd yields: one line for each request whose end the source recorded, in
// log order — "ok" or "failed", a tab, and its response time in whole
// milliseconds. That is the order Gatling feeds its digests in.
func EtalonSamples(w io.Writer, rd simlog.RunReader) error {
	buf := bufio.NewWriter(w)

	var wrote error

	err := Samples(rd, func(outcome model.Outcome, ms int64) {
		name := "failed"
		if outcome == model.OutcomeSuccess {
			name = "ok"
		}

		if _, err := fmt.Fprintf(buf, "%s\t%d\n", name, ms); err != nil && wrote == nil {
			wrote = err
		}
	})
	if err != nil {
		return err
	}

	if wrote != nil {
		return wrote
	}

	return buf.Flush()
}

// Etalon is what the real t-digest libraries give for one run, for all,
// successful and failed requests and each of GatlingRanks: every value Gatling
// 3.11's digest — AVLTreeDigest(100) of t-digest 3.1 — gave over the seeds of
// its generator, ascending, and the value MergingDigest(100) of t-digest 3.3
// gave. An outcome no request reached has neither. Samples and SHA256 say what
// the etalon was run over — how many lines of EtalonSamples it read, and their
// digest — which ties an etalon.tsv to the log beside it.
type Etalon struct {
	AVL     [3][4][]int64
	Merging [3][4]Value

	Samples int
	SHA256  string
}

// Gives reports whether Gatling 3.11's digest gives value as the percentile at
// rank of the column — 0 for all requests, 1 for successful, 2 for failed — and
// false for a rank Gatling does not report.
func (e Etalon) Gives(column int, rank float64, value int64) bool {
	j := slices.Index(GatlingRanks[:], rank)

	return j >= 0 && slices.Contains(e.AVL[column][j], value)
}

// ReadEtalon reads the output of the etalon: a comment, the samples it read and
// their SHA-256, a heading, and one line for each column and rank holding the
// AVL digest's values, comma-separated, and the MergingDigest value, or "-" for
// both where the column is empty. It refuses an etalon that does not say what it
// was run over, and one that gives a column and rank twice.
func ReadEtalon(data []byte) (Etalon, error) {
	columns := map[string]int{"all": 0, "ok": 1, "failed": 2}

	var (
		e    Etalon
		seen [3][4]bool
	)

	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if rest, ok := strings.CutPrefix(line, "# samples "); ok {
			if _, err := fmt.Sscanf(rest, "%d sha256 %64s", &e.Samples, &e.SHA256); err != nil || len(e.SHA256) != 64 {
				return Etalon{}, fmt.Errorf("etalon line %q does not hold a number of samples and their SHA-256", line)
			}

			continue
		}

		if strings.HasPrefix(line, "#") || line == "column\trank\tavl-3.1\tmerging-3.3" {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) != 4 {
			return Etalon{}, fmt.Errorf("etalon line %q does not hold a column, a rank and two digests", line)
		}

		c, ok := columns[fields[0]]
		if !ok {
			return Etalon{}, fmt.Errorf("etalon line %q names no column", line)
		}

		rank, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return Etalon{}, fmt.Errorf("etalon line %q: %w", line, err)
		}

		j := slices.Index(GatlingRanks[:], rank)
		if j < 0 {
			return Etalon{}, fmt.Errorf("etalon line %q names a rank Gatling does not report", line)
		}

		if seen[c][j] {
			return Etalon{}, fmt.Errorf("etalon line %q gives %s at rank %v a second time", line, fields[0], rank)
		}

		seen[c][j] = true

		if fields[2] == "-" && fields[3] == "-" {
			continue
		}

		for _, text := range strings.Split(fields[2], ",") {
			v, err := strconv.ParseInt(text, 10, 64)
			if err != nil {
				return Etalon{}, fmt.Errorf("etalon line %q: %w", line, err)
			}

			e.AVL[c][j] = append(e.AVL[c][j], v)
		}

		merging, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			return Etalon{}, fmt.Errorf("etalon line %q: %w", line, err)
		}

		e.Merging[c][j] = Value{N: merging, Present: true}
	}

	if e.SHA256 == "" {
		return Etalon{}, errors.New("the etalon does not say which samples it was run over")
	}

	for c := range seen {
		for j := range seen[c] {
			if !seen[c][j] {
				return Etalon{}, fmt.Errorf("the etalon has no line for column %d at rank %v", c, GatlingRanks[j])
			}
		}
	}

	return e, nil
}

// OrderOfEqualCentroids ends the note that describes the one difference from
// Gatling 3.11's digest constitution Principle II admits, so that a test of
// recordings known to need no such description can refuse it.
const OrderOfEqualCentroids = "the difference caio's order of equal centroids makes"

// ComparePercentiles holds the percentiles of observed to Gatling 3.11's
// (constitution Principle II). Each must be a value the etalon's AVL digest gives
// for the run, or differ from one across a boundary between two runs of equal
// response times (see acrossEqualRuns), which is described and not held. Where
// recorded is what Gatling 3.11.x or 3.12.x printed for the run — reference —
// each printed percentile must equal this tool's, or be another value the
// digest gives, which its unseeded generator can draw. It returns a sentence for
// each percentile that does not hold, and notes that describe, against the
// run's own response times in durations, what a later Gatling printed and what
// MergingDigest gives where either differs.
func ComparePercentiles(observed Observed, etalon Etalon, recorded Recorded, reference bool, durations [3][]int64) (differences, notes []string) {
	columns := [3]string{"all", "ok", "failed"}

	for i, figures := range observed.Outcomes {
		for j, rank := range GatlingRanks {
			ours, present := figures.Percentile(rank)
			if !present {
				continue
			}

			gives := etalon.AVL[i][j]
			equal := etalon.Gives(i, rank, ours)

			// caio keeps a merged centroid where it is, where t-digest 3.1 moves it
			// after the centroids of an equal mean; on a boundary between two runs
			// of equal response times the two can then round to the two sides of
			// it. That is the one known difference, and it is described, not held.
			reordered := !equal && len(gives) > 0 && acrossEqualRuns(durations[i], ours, nearest(gives, ours), rank)

			switch {
			case equal:
			case reordered:
				notes = append(notes, fmt.Sprintf("%s p%v = %d, where Gatling 3.11's digest gives %v for this log: the rank lies on a boundary between two runs of equal response times with nothing recorded between them, %s", columns[i], rank, ours, gives, OrderOfEqualCentroids))
			default:
				differences = append(differences, fmt.Sprintf("%s p%v = %d, where Gatling 3.11's digest gives %v for this log", columns[i], rank, ours, gives))
			}

			misplaced := func(v int64) string {
				return fmt.Sprintf("misplacing the rank by %.5f where the rule allows %.5f", RankMisplacement(durations[i], v, rank), RankTolerance(rank, len(durations[i])))
			}

			if printed := recorded.Percentiles[j][i]; printed.Present && printed.N != ours {
				switch {
				case reference && etalon.Gives(i, rank, printed.N) && reordered:
					notes = append(notes, fmt.Sprintf("Gatling printed %s p%v = %d and this tool %d: %s", columns[i], rank, printed.N, ours, OrderOfEqualCentroids))
				case reference && etalon.Gives(i, rank, printed.N) && equal:
					notes = append(notes, fmt.Sprintf("Gatling printed %s p%v = %d and this tool %d: Gatling 3.11's digest gives %v for this log, as its generator draws", columns[i], rank, printed.N, ours, gives))
				case reference:
					differences = append(differences, fmt.Sprintf("Gatling printed %s p%v = %d, this tool %d, and Gatling 3.11's digest gives %v for this log", columns[i], rank, printed.N, ours, gives))
				default:
					notes = append(notes, fmt.Sprintf("Gatling printed %s p%v = %d, %s, where Gatling 3.11's digest gives %v and this tool %d: tdunning/t-digest#230", columns[i], rank, printed.N, misplaced(printed.N), gives, ours))
				}
			}

			if merging := etalon.Merging[i][j]; merging.Present && merging.N != ours {
				notes = append(notes, fmt.Sprintf("MergingDigest gives %s p%v = %d, %s, where Gatling 3.11's digest gives %v and this tool %d", columns[i], rank, merging.N, misplaced(merging.N), gives, ours))
			}
		}
	}

	return differences, notes
}

// acrossEqualRuns reports whether ours and theirs, two estimates of the
// percentile at rank, differ the one way Principle II admits: across a boundary
// between two runs of equal response times. The recorded response times around
// the two — the largest at or below the smaller and the smallest at or above the
// greater — must have nothing recorded between them and must each have been
// recorded at least twice; the position rank/100·(n−1) must lie inside those two
// runs; and ours must keep the rank rule. Only there can the order of centroids
// of one mean change what a digest interpolates between, and only that far.
func acrossEqualRuns(sorted []int64, ours, theirs int64, rank float64) bool {
	low, high := min(ours, theirs), max(ours, theirs)

	// below is the last response time at or under low, above the first at or
	// over high; they are neighbours when nothing was recorded between them.
	below := sort.Search(len(sorted), func(k int) bool { return sorted[k] > low }) - 1
	above := sort.Search(len(sorted), func(k int) bool { return sorted[k] >= high })

	if below < 0 || above >= len(sorted) || above != below+1 {
		return false
	}

	first := sort.Search(len(sorted), func(k int) bool { return sorted[k] >= sorted[below] })
	last := sort.Search(len(sorted), func(k int) bool { return sorted[k] > sorted[above] }) - 1

	if below-first < 1 || last-above < 1 {
		return false
	}

	position := rank / 100 * float64(len(sorted)-1)
	if position < float64(first) || position > float64(last) {
		return false
	}

	return RankMisplacement(sorted, ours, rank) <= RankTolerance(rank, len(sorted))
}

// nearest returns the value of values closest to v.
func nearest(values []int64, v int64) int64 {
	best := values[0]

	for _, x := range values[1:] {
		if abs64(x-v) < abs64(best-v) {
			best = x
		}
	}

	return best
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}

	return v
}
