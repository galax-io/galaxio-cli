package report

import (
	"encoding/json"

	"github.com/galax-io/parsec/model"
)

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

// Opt is a value the source may not have recorded.
//
// The zero Opt is unset and is omitted from JSON under the omitzero tag, so a
// duration the source measured as 0 ms and a duration it never recorded do
// not read alike. It is a value rather than a pointer so that an optional
// field costs no allocation per record.
type Opt[T comparable] struct {
	value T
	isSet bool
}

// Some returns an Opt holding v.
func Some[T comparable](v T) Opt[T] {
	return Opt[T]{value: v, isSet: true}
}

// Get returns the value and whether it was set.
func (o Opt[T]) Get() (T, bool) {
	return o.value, o.isSet
}

// IsZero reports whether the value is unset; encoding/json consults it for
// the omitzero tag.
func (o Opt[T]) IsZero() bool {
	return !o.isSet
}

// MarshalJSON writes the held value; an unset Opt is omitted before this is
// reached.
func (o Opt[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.value)
}

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
type RequestRecord struct {
	Kind          Kind         `json:"kind"`
	Groups        []string     `json:"groups"`
	Name          string       `json:"name"`
	Start         int64        `json:"start,omitzero"`
	Duration      Opt[int64]   `json:"duration,omitzero"`
	Outcome       string       `json:"outcome"`
	Failure       Opt[Failure] `json:"failure,omitzero"`
	Scenario      Opt[string]  `json:"scenario,omitzero"`
	ResponseCode  Opt[string]  `json:"responseCode,omitzero"`
	BytesSent     Opt[int64]   `json:"bytesSent,omitzero"`
	BytesReceived Opt[int64]   `json:"bytesReceived,omitzero"`
}

// GroupRecord is one traversal of a group, closing. Groups is the group's
// own path, its own name last.
type GroupRecord struct {
	Kind              Kind       `json:"kind"`
	Groups            []string   `json:"groups"`
	Start             int64      `json:"start,omitzero"`
	Duration          Opt[int64] `json:"duration,omitzero"`
	CumulatedDuration Opt[int64] `json:"cumulatedDuration,omitzero"`
	Outcome           string     `json:"outcome"`
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
