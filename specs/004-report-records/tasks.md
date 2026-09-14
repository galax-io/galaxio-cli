# Tasks: Report a Gatling Run as Records

**Input**: Design documents from `/specs/004-report-records/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [contracts/cli.md](contracts/cli.md),
[contracts/records.md](contracts/records.md), [quickstart.md](quickstart.md)

**Tests**: Required by Constitution Principle III and never optional. Tests land in the
same commit as the change they cover: inside a task, write the test first, watch it fail,
then implement until green, then commit once. A task that only adds tests or evidence is
green on its own and still gets its own commit.

**Organization**: Tasks are grouped by user story from `spec.md`. The milestone has one
review boundary: the milestone PR on branch `110-report-records`, assigned to
`v0.13.0 Report dump`, closing #50 when it lands. Every task maps to exactly one green
commit (`go build ./... && go test ./...`), and every commit to one task; the commit line
is given at the end of each task.

**Delivery constraints** (from `AGENTS.md`): `gofmt -w .` before every commit; no
merge commits; the PR stays open for maintainer review. `github.com/galax-io/parsec v0.1.0`
was approved by the maintainer on 2026-09-15 and lands in T003 with the first code that
imports it.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with the other `[P]` tasks of the same phase — different
  files, no dependency on an incomplete task.
- **[Story]**: Maps the task to US1–US4 from `spec.md`.
- Every task names the exact files it changes.

## Required reading (constitution "Engineering Guidance")

Read before the task that touches the area: `golang-cli` and `golang-spf13-cobra` before
T008; `golang-error-handling` before T005–T008; `golang-testing` before every task;
`golang-naming` and `golang-documentation` before T003 and T015; `golang-benchmark` before
T011; `golang-safety`/`golang-security` before T006 and T007 (every log is untrusted
input); `golang-context` before T007. Record any disagreement with the constitution in
`research.md`; none is expected.

---

## Phase 1: Setup (Fixtures and Spec-First Delivery)

**Purpose**: Put the frozen recordings and the specification on the branch before any code.

- [X] T001 Commit the feature artifacts first: `specs/004-report-records/` (spec.md, plan.md, research.md, data-model.md, contracts/, quickstart.md, checklists/, tasks.md) and `.specify/feature.json` → commit `docs(speckit): add 004-report-records spec/plan/tasks (#50)`
- [X] T002 [P] Copy the corpus from the parsec v0.1.0 module cache (`$(go env GOMODCACHE)/github.com/galax-io/parsec@v0.1.0/testdata/corpus/gatling/`) into `internal/report/testdata/corpus/gatling/`: `3.11.5/simulation.log`, `3.12.0/simulation.log`, `3.13.1/simulation.log`, `3.14.9/simulation.log`, `3.15.1/simulation.log`, and the whole `lastrun/results/` tree (three run directories with their `simulation.log` and `lastRun.txt`; omit `index.html`); add `internal/report/testdata/corpus/gatling/PROVENANCE.md` stating the source (parsec v0.1.0, MIT), the recording dates, the expected request counts (36, 36, 102, 102, 102) and that recordings are never edited; add `internal/report/testdata/corpus/gatling/.gitattributes` with `**/simulation.log -text` so no checkout rewrites the binary logs → commit `test(report): add Gatling corpus recordings from parsec v0.1.0 (#50)`

**Checkpoint**: Spec on the branch; fixtures on disk; `go test ./...` unchanged and green.

---

## Phase 2: Foundational (Schema, Conversion, Source, Writer)

**Purpose**: The `internal/report/` package every user story runs through. No user story
work starts before T007 is green.

- [X] T003 Add the dependency with the first code that imports it: run `go get github.com/galax-io/parsec@v0.1.0 && go mod tidy`; create `internal/report/doc.go` (package converts parsec's stream into the documented record schema and computes nothing) and `internal/report/record.go` with the record kinds (`run`, `request`, `group`, `user`, `error`, `assertion`), one struct per kind with the JSON tags and key order of `contracts/records.md`, the generic `Opt[T comparable]` (`MarshalJSON`, `IsZero`, used with `omitzero`), and the `model.Field` → identifier table from `data-model.md` with `Field.String()` fallback; add `internal/report/record_test.go` covering `Opt` unset-omitted versus set-zero-written, `groups` empty-non-nil renders `[]`, every known `Field` maps to its identifier, and an out-of-range `Field` falls back → commit `feat(report): add parsec and the record schema (#50)`
- [X] T004 [P] Create `internal/report/convert.go` with `headerFrom(model.Run) RunRecord` and `recordFrom(model.Item) Record` (epoch-millisecond instants omitted for the zero time, durations in milliseconds, `absent` from `Capabilities.Absent()`, `warnings` as `{version, reason}`, `assertions` and `payload` base64 of the bytes verbatim, `failure` present iff outcome is failure with `type` omitted when empty, `outcome` from `Outcome.String()` never inferred); add `internal/report/convert_test.go` table-driven over hand-built `model` values for each kind and each absence rule, plus one binary payload containing `\x00` round-tripping through base64 → commit `feat(report): convert parsec items into records (#50)`
- [X] T005 [P] Create `internal/report/source.go` with `Locate(path string) (run.Location, error)` (empty path → `run.DefaultResultsRoot`, otherwise `run.Find`) and `internal/report/errors.go` with the exported error helpers the command maps: `IsNotFound(err)` naming the searched directory, `IsUnreadable(err)` for `*fs.PathError`; add `internal/report/source_test.go` over the corpus: a run directory → `FoundByPath`, its `simulation.log` → same location, a copy of a run directory renamed `simulation.log` → `FoundByPath` (the directory is examined for a log inside it), a run directory that also holds run directories beneath it → `FoundByPath` (a run, never a results root), `lastrun/results` → `FoundByLastRun` naming `corpussimulation-20260909022708912`, a copy of that root without `lastRun.txt` (in `t.TempDir()`) → `FoundByNewest`, a copy whose `lastRun.txt` names a deleted run → `FoundByNewest`, an empty temp directory → not-found naming it, an unreadable directory (`os.Chmod` 0, skipped when running as root or on Windows) → unreadable never not-found → commit `feat(report): locate a Gatling run (#50)`
- [X] T006 Add `Open(loc run.Location) (*Source, error)` to `internal/report/source.go` returning the open file and a `simlog.RunReader` (buffered read), with `Close()`; extend `internal/report/errors.go` with the parsec mappings: `*gatling.FormatError` → "not a Gatling simulation.log: <path>", `*gatling.VersionError` → parsec's text (names version and range), `*run.NotFoundError`, `*fs.PathError` passed through; add `internal/report/source_open_test.go`: each corpus log opens and `Run().ToolVersion` matches its directory name; `3.15.1/index.html` → not-a-log; a synthetic text header `RUN\tio.x.Sim\tsim\t1700000000000\t \t3.10.0\n` in `t.TempDir()` → version error quoting `3.11.5 through 3.12.0`; the 3.13.1 log with bytes 5–10 patched to `3.13.0` → version error; the same patched to `3.99.9` and the text header with `3.99.0` → opens with exactly one `Run().Warnings` entry → commit `feat(report): open a run through parsec's version gate (#50)`
- [X] T007 Create `internal/report/write.go` with `Write(ctx context.Context, rd simlog.RunReader, w io.Writer) (Summary, error)`: header first, then each item converted and encoded (`json.Encoder`, `SetEscapeHTML(false)`, one `\n` per record) before the next `Next`, through a 64 KiB `bufio.Writer` flushed at the end and before every error return; `ctx.Err()` checked each 1024 items; `Summary{Requests, Groups, Users, Errors, Assertions int; Truncated *gatling.TruncationError}`; on `*gatling.TruncationError` return the summary and a wrapped error saying where the log was cut and that the emitted records are what the run recorded; on `*gatling.SyntaxError` a wrapped error saying the N records already emitted do not form a complete run; on a write error stop with that error once; add `internal/report/write_test.go` with a `-update` flag that rewrites goldens, golden output `internal/report/testdata/golden/<version>.jsonl` for the five corpus versions (generate once, read them, commit them), the per-kind counts 36/12/12/6 and 102/12/12/6, every line parsing as a standalone JSON object with the first `kind` = `run`, the 3.15.1 log cut at 2000 bytes → 62 records plus header on the writer and a truncation error, a corrupted middle byte → syntax error wording, a synthetic text log holding only the RUN header, two USER lines and one ERROR line → header plus three records, zero requests and no error, a writer failing on the third write → exactly one error and no further writes, a cancelled context → stops → commit `feat(report): write a run as JSON Lines (#50)`

**Checkpoint**: `internal/report` converts, locates, opens and writes every corpus run;
`go test -race ./internal/report` green; coverage of the package ≥ 90%.

---

## Phase 3: User Story 1 — Report a named run for any supported Gatling version (Priority: P1) 🎯 MVP

**Goal**: `galaxio report gatling <run>` writes the record stream for a text or binary log.

**Independent Test**: `runCLI("report", "gatling", <3.12.0 dir>)` and the same for 3.15.1
exit 0 with 36 and 102 request lines; the log path and its directory produce identical
output; omitting the tool exits 2.

- [X] T008 [US1] Rewrite `cmd/galaxio/report.go` as the operational command: `Use: "report <tool> [PATH]"`, `Args: cobra.MaximumNArgs(2)` (0 args → `cmd.Help()`, exit 0, unchanged from v0.12.0), tool validated against the fixed list `[gatling]` with `UsageError` `unsupported tool "<x>": accepted tools: gatling`; `reportOptions{Tool, Path string; Quiet, Verbose bool; Stdout, Stderr io.Writer}`; `runReport(ctx, opts) (reportOutput, error)` calling `report.Locate`, `report.Open`, `report.Write`, wrapping parsec/IO failures into `RuntimeError` and returning `reportOutput{Dir, Log string; Found run.FoundBy; Summary report.Summary}`; keep the `Short`/`Long` text and drop "Operational subcommands are introduced separately"; create `cmd/galaxio/report_test.go` with `runCLI` tests through the corpus at `../../internal/report/testdata/corpus/gatling`: 3.12.0 and 3.15.1 directories → exit 0, first stdout line `"kind":"run"`, request-line count 36 and 102, empty `Error:` on stderr; `simulation.log` path equals directory output; `report <dir>` (no tool) → exit 2 naming `gatling`, empty stdout; `report jmeter <dir>` → exit 2; update `TestReportCommand` and `TestReportCommandRejectsArguments` in `cmd/galaxio/root_test.go` for the new help text (assert `report <tool> [PATH]`; the `-o` line is asserted from T013) and the "unsupported tool" message → commit `feat(report): make galaxio report an operational command (#50)`
- [X] T009 [US1] Surface a newer-than-verified version: in `cmd/galaxio/report.go` print `report: warning: <version>: <reason>` on stderr for each `Run().Warnings` entry (not suppressed by `--quiet`, since it is not informational) and keep exit 0; add a `runCLI` test in `cmd/galaxio/report_test.go` over a synthetic text log with version `3.99.0` written to `t.TempDir()` asserting exit 0, the warning on stderr, and `"warnings"` in the header line → commit `feat(report): report an unverified Gatling version (#50)`

**Checkpoint**: MVP — a named run of any supported version becomes JSON Lines.

---

## Phase 4: User Story 2 — Report the latest run without naming it (Priority: P1)

**Goal**: `galaxio report gatling` with no path finds the run and says which one it read.

**Independent Test**: a results root with three runs is reported by `lastRun.txt`, then by
newest once that file is removed; from a project directory with `target/gatling` no path
is needed; a root without runs exits 1 naming it.

- [X] T010 [US2] In `cmd/galaxio/report.go` print `report: reading <dir> (found by <rule>)` on stderr before writing unless `--quiet`, and under `--verbose` print `report: <format> log, Gatling <version>: N requests, N groups, N user events, N errors` after the stream; add `runCLI` tests in `cmd/galaxio/report_test.go`: `lastrun/results` → stderr says `found by lastRun.txt` and the `…022708912` directory; a temp copy without `lastRun.txt` → `found by newest`; `t.Chdir` into a temp project holding `target/gatling/<run>` and `report gatling` with no path → exit 0 and the same diagnostic; an empty temp directory as path → exit 1 and `no Gatling run under <dir>`; `--quiet` → empty stderr on success; `--verbose` → the count line with 102 requests → commit `feat(report): find the latest run and say which one was read (#50)`

**Checkpoint**: The common case needs no argument beyond the tool.

---

## Phase 5: User Story 3 — Stream a multi-gigabyte log (Priority: P2)

**Goal**: Memory does not grow with the log; the first line arrives immediately; a closed
pipe ends the process quietly.

**Independent Test**: `TestWriteAllocations` ≤ 4 allocations per record; `BenchmarkWrite`
over a synthetic multi-million-record log reports flat heap; the built binary piped into
`head -n 1` exits without a diagnostic flood.

- [X] T011 [P] [US3] Add `internal/report/synth_test.go` with a replay `io.Reader` that emits the 3.12.0 corpus header once and then its body lines N times with shifted timestamps (no file written), `TestWriteAllocations` in `internal/report/write_test.go` asserting `testing.AllocsPerRun` over 10 000 records ≤ 4 per record, and `BenchmarkWrite` (`ReportAllocs`, records/s and MB/s via `b.SetBytes`) over enough replayed records that the stream exceeds 2 GiB (about 16 000 000 lines of the 3.12.0 body; run with `-benchtime=1x`), with a `runtime.ReadMemStats` check that `HeapInuse` stays under 32 MiB so SC-003's two-gigabyte case is what is measured; fix any allocation found in `internal/report/write.go` or `convert.go`; record the measured figures in `specs/004-report-records/quickstart.md` §3 → commit `test(report): prove bounded memory and allocations per record (#50)`
- [X] T012 [P] [US3] Add `cmd/galaxio/report_integration_test.go` behind `//go:build integration`: build the binary into `t.TempDir()` with `go build`, run it against the 3.15.1 corpus with stdout piped into `head -n 1` through `sh -c`, assert the pipeline's first line is the header, the process ends within 5 s and stderr carries no `Error:` line; run it twice on the same log and assert byte-identical output (FR-023); `t.Skip` with a reason when `sh` or `head` is unavailable → commit `test(report): integration test for pipes and determinism (#50)`

**Checkpoint**: SC-003 evidenced by benchmark and integration test; figures in quickstart.

---

## Phase 6: User Story 4 — Fail with a message the engineer can act on (Priority: P2)

**Goal**: Every failure names what was at fault; usage and runtime failures are told apart
by the exit code; `-o` is reserved and rejected.

**Independent Test**: each scenario in spec US4 exits with its documented code, stdout is
empty or holds only complete records, and stderr names the path, directory, version, tool
or format.

- [X] T013 [US4] Add the `-o/--output` flag to `cmd/galaxio/report.go`: parse a comma-separated list; known names `stats`, `global_stats` (→ `UsageError` `report format "<name>" is not available yet: it arrives with milestone v0.15.0 Legacy stats.json`), `yml` (→ `UsageError` `report format "yml" is not available yet: the OpenNFR YAML report is postponed`), anything else (→ `UsageError` `unknown report format "<name>": known formats: stats, global_stats, yml`); help text says the stream is written when no format is requested; extend the help assertion in `TestReportCommand` (`cmd/galaxio/root_test.go`) to expect the `-o, --output` line (FR-024); add `runCLI` tests in `cmd/galaxio/report_test.go` for `-o stats`, `-o stats,global_stats`, `-o yml`, `-o json`, `-o text` → exit 2, empty stdout, the exact message → commit `feat(report): reserve -o for report formats (#50)`
- [X] T014 [US4] Add the remaining failure tests to `cmd/galaxio/report_test.go` through `runCLI`, fixing `cmd/galaxio/report.go` or `internal/report/errors.go` where a message falls short: synthetic text `3.10.0` → exit 1 naming version and range; 3.13.1 bytes patched to `3.13.0` in `t.TempDir()` → exit 1; 3.15.1 cut at 2000 bytes → exit 1, stdout holds header plus 62 complete lines, stderr says cut short and that emitted records are what the run recorded; `/nonexistent/path` → exit 1 naming the path and never "no run"; `3.15.1/index.html` → exit 1 not-a-log naming the path; a corrupted log → exit 1 saying emitted records are not a complete run; three positional arguments → exit 2; `--bogus` → exit 2; `execute` with a stdout writer that fails after two writes → exit 1 and exactly one `Error:` line → commit `test(report): cover every failure path and exit code (#50)`

**Checkpoint**: All four stories complete; `go test -race ./...` green.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T015 [P] Document the command in `README.md` § Reporting Ecosystem: replace "This milestone exposes the namespace and its help only…" with `galaxio report <tool> [PATH]`, the tool rule, the PATH rules, the `-o` reserved formats, the stderr diagnostics, the exit codes, a table of record kinds and keys, a `jq` example, and a link to `specs/004-report-records/contracts/records.md` as the full schema → commit `docs(readme): document galaxio report (#50)`
- [X] T016 Run and record the gates in `specs/004-report-records/quickstart.md` §6: `test -z "$(gofmt -l .)"`, `go vet ./...`, `go test -race -coverprofile=coverage.out ./...` with the total coverage figure (≥ 80%), `go mod tidy && git diff --exit-code -- go.mod go.sum`, `go mod verify`, `go test -tags=integration -race -count=1 ./...`, `govulncheck ./...`; paste the observed lines under §6 and tick this box → commit `docs(speckit): record 004-report-records gate evidence (#50)`
- [ ] T017 Walk `specs/004-report-records/quickstart.md` §1–§5 with the built binary and record each observed output next to its expected value; tick this box; then push the branch and open (or update) the milestone PR against `main` assigned to milestone `v0.13.0 Report dump` with `Closes #50` in the body, and leave it open for maintainer review → commit `docs(speckit): record 004-report-records quickstart evidence (#50)`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 first (spec-first commit); T002 needs only the parsec module
  cache (`go mod download github.com/galax-io/parsec@v0.1.0`, or the probe from planning).
- **Foundational (Phase 2)**: T003 first (it brings parsec into `go.mod`); T004 and T005
  in parallel after T003; T006 after T005; T007 after T004 and T006. Blocks every story.
- **US1 (Phase 3)**: T008 after T007; T009 after T008.
- **US2 (Phase 4)**: T010 after T008 (it edits the same command file).
- **US3 (Phase 5)**: T011 after T007; T012 after T008. T011 and T012 in parallel with each
  other and with Phase 4 and Phase 6 (different files) — but commits stay one per task.
- **US4 (Phase 6)**: T013 after T010 (same file, keeps the history linear); T014 after T013.
- **Polish (Phase 7)**: T015 after T013 (the flag surface is final); T016 after T014, T011,
  T012; T017 last.

### User Story Dependencies

- **US1 (P1)**: needs only Phase 2. The MVP.
- **US2 (P1)**: needs US1's command file; independently testable through the results-root
  fixtures.
- **US3 (P2)**: needs the writer (T007) and, for the pipe test, the command (T008);
  independently testable by benchmark and integration test.
- **US4 (P2)**: needs the command; independently testable per failure scenario.

### Parallel Opportunities

```text
Phase 1:  T001 | T002
Phase 2:  T003 → (T004 | T005) → T006 → T007
Phase 3:  T008 → T009
Phases 4–6 after T008:  T010 → T013 → T014   |   T011   |   T012
Phase 7:  T015 (after T013) | T016 (after T014, T011, T012) → T017
```

Every task is still its own commit on one linear branch; "parallel" means the work does
not wait, not that two tasks share a commit.

---

## Implementation Strategy

### MVP First (User Story 1)

1. T001–T002: spec on the branch, fixtures on disk.
2. T003–T007: `internal/report` complete and green against every corpus version.
3. T008–T009: `galaxio report gatling <run>` streams records.
4. **STOP and VALIDATE**: quickstart §1 by hand; `go test -race ./...`.

### Incremental Delivery

1. + US2 (T010): no path needed → quickstart §2.
2. + US3 (T011–T012): memory and pipe evidence → quickstart §3.
3. + US4 (T013–T014): reserved `-o`, every failure path → quickstart §4.
4. + Polish (T015–T017): README, gate evidence, PR ready for review.

---

## Notes

- Each task = one commit, message as given, `Co-Authored-By` trailer per the session's
  attribution rule; `gofmt -w .` before each commit; `go build ./... && go test ./...`
  green at every commit.
- Golden files under `internal/report/testdata/golden/` are generated once by T007 with
  `-update`, read by a person, and committed; never regenerated to make a later test pass.
- The Principle I deviation (no `-o text|json`) is justified in `plan.md` Complexity
  Tracking; the proposed constitution amendment is a separate PR, not a task here.
- Do not compute any statistic anywhere in this milestone (`-o stats`, `global_stats`
  are v0.14.0/v0.15.0).
- The PR is opened by T017 and left open; only the maintainer merges it.
