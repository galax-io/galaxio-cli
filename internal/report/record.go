package report

import "github.com/galax-io/parsec/model"

// Kind is the value of the "kind" key that opens every record.
type Kind string

// The record kinds, in the order a consumer meets them: the header first,
// then whatever the log recorded.
const (
	KindRun       Kind = "run"
	KindRequest   Kind = "request"
	KindGroup     Kind = "group"
	KindUser      Kind = "user"
	KindError     Kind = "error"
	KindAssertion Kind = "assertion"
)

// Warning is what the source's version gate raised about a run that was read
// anyway.
type Warning struct {
	Version string `json:"version"`
	Reason  string `json:"reason"`
}

// Failure is why a request failed. Type is omitted when the source records
// only free text, as Gatling does.
type Failure struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message"`
}

// RunRecord is the header: everything about the run that does not grow with
// its length. It is always the first line.
type RunRecord struct {
	Kind        Kind      `json:"kind"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Start       int64     `json:"start,omitzero"`
	Tool        string    `json:"tool"`
	ToolVersion string    `json:"toolVersion"`
	Absent      []string  `json:"absent"`
	Warnings    []Warning `json:"warnings,omitempty"`
	Assertions  []string  `json:"assertions,omitempty"`
}

// RequestRecord is one recorded request. Groups is always present: a
// request outside any group has an empty path, not an absent one.
//
// An optional field is a pointer so that a value the source did not record
// is omitted while a recorded zero is written as 0. The pointers a conversion
// returns refer to scratch storage the writer reuses, so a record is valid
// only until the next conversion — the same rule parsec applies to Groups.
type RequestRecord struct {
	Kind          Kind     `json:"kind"`
	Groups        []string `json:"groups"`
	Name          string   `json:"name"`
	Start         int64    `json:"start,omitzero"`
	Duration      *int64   `json:"duration,omitempty"`
	Outcome       string   `json:"outcome"`
	Failure       *Failure `json:"failure,omitempty"`
	Scenario      *string  `json:"scenario,omitempty"`
	ResponseCode  *string  `json:"responseCode,omitempty"`
	BytesSent     *int64   `json:"bytesSent,omitempty"`
	BytesReceived *int64   `json:"bytesReceived,omitempty"`
}

// GroupRecord is one traversal of a group, closing. Groups is the group's
// own path, its own name last.
type GroupRecord struct {
	Kind              Kind     `json:"kind"`
	Groups            []string `json:"groups"`
	Start             int64    `json:"start,omitzero"`
	Duration          *int64   `json:"duration,omitempty"`
	CumulatedDuration *int64   `json:"cumulatedDuration,omitempty"`
	Outcome           string   `json:"outcome"`
}

// UserRecord is a virtual user starting or ending a scenario.
type UserRecord struct {
	Kind     Kind   `json:"kind"`
	Scenario string `json:"scenario"`
	Event    string `json:"event"`
	At       int64  `json:"at,omitzero"`
}

// ErrorRecord is a failure the source recorded that belongs to no request.
type ErrorRecord struct {
	Kind    Kind   `json:"kind"`
	Message string `json:"message"`
	At      int64  `json:"at,omitzero"`
}

// AssertionRecord is an opaque declared-assertion payload a source yields
// among its events. Payload is base64 of the bytes verbatim.
type AssertionRecord struct {
	Kind    Kind   `json:"kind"`
	Payload string `json:"payload"`
}

// fieldNames maps a capability field to the identifier the header's "absent"
// list uses. The identifiers are this schema's own and are stable; parsec's
// Field.String() is written for people and may be reworded.
var fieldNames = map[model.Field]string{
	model.FieldSampleDuration:         "sample.duration",
	model.FieldSampleScenario:         "sample.scenario",
	model.FieldSampleResponseCode:     "sample.responseCode",
	model.FieldSampleBytesSent:        "sample.bytesSent",
	model.FieldSampleBytesReceived:    "sample.bytesReceived",
	model.FieldSampleFailureType:      "sample.failureType",
	model.FieldSampleUserIdentity:     "sample.userIdentity",
	model.FieldGroupDuration:          "group.duration",
	model.FieldGroupCumulatedDuration: "group.cumulatedDuration",
	model.FieldGroupOutcome:           "group.outcome",
	model.FieldConnectTiming:          "timing.connect",
	model.FieldDNSTiming:              "timing.dns",
	model.FieldTLSTiming:              "timing.tls",
	model.FieldRequirements:           "requirements",
	model.FieldIntervalSeries:         "intervalSeries",
}

// fieldName returns the schema identifier for f. A field this table does not
// know — one a later parsec added — falls back to parsec's own rendering so
// that nothing is dropped from the "absent" list.
func fieldName(f model.Field) string {
	if name, ok := fieldNames[f]; ok {
		return name
	}
	return f.String()
}
