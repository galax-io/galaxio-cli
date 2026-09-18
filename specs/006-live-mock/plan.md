# Implementation Plan: Serve the Live Gatling Runs from a Service That Does Real Work

**Branch**: `115-live-mock` | **Date**: 2026-09-18 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/006-live-mock/spec.md`
(milestone [`v0.15.0 Legacy stats.json`](https://github.com/galax-io/galaxio-cli/milestone/5),
issue [#117](https://github.com/galax-io/galaxio-cli/issues/117))

## Summary

The service of the hand-made run of 2026-09-18 becomes `reporttest.Mock()`, an `http.Handler` in
test support. It hashes a fixed buffer for every request, with three endpoints:

- `GET /`: about 4 ms of hashing;
- `GET /report`: about 0.2 s;
- `GET /export`: about 1 s.

`TestReportLiveGatling` loads the mock instead of its sleeping stub, and writes its own scenario
into the rendered project. Every user calls `/`; 7 % of users, chosen by their number, also call
`/report`, and 3 % call `/export`. The run is `Stability` at 3000 rpm, with the versions and the
comparisons unchanged. The committed recordings stay as they are, and `RECORDING.md` names the
tag that still holds the stub that served them.

## Technical Context

**Language/Version**: Go 1.27.1 (`go.mod`); Scala, as the template renders it, for the scenario.

**Primary Dependencies**: none new. `crypto/sha256`, `encoding/hex` and `net/http`.

**Storage**: N/A.

**Testing**: the mock is tested with Gatling, by the live run it serves, and has no Go test of
its own (maintainer, 2026-09-18). `TestReportLiveGatling` fails when Gatling prints a failed
request, or no successful request to one of the three endpoints. It reads the count Gatling
prints for each request name, in the console format of Gatling 3.11–3.13 and in that of 3.14 and
later. It stays behind the `integration` tag and `GALAXIO_LIVE_GATLING=1`. Two runs prove it:
all five versions against the mock (quickstart §2), and one against a mock with an endpoint
broken (quickstart §3). Gates: `gofmt`, `go vet` with and without the tag, `go test -race`,
coverage ≥ 80 %, `go mod tidy` diff.

**Target Platform**: as `TestReportLiveGatling`: a maintainer's machine with a JDK and sbt.

**Project Type**: Go CLI; this feature is test support and an integration test.

**Performance Goals**: the mock needs about 2.4 cores at the live test's 50 users a second
([research.md](research.md) §2). The ordinary suite does not change.

**Constraints**: no sleep and no random number in the mock. No environment variable is added.
The committed recordings do not change.

**Scale/Scope**: one handler, one integration test changed, one provenance note, spec artifacts.

## Constitution Check

*GATE: evaluated before Phase 0 research and re-checked after Phase 1 design.*

| # | Gate (constitution principle) | Status |
|---|---|---|
| I | Command Contract — every new command is a thin cobra wrapper over `runX(ctx, opts)`; offers `-o`, valued `text\|json` where it encodes one output and named by product where it selects among several, with a documented structure for any machine-readable form and no interim one invented; exits 0/1/2 via `UsageError`/`RuntimeError`; honours `--verbose`/`--quiet`/`--no-color`; experimental commands sit behind `internal/featureflags`. | PASS — no command, flag, output or exit code changes. |
| II | Report Arithmetic Lives Here — (report features only) statistics are computed in `internal/report/` over `parsec` primitives, not requested from the library; success and failure accumulated separately; counts, extremes, mean and deviation exact, with a refusal rather than an estimate where one cannot be kept exact; percentiles exact, or a deterministic estimate from a bounded-memory sketch that every output labels as one, its estimator recorded in `research.md`, its corpus values pinned in tests and any known divergence documented with an example; a Gatling run's percentiles equal to Gatling 3.11.x's digest over the same log, asserted by tests, save where the estimator orders centroids of an equal mean otherwise, which a test may describe across a boundary between two runs of equal response times; an output reproducing another tool's file names the fields that can differ from it and why; one pass, bounded memory, with the peak-memory goal stated in Technical Context; absence reported as absent; source detected by content. Mark N/A for non-report features. | PASS — no arithmetic changes. The live test keeps asserting the equality with Gatling 3.11's digest and with what Gatling printed, now on response times no table chose. The peak-memory goal is unchanged: heap under 32 MiB. The committed recordings and every pinned value stay. |
| III | Tests Land With The Change — stdlib `testing`, table-driven, golden files under `testdata/`; command-level tests through `runCLI` asserting exit code and output; integration tests behind the `integration` tag on real packs/registries/specs; race on; coverage stays ≥ 80%; every fix carries a regression test; test tasks are never optional. | PASS — the mock lands in the same commit as the Gatling run that tests it. The live test fails when Gatling prints a failed request, or no successful request to one of its endpoints, and a run against a broken endpoint shows that it does. The integration test stays behind its tag, renders the real published pack, and runs real Gatling and the real t-digest jars. The mock is not a mock under the principle: the system under test is the command, and the service stands in for whatever a user loads, which no test can reach (spec, Assumptions). |
| IV | Minimal, Explicit Dependencies — no new module unless named here with the reason the standard library or an existing dependency is insufficient, recorded in `research.md`, licence-compatible with GPL-2.0-only, and asked for first. | PASS — no module; `go.mod` and `go.sum` do not change. |
| V | Published Surfaces — any change to a command, flag, default, exit code, `-o json` structure, manifest/registry schema or generated output is listed; breaking ones are approved before implementation and will be committed with `!`; README updated in the same PR; deprecations keep working one minor release. | PASS — nothing published changes, so the README does not. The template is not changed: the scenario is written into the rendered copy the test owns. |
| VI | Idiomatic, Simple Go — gofmt/vet clean; errors as values wrapped into `UsageError`/`RuntimeError` at the boundary; no panic control flow; no dead or duplicated code; no refactor outside this issue's scope. | PASS — `liveSeed`, `livePlan` and `liveStub` go with the last call to them, and the provenance note points at the tag that keeps them. The mock is a table of three endpoints and one handler; it has no limit, since no run reaches one ([research.md](research.md) §1). |

**Post-design re-check**: gates I–VI PASS; the design adds nothing the pre-research check did not
cover.

## Project Structure

### Documentation (this feature)

```text
specs/006-live-mock/
├── plan.md              # This file
├── research.md          # Phase 0: the mock's work, who calls the long endpoints, the scenario
├── data-model.md        # Phase 1: the endpoints and the scenario
├── quickstart.md        # Phase 1: how to prove the feature end to end
├── checklists/requirements.md
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

No `contracts/`: the feature adds nothing a user or another system calls.

### Source Code (repository root)

```text
internal/report/
├── reporttest/
│   ├── mock.go                        # Mock: three endpoints, each hashing a fixed amount
│   └── replay.go                      # the package comment names the mock beside the logs
├── live_integration_test.go           # the mock instead of liveStub; the scenario copied into the project; the mock held to what Gatling printed
├── testdata/live/scenario/            # cases/HttpActions.scala and scenarios/HttpScenario.scala, copied over the rendered ones
└── testdata/live/gatling/RECORDING.md # where the stub that served the recordings can still be read
```

**Structure Decision**: the mock sits in `reporttest`, the package only tests import, beside the
comparisons the live test uses. The scenario is two Scala files under `testdata/live/scenario/`,
beside the recordings, and everything else of the project is rendered from the template.

## Complexity Tracking

No gate fails; nothing to justify.
