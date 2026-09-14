package report

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/galax-io/parsec/model"
)

func TestOptMarshalJSON(t *testing.T) {
	t.Parallel()

	type holder struct {
		Duration Opt[int64]   `json:"duration,omitzero"`
		Failure  Opt[Failure] `json:"failure,omitzero"`
	}

	tests := []struct {
		name     string
		value    holder
		expected string
	}{
		{
			name:     "unset values are omitted",
			value:    holder{},
			expected: `{}`,
		},
		{
			name:     "a recorded zero is written as zero",
			value:    holder{Duration: Some[int64](0)},
			expected: `{"duration":0}`,
		},
		{
			name:     "a set value is written",
			value:    holder{Duration: Some[int64](1504)},
			expected: `{"duration":1504}`,
		},
		{
			name:     "a struct value nests and omits its own empty type",
			value:    holder{Failure: Some(Failure{Message: "status.find.is(200), found 500"})},
			expected: `{"failure":{"message":"status.find.is(200), found 500"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(got) != tt.expected {
				t.Fatalf("Marshal = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestOptGet(t *testing.T) {
	t.Parallel()

	if v, ok := Some("x").Get(); !ok || v != "x" {
		t.Fatalf("Some(\"x\").Get() = %q, %v; want \"x\", true", v, ok)
	}
	var unset Opt[string]
	if v, ok := unset.Get(); ok || v != "" {
		t.Fatalf("unset.Get() = %q, %v; want \"\", false", v, ok)
	}
	if unset != (Opt[string]{}) {
		t.Fatalf("unset Opt must equal the zero Opt")
	}
}

func TestRequestRecordGroupsAlwaysPresent(t *testing.T) {
	t.Parallel()

	got, err := json.Marshal(RequestRecord{Kind: KindRequest, Groups: []string{}, Name: "GET /ok", Outcome: "success"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	expected := `{"kind":"request","groups":[],"name":"GET /ok","outcome":"success"}`
	if string(got) != expected {
		t.Fatalf("Marshal = %s, want %s", got, expected)
	}
}

func TestRunRecordKeyOrder(t *testing.T) {
	t.Parallel()

	got, err := json.Marshal(RunRecord{
		Kind:        KindRun,
		ID:          "sim",
		Name:        "io.x.Sim",
		Start:       1700000000000,
		Tool:        "gatling",
		ToolVersion: "3.15.1",
		Absent:      []string{},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	expected := `{"kind":"run","id":"sim","name":"io.x.Sim","start":1700000000000,"tool":"gatling","toolVersion":"3.15.1","absent":[]}`
	if string(got) != expected {
		t.Fatalf("Marshal = %s, want %s", got, expected)
	}
}

func TestFieldName(t *testing.T) {
	t.Parallel()

	seen := map[string]model.Field{}
	for _, f := range model.FieldsKnown() {
		name, ok := fieldNames[f]
		if !ok {
			t.Errorf("field %v has no schema identifier; add it to fieldNames", f)
			continue
		}
		if strings.ContainsAny(name, " \t") {
			t.Errorf("field %v renders as %q, which is not an identifier", f, name)
		}
		if prev, dup := seen[name]; dup {
			t.Errorf("fields %v and %v share the identifier %q", prev, f, name)
		}
		seen[name] = f
		if got := fieldName(f); got != name {
			t.Errorf("fieldName(%v) = %q, want %q", f, got, name)
		}
	}

	if got, want := fieldName(model.Field(60000)), model.Field(60000).String(); got != want {
		t.Fatalf("fieldName(unknown) = %q, want the parsec rendering %q", got, want)
	}
}
