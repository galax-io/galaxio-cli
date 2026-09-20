package report

import (
	"errors"
	"fmt"
	"math"
)

// ExportStats is the validated numeric view used by both legacy products.
// Columns are total, OK, KO; Bands are under, between, over, failed.
// It is not an intermediate serialized format.
type ExportStats struct {
	Columns [3]ExportColumn
	Bands   [4]ExportBand
	Bounds  Bands
}

// ExportColumn uses Count == 0 to distinguish legacy empty-column placeholders
// from populated measurements. Empty columns contain only numeric zeros.
type ExportColumn struct {
	Count       int
	Min         int64
	Max         int64
	Mean        int64
	StdDev      int64
	Percentiles [4]int64
	Rate        float64
}

// ExportBand carries an exact population count and its binary64 percentage.
type ExportBand struct {
	Count      int
	Percentage float64
}

// Export returns a complete legacy numeric view or refuses a missing populated
// measurement. It reuses the exact moments and existing deterministic sketches,
// and does not validate group metrics which global statistics do not represent.
func (s Summary) Export() (ExportStats, error) {
	opts, err := s.Options.Normalize()
	if err != nil {
		return ExportStats{}, err
	}
	if len(opts.Percentiles) != 4 {
		return ExportStats{}, errors.New("legacy statistics require exactly four normalized percentile ranks")
	}
	if s.Tally.Unknown != 0 {
		return ExportStats{}, errors.New("request outcome is unknown")
	}
	all := s.All()
	countsAgree := s.Tally.Requests == all.count &&
		s.Tally.Successes == s.OK.count && s.Tally.Failures == s.Failed.count
	if !countsAgree || s.Tally.Requests < 0 {
		return ExportStats{}, errors.New("request counts are inconsistent")
	}
	view := ExportStats{Bounds: opts.Bands}
	for i, figures := range []Figures{all, s.OK, s.Failed} {
		column, err := exportColumn(figures, opts.Percentiles)
		if err != nil {
			return ExportStats{}, fmt.Errorf("%s requests: %w", [...]string{"total", "ok", "ko"}[i], err)
		}
		if column.Count != 0 {
			rate, ok := s.Rate(column.Count)
			if !ok || !validExportFloat(rate) {
				return ExportStats{}, errors.New("request rate requires a usable whole-run span")
			}
			column.Rate = rate
		}
		view.Columns[i] = column
	}
	remaining := s.OK.count
	for i, count := range []int{s.Under, s.Between, s.Over} {
		if count < 0 || count > remaining {
			return ExportStats{}, fmt.Errorf("response-time band %d count is inconsistent", i+1)
		}
		remaining -= count
	}
	if remaining != 0 {
		return ExportStats{}, errors.New("response-time bands lack successful timings")
	}
	for i, count := range []int{s.Under, s.Between, s.Over, s.Failed.count} {
		percentage := 0.0
		if all.count != 0 {
			percentage = float64(count) / float64(all.count) * 100
		}
		if !validExportFloat(percentage) {
			return ExportStats{}, fmt.Errorf("response-time band %d percentage is not finite", i+1)
		}
		view.Bands[i] = ExportBand{Count: count, Percentage: percentage}
	}
	return view, nil
}

func exportColumn(figures Figures, ranks []float64) (ExportColumn, error) {
	if figures.count < 0 || figures.timed != figures.count {
		return ExportColumn{}, errors.New("timing is missing from a populated request column")
	}
	if figures.count == 0 {
		return ExportColumn{}, nil
	}
	column := ExportColumn{Count: figures.count}
	readings := []struct {
		name string
		read func() (int64, bool)
		dest *int64
	}{
		{name: "minimum", read: figures.Min, dest: &column.Min},
		{name: "maximum", read: figures.Max, dest: &column.Max},
		{name: "mean", read: figures.Mean, dest: &column.Mean},
		{name: "standard deviation", read: figures.StdDev, dest: &column.StdDev},
	}
	for _, reading := range readings {
		value, ok := reading.read()
		if !ok || value < 0 {
			return ExportColumn{}, fmt.Errorf("%s timing is unavailable", reading.name)
		}
		*reading.dest = value
	}
	for i, rank := range ranks {
		value, ok := figures.Percentile(rank)
		if !ok || value < 0 {
			return ExportColumn{}, fmt.Errorf("percentile %v is unavailable", rank)
		}
		column.Percentiles[i] = value
	}
	return column, nil
}

func validExportFloat(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
