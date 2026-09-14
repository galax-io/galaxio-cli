# Implementation Plan: Report a Gatling Run as Records

**Branch**: `110-report-records` | **Date**: 2026-09-15 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/004-report-records/spec.md`
(milestone [`v0.13.0 Report dump`](https://github.com/galax-io/galaxio-cli/milestone/3),
issue [#50](https://github.com/galax-io/galaxio-cli/issues/50))

## Summary

Make `galaxio report <tool> [PATH]` an operational command on the namespace milestone
v0.12.0 reserved; `gatling` is the only tool this milestone accepts. It locates a Gatling run (an explicit run directory or `simulation.log`, a
results root, or the default `target/gatling`), opens the log whatever Gatling wrote it,
and streams one record per line to standard output: a run header, then requests, group
traversals, virtual-user events and run-level errors in log order. With no `-o` it writes JSON
Lines; `-o` takes a comma-separated list of report formats — `stats`, `global_stats`
(Gatling's legacy files, v0.14.0/v0.15.0) and `yml` (OpenNFR-style YAML, postponed) — every
one reserved and rejected until built. There is no `-o json` and no `-o text`. Run discovery, format
detection, the version gate and the canonical records come from
`github.com/galax-io/parsec v0.1.0`, the first new module since the constitution was
ratified and the plan's ask-first decision. This repository adds `internal/report/`, which
converts parsec's stream into the documented record schema and encodes it without
retaining records, and computes nothing. Gatling's legacy `stats.json`/`global_stats.json` and
the OpenNFR-style YAML report are reserved `-o` names, deliberately out of scope.

## Technical Context

**Language/Version**: Go 1.27.1 (`go.mod` `go 1.27.1`; CI uses `go-version-file`).
`omitzero` and `testing.T.Chdir` (both Go 1.24+) are relied on.

**Primary Dependencies**: `github.com/spf13/cobra v1.10.2` (existing) for the command tree.
**New, ask-first**: `github.com/galax-io/parsec v0.1.0` — MIT, `go 1.25`, zero transitive
modules (`go list -m all` in a probe module lists only parsec), API frozen at v0.1.0. It
supplies `gatling/run.Find`, `gatling/simlog.NewRunReader`, `simlog.Supported`, the
`model` types and the `gatling` error types. Rationale and alternatives are in
[research.md](research.md) §1. **Approved by the maintainer on 2026-09-15**; the module is
added by the first implementation task together with the code that imports it.

**Storage**: N/A. The command reads a `simulation.log` from the filesystem and writes to
standard output; it creates no file.

**Testing**: Standard-library `testing`, table-driven. Golden JSON Lines output per corpus
version under `internal/report/testdata/`; command-level tests through `runCLI`;
one integration test behind the `integration` tag driving the built binary through a real
pipe; a `testing.B` over a synthetic multi-million-record text log for the memory goal.
Gates: `gofmt`, `go vet`, `go test -race`, coverage ≥ 80%, `go mod tidy` diff.

**Target Platform**: The existing static, cross-platform CLI (macOS, Linux, Windows on
amd64/arm64) and the distroless image. parsec is pure Go, so `CGO_ENABLED=0` still holds.

**Project Type**: Go CLI; one new command file, one new internal package, versioned spec
artifacts.

**Performance Goals**: The header is flushed to the reader before the first item is read
from the log, so the first line reaches a pipe well inside the one second SC-003 asks for
whatever the log's size; records follow in 64 KiB chunks. Throughput is measured, not gated: the benchmark records records/s and MB/s on the
synthetic log and the figure goes into `quickstart.md`.

**Constraints**: **Peak memory goal (Principle II): heap in use stays under 32 MiB for a
log of any size, and the writer allocates at most 4 objects per record**, measured by the
benchmark and an `AllocsPerRun` test. No statistic is computed (Principle II). Exit codes
0/1/2 via `RuntimeError`/`UsageError`. Output is deterministic. Stdout carries only
records; every diagnostic goes to stderr. Licence: GPL-2.0-only consuming MIT parsec, per
milestone v0.12.0.

**Scale/Scope**: Five corpus versions (3.11.5, 3.12.0 text; 3.13.1, 3.14.9, 3.15.1 binary)
of 36 or 102 requests each for golden coverage; real logs of gigabytes for the memory goal.
One command made operational, one output encoding, ~5 source files in `internal/report/`, README
section, spec artifacts.

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

| # | Gate (constitution principle) | Status |
|---|---|---|
| I | Command Contract — every new command is a thin cobra wrapper over `runX(ctx, opts)`; offers `-o text\|json` with a documented JSON structure; exits 0/1/2 via `UsageError`/`RuntimeError`; honours `--verbose`/`--quiet`/`--no-color`; experimental commands sit behind `internal/featureflags`. | **FAIL (justified)** on one clause: the command offers neither `-o text` nor `-o json` — the record stream is the unnamed default and `-o` carries only reserved report-format names — by maintainer decision on 2026-09-15 (spec FR-013, Assumptions); see Complexity Tracking. Every other clause passes: `report.go` is a cobra wrapper over `runReport(ctx, opts) (reportOutput, error)`; because the command streams, `opts` carries the stdout/stderr writers and the returned output is the run location and counts (the seam a test needs). The JSON structure is [contracts/records.md](contracts/records.md) and every line ends with one newline. Usage failures — an unsupported tool, a third argument, any `-o` value while all are reserved — → `UsageError` (exit 2); parsec and I/O errors → `RuntimeError` (exit 1) naming the path, directory or version. `--quiet` silences the run-chosen diagnostic; `--verbose` adds a count summary on stderr. Not gated: the command is the milestone's deliverable, not an experiment (research.md §6). |
| II | Report Arithmetic Lives Here — statistics computed in `internal/report/` over `parsec` primitives; success and failure accumulated separately; one pass, bounded memory with the goal stated; absence reported as absent; source detected by content. | PASS — nothing is computed: `internal/report/` converts and encodes; the only numbers are the per-kind counts printed under `--verbose`, kept as separate success/failure-agnostic tallies and never a statistic. One pass over `RunReader.Next`, each item encoded before the next is read; peak-memory goal stated above and measured. Absent values are omitted from records and the header lists what the source never provides (`absent`). Format detection is parsec's, by leading bytes; no flag overrides it (none is needed: `simlog` reads both formats). |
| III | Tests Land With The Change — stdlib `testing`, table-driven, golden files under `testdata/`; command tests through `runCLI`; integration tests behind the tag on real resources; race on; coverage ≥ 80%; regression tests; test tasks never optional. | PASS — golden JSON Lines per corpus version; conversion unit tests for absence, failure, zero time, assertion encoding; error-mapping tests for not-found, unreadable, not-a-log, below-range (synthetic text 3.10.0), 3.13.0 (3.13.1 log with its version bytes patched), newer-than-range warning (synthetic 3.99.0), truncation (corpus log cut mid-record), syntax error, write failure; command tests through `runCLI` for every acceptance scenario and exit code, including results-root selection by `lastRun.txt` and by newest and the no-path default via `t.Chdir`; integration test drives the built binary through `head -n 1`; benchmark plus `AllocsPerRun` test for the memory goal. Race detector on in CI. |
| IV | Minimal, Explicit Dependencies — no new module unless named here with the reason the standard library or an existing dependency is insufficient, recorded in `research.md`, licence-compatible with GPL-2.0-only, and asked for first. | PASS — one module, `github.com/galax-io/parsec v0.1.0`: MIT (compatible, milestone v0.12.0), zero transitive modules, pure Go. The standard library has no Gatling codec; the binary format has a string cache and JVM-compact strings that took parsec five milestones and a recorded corpus to get right, and Principle II requires this CLI to build on parsec's definitions rather than re-derive them. Recorded in research.md §1. **Approved by the maintainer on 2026-09-15 in the planning conversation.** The `go get` lands in the first implementation task with the code that imports it (an unused requirement would not survive `go mod tidy`); `go mod tidy` leaves no diff; `govulncheck` runs before the milestone tag. |
| V | Published Surfaces — any change to a command, flag, default, exit code, `-o json` structure, manifest/registry schema or generated output is listed; breaking ones approved before implementation and committed with `!`; README updated in the same PR; deprecations keep working one minor release. | PASS — additive: `galaxio report` with no arguments still prints help and exits 0 (spec FR-024), so the v0.12.0 surface is unchanged; `galaxio report <x>` still exits 2, now as an unsupported tool. New: the `<tool>` and `[PATH]` arguments, the local `-o/--output` flag with its reserved names (`stats`, `global_stats`, `yml`, all rejected), the documented JSON Lines schema (a published surface from this release) and the new exit paths. Nothing deprecated, nothing breaking. README documents the command and schema in the same PR. |
| VI | Idiomatic, Simple Go — gofmt/vet clean; errors as values wrapped into `UsageError`/`RuntimeError` at the boundary; no panic control flow; no dead or duplicated code; no refactor outside this issue's scope. | PASS — errors from parsec are inspected with `errors.As` and wrapped once at the command boundary; with one encoding there is no encoder interface — `Write` writes JSON Lines to an `io.Writer` directly; no options struct, registry or plugin abstraction; no change to existing commands beyond the parent's help text and its test. Required skills (`golang-cli`, `golang-spf13-cobra`, `golang-error-handling`, `golang-testing`, `golang-naming`, `golang-documentation`) are read before the corresponding code is written; consulted ones are listed in research.md §14. |

**Post-design re-check**: gates II–VI PASS; gate I keeps its single justified FAIL (no
`-o text`). The data model keeps absent-versus-zero honest with pointer fields into
per-write scratch storage, at no allocation per record (Principle II and VI); the contracts
add no flag beyond `-o`; the dependency decision is unchanged and approved.

## Project Structure

### Documentation (this feature)

```text
specs/004-report-records/
├── plan.md              # This file
├── research.md          # Phase 0: decisions, evidence, alternatives
├── data-model.md        # Phase 1: run location, header, record kinds, encodings
├── quickstart.md        # Phase 1: how to prove the feature end to end
├── contracts/
│   ├── cli.md           # Command, arguments, flags, diagnostics, exit codes
│   └── records.md       # The JSON Lines schema and the reserved report formats
├── checklists/requirements.md
└── tasks.md             # Phase 2 output (/speckit-tasks) — not created here
```

### Source Code (repository root)

```text
cmd/galaxio/
├── report.go                 # rewritten: operational command, reportOptions, runReport, error mapping
├── report_test.go            # command-level tests through runCLI and runReport
└── root_test.go              # TestReportCommand / TestReportCommandRejectsArguments updated

internal/report/
├── doc.go                    # package purpose: convert and encode; compute nothing
├── record.go                 # Record kinds, per-kind structs, Opt[T], field-name table
├── convert.go                # headerFrom(model.Run), recordFrom(model.Item)
├── write.go                   # Write(ctx, RunReader, io.Writer) (Summary, error): JSON Lines via bufio + encoding/json
├── source.go                 # Locate(path) and Open(location): run.Find + simlog.NewRunReader
├── *_test.go                 # golden, conversion, error-mapping, alloc and benchmark tests
└── testdata/
    ├── corpus/gatling/
    │   ├── PROVENANCE.md                     # copied from parsec v0.1.0 (MIT), never edited
    │   ├── 3.11.5/simulation.log             # text, 36 requests
    │   ├── 3.12.0/simulation.log             # text, 36 requests
    │   ├── 3.13.1/simulation.log             # binary, 102 requests
    │   ├── 3.14.9/simulation.log             # binary, 102 requests
    │   ├── 3.15.1/simulation.log             # binary, 102 requests
    │   └── lastrun/results/                  # three runs + lastRun.txt for selection tests
    └── golden/
        └── <version>.jsonl                   # expected output per corpus version

README.md                     # "Reporting Ecosystem": galaxio report, flags, exit codes, schema
go.mod / go.sum               # + github.com/galax-io/parsec v0.1.0 (after approval)
```

**Structure Decision**: `internal/report/` is a single flat package, the home the
constitution already names for report code; the record writer is its first resident and later
milestones (summary, `stats.json`) add files or a subpackage beside it. The corpus lives
once, next to the package that converts it; command tests in `cmd/galaxio` reach it by the
relative path `../../internal/report/testdata/corpus` rather than duplicating frozen
recordings. Command wiring stays in `cmd/galaxio/report.go`, already registered on root;
no subcommand is added (research.md §12).

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Principle I: `report` offers neither `-o text` nor `-o json` | Maintainer decision, 2026-09-15: `-o` on `report` selects a *report format* (`stats`, `global_stats`, `yml` — all reserved now), not an encoding of one output; the record stream is the unnamed default with one encoding shared with the sidecar. A `text` rendering would be a second published surface (Principle V) with no consumer, and a `json` name would collide with the report-format meaning of `-o`. | Complying by adding `-o text\|json` was rejected by the maintainer, repeatedly. A follow-up constitution amendment should say that on `report` commands `-o` names a report format and the default output is the machine-readable one, so later report commands do not each need this row. |

The `scratch` struct in `internal/report/` is not a gate violation: optional fields are
pointers into it, which is how `encoding/json` tells "duration 0 ms" from "no duration"
without an allocation per record. The generic `Opt[T]` first planned here was measured and
replaced (research.md §4).
