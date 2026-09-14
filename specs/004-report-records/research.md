# Research: Report a Gatling Run as Records

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Date**: 2026-09-15

Every decision below was checked against the published `github.com/galax-io/parsec v0.1.0`
API (`go doc` in a throwaway module) and its recorded corpus, run through
`simlog.NewRunReader` on 2026-09-15. Quoted counts and error texts are what that run
printed, not what documentation promised.

## 1. Dependency: `github.com/galax-io/parsec v0.1.0` (ask-first)

**Decision**: Add `github.com/galax-io/parsec v0.1.0` as a direct dependency, pinned to
that tag. **Approved by the maintainer on 2026-09-15** ("да, добавляй parsec"), after the
question of why it is needed in this milestone was answered: every user story starts by
reading `simulation.log`, and parsec is the only reader; without it the milestone is a
command skeleton with no user-visible result. It provides `gatling/run.Find` (run discovery),
`gatling/simlog.NewRunReader` (format detection by leading bytes, version gate, canonical
`model.Item` stream), `simlog.Supported()` (the version range, read at run time), and the
error types the command maps to exit codes.

**Rationale**:
- The standard library has no Gatling codec. The text log is a tab-separated grammar with
  its own escaping rules; the binary log (3.13+) is an undocumented stream with a string
  cache, JVM-compact Latin-1/UTF-16 strings and big-endian framing. parsec spent five
  milestones and a recorded corpus of five versions getting both right; re-implementing
  them here would duplicate that work and fork the definitions Principle II says this
  repository must not re-derive.
- Licence: MIT, classified GPL-compatible by the FSF (milestone v0.12.0, issue #48).
- Footprint: `go 1.25`, zero transitive modules (`go list -m all` shows only parsec), pure
  Go, so `CGO_ENABLED=0` and the distroless image are unaffected.
- Stability: from v0.1.0 the exported surface of `model`, `gatling`, `gatling/text`,
  `gatling/binary`, `gatling/simlog` and `gatling/run` is a contract; a breaking change
  costs a MINOR release and a deprecation window. The `RunReader` method set is frozen.
- Hygiene: `go mod tidy` will leave no diff; `go mod verify` and `govulncheck ./...` run
  before the milestone tag (parsec has no dependencies, so the scan covers the standard
  library and this module only).

**Alternatives considered**:
- Hand-written text-only parser in this repository: rejected — would leave every run from
  3.13 on unreadable, which is the very gap the issue names, and would re-derive
  definitions parsec owns.
- Vendoring parsec (`go mod vendor`): rejected — the module proxy and checksum database
  already give reproducibility, and `main` carries no `vendor/` today.
- Copying parsec's source into `internal/`: rejected — forks the definitions and loses the
  corpus-bound version gate updates.

**Skills consulted**: `golang-dependency-management` (ask before `go get`; commit `go.sum`;
tidy; `govulncheck` before release). `golang-pkg-go-dev` is the check the implementer runs
on the module page before pinning (importers, versions, vulnerabilities) and records the
result in the task's commit body.

## 2. Package layout: a flat `internal/report/`

**Decision**: One package `internal/report/` holding the record schema, conversion,
encoding and source resolution. No subpackage yet.

**Rationale**: The constitution names `internal/report/` as the home of report code. With
one command it has six files; a subpackage per command would be speculative structure
(Principle VI). When summary arithmetic arrives (v0.14.0) it can sit beside these files or
split into `internal/report/stats` if the package grows past reading comfortably.

**Alternatives considered**: `internal/report/records` + `internal/report/source` — rejected
as premature; `cmd/galaxio` only — rejected because conversion and encoding must be
testable without a cobra command and will be reused by later commands.

## 3. Record schema: field names and the absent list

**Decision**: The JSON Lines schema is this repository's own, documented in
[contracts/records.md](contracts/records.md), with camelCase keys and a `kind`
discriminator on every line (`run`, `request`, `group`, `user`, `error`, `assertion`). The
header's `absent` list uses stable dotted identifiers mapped from `model.Field` by a table
in `record.go` (for example `sample.responseCode`), not parsec's `Field.String()` text,
which renders with spaces ("sample response code") and is written for people. A `Field`
this repository does not know yet (added by a later parsec MINOR) falls back to
`Field.String()` so nothing is silently dropped.

**Evidence**: On every corpus run parsec reports the same eleven absent fields for Gatling:
sample scenario, response code, bytes sent, bytes received, failure type, user identity,
connect/dns/tls timing, requirements, interval series. The header therefore carries an
`absent` array of eleven identifiers for a Gatling run.

**Alternatives considered**: OpenNFR `loadtest.*` attribute names — postponed by the
maintainer on 2026-09-15 together with the YAML encoding; the table in `record.go` is the
one place a later rename touches. Emitting parsec's `Field.String()` verbatim — rejected:
a documented schema should not change when a library reworded a message.

## 4. Absent versus zero: pointer fields into per-write scratch storage

**Decision**: Optional fields (`duration`, `cumulatedDuration`, `scenario`, `responseCode`,
`bytesSent`, `bytesReceived`, `failure`) are pointers tagged `omitempty`, and the
conversion points them into a `scratch` struct the writer allocates once per run. A nil
pointer is omitted; a pointer to a recorded `0` is written as `0`. Instants (`start`, `at`)
are `int64` epoch milliseconds tagged `omitzero`: parsec guarantees a recorded instant is
never the zero `time.Time`, and no recorded instant is before 1970, so `0` can only mean
"could not be resolved" and is correctly omitted. Optional strings that are never
"recorded as empty" (`description`, `failure.type`) use `omitempty`. `groups` is always
present on request and group records — a request outside any group has a known-empty
path, rendered `[]` from a shared empty slice.

**Rationale**: FR-012 forbids rendering absence as zero. The first design was a generic
`Opt[T]` with `IsZero` and `MarshalJSON` methods; measured on the 64 MiB replay it cost
9.8 allocations per record, all of them `encoding/json` boxing the value to call those
methods through reflection (profile: `reflect.packEface` under the struct arshaler and
`bytes.Clone` of every `MarshalJSON` result). Pointers into reused storage are followed by
the encoder without any call and any allocation: the same replay then allocates 475 objects
in total for 940 000 records, and the 2 GiB replay 87 objects for 30 million. A record's
pointers are valid until the next conversion, the same rule parsec applies to `Groups`, and
the writer encodes each record before converting the next.

**Alternatives considered**: `Opt[T]` without `IsZero` (reflect's own zero check has the
same semantics) — halved the cost but each set field still paid for `MarshalJSON` plus the
encoder's clone of its result, 3 allocations per set field; a pointer-receiver
`MarshalJSON` with `strconv` — no cheaper, and a struct marshalled by value silently
renders `{}` for the field; a hand-written JSON appender — zero allocations but a second
encoder to keep correct; `encoding/json/v2` with `MarshalJSONTo` — not importable on this
toolchain without an experiment flag.

## 5. Assertion payloads: base64 of the bytes parsec delivered

**Decision**: `assertions` (header) and the `assertion` record's `payload` carry
`base64` of the payload bytes exactly as parsec's `Run.Assertions` / `Item.Assertion`
hold them. One rule for both log formats.

**Rationale**: Binary logs carry raw bytes (the 3.15.1 corpus payloads include `\x00` and
`\x80`); `encoding/json` would replace invalid UTF-8 with U+FFFD, which is lossy. Text logs
carry the payload as base64 text already, so this rule double-encodes them — accepted for
uniformity, since no consumer decodes the payload before milestone v0.17.0 and a single
documented rule is easier to hold than a per-format one.

**Evidence**: 3.11.5/3.12.0 runs carry 3 assertions on the header; 3.13.1–3.15.1 carry 10.
`ItemAssertion` never appears for Gatling (both formats write payloads before the events),
but the record kind exists so a future source that interleaves them loses nothing
(spec FR-008, amended in this plan's commit).

## 6. Not gated behind `internal/featureflags`

**Decision**: `galaxio report` ships ungated.

**Rationale**: The constitution gates *experimental* commands. The command is the deliverable of
milestone v0.13.0, the first result a user can see from the initiative, with a documented
schema that Principle V makes a published surface. Gating it would contradict releasing
it. `featureflags.commandFlags` stays empty.

## 7. Run selection, default root and diagnostics

**Decision**: An empty path becomes `run.DefaultResultsRoot` (`target/gatling`) resolved
against the working directory; otherwise the argument goes to `run.Find` unchanged, which
treats a path holding `simulation.log` (or the log itself) as the run and anything else as
a results root searched by `lastRun.txt` then by newest. The command prints
`report: reading <dir> (found by <rule>)` on stderr unless `--quiet`.

**Evidence**: `run.Find(<lastrun corpus root>)` returned the run named by `lastRun.txt`
with `Found=lastRun.txt`; a missing root returned `*run.NotFoundError` reading
`gatling: no Gatling run under <dir>`; an unreadable root wraps `*fs.PathError`. parsec's
own documentation asks callers that report which run they read to report the rule too,
because "newest" is a guess from modification times.

**Alternatives considered**: Also searching Gradle's `build/reports/gatling` when the
default is empty — rejected (spec Assumptions): a guessed root can report a run nobody asked
about. `--results-root` flag — rejected: the positional argument already accepts a root.

## 8. Failure semantics and exit codes

**Decision** (all wrap into `RuntimeError`, exit 1, unless noted):

| Condition | parsec signal | Behaviour |
|---|---|---|
| No run under the path | `*run.NotFoundError` | message names the directory searched |
| Path unreadable / missing | `*fs.PathError` | message names the path and the OS reason; never "no run" |
| Not a Gatling log | `*gatling.FormatError` | `not a Gatling simulation.log: <path>` plus parsec's detail |
| Version below range, or 3.13.0 | `*gatling.VersionError` | parsec's text names the version and range, e.g. `version 3.10.0 is below the supported range 3.11.5 through 3.12.0` |
| Version above range | `model.Run.Warnings` non-empty | records emitted, exit 0, warning in header and on stderr |
| Log cut short | `*gatling.TruncationError` from `Next` | every record already read is emitted and flushed, then exit 1: `log cut short at byte 1991: 9 trailing bytes could not be decoded; the 62 records emitted are what the run recorded` |
| Damaged log | `*gatling.SyntaxError` from `Next` | stop, exit 1, message adds `the N records already emitted do not form a complete run` |
| Write to stdout fails | `error` from the encoder | stop at the first failure, one `RuntimeError`, no per-record diagnostics |
| Reader closed the pipe (fd 1) | `SIGPIPE` | the Go runtime ends the process quietly by signal before any diagnostic; documented, and the integration test proves it |
| tool other than `gatling`, any `-o` value (reserved `stats`/`global_stats`/`yml` or unknown), a third argument, unknown flag | — | `UsageError`, exit 2, empty stdout |

**Evidence**: the messages in the table are what the probe printed for the corpus. The
truncation probe (3.15.1 log cut at 2000 bytes) delivered 62 items before the error, and
`errors.As` matched `*gatling.TruncationError`.

## 9. Streaming, memory goal and how it is measured

**Decision**: `Write` loops `Next`, converts each `model.Item` into the record struct for
its kind, and encodes it before calling `Next` again; parsec's reused
`Groups` slice is therefore never aliased across calls and needs no copy. Stdout is
wrapped in a 64 KiB `bufio.Writer` flushed once right after the header — so a reader sees
the run's identity and warnings before the log is read, at the cost of one write per run —
then at the end and before any error return. The
JSON encoder is one `json.Encoder` with `SetEscapeHTML(false)`. **Goal: heap in use under
32 MiB regardless of log size; ≤ 4 allocations per record.**

**Measurement**: `BenchmarkWrite` feeds a synthetic text log built by replaying the body
lines of the 3.12.0 corpus N million times through an `io.Reader` (no file on disk) and
reports records/s, MB/s and `ReportAllocs`; `TestWriteAllocations` uses
`testing.AllocsPerRun` over 10 000 records and asserts the per-record ceiling; the
quickstart records the observed figures. First-record latency is structural (header written
before the second `Next`) and additionally proven by the integration test that pipes the
binary into `head -n 1`.

**Skills**: `golang-benchmark` before writing the benchmark; `golang-performance` only if the
profile shows a hot spot; `golang-design-patterns` for the `io.Reader`/encoder pipeline;
`golang-safety`/`golang-security` because every log is untrusted input (parsec bounds line
and string length; this package adds no unbounded buffer); `golang-context` for honouring
`ctx.Err()` in the loop; `golang-concurrency` is *not* needed — one goroutine reads, and
parsec's readers must not be shared.

## 10. Testing the version gate without a 3.13.0 recording

**Decision**: The below-range case uses a synthetic text header claiming `3.10.0`; the
3.13.0 refusal patches the six version bytes at offset 5 of the 3.13.1 corpus log
(`00 | 00 00 00 06 | "3.13.1"` → `"3.13.0"`, same length, done in memory by the test); the
newer-than-range warning uses a text header claiming `3.99.0` (decoded with a warning by the
probe) and the same byte patch to `3.99.9` for binary.

**Rationale**: parsec's gate reads the version from the RUN record before anything else, so
patching only that string exercises exactly the gate; no fabricated binary layout is
needed and no recording is edited on disk.

## 11. One unnamed output; `-o` reserved for report formats

**Decision**: With no `-o` the command writes JSON Lines. `-o` parses a comma-separated
list of report-format names; the known names are `stats` and `global_stats` (Gatling's
legacy `js/stats.json` and `js/global_stats.json`, computed by milestones v0.14.0 and
v0.15.0) and `yml` (the postponed OpenNFR-style YAML report). In this milestone every known
name is rejected with a usage error naming the milestone that delivers it, and an unknown
name is rejected listing the known ones. There is no `-o json` and no `-o text`. With a
single encoding there is no encoder interface: `Write` writes JSON Lines straight to an
`io.Writer`.

**Rationale**: Maintainer decision on 2026-09-15, repeated: on `report`, `-o` names a report
format, not an encoding of the record stream; the stream is the default and has no flag
name. This is the one Principle I clause the feature does not meet; the plan's Complexity
Tracking row records the justification and proposes a constitution amendment.

**Alternatives considered**: `-o json` naming the stream — removed at the maintainer's
instruction; `-o text` — rejected; treating a reserved name as accepted-but-unimplemented
(exit 1) — rejected: an unimplemented value is a usage error, and a script that passes
`stats` today must fail loudly rather than receive the record stream; parsing `-o` as a
list only when the first format lands — rejected: the comma form is part of the surface the
maintainer specified, so its grammar is fixed and tested now.

## 12. No subcommand: `report` itself is the operational command

**Decision**: There is no `dump` subcommand. `cmd/galaxio/report.go`, today a help-only
namespace parent, becomes the operational command `galaxio report <tool> [PATH] [-o …]`:
`Args: cobra.MaximumNArgs(2)`, `RunE` → help when no arguments, otherwise `runReport`;
local `-o`. `galaxio report` with no arguments still prints help and exits 0.

**Rationale**: Maintainer decision on 2026-09-15, repeated: the issue's `report dump` wording
was a direction to evaluate, not the name. One command, `galaxio report <tool> [PATH]`,
with `-o` reserved for report formats, is the shape the maintainer wants for the whole
`report` surface, so the parent carries it.

**Consequences**: `TestReportCommand` (help on no arguments) keeps passing;
`TestReportCommandRejectsArguments` (`report unexpected` → exit 2) keeps its exit code but
its expected message becomes "unsupported tool". The v0.12.0 README sentence "Operational
report subcommands are introduced separately" is replaced. No published behaviour changes
(plan gate V).

## 13. Which tool produced the run

**Decision**: The user names the tool as the first positional argument (`gatling`); it is
never inferred from a file name. Within the named tool, detection is by content:
`run.Find` treats a directory holding `simulation.log` as the run and `simlog.NewRunReader`
identifies text versus binary from the leading bytes. The header's `tool` is
`model.Run.Tool`, stated by the source. An unknown tool name is a usage error listing the
accepted tools; a path that is not a Gatling run is a runtime error naming the path.

**Rationale**: Maintainer decision on 2026-09-15 (the shape is `galaxio report <tool>`).
It is also the "explicit override takes precedence" half of Principle II, made mandatory
rather than optional, and it means a second tool (JMeter, v0.19.0) is one more accepted
name and one more `case`, with no signature-ordering question to settle.

**Implementation note**: the tool name is validated in the cobra layer against a fixed
list (`gatling`); `runReport` receives it and selects the parsec path. No registry or
plugin table (Principle VI) until a second tool exists.

## 14. Skills classification for this feature

Required before code (constitution table): `golang-cli`, `golang-spf13-cobra`
(`report.go`), `golang-error-handling` (error mapping), `golang-testing` (all tests),
`golang-naming` and `golang-documentation` (exported identifiers in `internal/report/`,
README). Consulted: `golang-dependency-management` (read for §1), `golang-pkg-go-dev`,
`golang-project-layout` (§2), `golang-benchmark` (§9), `golang-design-patterns` (§9),
`golang-safety`, `golang-security`, `golang-context`. No skill contradicted the
constitution; nothing to record.
