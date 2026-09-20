// Package legacy renders Gatling's former stats.json products.
package legacy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/galax-io/galaxio-cli/internal/report"
	"github.com/galax-io/parsec/model"
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

type treeNode struct {
	Type          string                `json:"type"`
	Name          string                `json:"name"`
	Path          string                `json:"path"`
	PathFormatted string                `json:"pathFormatted"`
	Stats         statistics            `json:"stats"`
	Contents      *map[string]*treeNode `json:"contents,omitempty"`
}

// Render creates the selected JSON documents from one completed scan.
func Render(source *report.Tree, products []Product) (map[Product][]byte, error) {
	if source == nil {
		return nil, errors.New("legacy export requires a completed scan")
	}

	root, err := buildTree(source)
	if err != nil {
		return nil, err
	}

	documents := make(map[Product][]byte, len(products))
	for _, product := range products {
		var value any
		switch product {
		case Stats:
			value = root
		case GlobalStats:
			value = root.Stats
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

func buildTree(source *report.Tree) (*treeNode, error) {
	rootView, err := source.Summary.Export()
	if err != nil {
		return nil, err
	}

	rootContents := make(map[string]*treeNode)
	root := &treeNode{
		Type:          "GROUP",
		Name:          "All Requests",
		PathFormatted: identifier("", "group_"),
		Stats:         makeStatistics("All Requests", rootView),
		Contents:      &rootContents,
	}
	nodes := map[model.Position]*treeNode{{}: root}

	for _, sourceNode := range source.Nodes {
		if sourceNode == nil {
			return nil, errors.New("legacy export contains a nil node")
		}
		parent := nodes[sourceNode.Parent]
		if parent == nil || parent.Contents == nil {
			return nil, fmt.Errorf("legacy export is missing parent for %s", sourceNode.Position)
		}

		groups := sourceNode.Position.Groups()
		name := sourceNode.Position.Name()
		prefix := "req_"
		typeName := "REQUEST"
		segments := append([]string{}, groups...)
		var contents *map[string]*treeNode
		if sourceNode.Position.Kind() == model.PositionGroup {
			prefix = "group_"
			typeName = "GROUP"
			if len(groups) > 0 {
				name = groups[len(groups)-1]
			}
			children := make(map[string]*treeNode)
			contents = &children
		} else {
			segments = append(segments, name)
		}
		path := strings.Join(segments, " / ")
		view, err := sourceNode.Export(source.Summary.Tally.Bounds)
		if err != nil {
			return nil, err
		}
		child := &treeNode{
			Type:          typeName,
			Name:          name,
			Path:          path,
			PathFormatted: identifier(path, prefix),
			Stats:         makeStatistics(name, view),
			Contents:      contents,
		}
		key := uniqueKey(*parent.Contents, identifier(name, prefix))
		(*parent.Contents)[key] = child
		nodes[sourceNode.Position] = child
	}

	return root, nil
}

func uniqueKey(contents map[string]*treeNode, key string) string {
	if _, exists := contents[key]; !exists {
		return key
	}
	for suffix := 2; ; suffix++ {
		candidate := key + "-" + strconv.Itoa(suffix)
		if _, exists := contents[candidate]; !exists {
			return candidate
		}
	}
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

func identifier(value, prefix string) string {
	trimmed := strings.TrimFunc(value, func(r rune) bool { return r <= 0x20 })
	if trimmed == "" {
		trimmed = "missing_name"
	}

	clean := make([]rune, 0, 15)
	for _, r := range strings.ToLower(trimmed) {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			r = '-'
		}
		clean = append(clean, r)
		if len(clean) == 15 {
			break
		}
	}

	var hash uint32
	for _, unit := range utf16.Encode([]rune(trimmed)) {
		hash = hash*31 + uint32(unit)
	}
	return prefix + string(clean) + "-" + strconv.FormatInt(int64(int32(hash)), 10)
}
