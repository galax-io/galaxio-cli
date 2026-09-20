package report

import (
	"context"
	"testing"
	"time"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/model"
)

func TestSummaryExport(t *testing.T) {
	items := []model.Item{
		{Kind: model.ItemSample, Sample: model.Sample{
			Name: "ok", Start: time.UnixMilli(1000), Duration: model.Some(10 * time.Millisecond), Outcome: model.OutcomeSuccess,
		}},
		{Kind: model.ItemSample, Sample: model.Sample{
			Name: "failed", Start: time.UnixMilli(1020), Duration: model.Some(20 * time.Millisecond), Outcome: model.OutcomeFailure,
		}},
	}
	summary, err := Scan(context.Background(), reporttest.Items(model.Run{}, nil, items...), DefaultOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}

	view, err := summary.Export()
	if err != nil {
		t.Fatal(err)
	}
	if view.Columns[0].Count != 2 || view.Columns[1].Count != 1 || view.Columns[2].Count != 1 {
		t.Fatalf("export counts = %d/%d/%d", view.Columns[0].Count, view.Columns[1].Count, view.Columns[2].Count)
	}
	if view.Columns[1].Min != 10 || view.Columns[2].Max != 20 {
		t.Fatalf("export timings = %+v", view.Columns)
	}
	if view.Bands[0].Count != 1 || view.Bands[3].Count != 1 {
		t.Fatalf("export bands = %+v", view.Bands)
	}
}

func TestSummaryExportEmptyOutcome(t *testing.T) {
	item := model.Item{Kind: model.ItemSample, Sample: model.Sample{
		Name: "ok", Start: time.UnixMilli(1000), Duration: model.Some(10 * time.Millisecond), Outcome: model.OutcomeSuccess,
	}}
	summary, err := Scan(context.Background(), reporttest.Items(model.Run{}, nil, item), DefaultOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}
	view, err := summary.Export()
	if err != nil {
		t.Fatal(err)
	}
	if view.Columns[2] != (ExportColumn{}) {
		t.Fatalf("empty failed column = %+v, want zero placeholders", view.Columns[2])
	}
}
