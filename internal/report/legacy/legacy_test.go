package legacy

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/galax-io/galaxio-cli/internal/report"
	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/model"
)

func testTree(t *testing.T) *report.Tree {
	t.Helper()

	items := []model.Item{
		{Kind: model.ItemSample, Sample: model.Sample{
			Groups:   []string{"outer"},
			Name:     `say "hi" \\ 雪`,
			Start:    time.Unix(1, 0),
			Duration: model.Some(10 * time.Millisecond),
			Outcome:  model.OutcomeSuccess,
		}},
		{Kind: model.ItemSample, Sample: model.Sample{
			Name:     "failed",
			Start:    time.Unix(2, 0),
			Duration: model.Some(20 * time.Millisecond),
			Outcome:  model.OutcomeFailure,
		}},
	}
	source := &report.Source{Reader: reporttest.Items(model.Run{}, nil, items...)}
	tree, err := source.ScanTree(t.Context(), report.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestRender(t *testing.T) {
	products, err := Select("stats, global_stats, stats")
	if err != nil || len(products) != 2 {
		t.Fatalf("Select = %v, %v", products, err)
	}
	documents, err := Render(testTree(t), products)
	if err != nil {
		t.Fatal(err)
	}

	var global struct {
		Name             string
		NumberOfRequests struct{ Total, OK, KO int }
	}
	if err := json.Unmarshal(documents[GlobalStats], &global); err != nil {
		t.Fatal(err)
	}
	if global.Name != "All Requests" || global.NumberOfRequests.Total != 2 ||
		global.NumberOfRequests.OK != 1 || global.NumberOfRequests.KO != 1 {
		t.Fatalf("global statistics = %+v", global)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(documents[GlobalStats], &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"name", "numberOfRequests", "minResponseTime", "maxResponseTime",
		"meanResponseTime", "standardDeviation", "percentiles1", "percentiles2",
		"percentiles3", "percentiles4", "group1", "group2", "group3", "group4",
		"meanNumberOfRequestsPerSecond",
	} {
		if _, exists := fields[name]; !exists {
			t.Fatalf("global_stats.json is missing %q", name)
		}
	}
	if len(fields) != 15 {
		t.Fatalf("global_stats.json fields = %d, want 15", len(fields))
	}

	repeated, err := Render(testTree(t), products)
	if err != nil {
		t.Fatal(err)
	}
	for product, document := range documents {
		if !bytes.Equal(document, repeated[product]) {
			t.Fatalf("repeated %s render differs", product)
		}
		if !bytes.HasPrefix(document, []byte("{\n    \"")) {
			t.Fatalf("%s is not indented with four spaces", product)
		}
		if len(document) == 0 || document[len(document)-1] != '\n' {
			t.Fatalf("%s has no trailing newline", product)
		}
	}

	var root treeNode
	if err := json.Unmarshal(documents[Stats], &root); err != nil {
		t.Fatal(err)
	}
	if root.Type != "GROUP" || root.Contents == nil || len(*root.Contents) != 2 {
		t.Fatalf("tree root = %+v", root)
	}
	found := false
	var visit func(*treeNode)
	visit = func(node *treeNode) {
		if node.Name == `say "hi" \\ 雪` {
			found = true
		}
		if node.Contents != nil {
			for _, child := range *node.Contents {
				visit(child)
			}
		}
	}
	visit(&root)
	if !found {
		t.Fatal("stats.json lost the decoded request name")
	}
}
