# Tasks: Read a Finished Gatling Run

**Input**: Design documents from `/specs/004-report-records/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [contracts/cli.md](contracts/cli.md),
[quickstart.md](quickstart.md)

**Tests**: Required by Constitution Principle III and never optional. Tests land in the same
commit as the change they cover: inside a task, write the test first, watch it fail, then
implement until green, then commit once. A task that only adds tests or evidence is green on
its own and still gets its own commit.

**Organization**: One milestone, one review boundary: the milestone PR on branch
`110-report-records`, assigned to `v0.13.0 Read a run`, closing #50 when it lands. Every
task maps to exactly one green commit (`go build ./... && go test ./...`), and every commit
to one task; the commit line is given at the end of each task.

**History**: this branch was rebuilt three times — on 2026-09-16 after the maintainer
withdrew the record stream, and twice more after full reviews of the rebuilt branch found
defects. AGENTS.md forbids add-then-remove inside a PR and asks for intent rather than path,
so each task below is one commit carrying its final content, and what the reviews found and
how it was answered is recorded in the PR body. Every commit was checked out on its own and
built and tested.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with the other `[P]` tasks of the same phase.
- **[Story]**: Maps the task to US1–US4 from `spec.md`.
- Every task names the exact files it changes.

## Required reading (constitution "Engineering Guidance")

`golang-cli` and `golang-spf13-cobra` before T004 and T005; `golang-error-handling` before
T003, T004 and T005; `golang-testing` before every task; `golang-naming` and
`golang-documentation` before T003 and T007; `golang-benchmark` before T003's benchmark;
`golang-safety` and `golang-security` before T003 (every log is untrusted input);
`golang-context` before T003.

---

## Phase 1: Setup

- [X] T001 Commit the feature artifacts first: `specs/004-report-records/` (spec, plan, research, data-model, contracts/cli.md, quickstart, checklists, tasks) and `.specify/feature.json` → commit `docs(speckit): add 004-report-records spec/plan/tasks (#50)`
- [X] T002 [P] Copy the corpus from the parsec v0.1.0 module cache into `internal/report/testdata/corpus/gatling/`: the `simulation.log` of 3.11.5, 3.12.0, 3.13.1, 3.14.9 and 3.15.1, and the `lastrun/results/` tree of three runs with its `lastRun.txt`; add `PROVENANCE.md` recording the source (parsec v0.1.0, MIT), the recording dates and each entry's own Gatling total, and `.gitattributes` marking every `simulation.log` as `-text` → commit `test(report): add Gatling corpus recordings from parsec v0.1.0 (#50)`

**Checkpoint**: spec on the branch, fixtures on disk, `go test ./...` unchanged and green.

---

## Phase 2: Foundational — reading a run (US1, US2, US3)

- [X] T003 [US1] Add the dependency with the code that imports it (`go get github.com/galax-io/parsec@v0.1.0 && go mod tidy`) and build the reading package. `internal/report/doc.go`: the package reads and counts and declares nothing parsec defines. `internal/report/source.go`: `ErrNoPath` wraps `run.ErrNoPath` rather than restating it, so a caller checking parsec's sentinel sees this package's refusal as the same condition; `Locate` resolves an empty path to `run.DefaultResultsRoot` and otherwise hands the path to `run.Find`, then tells apart the three things parsec folds into one absence — stat'ing `filepath.Clean(path)`, the path parsec searched, so that `a/nope/../b` is not called unreadable when parsec read `a/b` perfectly — reporting a named path that does not exist as a read failure, one that exists and is not a directory as not a run at all, and passing an absent *default* root and any directory parsec could not read through with parsec's own message; `Open` identifies the format with `ReadAt` over `gatling.DetectSize` bytes so that a short read cannot make a readable log report an unknown format, and adds the log's path only to a refusal that does not already name it; `Source.Scan` walks the run once and refuses a second walk, because a reader yields its items once and a spent one returns an empty tally and no error, which is what a run that recorded nothing returns; `Close` spends it too. `internal/report/scan.go`: `Tally` and `Scan` walk the stream once, extend one `model.Bounds`, count requests split by `model.Outcome` with a lost outcome counted apart, group traversals, virtual-user events, run-level errors, assertion payloads carried among the events, and items of a kind this release does not know, check `ctx.Err()` every 1024 items, and return the tally with a wrapped error for a truncation and for any other read failure. `internal/report/reporttest/replay.go`: `Split` caps the header to its own length so appending to it cannot reach the body, refuses a log whose last line is unterminated, and `Replay` treats a negative count as zero. Tests cover every corpus version against its own Gatling total, the seven run-selection rules, a dangling symlink, a path needing cleaning, a path that is a file, an absent default root, the version gate below and above the range and 3.13.0 by patching the recorded version bytes, a not-a-Gatling log, a missing and an unreadable path, a truncation, a damaged log, cancellation, a second walk, a run with no items, every item kind including a lost outcome, the header-versus-body aliasing, and the 32 MiB memory goal as a test the ordinary suite runs rather than a benchmark nothing runs → commit `feat(report): read a Gatling run and count what it holds (#50)`

---

## Phase 3: The command (US1, US2, US4)

- [X] T004 Let the kind of a failure decide the exit code, and let an interrupt stop the work. `cmd/galaxio/root.go` guessed the code back by looking for the words "unknown command" anywhere in the message, which a caller's own path or a log byte parsec quotes can imitate. Give the root `cobra.NoArgs` and wrap every argument validator in the tree once in `UsageError`, cobra's own generated `completion <shell>` commands included — they carry a bare `NoArgs` this package never sees, so their usage errors would otherwise exit like runtime failures — leaving an already-usage error alone so wrapping cannot nest. Run the CLI under a context an interrupt cancels (`cmd/galaxio/main.go` with `signal.NotifyContext`, `executeContext` with `cmd.ExecuteContext`), so that the `ctx.Err()` poll the read makes every 1024 items can fire at all. Tests pin both directions of the classification — a runtime failure whose message merely contains the words exits 1, a real unknown command exits 2 — and drive a cancelled read through the command → commit `fix(cli): decide the exit code by the kind of failure (#50)`
- [X] T005 [US1] Rewrite `cmd/galaxio/report.go` as the operational command: `Use: "report <tool> [PATH]"`, `Args: cobra.MaximumNArgs(2)` classified by the root's wrapper, `-o` checked by the flag's own type as it is parsed — the only stage that runs before cobra answers `--help`, so a reserved format can never be accepted and discarded — the tool validated against `[gatling.Tool]`, parsec's own spelling, and every known `-o` format rejected as not available yet naming its milestone with an unknown one listing the known; `reportOptions`/`reportOutput` carrying parsec's own `run.Location`, `model.Run`, `gatling.Format` and the tally, with `PathSet` carrying the distinction the empty string cannot so that the refusal of a path given as empty lives behind the seam a test can reach; `runReport` locating, opening, printing each version warning even under `--quiet`, scanning, naming the log on the scan's error once so that a failed write joins it rather than replacing it, and printing the aligned report unless `--quiet`, with a truncation still reporting the counts and a damaged log reporting none; `formatReport` rendering the block of `contracts/cli.md`, omitting every fact the source did not record including `run` and `id`, quoting anything from the log that is not printable so that a log cannot colour the terminal or erase the line it is on, counting the assertions the run declared wherever the source put them, and `--verbose` adding what the source can never record. Tests cover both formats against their own Gatling totals, the line order the contract pins, log path equals directory, the missing and unsupported tool, the empty argument through both the command and the seam, the unverified-version warning under `--quiet`, the three selection rules, the default root through `t.Chdir`, `--verbose`, every reserved and unknown `-o` value with and without a tool and with `--help`, a cancelled read, a hostile log, every failure path with its exit code and the log path in its message, a write failure reported once and a write failure beside a truncation, the report's omission branches, and the block's alignment asserted against what the report renders rather than a second copy of its keys → commit `feat(report): make galaxio report read a run (#50)`
- [X] T006 [US3] Add `cmd/galaxio/report_integration_test.go` behind `//go:build integration`: build the binary, read a replayed run of 720 000 requests and assert the reported counts, assert two reads of one run are byte-identical, and assert the exit code of a run that reads, a directory with no run, an unsupported tool, a reserved report format and no arguments → commit `test(report): integration tests for reading, determinism and exit codes (#50)`

**Checkpoint**: all four stories complete; `go test -race ./...` and the integration suite green.

---

## Phase 4: Documentation and evidence

- [X] T007 [P] Document the command in `README.md` § Reporting Ecosystem: `galaxio report <tool> [PATH]`, the tool rule, the PATH rules, an example report, what each line means, the reserved `-o` formats and why there is no `-o json`, and the exit codes including the truncated-versus-damaged distinction and the boundary-cut run no reading can catch → commit `docs(readme): document galaxio report (#50)`
- [X] T008 Rename milestone 3 from `v0.13.0 Report dump` to `v0.13.0 Read a run` and rewrite its description, since both named the withdrawn record dump, and amend issue #50 with the withdrawal, what the milestone now delivers and the restated acceptance criteria, retitling it to the problem the milestone removes. The spec artefacts already cite the new title, so this task carries no file change of its own beyond its evidence → commit `docs(speckit): record the milestone and issue rename (#50)`
- [ ] T009 Run and record the gates in `specs/004-report-records/quickstart.md` §6 at the branch's final commit: `gofmt -l .`, `go vet ./...` with and without the integration tag, `go test -race -coverprofile=coverage.out ./...` with the total coverage figure (≥ 80%), `go mod tidy && git diff --exit-code -- go.mod go.sum`, `go mod verify`, `go test -tags=integration -race -count=1 ./...`, `govulncheck ./...` → commit `docs(speckit): record 004-report-records gate evidence (#50)`
- [ ] T010 Walk `specs/004-report-records/quickstart.md` §1–§5 with the built binary, record each observed output next to its expected value including the memory figures in §3, tick this box, then push the branch and update the milestone PR against `main`, assigned to milestone `v0.13.0 Read a run` with `Closes #50` in the body, and leave it open for maintainer review → commit `docs(speckit): record 004-report-records quickstart evidence (#50)`

---

## Dependencies & Execution Order

- **T001** first (spec-first). **T002** needs only the parsec module cache and runs beside it.
- **T003** after T002 (its tests read the corpus). Blocks everything below.
- **T004** is independent of the report command and lands before it, because the command's
  exit codes rest on it.
- **T005** after T003 and T004. **T006** after T005.
- **T007** after T005 (the surface is final). **T008** after T007, since it renames the
  milestone those documents cite. **T009** after T006. **T010** last.

```text
Phase 1:  T001 | T002
Phase 2:  T003
Phase 3:  T004 → T005 → T006
Phase 4:  T007 → T008 | T009 → T010
```

Every task is its own commit on one linear branch; parallel means the work does not wait,
not that two tasks share a commit.

---

## Implementation Strategy

1. T001–T002: spec on the branch, fixtures on disk.
2. T003: `internal/report` reads every corpus version, in one pass, inside the memory goal.
3. T004–T006: the exit-code contract, then the command, then the binary exercised end to end.
4. T007–T010: README, the milestone and issue renamed, gate evidence, quickstart evidence.

---

## Notes

- Each task is one commit, message as given, with the session's attribution trailer;
  `gofmt -w .` before each commit; `go build ./... && go test ./...` green at every commit.
- No statistic is computed anywhere in this milestone, and no machine-readable format is
  written. `stats.json` is #52, over the statistics of #51, and must carry Gatling's schema
  and Gatling's numbers with every possible divergence named.
- The Principle I deviation (no `-o text|json`) is justified in `plan.md` Complexity
  Tracking against constitution v2.0.0, which is what is in force while this milestone is
  reviewed. The amendment it proposes is PR #113, in milestone v0.14.0; it lands after this
  one, so nothing here depends on it.
- Issue #50 was amended and retitled by T008, so the issue this PR closes describes the
  reading that was built rather than the withdrawn record dump.
- The PR is left open for maintainer review; only the maintainer merges it.
