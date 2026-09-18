# Implementation Plan: Summarise a Finished Gatling Run

**Branch**: `113-report-summary` | **Date**: 2026-09-17 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/005-report-summary/spec.md`
(milestone [`v0.14.0 Report summary`](https://github.com/galax-io/galaxio-cli/milestone/4),
issue [#51](https://github.com/galax-io/galaxio-cli/issues/51))

## Summary

`galaxio report gatling [PATH]` keeps the run description of milestone v0.13.0 and prints
below it the summary of the run as a whole: counts, shares, minimum, maximum, mean, standard
deviation, percentiles and the mean request rate, each for all, ok and failed requests, and
the response-time bands. The layout is this tool's own and tool-independent — a headline, a
response-time table with one row per outcome, the bands as bars — chosen by the maintainer
from three mock-ups; the outcomes are `ok` and `failed` everywhere, which changes one word
of the v0.13.0 description. `--percentiles` and `--bounds` change the ranks and the band
boundaries from Gatling's defaults. **No figure per request or group is printed and no file
is written**: those belong to the `-o` products of later milestones. Everything is computed
from the log in the single pass the command already makes; the HTML report is not needed.
While a long log is read, a terminal sees a six-line block on standard error — a progress
bar, the time left and the figures so far — redrawn in place from the same accumulators and
erased before the report; pipelines and `--quiet` see nothing of it, and standard output
does not change by a byte.

Every non-percentile figure is exact and is verified against what Gatling itself recorded
for the corpus. Percentiles come from `github.com/caio/go-tdigest/v5` read with its default
quantile (maintainer decision, 2026-09-17), are labelled as this tool's estimates, and are
pinned in tests — including the known case, 1427 ms printed for the 3.13.1 run where the
request at the rank took 1502 ms. The upstream pull request that removes that case,
[caio/go-tdigest#42](https://github.com/caio/go-tdigest/pull/42), is noted in the README and
is not part of this milestone.

**Prerequisite met** (clarified 2026-09-17): the percentile clause of constitution
Principle II was amended first, in its own issue and pull request inside this milestone —
[#114](https://github.com/galax-io/galaxio-cli/issues/114), merged as
[#115](https://github.com/galax-io/galaxio-cli/pull/115) on 2026-09-17, constitution
v2.2.0 — and this plan is checked against the ratified text. See gate II and
[research.md](research.md) §13.

## Technical Context

**Language/Version**: Go 1.27.1 (`go.mod`; CI uses `go-version-file`).

**Primary Dependencies**: `github.com/spf13/cobra v1.10.2` and
`github.com/galax-io/parsec v0.1.0` (existing). **New**: `github.com/caio/go-tdigest/v5
v5.0.0` — MIT, the only module it links, standard-library-only outside its tests, builds
with `CGO_ENABLED=0`, no known advisory; named by the maintainer on 2026-09-17. Used at its
defaults: compression 100 and its own constant-seeded generator. The Principle IV case is
[research.md](research.md) §1. Colour, the progress block and the recognition of a terminal
use the standard library only.

**Storage**: N/A — the command creates, changes and removes no file.

**Testing**: Standard-library `testing`, table-driven. Unit tests for the exact arithmetic
(adversarial sets for the half-up mean and deviation — three requests of 0 ms and three of
1 ms must give a mean of 1 — single request, equal requests, sums beyond 64 bits), the
bands and their bar cells, the rate, group traversals taking part in nothing, requests with
no end and lost outcomes. Corpus tests comparing every non-percentile whole-run figure with
the `global_stats.json` Gatling wrote (3.11.5, 3.12.0, 3.13.1) and with its console (3.13.1,
3.14.9, 3.15.1); no test reads a percentile Gatling recorded. Pinned percentiles per corpus
run. Golden files for the console summary, plain and coloured. Command tests through
`runCLI` for every acceptance scenario and exit code, including the `failed` wording of the
description, no escape sequence off a terminal, and the run directory left byte-for-byte as
it was. The progress block is tested through an injected redraw check and clock: the frames
drawn, control sequences included; that it is erased before an error and on an interrupt;
that nothing is drawn for a short read, a redirected standard error or a `dumb` terminal;
and that standard output is identical with and without it. An ordinary test for the memory
goal; a `testing.B` for throughput, with and without the block; an integration test behind
the tag for one hundred identical outputs. Gates: `gofmt`, `go vet`, `go test -race`,
coverage ≥ 80 %, `go mod tidy` diff.

**Target Platform**: The existing static, cross-platform CLI and the distroless image; the
new module is pure Go. A terminal whose `TERM` is unset or `dumb` — the classic Windows
console included — gets plain text and no progress block.

**Project Type**: Go CLI; one command (three files), one internal package, versioned spec
artifacts.

**Performance Goals**: One forward pass; one digest insertion per request, measured at
about 0.3 µs. Throughput is measured and recorded in quickstart.md, not gated; so is the
cost of the progress block, which SC-010 caps at 5 %.

**Constraints**: **Peak memory goal: heap in use stays under 32 MiB for a log of any size
and any number of distinct request names** — the goal of v0.13.0, unchanged. The summary
adds two digests and a handful of integers: 0.05 MiB measured at ten million requests and
0.06 MiB at a hundred million ([research.md](research.md) §8); nothing is kept per name.
Enforced by a test in the ordinary suite. Counts, extremes, mean and deviation are computed
in integers. Output is deterministic. Exit codes 0/1/2 via `RuntimeError`/`UsageError`. No
machine-readable form and no file is added. Licence: GPL-2.0-only consuming MIT modules.

**Scale/Scope**: Five corpus runs of 36 or 102 requests; synthetic replays of gigabytes for
the memory goal and the progress block. Three command files, four source files in
`internal/report/`, recorded reference data added to the corpus, a README section, spec
artifacts.

## Constitution Check

*GATE: evaluated before Phase 0 research and re-checked after Phase 1 design.*

| # | Gate (constitution principle) | Status |
|---|---|---|
| I | Command Contract — every new command is a thin cobra wrapper over `runX(ctx, opts)`; offers `-o`, valued `text\|json` where it encodes one output and named by product where it selects among several, with a documented structure for any machine-readable form and no interim one invented; exits 0/1/2 via `UsageError`/`RuntimeError`; honours `--verbose`/`--quiet`/`--no-color`; experimental commands sit behind `internal/featureflags`. | PASS — no new command: `runReport(ctx, opts) (reportOutput, error)` gains the summary and stays the test seam. `-o` is untouched: its three product names stay reserved and rejected, no `-o json`, `-o text` or other machine-readable form is invented, and no option names an output file — the products of `-o` remain the only files this command is to write. Bad `--percentiles` and `--bounds` are `UsageError` (exit 2), refused while the flag is parsed; a span of zero, a lost outcome and the endings of v0.13.0 are `RuntimeError` (exit 1) naming the span or the count. `--quiet` silences the summary and the progress block. `--no-color` and `NO_COLOR`, which the root already merges, are honoured by this command for the first time: they switch its colours off. The progress block is a diagnostic: standard error only, a redraw-capable terminal only, never under `--quiet`, and erased before any error is written. |
| II | Report Arithmetic Lives Here — (report features only) statistics are computed in `internal/report/` over `parsec` primitives, not requested from the library; success and failure accumulated separately; counts, extremes, mean and deviation exact, with a refusal rather than an estimate where one cannot be kept exact; percentiles exact, or a deterministic estimate from a bounded-memory sketch that every output labels as one, its estimator recorded in `research.md`, its corpus values pinned in tests and any known divergence documented with an example; no parity with another tool's percentiles claimed or tested for; one pass, bounded memory, with the peak-memory goal stated in Technical Context; absence reported as absent; source detected by content. Mark N/A for non-report features. | PASS against v2.2.0 (#115) — all arithmetic is in `internal/report/` over `model.Bounds` and `model.Outcome`, neither re-derived; ok and failed are accumulated apart and a failure reaches no figure of ok requests; counts, extremes, mean and deviation are exact integer arithmetic that cannot wrap for any run the command can read — a 128-bit sum and a 192-bit sum of squares (research.md §4) — so the refusal is never reached; percentiles are a deterministic estimate — `caio/go-tdigest/v5` at compression 100 with its constant-seeded generator, read with `Quantile` and rounded half up, recorded in research.md §1–§2 — that says so wherever it is printed: the summary's closing line and the progress block's second line; their values for the five corpus runs are pinned by T007, and the README documents the 1427-for-1502 divergence with its example (FR-020); they are never presented as Gatling's and no test reads a percentile Gatling recorded; one pass, nothing retained, goal stated and tested; an empty outcome, an unbounded run and a request with no end are reported as absent, never as zero. |
| III | Tests Land With The Change — stdlib `testing`, table-driven, golden files under `testdata/`; command-level tests through `runCLI` asserting exit code and output; integration tests behind the `integration` tag on real packs/registries/specs; race on; coverage stays ≥ 80%; every fix carries a regression test; test tasks are never optional. | PASS — see Testing above: every arithmetic rule has a table-driven unit test with the adversarial case that distinguishes it from the rejected alternative; the whole-run figures of every corpus run are compared with what Gatling recorded; golden files for the plain and the coloured summary; `runCLI` tests for each acceptance scenario and exit code; the memory goal is an ordinary test; integration test on the built binary. The v0.13.0 tests that quote `ko` change with the line they quote. Tests precede or accompany each task; none is optional. |
| IV | Minimal, Explicit Dependencies — no new module unless named here with the reason the standard library or an existing dependency is insufficient, recorded in `research.md`, licence-compatible with GPL-2.0-only, and asked for first. | PASS — one new module, `github.com/caio/go-tdigest/v5 v5.0.0`, named by the maintainer on 2026-09-17: MIT; the only module linked; bounded-memory percentiles are not in the standard library. `go.sum` also gains checksums of its two test-only dependencies (`gonum`, BSD-3-Clause; `leesper/go_rng`, Apache-2.0), which are neither required in `go.mod`, nor compiled, nor distributed — stated in research.md §1 so the Apache-2.0 line in `go.sum` surprises nobody. `influxdata/tdigest` was excluded for being Apache-2.0. Recognising a terminal uses the standard library (`ModeCharDevice` and `TERM`), colour and the progress block are a few fixed escape sequences, and the terminal's width is never asked for — so `golang.org/x/term` is not added. `go mod tidy` leaves no diff; `govulncheck` is run by hand and recorded, as in v0.13.0. |
| V | Published Surfaces — any change to a command, flag, default, exit code, `-o json` structure, manifest/registry schema or generated output is listed; breaking ones are approved before implementation and will be committed with `!`; README updated in the same PR; deprecations keep working one minor release. | PASS — listed: two new flags on `galaxio report` (`--percentiles`, default `50,75,95,99`; `--bounds`, default `800,1200`), neither with a shorthand; new lines on standard output below the v0.13.0 block; colour on a terminal, off with `--no-color` or `NO_COLOR`; one new warning on standard error; a progress block on standard error while reading, shown to a redraw-capable terminal only and leaving standard output and the exit code identical. **One word of published text changes**: the requests line of the description says `18 failed` where v0.13.0 said `18 ko` — approved by the maintainer in clarification. It is not a breaking change as Principle V defines one: the text report is not a parseable output, and the forms a program reads are the `-o` products, none of which exists yet; it is called out because a script may grep it. **Two exit codes change from 0 to 1** — a run whose span is zero and a run holding a request whose outcome the source lost — as issue #51 decides; no Gatling log is known to produce either, so no existing invocation on a real run changes, and the spec calls it out. README documents all of it in the same PR, with the percentile limitation and the upstream pull request. |
| VI | Idiomatic, Simple Go — gofmt/vet clean; errors as values wrapped into `UsageError`/`RuntimeError` at the boundary; no panic control flow; no dead or duplicated code; no refactor outside this issue's scope. | PASS — the v0.13.0 walk stays the only loop: `Scan` is restructured to return a `Summary` that carries the `Tally`, as an explicit task and its own commit, so no second pass and no dead tally-only path remain. Nothing is computed that nothing shows: no accumulator per request or group exists until a product that carries one does. The digest library panics on a quantile outside [0, 1]; ranks are validated at the flag, so no `recover` is needed and none is used. The progress block starts no goroutine and no timer: it is redrawn from the walk's existing every-1024-items check, so there is nothing to race and nothing to stop, and an interrupt reaches it as the cancelled context the walk already returns on. No parsec definition is restated. Required skills are read before the corresponding code (research.md §14). |

**Post-design re-check**: gates I–VI PASS, gate II against constitution v2.2.0 as #115
ratified it. The re-check after that merge (T003) found two gaps and closed both in the
design before any code: the progress block printed percentiles without saying they are
estimates, and a 64-bit sum of durations could wrap on a hostile log (research.md §15 and
§4).

## Project Structure

### Documentation (this feature)

```text
specs/005-report-summary/
├── plan.md              # This file
├── research.md          # Phase 0: decisions, measurements, alternatives
├── data-model.md        # Phase 1: options, outcome figures, bands, span, read progress
├── quickstart.md        # Phase 1: how to prove the feature end to end
├── contracts/cli.md     # Flags, console layout, colour, progress block, exit codes
├── checklists/requirements.md
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
cmd/galaxio/
├── report.go                  # + two flags and their pflag.Value types, options, runReport wiring; "failed" in the requests line
├── report_summary.go          # the summary text: headline, response-time table, bands as bars, closing line; colour
├── report_progress.go         # the progress block: when it is shown, what it carries, how it is redrawn and erased
├── report_test.go             # + flag validation, exit codes, --quiet, the run directory left untouched
├── report_summary_test.go     # golden summary per corpus run, plain and coloured; absence; custom ranks and bounds
├── report_progress_test.go    # frames under a fake clock, erased before errors and on interrupt, silent off a terminal
├── report_integration_test.go # + one hundred identical outputs on the built binary
└── testdata/report/           # golden files: <version>.summary.golden, 3.13.1.summary.color.golden

internal/report/
├── doc.go                     # package purpose: read a run, count it and summarise it
├── scan.go                    # the one walk; Scan returns a Summary that carries the Tally; calls the tick it is given
├── source.go                  # + bytes read and the size seen at opening, for the progress bar
├── summary.go                 # Options, Summary, the two outcomes, bands, span in seconds
├── moments.go                 # exact count, extremes, sum, 128-bit sum of squares; half-up mean and deviation
├── percentiles.go             # one digest per outcome; merged read for all requests
├── *_test.go                  # arithmetic, corpus-against-Gatling, pinned percentiles, memory, benchmark
└── testdata/corpus/gatling/   # + what Gatling recorded: global_stats.json, console.txt; PROVENANCE.md

README.md                      # "Reporting Ecosystem": the summary and its layout, the two flags, every figure's
                               # definition, the percentile limitation with its example and caio/go-tdigest#42,
                               # the progress block, "failed" in the description example
go.mod / go.sum                # + github.com/caio/go-tdigest/v5 v5.0.0
```

**Structure Decision**: `internal/report/` stays one flat package, as milestone v0.13.0
left it and as the constitution names it; the arithmetic is split by what a reader looks
for — the walk, the summary, the exact moments, the percentiles. The command gains a second
file because laying the summary out would double `report.go`, and a third for the progress
block, which is about a terminal and not about the report; all three are one command.
Recorded reference data lives beside the logs it belongs to, once.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| One milestone, one pull request: the amendment was a second pull request in milestone v0.14.0, #115 | It had to be merged before the implementation it governs, so it could not ride in the implementation's pull request. Explicitly requested by the maintainer in clarification, which is what the rule requires for a split; #113 did the same for #112 in this milestone. | Putting the amendment first in the same pull request keeps one PR but lets the implementation be reviewed against a constitution that is not yet ratified on `main`. |
