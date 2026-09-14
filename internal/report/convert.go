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
// when the source set it.
func requestFrom(s model.Sample) RequestRecord {
	rec := RequestRecord{
		Kind:          KindRequest,
		Groups:        groups(s.Groups),
		Name:          s.Name,
		Start:         millis(s.Start),
		Duration:      durationMillis(s.Duration),
		Outcome:       s.Outcome.String(),
		Scenario:      opt(s.Scenario),
		ResponseCode:  opt(s.ResponseCode),
		BytesSent:     opt(s.BytesSent),
		BytesReceived: opt(s.BytesReceived),
	}
	if f, ok := s.Failure.Get(); ok {
		rec.Failure = Some(Failure{Type: f.Type, Message: f.Message})
	}
	return rec
}

func groupFrom(g model.GroupSample) GroupRecord {
	return GroupRecord{
		Kind:              KindGroup,
		Groups:            groups(g.Groups),
		Start:             millis(g.Start),
		Duration:          durationMillis(g.Duration),
		CumulatedDuration: durationMillis(g.CumulatedDuration),
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

// durationMillis keeps the set/unset distinction: an unset duration stays
// unset, a recorded 0 ms becomes Some(0).
func durationMillis(d model.Opt[time.Duration]) Opt[int64] {
	v, ok := d.Get()
	if !ok {
		return Opt[int64]{}
	}
	return Some(v.Milliseconds())
}

func opt[T comparable](o model.Opt[T]) Opt[T] {
	v, ok := o.Get()
	if !ok {
		return Opt[T]{}
	}
	return Some(v)
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
