package report

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/galax-io/parsec/model"
)

func ptr[T any](v T) *T {
	return &v
}

func TestOptionalFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    RequestRecord
		expected string
	}{
		{
			name:     "unrecorded values are omitted",
			value:    RequestRecord{Kind: KindRequest, Groups: []string{}, Name: "GET /ok", Outcome: "success"},
			expected: `{"kind":"request","groups":[],"name":"GET /ok","outcome":"success"}`,
		},
		{
			name:     "a recorded zero is written as zero",
			value:    RequestRecord{Kind: KindRequest, Groups: []string{}, Name: "GET /ok", Duration: ptr[int64](0), Outcome: "success"},
			expected: `{"kind":"request","groups":[],"name":"GET /ok","duration":0,"outcome":"success"}`,
		},
		{
			name:     "a failure nests and omits its own empty type",
			value:    RequestRecord{Kind: KindRequest, Groups: []string{}, Name: "GET /fail", Outcome: "failure", Failure: &Failure{Message: "status.find.is(200), found 500"}},
			expected: `{"kind":"request","groups":[],"name":"GET /fail","outcome":"failure","failure":{"message":"status.find.is(200), found 500"}}`,
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
