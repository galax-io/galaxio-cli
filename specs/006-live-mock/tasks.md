# Tasks: Serve the Live Gatling Runs from a Service That Does Real Work

**Input**: Design documents from `/specs/006-live-mock/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [quickstart.md](quickstart.md)

**Tests**: Required by Constitution Principle III and never optional. A test lands in the same
commit as the code it covers.

**Organization**: One milestone, one review boundary: the milestone PR on branch `115-live-mock`,
assigned to `v0.15.0 Legacy stats.json`, closing #117 when it lands. Every task maps to exactly
one green commit (`go build ./... && go test ./...`), and every commit to one task; the commit
line is given at the end of each task, and each commit ticks its own task. #52 joins the same
pull request later.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with the other `[P]` tasks of the same phase.
- **[Story]**: Maps the task to US1–US2 from `spec.md`.
- Every task names the exact files it changes.

## Required reading (constitution "Engineering Guidance")

`golang-testing`, `golang-naming` and `golang-documentation` before T002, which exports
`reporttest.Mock` and changes the live test. `galaxio-gatling-pro` before T002, which places the
scenario the test copies into the rendered project.

---

## Phase 1: Setup

- [X] T001 Commit the feature artifacts first: `specs/006-live-mock/` (spec, plan, research, data-model, quickstart, checklists, tasks) and `.specify/feature.json` → commit `docs(speckit): add 006-live-mock spec/plan/tasks (#117)`

---

## Phase 2: User Story 1 + User Story 2 — the live runs load the mock (Priority: P1, P2) 🎯 MVP

**Goal**: `TestReportLiveGatling` loads a mock that does real work. Every user calls `/`, and users chosen by their number also call `/report` (7 %) and `/export` (3 %). The mock is held to what Gatling printed. Every comparison stays as it was.

**Independent Test**: quickstart §2 and §3, all five versions against the mock and one against a broken endpoint.

The mock, US1 and US2 land in one commit. The mock is tested only by the Gatling run it serves (research.md §5), so it cannot land before that run does. The long endpoints are called only by the scenario the test copies in.

- [ ] T002 [US1] [US2] Serve the live Gatling runs from the mock:
  - **The mock.** Add `internal/report/reporttest/mock.go` with `Mock() http.Handler`: a `ServeMux` with `/{$}`, `/report` and `/export`, registered without a method so the handler decides. Each handler refuses every method but `GET` — including `HEAD`, which a `GET` pattern also matches — with `405` and `Allow: GET`, before any work; a `GET` hashes the fixed 4 KiB block `byte(i * 31)` 2 000, 128 000 or 640 000 times into one SHA-256 and answers `200`, `application/json`, `{"digest":"<first 8 bytes of the sum, hex>"}`. Nothing sleeps, reads a clock or draws a random number (data-model.md). Widen the package comment in `internal/report/reporttest/replay.go` to name the mock beside the logs and the comparisons.
  - **Serve the runs from it.** In `internal/report/live_integration_test.go`, serve the run from `httptest.NewServer(reporttest.Mock())`. Remove `liveSeed`, `livePlan` and `liveStub`, and the imports only they used.
  - **Commit the scenario, render the rest.** Add `internal/report/testdata/live/scenario/cases/HttpActions.scala` and `scenarios/HttpScenario.scala`: the template's `getMainPage` for `GET /`, plus `getReport` and `getExport` for the mock's other two endpoints, and the flow that sends them. `writeLiveScenario` copies every file under `testdata/live/scenario/` over the rendered file of the same path after `template init`, failing and naming the path when the template no longer renders it. Nothing else of the project is committed; it stays rendered from `gatling/scala-sbt`.
  - **Send the long endpoints.** Every user runs `HttpActions.getMainPage`; `doIf(session => session.userId % 100 < 7)` adds `getReport` and `doIf(session => session.userId % 100 >= 97)` adds `getExport`.
  - **Hold the mock to Gatling.** Add `servedEverything`: fail when Gatling's Global Information prints a failed request, or when the last progress line of `GET /`, `GET /report` or `GET /export` prints no successful one. Read both console formats, up to 3.13 and from 3.14.
  - **Rewrite the test's comment** to describe the mock, the scenario and the check.
  - **Provenance.** In `internal/report/testdata/live/gatling/RECORDING.md`, say that the stub which served the recordings is `livePlan` in `internal/report/live_integration_test.go` as tagged `v0.14.0`, and that `TestReportLiveGatling` now loads `reporttest.Mock`.

  Run quickstart §2 and §3 before committing → commit `test(report): serve the live Gatling runs from a mock that does real work (#117)`

**Checkpoint**: all five versions pass against the mock with three request names and the same counts in each; a broken endpoint fails the test on that endpoint; every method but `GET` is refused before any work.

---

## Phase 3: Polish & Validation

- [ ] T003 Run quickstart.md §1 to §5 and record what was observed, dated, under each section of `specs/006-live-mock/quickstart.md` → commit `docs(speckit): record 006-live-mock quickstart evidence (#117)`

---

## Dependencies & Execution Order

- T001 → T002 → T003, in that order.
- No `[P]` task: each task touches what the previous one made.

## Stories

| Story | Tasks | Test |
|---|---|---|
| US1 — response times nobody chose | T002 | quickstart §2 and §3 |
| US2 — long requests beside short ones | T002 | quickstart §2: three request names, equal counts across versions |

## Implementation Strategy

T001 and T002 are the whole feature. T003 proves it and records the evidence in the pull request.
