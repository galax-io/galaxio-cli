package report

import (
	"errors"
	"testing"
	"time"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/model"
)

func TestSourceScanTree(t *testing.T) {
	items := []model.Item{
		{Kind: model.ItemSample, Sample: model.Sample{
			Groups: []string{"outer", "inner"}, Name: "ok", Start: time.UnixMilli(1000),
			Duration: model.Some(10 * time.Millisecond), Outcome: model.OutcomeSuccess,
		}},
		{Kind: model.ItemSample, Sample: model.Sample{
			Groups: []string{"outer"}, Name: "failed", Start: time.UnixMilli(1020),
			Duration: model.Some(20 * time.Millisecond), Outcome: model.OutcomeFailure,
		}},
	}
	source := &Source{Reader: reporttest.Items(model.Run{}, nil, items...)}
	tree, err := source.ScanTree(t.Context(), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if tree.Summary.Tally.Requests != 2 || len(tree.Nodes) != 4 {
		t.Fatalf("tree = %d requests, %d nodes", tree.Summary.Tally.Requests, len(tree.Nodes))
	}

	position := model.NewSamplePosition([]string{"outer", "inner"}, "ok")
	node, ok := tree.Node(position)
	if !ok {
		t.Fatalf("tree has no node for %s", position)
	}
	view, err := node.Export(tree.Summary.Tally.Bounds)
	if err != nil {
		t.Fatal(err)
	}
	if view.Columns[0].Count != 1 || view.Columns[1].Count != 1 || view.Columns[2].Count != 0 {
		t.Fatalf("node counts = %+v", view.Columns)
	}

	if _, err := source.ScanTree(t.Context(), DefaultOptions()); !errors.Is(err, ErrSpent) {
		t.Fatalf("second ScanTree = %v, want ErrSpent", err)
	}
}

func TestSourceScanTreeRejectsMissingTiming(t *testing.T) {
	item := model.Item{Kind: model.ItemSample, Sample: model.Sample{
		Name: "unfinished", Start: time.UnixMilli(1000), Outcome: model.OutcomeSuccess,
	}}
	source := &Source{Reader: reporttest.Items(model.Run{}, nil, item)}
	if _, err := source.ScanTree(t.Context(), DefaultOptions()); err == nil {
		t.Fatal("ScanTree accepted a populated request without timing")
	}
}
