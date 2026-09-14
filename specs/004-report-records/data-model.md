# Data Model: Report a Gatling Run as Records

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Contracts**:
[cli.md](contracts/cli.md), [records.md](contracts/records.md)

Nothing here is persisted. The model is the shape of what the command reads (from
parsec) and what it writes (the record stream). Field names on the right are the JSON keys
fixed in [contracts/records.md](contracts/records.md).

## Run location

Where the run's artefacts sit and how they were chosen. Produced by
`gatling/run.Find`; consumed by `Open` and by the stderr diagnostic. Discarded once the
log is open.

| Attribute | Meaning | Source |
|---|---|---|
| `Dir` | the run directory, cleaned; absolute only if the argument was | `run.Location.Dir` |
| `Log` | `Dir/simulation.log` | `run.Location.Log` |
| `Found` | rule that chose it: `path`, `lastRun.txt`, `newest` | `run.Location.Found` |

Resolution: empty argument → `target/gatling`; a path holding `simulation.log` (or the log
itself) → `path`; otherwise a results root searched by `lastRun.txt` (if present and its
run still exists) then by newest modification, ties broken by the run id's timestamp.

Validation: a root with no run is `*run.NotFoundError` (names `Dir`); an unreadable root
is `*fs.PathError` and is never reported as "no run".

## Header record (`kind: "run"`)

The first line of every output. Everything about the run that does not grow with its
length. Built from `model.Run` by `headerFrom`.

| Key | Type | Presence | Source and rule |
|---|---|---|---|
| `kind` | string | always `"run"` | — |
| `id` | string | always | `Run.ID`; Gatling records the simulation id, so it repeats across runs of one simulation — key stored results by `id` + `start` |
| `name` | string | always | `Run.Name` (simulation class) |
| `description` | string | when non-empty | `Run.Description`; Gatling's lone space is already empty in parsec |
| `start` | int64 epoch ms | when resolved | `Run.Start.UnixMilli()`; omitted for the zero time |
| `tool` | string | always | `Run.Tool` (`"gatling"`) |
| `toolVersion` | string | always | `Run.ToolVersion` as the log wrote it |
| `absent` | []string | always (may be `[]`) | `Run.Capabilities.Absent()` mapped by the field table below |
| `warnings` | []Warning | when non-empty | `Run.Warnings` → `{version, reason}` |
| `assertions` | []string (base64) | when non-empty | `Run.Assertions`, each payload base64-encoded verbatim |

**Field table** (`model.Field` → `absent` identifier): `FieldSampleDuration` →
`sample.duration`; `FieldSampleScenario` → `sample.scenario`; `FieldSampleResponseCode` →
`sample.responseCode`; `FieldSampleBytesSent` → `sample.bytesSent`;
`FieldSampleBytesReceived` → `sample.bytesReceived`; `FieldSampleFailureType` →
`sample.failureType`; `FieldSampleUserIdentity` → `sample.userIdentity`;
`FieldGroupDuration` → `group.duration`; `FieldGroupCumulatedDuration` →
`group.cumulatedDuration`; `FieldGroupOutcome` → `group.outcome`; `FieldConnectTiming` →
`timing.connect`; `FieldDNSTiming` → `timing.dns`; `FieldTLSTiming` → `timing.tls`;
`FieldRequirements` → `requirements`; `FieldIntervalSeries` → `intervalSeries`. An unknown
`Field` falls back to `Field.String()`.

For every Gatling run in the corpus `absent` is the same eleven entries: `sample.scenario`,
`sample.responseCode`, `sample.bytesSent`, `sample.bytesReceived`, `sample.failureType`,
`sample.userIdentity`, `timing.connect`, `timing.dns`, `timing.tls`, `requirements`,
`intervalSeries`.

## Request record (`kind: "request"`)

One recorded operation. Built from `model.Sample` (`ItemSample`).

| Key | Type | Presence | Rule |
|---|---|---|---|
| `groups` | []string | always (`[]` outside any group) | `Sample.Groups`, outermost first; encoded before the next `Next`, so parsec's reused slice is never aliased |
| `name` | string | always | `Sample.Name` |
| `start` | int64 epoch ms | when resolved | `Sample.Start` |
| `duration` | int64 ms | when set | `Sample.Duration` (`Opt`) — a recorded `0` is written as `0` |
| `outcome` | string | always | `Sample.Outcome.String()`: `success`, `failure` (`unknown` only if a source lost it) |
| `failure` | object | iff `outcome == "failure"` | `{type?, message}` from `Sample.Failure`; `type` omitted when empty (Gatling never records one) |
| `scenario` | string | when set | `Sample.Scenario` (never for Gatling; `absent` says so) |
| `responseCode` | string | when set | `Sample.ResponseCode` (never for Gatling) |
| `bytesSent` / `bytesReceived` | int64 | when set | `Sample.BytesSent/BytesReceived` (never for Gatling) |

Invariant (Principle II): `outcome` is read from the source, never inferred from `failure`.

## Group record (`kind: "group"`)

One traversal of a group, closing. Built from `model.GroupSample` (`ItemGroup`).

| Key | Type | Presence | Rule |
|---|---|---|---|
| `groups` | []string | always | the group's own path, its own name last |
| `start` | int64 epoch ms | when resolved | `GroupSample.Start` |
| `duration` | int64 ms | when set | wall clock across the traversal, pauses included |
| `cumulatedDuration` | int64 ms | when set | sum of enclosed operation durations; not derived from `duration` |
| `outcome` | string | always | the group's own outcome, not the conjunction of its requests |

## User event record (`kind: "user"`)

A virtual user starting or ending a scenario. Built from `model.UserEvent` (`ItemUser`).

| Key | Type | Presence | Rule |
|---|---|---|---|
| `scenario` | string | always | `UserEvent.Scenario` |
| `event` | string | always | `UserEvent.Kind.String()`: `start` or `end` |
| `at` | int64 epoch ms | when resolved | `UserEvent.At` |

## Run error record (`kind: "error"`)

A failure that belongs to no sample. Built from `model.RunError` (`ItemError`).

| Key | Type | Presence | Rule |
|---|---|---|---|
| `message` | string | always | verbatim, trailing space included (Gatling writes one) |
| `at` | int64 epoch ms | when resolved | `RunError.At` |

## Assertion record (`kind: "assertion"`)

An opaque payload a source wrote among its events (`ItemAssertion`). Gatling writes all of
its payloads before the events, so for Gatling they land on the header and this record
never appears; it exists so the stream is lossless for a source that interleaves them.

| Key | Type | Presence | Rule |
|---|---|---|---|
| `payload` | string (base64) | always | `Item.Assertion`, base64 of the bytes verbatim |

## Optional fields

An optional field is a pointer (`*int64`, `*string`, `*Failure`) tagged `omitempty`: nil is
omitted, a pointer to a recorded zero is written as `0`. The conversion points each one
into a `scratch` struct the writer allocates once, so an optional field costs no allocation
per record; a record is therefore valid only until the next conversion, exactly as parsec's
reused `Groups` slice is. The measured effect and the rejected `Opt[T]` design are in
research.md §4.

## Summary (returned by `Write`, not emitted)

Counts per kind (`Requests`, `Groups`, `Users`, `Errors`, `Assertions`) and whether the
stream ended cleanly, truncated (with parsec's position and dropped-byte count) or failed.
Printed on stderr under `--verbose`; asserted by tests against the console summary
(36 requests for 3.11.5/3.12.0, 102 for 3.13.1/3.14.9/3.15.1). These are tallies of items
seen, not statistics, and are never written to stdout.

## Encoding

One schema, one encoding: **JSON Lines**, written whenever no report format is requested
(there is no `-o json`). One `encoding/json` object per line, key order as in the tables
above, HTML escaping off, exactly one `\n` per record, no preamble or trailer. `Write`
writes it to an `io.Writer` through a 64 KiB `bufio.Writer`; there is no encoder interface.
`-o` requests report formats (`stats`, `global_stats`, `yml`), all reserved for later
milestones and rejected as usage errors now.
