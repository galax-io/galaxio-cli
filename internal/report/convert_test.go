package report

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/galax-io/parsec/model"
)

func at(ms int64) time.Time {
	return time.UnixMilli(ms).UTC()
}

func marshal(t *testing.T, v any) string {
	t.Helper()

	got, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return string(got)
}

func TestHeaderFrom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		run      model.Run
		expected string
	}{
		{
			name: "full header with warnings and binary assertion payloads",
			run: model.Run{
				ID:           "corpussimulation",
				Name:         "io.galaxio.parsec.corpus.CorpusSimulation",
				Description:  "smoke",
				Start:        at(1788670094356),
				Tool:         "gatling",
				ToolVersion:  "3.99.0",
				Capabilities: model.NewCapabilities(model.FieldSampleDuration, model.FieldGroupDuration, model.FieldGroupCumulatedDuration, model.FieldGroupOutcome),
				Warnings:     []model.Warning{{Version: "3.99.0", Reason: "above the verified range"}},
				Assertions:   []string{"\x00\x01\x80Y@", "plain"},
			},
			expected: `{"kind":"run","id":"corpussimulation","name":"io.galaxio.parsec.corpus.CorpusSimulation","description":"smoke","start":1788670094356,"tool":"gatling","toolVersion":"3.99.0","absent":["sample.scenario","sample.responseCode","sample.bytesSent","sample.bytesReceived","sample.failureType","sample.userIdentity","timing.connect","timing.dns","timing.tls","requirements","intervalSeries"],"warnings":[{"version":"3.99.0","reason":"above the verified range"}],"assertions":["AAGAWUA=","cGxhaW4="]}`,
		},
		{
			name: "no description, unresolved start, nothing absent, no warnings, no assertions",
			run: model.Run{
				ID:           "sim",
				Name:         "io.x.Sim",
				Tool:         "gatling",
				ToolVersion:  "3.12.0",
				Capabilities: model.NewCapabilities(model.FieldsKnown()...),
			},
			expected: `{"kind":"run","id":"sim","name":"io.x.Sim","tool":"gatling","toolVersion":"3.12.0","absent":[]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := marshal(t, headerFrom(tt.run)); got != tt.expected {
				t.Fatalf("headerFrom =\n%s\nwant\n%s", got, tt.expected)
			}
		})
	}
}

func TestRequestFrom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sample   model.Sample
		expected string
	}{
		{
			name: "success outside any group",
			sample: model.Sample{
				Name:     "GET /ok",
				Start:    at(1788670094894),
				Duration: model.Some(4 * time.Millisecond),
				Outcome:  model.OutcomeSuccess,
			},
			expected: `{"kind":"request","groups":[],"name":"GET /ok","start":1788670094894,"duration":4,"outcome":"success"}`,
		},
		{
			name: "failure inside a nested group carries the message and no empty type",
			sample: model.Sample{
				Groups:   []string{"outer", "inner, with comma"},
				Name:     "GET /fail",
				Start:    at(1788670095021),
				Duration: model.Some(3 * time.Millisecond),
				Outcome:  model.OutcomeFailure,
				Failure:  model.Some(model.Failure{Message: "status.find.is(200), found 500"}),
			},
			expected: `{"kind":"request","groups":["outer","inner, with comma"],"name":"GET /fail","start":1788670095021,"duration":3,"outcome":"failure","failure":{"message":"status.find.is(200), found 500"}}`,
		},
		{
			name: "a recorded zero duration is written and an unresolved start is omitted",
			sample: model.Sample{
				Name:     "GET /instant",
				Duration: model.Some(time.Duration(0)),
				Outcome:  model.OutcomeSuccess,
			},
			expected: `{"kind":"request","groups":[],"name":"GET /instant","duration":0,"outcome":"success"}`,
		},
		{
			name: "no recorded end leaves duration absent",
			sample: model.Sample{
				Name:    "GET /never-completed",
				Start:   at(1788670094894),
				Outcome: model.OutcomeFailure,
				Failure: model.Some(model.Failure{Type: "timeout", Message: "never completed"}),
			},
			expected: `{"kind":"request","groups":[],"name":"GET /never-completed","start":1788670094894,"outcome":"failure","failure":{"type":"timeout","message":"never completed"}}`,
		},
		{
			name: "optional fields another source records are written when set",
			sample: model.Sample{
				Name:          "POST /checkout",
				Start:         at(1788670094894),
				Duration:      model.Some(120 * time.Millisecond),
				Outcome:       model.OutcomeSuccess,
				Scenario:      model.Some("Checkout"),
				ResponseCode:  model.Some("201"),
				BytesSent:     model.Some[int64](0),
				BytesReceived: model.Some[int64](812),
			},
			expected: `{"kind":"request","groups":[],"name":"POST /checkout","start":1788670094894,"duration":120,"outcome":"success","scenario":"Checkout","responseCode":"201","bytesSent":0,"bytesReceived":812}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := marshal(t, requestFrom(tt.sample, &scratch{})); got != tt.expected {
				t.Fatalf("requestFrom =\n%s\nwant\n%s", got, tt.expected)
			}
		})
	}
}

func TestGroupFrom(t *testing.T) {
	t.Parallel()

	got := marshal(t, groupFrom(model.GroupSample{
		Groups:            []string{"outer", "inner, with comma"},
		Start:             at(1788670095016),
		Duration:          model.Some(1504 * time.Millisecond),
		CumulatedDuration: model.Some(1503 * time.Millisecond),
		Outcome:           model.OutcomeFailure,
	}, &scratch{}))
	expected := `{"kind":"group","groups":["outer","inner, with comma"],"start":1788670095016,"duration":1504,"cumulatedDuration":1503,"outcome":"failure"}`
	if got != expected {
		t.Fatalf("groupFrom =\n%s\nwant\n%s", got, expected)
	}

	got = marshal(t, groupFrom(model.GroupSample{Groups: []string{"outer"}, Outcome: model.OutcomeSuccess}, &scratch{}))
	expected = `{"kind":"group","groups":["outer"],"outcome":"success"}`
	if got != expected {
		t.Fatalf("groupFrom without timings =\n%s\nwant\n%s", got, expected)
	}
}

func TestUserFrom(t *testing.T) {
	t.Parallel()

	got := marshal(t, userFrom(model.UserEvent{Scenario: "Corpus recording", Kind: model.UserStart, At: at(1788670094885)}))
	expected := `{"kind":"user","scenario":"Corpus recording","event":"start","at":1788670094885}`
	if got != expected {
		t.Fatalf("userFrom =\n%s\nwant\n%s", got, expected)
	}

	got = marshal(t, userFrom(model.UserEvent{Scenario: "Corpus recording", Kind: model.UserEnd}))
	expected = `{"kind":"user","scenario":"Corpus recording","event":"end"}`
	if got != expected {
		t.Fatalf("userFrom with unresolved time =\n%s\nwant\n%s", got, expected)
	}
}

func TestErrorFrom(t *testing.T) {
	t.Parallel()

	got := marshal(t, errorFrom(model.RunError{Message: "unresolvable url: No attribute named 'undefinedAttribute' is defined ", At: at(1788670096632)}))
	expected := `{"kind":"error","message":"unresolvable url: No attribute named 'undefinedAttribute' is defined ","at":1788670096632}`
	if got != expected {
		t.Fatalf("errorFrom =\n%s\nwant\n%s", got, expected)
	}
}

func TestAssertionFrom(t *testing.T) {
	t.Parallel()

	raw := "\x00\x01\x01\x00\x01\x05\x00\x00\x00\x00\x00\x00\x80Y@"
	rec := assertionFrom(raw)
	if rec.Kind != KindAssertion {
		t.Fatalf("Kind = %q, want %q", rec.Kind, KindAssertion)
	}
	decoded, err := base64.StdEncoding.DecodeString(rec.Payload)
	if err != nil {
		t.Fatalf("payload is not base64: %v", err)
	}
	if string(decoded) != raw {
		t.Fatalf("payload round trip = %q, want %q", decoded, raw)
	}
}
