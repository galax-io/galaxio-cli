# Implementation Plan: Read a Finished Gatling Run

**Branch**: `110-report-records` | **Date**: 2026-09-16 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/004-report-records/spec.md`
(milestone [`v0.13.0 Read a run`](https://github.com/galax-io/galaxio-cli/milestone/3),
issue [#50](https://github.com/galax-io/galaxio-cli/issues/50))

## Summary

Make `galaxio report <tool> [PATH]` an operational command that reads a finished Gatling
run and reports what it holds. It locates the run (an explicit run directory or
`simulation.log`, a results root, or the default `target/gatling`), opens the log whatever
Gatling wrote it, walks it once, and prints the tool and version, the log format, the run's
identity and start, the span the library bounds it by, and the counts of requests, group
traversals, virtual-user events and run-level errors, requests split into successes and
failures. Run discovery, format detection, the version gate and the canonical records come
from `github.com/galax-io/parsec v0.1.0`.

This repository adds `internal/report/`, which is glue over parsec plus one counting fold.
It computes no statistic and writes no data format. The record stream of the first revision
is withdrawn (spec, Withdrawn from this milestone); `stats.json` identical to Gatling's own
is the later work of #51 and #52.

## Technical Context

**Language/Version**: Go 1.27.1 (`go.mod`; CI uses `go-version-file`). `testing.T.Chdir`
(Go 1.24+) is relied on.

**Primary Dependencies**: `github.com/spf13/cobra v1.10.2` (existing) for the command tree.
`github.com/galax-io/parsec v0.1.0` — MIT, `go 1.25`, zero transitive modules, API frozen at
v0.1.0; approved by the maintainer on 2026-09-15. It supplies `gatling/run.Find`,
`gatling/simlog.NewRunReader`, the `model` types and the `gatling` error types. Rationale in
[research.md](research.md) §1.

**Storage**: N/A. The command reads a `simulation.log` and writes to standard output; it
creates no file.

**Testing**: Standard-library `testing`, table-driven. Command-level tests through `runCLI`
against the recorded corpus; unit tests for the fold and the error mapping; integration
tests behind the `integration` tag driving the built binary — a large replayed run,
determinism of two reads, and the exit codes; a `testing.B` over a synthetic
multi-gigabyte log for throughput, and an ordinary test for the memory goal, which a
benchmark no CI job runs could not gate. Gates: `gofmt`, `go vet`, `go test -race`, coverage
≥ 80%, `go mod tidy` diff.

**Target Platform**: The existing static, cross-platform CLI and the distroless image.
parsec is pure Go, so `CGO_ENABLED=0` still holds.

**Project Type**: Go CLI; one command file, one internal package, versioned spec artifacts.

**Performance Goals**: Reading is one forward pass; nothing is retained. Throughput is
measured, not gated, and the figure goes into `quickstart.md`.

**Constraints**: **Peak memory goal: heap in use stays under 32 MiB for a log of any size**,
enforced by `TestScanMemoryDoesNotGrowWithTheLog` in the ordinary suite, which CI runs. No statistic is computed. No vocabulary is declared that
duplicates one parsec defines. Exit codes 0/1/2 via `RuntimeError`/`UsageError`. Output is
deterministic. Licence: GPL-2.0-only consuming MIT parsec, per milestone v0.12.0.

**Scale/Scope**: Five corpus versions (3.11.5, 3.12.0 text; 3.13.1, 3.14.9, 3.15.1 binary)
of 36 or 102 requests each; real logs of gigabytes for the memory goal. One command file,
three source files in `internal/report/` and one test-fixture subpackage, README section, spec artifacts.

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

| # | Gate (constitution principle) | Status |
|---|---|---|
| I | Command Contract — thin cobra wrapper over `runX(ctx, opts)`; `-o text\|json` with a documented JSON structure; exits 0/1/2 via `UsageError`/`RuntimeError`; honours the root flags; experimental commands behind `internal/featureflags`. | **FAIL (justified)** on one clause: the command offers neither `-o text` nor `-o json`. On `report`, `-o` names a *report format* (`stats`, `global_stats`, `yml`), every one reserved for a later milestone and rejected; the command's own output is a human-readable report with no machine format at all in this milestone. Maintainer decisions of 2026-09-15 and 2026-09-16; see Complexity Tracking. Every other clause passes: `report.go` is a cobra wrapper over `runReport(ctx, opts) (reportOutput, error)`; usage failures → `UsageError` (exit 2), parsec and I/O failures → `RuntimeError` (exit 1) naming the path, directory or version; `--quiet` suppresses the report, `--verbose` adds what the source cannot record. Not gated: the command is the milestone's deliverable, not an experiment. |
| II | Report Arithmetic Lives Here — statistics computed in `internal/report/` over parsec primitives; success and failure accumulated separately; one pass, bounded memory with the goal stated; absence reported as absent; source detected by content. | PASS — nothing is computed: this milestone reads and counts. The counts are tallies of records walked, never a mean, percentile, range or rate, and successes and failures are tallied apart from the outcome parsec recorded, never inferred. One pass over `RunReader.Next`, nothing retained; peak-memory goal stated above and measured. The run's span is parsec's `model.Bounds`, not re-derived here. What the source cannot record is reported from parsec's `Capabilities` as absent. Format detection is parsec's, by leading bytes. The statistics this principle governs arrive with #51, over this same pass. |
| III | Tests Land With The Change — stdlib `testing`, table-driven, golden files under `testdata/`; command tests through `runCLI`; integration tests behind the tag; race on; coverage ≥ 80%; regression tests; test tasks never optional. | PASS — corpus-driven tests for every supported version asserting the counts against each recording's own Gatling console summary; unit tests for the fold over hand-built items, including a lost outcome and a run with no requests; error-mapping tests for not-found, unreadable, not-a-log, below-range, 3.13.0, newer-than-range, truncation and syntax error; command tests through `runCLI` for every acceptance scenario and exit code, including results-root selection and the no-path default via `t.Chdir`; integration test drives the built binary; benchmark for the memory goal. Race detector on in CI. |
| IV | Minimal, Explicit Dependencies — no new module without rationale, licence compatibility and approval. | PASS — one module, `github.com/galax-io/parsec v0.1.0`: MIT, zero transitive modules, pure Go, approved 2026-09-15. `go mod tidy` leaves no diff. `govulncheck ./...` was run by hand and its clean result recorded in quickstart.md §6; no CI job runs it, and adding one is a change to the constitution's Quality Gates table and so a separate, ask-first change. |
| V | Published Surfaces — command, flag, default, exit code, output structure changes listed; breaking ones approved; README updated in the same PR. | PASS — additive: `galaxio report` with no arguments still prints help and exits 0, so the v0.12.0 surface is unchanged, and `galaxio report <x>` still exits 2, now as an unsupported tool. New: the `<tool>` and `[PATH]` arguments, the `-o` flag with its reserved names, the human-readable report and the new exit paths. **No machine-readable format is published by this milestone**, which is deliberate: the first published data surface will be `stats.json`, and it must be identical to Gatling's own. README documents the command in the same PR. |
| VI | Idiomatic, Simple Go — gofmt/vet clean; errors as values wrapped at the boundary; no panic control flow; no dead or duplicated code; no refactor outside scope. | PASS — parsec's errors are inspected with `errors.As` and wrapped once at the command boundary. **No type in `internal/report/` restates a parsec definition**: the run description, the bounds, the outcome, the optional value and the capability set are parsec's own, used directly; this package declares two types of its own — the tally of what it walked, and the open source that pairs parsec's reader with the format detected for the report — and neither restates a parsec definition. Required skills are read before the corresponding code is written. |

**Post-design re-check**: gates II–VI PASS; gate I keeps its single justified FAIL. The
withdrawal of the record stream removed the package's own record vocabulary, which is what
Principle VI's no-duplication clause and the maintainer's 2026-09-16 decision both required.

## Project Structure

### Documentation (this feature)

```text
specs/004-report-records/
├── plan.md              # This file
├── research.md          # Phase 0: decisions, evidence, alternatives
├── data-model.md        # Phase 1: run location, run identity, span, tally
├── quickstart.md        # Phase 1: how to prove the feature end to end
├── contracts/cli.md     # Command, arguments, flags, output, exit codes
├── checklists/requirements.md
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
cmd/galaxio/
├── report.go                 # operational command, reportOptions, runReport, error mapping
├── report_test.go            # command-level tests through runCLI and runReport
├── report_integration_test.go # built binary, behind the integration tag
└── root_test.go              # TestReportCommand expectations for the new help text

internal/report/
├── doc.go                    # package purpose: read a run and count what it holds
├── source.go                 # Locate and Open: run.Find + simlog.NewRunReader, read failures named
├── scan.go                   # Scan(ctx, RunReader) (Tally, error): one pass, counts, model.Bounds, endings wrapped
├── *_test.go                 # corpus, fold, error-mapping and benchmark tests
├── reporttest/replay.go      # synthetic logs of any size for the memory benchmark
└── testdata/corpus/gatling/  # frozen recordings copied from parsec v0.1.0 (MIT)

README.md                     # "Reporting Ecosystem": galaxio report, arguments, exit codes
go.mod / go.sum               # + github.com/galax-io/parsec v0.1.0
```

**Structure Decision**: `internal/report/` is a single flat package, the home the
constitution names for report code. It holds glue and one fold; the statistics of #51 are
added over the same pass, and the `stats.json` writer of #52 beside them. The corpus lives
once, next to the package that reads it; command tests reach it by relative path rather than
duplicating frozen recordings.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Principle I: `report` offers neither `-o text` nor `-o json`, and publishes no machine-readable output at all | Maintainer decisions of 2026-09-15 and 2026-09-16. On `report`, `-o` selects a report format, not an encoding; the three names it will ever take (`stats`, `global_stats`, `yml`) belong to later milestones and are reserved now so that `-o json` can never come to mean something else. The first machine-readable output this command publishes must be `stats.json` identical to Gatling's own, and shipping an interim format would publish a surface (Principle V) that is then withdrawn. | Complying by adding `-o text\|json` was rejected by the maintainer, repeatedly; an interim JSON Lines record stream was built and withdrawn on 2026-09-16 for the same reason. A follow-up constitution amendment should state that on `report` commands `-o` names a report format, so later report commands do not each need this row. |
