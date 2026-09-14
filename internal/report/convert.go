package report

import (
	"encoding/base64"
	"time"

	"github.com/galax-io/parsec/model"
)

// noGroups is the path of a request outside any group. It is shared and
// never written to, so every such record renders "groups":[] without
// allocating.
var noGroups = []string{}

// scratch is the storage the optional fields of a record point at. One
// scratch serves every record a writer produces, so an optional field costs
// no allocation: encoding/json follows the pointer and writes the value, or
// omits the field when the pointer is nil.
type scratch struct {
	duration      int64
	cumulated     int64
	bytesSent     int64
	bytesReceived int64
	scenario      string
	responseCode  string
	failure       Failure
}

// headerFrom builds the run header. Warnings and assertions are copied only
// when present so that a run without them omits the keys.
func headerFrom(run model.Run) RunRecord {
	absent := run.Capabilities.Absent()
	names := make([]string, len(absent))
	for i, f := range absent {
		names[i] = fieldName(f)
	}

	rec := RunRecord{
		Kind:        KindRun,
		ID:          run.ID,
		Name:        run.Name,
		Description: run.Description,
		Start:       millis(run.Start),
		Tool:        run.Tool,
		ToolVersion: run.ToolVersion,
		Absent:      names,
	}
	if len(run.Warnings) > 0 {
		rec.Warnings = make([]Warning, len(run.Warnings))
		for i, w := range run.Warnings {
			rec.Warnings[i] = Warning{Version: w.Version, Reason: w.Reason}
		}
	}
	if len(run.Assertions) > 0 {
		rec.Assertions = make([]string, len(run.Assertions))
		for i, a := range run.Assertions {
			rec.Assertions[i] = payload(a)
		}
	}
	return rec
}

// requestFrom converts one sample. Outcome is what the source recorded and is
// never derived from whether a failure is present; Failure is copied exactly
// when the source set it. The record's optional fields point into sc.
func requestFrom(s model.Sample, sc *scratch) RequestRecord {
	rec := RequestRecord{
		Kind:          KindRequest,
		Groups:        groups(s.Groups),
		Name:          s.Name,
		Start:         millis(s.Start),
		Duration:      keepDuration(s.Duration, &sc.duration),
		Outcome:       s.Outcome.String(),
		Scenario:      keep(s.Scenario, &sc.scenario),
		ResponseCode:  keep(s.ResponseCode, &sc.responseCode),
		BytesSent:     keep(s.BytesSent, &sc.bytesSent),
		BytesReceived: keep(s.BytesReceived, &sc.bytesReceived),
	}
	if f, ok := s.Failure.Get(); ok {
		sc.failure = Failure{Type: f.Type, Message: f.Message}
		rec.Failure = &sc.failure
	}
	return rec
}

// groupFrom converts one group traversal; its optional fields point into sc.
func groupFrom(g model.GroupSample, sc *scratch) GroupRecord {
	return GroupRecord{
		Kind:              KindGroup,
		Groups:            groups(g.Groups),
		Start:             millis(g.Start),
		Duration:          keepDuration(g.Duration, &sc.duration),
		CumulatedDuration: keepDuration(g.CumulatedDuration, &sc.cumulated),
		Outcome:           g.Outcome.String(),
	}
}

func userFrom(u model.UserEvent) UserRecord {
	return UserRecord{
		Kind:     KindUser,
		Scenario: u.Scenario,
		Event:    u.Kind.String(),
		At:       millis(u.At),
	}
}

func errorFrom(e model.RunError) ErrorRecord {
	return ErrorRecord{
		Kind:    KindError,
		Message: e.Message,
		At:      millis(e.At),
	}
}

func assertionFrom(p string) AssertionRecord {
	return AssertionRecord{Kind: KindAssertion, Payload: payload(p)}
}

// millis renders an instant as epoch milliseconds, or 0 for the zero Time,
// which parsec uses for an instant the source could not resolve and which
// omitzero then drops. No recorded instant is before 1970, so 0 is never a
// real value.
func millis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

// keep stores the value the source recorded in slot and returns slot, or
// returns nil when the source recorded nothing, so that a recorded zero and
// an absent value never look alike.
func keep[T comparable](o model.Opt[T], slot *T) *T {
	v, ok := o.Get()
	if !ok {
		return nil
	}
	*slot = v
	return slot
}

// keepDuration is keep for a duration, stored in milliseconds.
func keepDuration(o model.Opt[time.Duration], slot *int64) *int64 {
	d, ok := o.Get()
	if !ok {
		return nil
	}
	*slot = d.Milliseconds()
	return slot
}

// groups substitutes the shared empty path for a nil one so that the key is
// always present. The slice is parsec's, reused between calls to Next; the
// record is encoded before the next call, so it is never aliased.
func groups(g []string) []string {
	if len(g) == 0 {
		return noGroups
	}
	return g
}

// payload encodes an opaque assertion payload. Binary logs carry raw bytes
// that are not valid UTF-8, which encoding/json would replace with U+FFFD;
// base64 keeps them intact. Text logs already carry base64 text, and one
// rule for both formats is worth the double encoding.
func payload(p string) string {
	return base64.StdEncoding.EncodeToString([]byte(p))
}
