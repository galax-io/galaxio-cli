// Package legacy renders Gatling's former stats.json products.
package legacy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/galax-io/galaxio-cli/internal/report"
)

// Product identifies one file under a run's js directory.
type Product string

const (
	Stats       Product = "stats"
	GlobalStats Product = "global_stats"
)

func (p Product) Filename() string {
	switch p {
	case Stats:
		return "stats.json"
	case GlobalStats:
		return "global_stats.json"
	default:
		return ""
	}
}

// Select parses a comma-separated product list and removes duplicates.
func Select(value string) ([]Product, error) {
	products := make([]Product, 0, 2)
	seen := make(map[Product]bool, 2)
	for _, token := range strings.Split(value, ",") {
		product := Product(strings.TrimSpace(token))
		if product != Stats && product != GlobalStats {
			return nil, fmt.Errorf("unknown report product %q", product)
		}
		if !seen[product] {
			seen[product] = true
			products = append(products, product)
		}
	}
	return products, nil
}

type integerValues struct {
	Total int64 `json:"total"`
	OK    int64 `json:"ok"`
	KO    int64 `json:"ko"`
}

type floatValues struct {
	Total float64 `json:"total"`
	OK    float64 `json:"ok"`
	KO    float64 `json:"ko"`
}

type distribution struct {
	Name       string  `json:"name"`
	HTMLName   string  `json:"htmlName"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type statistics struct {
	Name                          string        `json:"name"`
	NumberOfRequests              integerValues `json:"numberOfRequests"`
	MinResponseTime               integerValues `json:"minResponseTime"`
	MaxResponseTime               integerValues `json:"maxResponseTime"`
	MeanResponseTime              integerValues `json:"meanResponseTime"`
	StandardDeviation             integerValues `json:"standardDeviation"`
	Percentiles1                  integerValues `json:"percentiles1"`
	Percentiles2                  integerValues `json:"percentiles2"`
	Percentiles3                  integerValues `json:"percentiles3"`
	Percentiles4                  integerValues `json:"percentiles4"`
	Group1                        distribution  `json:"group1"`
	Group2                        distribution  `json:"group2"`
	Group3                        distribution  `json:"group3"`
	Group4                        distribution  `json:"group4"`
	MeanNumberOfRequestsPerSecond floatValues   `json:"meanNumberOfRequestsPerSecond"`
}

// Render creates the selected JSON documents from one completed scan.
func Render(source *report.Tree, products []Product) (map[Product][]byte, error) {
	if source == nil {
		return nil, errors.New("legacy export requires a completed scan")
	}

	view, err := source.Summary.Export()
	if err != nil {
		return nil, err
	}
	global := makeStatistics("All Requests", view)

	documents := make(map[Product][]byte, len(products))
	for _, product := range products {
		var value any
		switch product {
		case GlobalStats:
			value = global
		default:
			return nil, fmt.Errorf("invalid legacy product %q", product)
		}

		data, err := marshal(value)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", product, err)
		}
		documents[product] = data
	}

	return documents, nil
}

func marshal(value any) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func makeStatistics(name string, view report.ExportStats) statistics {
	values := func(read func(report.ExportColumn) int64) integerValues {
		return integerValues{
			Total: read(view.Columns[0]),
			OK:    read(view.Columns[1]),
			KO:    read(view.Columns[2]),
		}
	}
	count := func(column report.ExportColumn) int64 { return int64(column.Count) }
	percentile := func(index int) func(report.ExportColumn) int64 {
		return func(column report.ExportColumn) int64 { return column.Percentiles[index] }
	}

	lower := strconv.FormatInt(view.Bounds.Lower, 10)
	upper := strconv.FormatInt(view.Bounds.Upper, 10)
	groups := [4]distribution{
		{Name: "t < " + lower + " ms", HTMLName: "t < " + lower + " ms"},
		{Name: lower + " ms <= t < " + upper + " ms", HTMLName: "t >= " + lower + " ms <br> t < " + upper + " ms"},
		{Name: "t >= " + upper + " ms", HTMLName: "t >= " + upper + " ms"},
		{Name: "failed", HTMLName: "failed"},
	}
	for i := range groups {
		groups[i].Count = view.Bands[i].Count
		groups[i].Percentage = view.Bands[i].Percentage
	}

	return statistics{
		Name:             name,
		NumberOfRequests: values(count),
		MinResponseTime:  values(func(column report.ExportColumn) int64 { return column.Min }),
		MaxResponseTime:  values(func(column report.ExportColumn) int64 { return column.Max }),
		MeanResponseTime: values(func(column report.ExportColumn) int64 { return column.Mean }),
		StandardDeviation: values(func(column report.ExportColumn) int64 {
			return column.StdDev
		}),
		Percentiles1: values(percentile(0)),
		Percentiles2: values(percentile(1)),
		Percentiles3: values(percentile(2)),
		Percentiles4: values(percentile(3)),
		Group1:       groups[0],
		Group2:       groups[1],
		Group3:       groups[2],
		Group4:       groups[3],
		MeanNumberOfRequestsPerSecond: floatValues{
			Total: view.Columns[0].Rate,
			OK:    view.Columns[1].Rate,
			KO:    view.Columns[2].Rate,
		},
	}
}
