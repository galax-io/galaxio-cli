package reporttest

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// GatlingRanks are the percentile ranks Gatling reports by default, which no
// recorded run changed.
var GatlingRanks = [4]float64{50, 75, 95, 99}

// Versions are the Gatling releases the corpus holds a run of, oldest first.
var Versions = []string{"3.11.5", "3.12.0", "3.13.1", "3.14.9", "3.15.1"}

// IsReference reports whether a recording of this Gatling version is one whose
// printed percentiles this tool's must equal: 3.11.x and 3.12.x compute them
// over t-digest 3.1, and later versions over the miscount of
// tdunning/t-digest#230 (research §18).
func IsReference(version string) bool {
	return strings.HasPrefix(version, "3.11.") || strings.HasPrefix(version, "3.12.")
}

// Recorded is what Gatling itself printed or wrote about a run as a whole: each
// figure for all, successful and failed requests, in that order, and the four
// response-time bands. A figure Gatling left empty is marked absent.
type Recorded struct {
	Count                  [3]int64
	Min, Max, Mean, StdDev [3]Value
	Percentiles            [4][3]Value
	Rate                   [3]float64
	Bands                  [4]int64
	Shares                 [4]float64
	// Rounded says the rates and shares were printed to two decimals, as a
	// console prints them, rather than written as the doubles themselves.
	Rounded bool
}

// Value is one recorded figure, or its absence.
type Value struct {
	N       int64
	Present bool
}

var (
	oldGlobalLine = regexp.MustCompile(`^> (.+?)\s+(\S+) \(OK=(\S+)\s+KO=(\S+)\s*\)$`)
	newGlobalLine = regexp.MustCompile(`^> (.+?)\s*\|\s*(\S+)\s*\|\s*(\S+)\s*\|\s*(\S+)\s*$`)
	bandLine      = regexp.MustCompile(`^> .+?\s+([\d,]+)\s+\(\s*([\d.]+)%\)$`)
)

// GlobalInformation returns the last Global Information block Gatling printed on
// its console, byte for byte: from the rule of = signs that opens it to the one
// that closes it, with the response-time distribution and the errors between.
// A live recording keeps that much of the console and nothing around it.
func GlobalInformation(console string) (string, error) {
	lines := strings.SplitAfter(console, "\n")

	heading := -1

	for i, line := range lines {
		if strings.HasPrefix(line, "---- Global Information") {
			heading = i
		}
	}

	if heading < 1 || !strings.HasPrefix(lines[heading-1], "====") {
		return "", errors.New("no Global Information block opened by a rule")
	}

	for i := heading + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "====") {
			return strings.Join(lines[heading-1:i+1], ""), nil
		}
	}

	return "", errors.New("the Global Information block is never closed")
}

// ParseConsole reads the last Global Information block and the response-time
// distribution after it from what Gatling printed on its console. It reads both
// layouts: the parenthesised one up to 3.13.x, and the table with | separators
// from 3.14.0.
func ParseConsole(console string) (Recorded, error) {
	lines := strings.Split(strings.ReplaceAll(console, "\r\n", "\n"), "\n")

	start := -1

	for i, line := range lines {
		if strings.HasPrefix(line, "---- Global Information") {
			start = i
		}
	}

	if start < 0 {
		return Recorded{}, errors.New("no Global Information block")
	}

	r := Recorded{Rounded: true}
	seen := map[string]bool{}
	bands := 0

	for _, line := range lines[start+1:] {
		switch {
		case strings.HasPrefix(line, "---- Response Time Distribution"):
			continue
		case strings.HasPrefix(line, "----") || strings.HasPrefix(line, "===="):
			if bands > 0 {
				return r, checkSeen(seen, bands)
			}

			continue
		case !strings.HasPrefix(line, "> "):
			continue
		}

		fields := oldGlobalLine.FindStringSubmatch(line)
		if fields == nil {
			fields = newGlobalLine.FindStringSubmatch(line)
		}

		if fields != nil {
			key, err := globalKey(fields[1])
			if err != nil {
				return Recorded{}, err
			}

			if err := r.set(key, fields[2:]); err != nil {
				return Recorded{}, fmt.Errorf("%s: %w", key, err)
			}

			seen[key] = true

			continue
		}

		if band := bandLine.FindStringSubmatch(line); band != nil && bands < 4 {
			count, err := strconv.ParseInt(strings.ReplaceAll(band[1], ",", ""), 10, 64)
			if err != nil {
				return Recorded{}, fmt.Errorf("band %q: %w", line, err)
			}

			share, err := strconv.ParseFloat(band[2], 64)
			if err != nil {
				return Recorded{}, fmt.Errorf("band %q: %w", line, err)
			}

			r.Bands[bands], r.Shares[bands] = count, share
			bands++
		}
	}

	return r, checkSeen(seen, bands)
}

func checkSeen(seen map[string]bool, bands int) error {
	for _, key := range []string{"count", "min", "max", "mean", "std", "p50", "p75", "p95", "p99", "rate"} {
		if !seen[key] {
			return fmt.Errorf("the Global Information block has no %s line", key)
		}
	}

	if bands != 4 {
		return fmt.Errorf("found %d response-time bands, want 4", bands)
	}

	return nil
}

// globalKey names a Global Information line whichever wording the version used.
func globalKey(label string) (string, error) {
	label = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(label), "(ms)"), "(rps)"))

	switch label {
	case "request count":
		return "count", nil
	case "min response time":
		return "min", nil
	case "max response time":
		return "max", nil
	case "mean response time":
		return "mean", nil
	case "std deviation", "response time std deviation":
		return "std", nil
	case "response time 50th percentile":
		return "p50", nil
	case "response time 75th percentile":
		return "p75", nil
	case "response time 95th percentile":
		return "p95", nil
	case "response time 99th percentile":
		return "p99", nil
	case "mean requests/sec", "mean throughput":
		return "rate", nil
	default:
		return "", fmt.Errorf("unknown Global Information line %q", label)
	}
}

func (r *Recorded) set(key string, texts []string) error {
	if key == "rate" {
		for i, text := range texts {
			// A console prints the rate of an outcome no request reached as "-",
			// which is a rate of 0 over the run's span.
			if text == "-" {
				continue
			}

			v, err := strconv.ParseFloat(strings.ReplaceAll(text, ",", ""), 64)
			if err != nil {
				return err
			}

			r.Rate[i] = v
		}

		return nil
	}

	values := [3]Value{}

	for i, text := range texts {
		if text == "-" {
			continue
		}

		n, err := strconv.ParseInt(strings.ReplaceAll(text, ",", ""), 10, 64)
		if err != nil {
			return err
		}

		values[i] = Value{N: n, Present: true}
	}

	switch key {
	case "count":
		for i, v := range values {
			r.Count[i] = v.N
		}
	case "min":
		r.Min = values
	case "max":
		r.Max = values
	case "mean":
		r.Mean = values
	case "std":
		r.StdDev = values
	case "p50":
		r.Percentiles[0] = values
	case "p75":
		r.Percentiles[1] = values
	case "p95":
		r.Percentiles[2] = values
	case "p99":
		r.Percentiles[3] = values
	}

	return nil
}

// ReadGlobalStats reads the global_stats.json Gatling wrote up to 3.13.x. The
// file writes 0 for every figure of an outcome no request reached, so such an
// outcome's timing figures are marked absent.
func ReadGlobalStats(data []byte) (Recorded, error) {
	type triple struct {
		Total float64 `json:"total"`
		OK    float64 `json:"ok"`
		KO    float64 `json:"ko"`
	}

	type band struct {
		Count      int64   `json:"count"`
		Percentage float64 `json:"percentage"`
	}

	var file struct {
		NumberOfRequests triple `json:"numberOfRequests"`
		MinResponseTime  triple `json:"minResponseTime"`
		MaxResponseTime  triple `json:"maxResponseTime"`
		MeanResponseTime triple `json:"meanResponseTime"`
		StdDev           triple `json:"standardDeviation"`
		Percentiles1     triple `json:"percentiles1"`
		Percentiles2     triple `json:"percentiles2"`
		Percentiles3     triple `json:"percentiles3"`
		Percentiles4     triple `json:"percentiles4"`
		Group1           band   `json:"group1"`
		Group2           band   `json:"group2"`
		Group3           band   `json:"group3"`
		Group4           band   `json:"group4"`
		Rate             triple `json:"meanNumberOfRequestsPerSecond"`
	}

	if err := json.Unmarshal(data, &file); err != nil {
		return Recorded{}, err
	}

	count := [3]int64{int64(file.NumberOfRequests.Total), int64(file.NumberOfRequests.OK), int64(file.NumberOfRequests.KO)}

	values := func(t triple) [3]Value {
		var v [3]Value

		for i, n := range [3]float64{t.Total, t.OK, t.KO} {
			v[i] = Value{N: int64(n), Present: count[i] > 0}
		}

		return v
	}

	return Recorded{
		Count:       count,
		Min:         values(file.MinResponseTime),
		Max:         values(file.MaxResponseTime),
		Mean:        values(file.MeanResponseTime),
		StdDev:      values(file.StdDev),
		Percentiles: [4][3]Value{values(file.Percentiles1), values(file.Percentiles2), values(file.Percentiles3), values(file.Percentiles4)},
		Rate:        [3]float64{file.Rate.Total, file.Rate.OK, file.Rate.KO},
		Bands:       [4]int64{file.Group1.Count, file.Group2.Count, file.Group3.Count, file.Group4.Count},
		Shares:      [4]float64{file.Group1.Percentage, file.Group2.Percentage, file.Group3.Percentage, file.Group4.Percentage},
	}, nil
}

// Figures is what a comparison reads of one outcome's figures.
type Figures interface {
	Count() int
	Min() (int64, bool)
	Max() (int64, bool)
	Mean() (int64, bool)
	StdDev() (int64, bool)
	Percentile(rank float64) (int64, bool)
}

// Observed is what a summary reported about a run, in the shape a comparison
// with Gatling reads it: all, successful and failed requests in that order.
type Observed struct {
	Outcomes [3]Figures
	Rate     func(count int) (float64, bool)
	Share    func(count int) (float64, bool)
	Bands    [4]int
}

// Compare returns, one sentence each, every whole-run figure of observed that
// differs from what Gatling recorded and every percentile that breaks the rank
// rule over the run's own response times, durations holding them for all,
// successful and failed requests. Gatling's own percentiles are not read here:
// ComparePercentiles holds this tool's to them.
func Compare(observed Observed, recorded Recorded, durations [3][]int64) (differences []string) {
	differ := func(format string, args ...any) { differences = append(differences, fmt.Sprintf(format, args...)) }
	columns := [3]string{"all", "ok", "failed"}

	for i, figures := range observed.Outcomes {
		if got := int64(figures.Count()); got != recorded.Count[i] {
			differ("%s count = %d, Gatling recorded %d", columns[i], got, recorded.Count[i])
		}

		for _, figure := range []struct {
			name     string
			read     func() (int64, bool)
			recorded Value
		}{
			{"min", figures.Min, recorded.Min[i]},
			{"max", figures.Max, recorded.Max[i]},
			{"mean", figures.Mean, recorded.Mean[i]},
			{"std", figures.StdDev, recorded.StdDev[i]},
		} {
			got, present := figure.read()
			if present != figure.recorded.Present || (present && got != figure.recorded.N) {
				differ("%s %s = %d (present %v), Gatling recorded %d (present %v)", columns[i], figure.name, got, present, figure.recorded.N, figure.recorded.Present)
			}
		}

		rate, ok := observed.Rate(figures.Count())
		if !ok || !matches(rate, recorded.Rate[i], recorded.Rounded) {
			differ("%s rate = %v (present %v), Gatling recorded %v", columns[i], rate, ok, recorded.Rate[i])
		}

		sorted := durations[i]
		if len(sorted) == 0 {
			continue
		}

		for _, rank := range []float64{1, 5, 25, 50, 75, 90, 95, 99, 99.9, 100} {
			got, _ := figures.Percentile(rank)
			if misplaced, tolerance := RankMisplacement(sorted, got, rank), RankTolerance(rank, len(sorted)); misplaced > tolerance {
				differ("%s p%v = %d misplaces the rank by %.5f, over the rule's %.5f", columns[i], rank, got, misplaced, tolerance)
			}
		}
	}

	for i, count := range observed.Bands {
		if int64(count) != recorded.Bands[i] {
			differ("band %d = %d, Gatling recorded %d", i+1, count, recorded.Bands[i])
		}

		share, ok := observed.Share(count)
		if !ok || !matches(share, recorded.Shares[i], recorded.Rounded) {
			differ("band %d share = %v (present %v), Gatling recorded %v", i+1, share, ok, recorded.Shares[i])
		}
	}

	return differences
}

// matches compares a figure with what Gatling recorded: exactly when Gatling
// wrote the double, and to the two decimals a console prints otherwise.
func matches(got, recorded float64, rounded bool) bool {
	if !rounded {
		return got == recorded
	}

	return math.Abs(got-recorded) <= 0.005+1e-9
}
