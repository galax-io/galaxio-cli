# Tasks: Summarise a Finished Gatling Run

**Input**: Design documents from `/specs/005-report-summary/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [contracts/cli.md](contracts/cli.md),
[quickstart.md](quickstart.md)

**Tests**: Required by Constitution Principle III and never optional. Tests land in the same
commit as the change they cover: inside a task, write the test first, watch it fail, then
implement until green, then commit once. A task that only adds tests or evidence is green on
its own and still gets its own commit.

**Organization**: One milestone, one review boundary: the milestone PR on branch
`113-report-summary`, assigned to `v0.14.0 Report summary`, closing #51 when it lands. Every
task maps to exactly one green commit (`go build ./... && go test ./...`), and every commit
to one task; the commit line is given at the end of each task. Each commit carries its final
content — AGENTS.md forbids add-then-remove inside a PR — which is why the figures of US1 and
US2 land in `internal/report/` before the summary is printed once, in its final layout,
rather than story by story with a golden file rewritten on the way. Every task still carries
the story it serves, and "Stories" below maps each story to its tasks and its test.

**One prerequisite outside this pull request, met on 2026-09-17 by #115** (spec,
Clarifications; plan, Summary): the percentile clause of constitution Principle II is
amended first, in its own issue and its own pull request inside milestone `v0.14.0`, with
the wording proposed in [research.md](research.md) §13, gate II of
`.specify/templates/plan-template.md` updated with it. T001 and T002 touch no implementation
and may land before it. **No task from T003 on starts until that pull request is merged into
`main`** — it was, as #115, constitution v2.2.0.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with the other `[P]` tasks of the same phase.
- **[Story]**: Maps the task to US1–US6 from `spec.md`.
- Every task names the exact files it changes.

## Required reading (constitution "Engineering Guidance")

`golang-testing` before every task. `golang-refactoring` and `golang-design-patterns` before
T004 (the `Scan` restructuring, the options struct) and T014 (the tick). `golang-naming` and
`golang-documentation` before T004–T007 and T017 (the exported surface of `internal/report`,
the README). `golang-dependency-management` and `golang-pkg-go-dev` before T007 (pin the
module only after looking at its versions and advisories). `golang-cli` and
`golang-spf13-cobra` before T009–T011 and T015. `golang-error-handling` before T013 (the
joined runtime errors). `golang-safety` and `golang-security` before T009 and T015 (every
log is untrusted input; escape sequences are written to a terminal). `golang-benchmark`,
then `golang-performance`, before T012 and the benchmark of T015.

---

## Phase 1: Setup

- [X] T001 Commit the feature artifacts first: `specs/005-report-summary/` (spec, plan, research, data-model, contracts/cli.md, quickstart, checklists, tasks) and `.specify/feature.json` → commit `docs(speckit): add 005-report-summary spec/plan/tasks (#51)`
- [X] T002 [P] Copy what Gatling itself recorded for the corpus runs from the parsec v0.1.0 module cache (`$(go env GOMODCACHE)/github.com/galax-io/parsec@v0.1.0/testdata/corpus/gatling/<version>/`) into `internal/report/testdata/corpus/gatling/<version>/`, unchanged: `global_stats.json` for 3.11.5 and 3.12.0, `js/global_stats.json` for 3.13.1, and `console.txt` for 3.13.1, 3.14.9 and 3.15.1. Do **not** copy any `stats.json`: nothing reads the rows per request before #52. Add one row per file to `internal/report/testdata/corpus/gatling/PROVENANCE.md` (source module and version, MIT, which run it belongs to, that it is Gatling's own output) and keep `.gitattributes` treating the JSON and text files as they are → commit `test(report): add what Gatling recorded for the corpus runs (#51)`

**Checkpoint**: spec on the branch, reference data on disk, `go test ./...` unchanged and green. Stop here until the Principle II amendment is merged.

---

## Phase 2: Foundational — one walk, one summary

- [X] T003 After the amendment is merged: rebase `113-report-summary` onto `main`, re-read `.specify/memory/constitution.md` Principle II as amended, and re-check `specs/005-report-summary/plan.md` against it: set gate II of the Constitution Check to PASS with the clause it now passes quoted, rewrite the "Blocked on one prerequisite" paragraph of the Summary and the post-design re-check to say the amendment has landed (version and pull request named), and remove the first row of Complexity Tracking, keeping the second, which records the split the maintainer asked for. If the merged wording differs from research.md §13, change §13 to quote what was ratified → commit `docs(speckit): re-check the 005 plan against the amended Principle II (#51)`
- [X] T004 Restructure the walk so that the statistics fold over the pass v0.13.0 already makes, with no second pass and no tally-only path left behind. New `internal/report/summary.go`: `Options` (percentile ranks, band boundaries in whole milliseconds) with `DefaultOptions()` returning 50, 75, 95, 99 and 800, 1200, and a validating constructor or method that refuses a rank not above 0 or above 100, an empty rank list, negative or non-increasing boundaries, and that sorts ranks ascending and drops duplicates; `Summary`, which for now carries the `Tally` and the `Options` in force. `internal/report/scan.go`: `Scan(ctx, rd, opts) (Summary, error)` replaces `Scan(ctx, rd) (Tally, error)` — same loop, same every-1024-items context check, same wrapped errors for a truncation and for any other read failure, the summary of what was read returned beside either. `internal/report/source.go`: `Source.Scan(ctx, opts)` follows, still refusing a second walk with `ErrSpent`. `internal/report/doc.go`: the package reads a run, counts it and summarises it. `cmd/galaxio/report.go`: `reportOutput` carries the `report.Summary` instead of the bare tally and `formatReport` reads `out.Summary.Tally`; output unchanged. Update every existing test and `internal/report/scan_bench_test.go` to the new signature, and add table tests for `Options` validation (each refusal, ordering, duplicates, defaults). No behaviour a user can see changes → commit `refactor(report): return a summary that carries the tally (#51)`

**Checkpoint**: `go test -race ./...` green, `galaxio report gatling` prints exactly what v0.13.0 printed.

---

## Phase 3: The figures (US1, US2)

- [X] T005 [US1] Compute the exact figures of the whole run. New `internal/report/moments.go`: the figures of one outcome per [data-model.md](data-model.md) — count, timed, minimum and maximum in whole milliseconds, the sum as a 128-bit pair and the sum of squares as three 64-bit words, both maintained with `math/bits`, a negative duration counting as no recorded end — and the arithmetic done when the figures are read, in integers with `math/big`: mean `floor((2·sum + timed) / (2·timed))`; population deviation about the unrounded mean, `k = floor(sqrt(num/den))` raised by one when `4·num >= (2k+1)²·den`, with `num = timed·Σx² − (Σx)²` and `den = timed²`; every derived figure **absent** (a value plus `ok bool`, never 0) while `timed` is zero. `internal/report/summary.go`: `Summary` gains the ok and failed outcomes, a method giving the figures of all requests by adding counts and sums and combining extremes (nothing stored for it), the count of requests with no recorded end, the run span in whole seconds `(ms + 999) / 1000` from `Tally.Bounds` with a state for "span is zero" and "bounds unresolved", and the three rates `count / seconds`, absent when there is no divisor and 0 for a count of zero over a known span. `internal/report/scan.go`: a sample feeds the outcome `model.OutcomeSuccess`/`model.OutcomeFailure` names, using `Sample.Duration.Get()` — a sample without a duration is counted and takes part in no timing figure; a sample of any other outcome stays in `Tally.Unknown` and feeds nothing; a group traversal feeds nothing (FR-015). Tests: table-driven arithmetic with the case that tells each rule from the rejected one — three requests of 0 ms and three of 1 ms give mean 1 and deviation 1; a mean of exactly 2.5 gives 3 (banker's rounding would give 2); a set where the sample deviation rounds differently from the population one; one request (deviation 0, every figure its time); equal requests; sums that carry past 64 bits and past 128 bits, from accumulators set near those boundaries and checked against `math/big`; a failure reaching all-request figures and no ok figure; an empty outcome absent; no-end and lost-outcome samples through the fake reader of `internal/report/scan_test.go`. Corpus test in `internal/report/summary_test.go`: for 3.11.5, 3.12.0 and 3.13.1 read `global_stats.json` into a struct that declares **only** `numberOfRequests`, `minResponseTime`, `maxResponseTime`, `meanResponseTime`, `standardDeviation` and `meanNumberOfRequestsPerSecond` (total/ok/ko each) — no `percentiles1`–`4` field may appear in any `_test.go` (FR-019) — and compare every one with the summary; for 3.13.1, 3.14.9 and 3.15.1 compare with a table in the test that cites the line of `console.txt` each value was read from → commit `feat(report): compute the exact figures of a run (#51)`
- [X] T006 [US1] Count the response-time bands. `internal/report/summary.go`: four counters — ok responses under the lower boundary, from it to under the upper one, at or above the upper one, and failed requests whatever their time — fed from the same place T005 feeds the outcomes, using `Options` boundaries; each share as `float64(count) / float64(total) * 100` in exactly that association, absent when the run holds no request. How long a band's bar is drawn is the console's business and lands with it in T009. A timed-less ok request belongs to no timing band and is not failed: test and document that the three timing bands then add up to the timed ok requests. Tests: boundaries are lower-inclusive (799, 800, 1199, 1200 ms at the defaults); a failed request of 5 000 ms lands in `failed` only; `12/36*100` equals `33.33333333333333` and not `33.333333333333336`; custom boundaries through `Options`. Extend the corpus test of T005 with `group1`–`group4` count and percentage from `global_stats.json` and with the distribution lines of `console.txt` → commit `feat(report): count the response-time bands (#51)`
- [X] T007 [US2] Estimate percentiles with the digest the maintainer named, adding the dependency in the commit that imports it: `go get github.com/caio/go-tdigest/v5@v5.0.0 && go mod tidy`, then check `go.mod` gained one `require` line and `go.sum` six lines, two of them the checksums of the library's test-only dependencies (research.md §1), and that `go version -m` on the built binary lists the one new module. New `internal/report/percentiles.go`: one digest per outcome at the library's defaults (compression 100, its own constant-seeded generator), created on the first timed request of that outcome and fed every timed request's milliseconds; percentiles read with `Quantile(rank/100)` and rounded half up to a whole millisecond; all requests read from a **fresh** digest at the defaults into which the ok digest and then the failed one are merged at the moment of reading — never a clone, whose seed the library draws from the original's generator, and never a third digest (research.md §3); an outcome with no timed request has no digest and every percentile absent; ranks come validated from `Options`, so the library's panic on a quantile outside [0, 1] cannot be reached and no `recover` is written. Tests in `internal/report/percentiles_test.go`: the table of research.md §2 pinned for all five corpus runs and all three outcomes, the 3.13.1 value of **1427** for the 95th percentile of all requests included and commented as the known divergence with the link to caio/go-tdigest#42 (FR-021), so that taking the upstream read later fails this test on purpose; two reads of every corpus run give identical percentiles; the merged read equals a digest fed every request directly on the corpus runs; a single request; equal requests; ranks outside a percentile's range; and reading the percentiles of all requests every hundred requests of a long walk leaves every quantile of every digest as an unread walk leaves it → commit `feat(report): estimate percentiles with a t-digest (#51)`

**Checkpoint**: `internal/report` produces every figure of [data-model.md](data-model.md); nothing a user sees has changed yet.

---

## Phase 4: The summary on the console (US1, US2) 🎯 MVP

- [X] T008 [US1] Call failed requests `failed`. `cmd/galaxio/report.go`: the requests line of `formatReport` prints `%d (%d ok, %d failed)` (FR-005); no other line of the description changes. Update what quotes the old word: the example block in `README.md` § "Read a finished run", `cmd/galaxio/report_test.go`, `cmd/galaxio/report_integration_test.go`, and any `ko` in the expectations or comments of `internal/report/scan_test.go`. Add a command test asserting that no line of the report for any of the five corpus runs contains the word `ko` in any case. The commit body says that one word of v0.13.0's text report changed, because the commit message is the changelog entry → commit `feat(report): say failed, not ko, in the run description (#51)`
- [ ] T009 [US1] Print the summary. New `cmd/galaxio/report_summary.go`: `formatSummary` renders the four parts of [contracts/cli.md](contracts/cli.md) § "The layout" from a `report.Summary` and one boolean, colour or not — the headline (three segments six spaces apart on one line while that line fits 100 columns, one segment per line beyond that, measured in characters, never from the terminal); the `response time, ms` table with a 20-wide label column, figures right-aligned in 7 and a column growing to keep two spaces before a longer value, rows `all`, `✓ ok`, `✗ failed`, columns `min`, `mean`, `std`, one `pNN` per rank ascending (`p99.9` for 99.9), `max`; the four band lines as bar, share, count, label — the bar `floor((40·count + total) / (2·total))` cells of 20 as a pure function, at least 1 for a band that holds a request and at most 19 for a band that does not hold them all — with the labels `ok under <LOW> ms`, `ok <LOW> to <HIGH> ms`, `ok <HIGH> ms and over`, `failed`; and the closing line `times in ms · percentiles are galaxio's t-digest estimates, interpolated; not Gatling's`. Numbers: whole milliseconds; rates and shares with at most two decimals, trailing zeros removed, a point, no thousands separator; an absent figure is `-`, never `0`. Colour: SGR 32 for `✓ … ok`, SGR 31 for `✗ … failed` and the filled part of the failed bar only when something failed, SGR 2 for the table heading, the unfilled part of every bar and the closing line, SGR 0 after each; with colour off not one escape byte. `cmd/galaxio/report.go`: `reportOptions` gains `Color bool`, set by the cobra wrapper when standard output is a character device (`os.ModeCharDevice`), `TERM` is set and is not `dumb`, and the root's merged `--no-color`/`NO_COLOR` option is off (add the accessor beside `isQuiet` in `cmd/galaxio/root.go`); `runReport` scans with `report.DefaultOptions()`, writes the description, a blank line and the summary in one write, keeps printing both for a log cut short and neither for a damaged log, and keeps `--quiet` silent; the command's `Long` help says what the summary is and that the command writes no file. Tests: the bar rule at 0, a sliver, a half cell, 19.6 of 20 and all; golden files `cmd/galaxio/testdata/report/<version>.summary.golden` for all five corpus runs and `3.13.1.summary.color.golden`, the 3.13.1 one equal to the block in the contract byte for byte; every acceptance scenario of US1 through `runCLI`; a directory holding only the log prints what the full directory prints; two runs of one log are identical; hand-built summaries for a run with no request (every share `-`), nothing failed (`0`, `0 %`, `0` and `-`, and no SGR 31 with colour on), an untimed run (every rate `-`), a headline past 100 columns, a value wider than its column; no row for a request or group appears for a run with several; **the command creates, changes and removes no file** — snapshot the run directory and an empty working directory entered with `t.Chdir` before and after → commit `feat(report): print the summary of a run (#51)`

**Checkpoint**: MVP — `galaxio report gatling $C/3.13.1` prints the block of [contracts/cli.md](contracts/cli.md) at the default ranks and boundaries. US1 is complete and testable on its own.

---

## Phase 5: Options (US2, US3)

- [ ] T010 [US2] Add `--percentiles`. `cmd/galaxio/report.go`: a `pflag.Value` that validates in `Set`, as `-o` does, so a bad value is refused while it is parsed — before `--help` is answered and before any work: comma-separated ranks, each a number above 0 and at most 100, default `50,75,95,99`, no shorthand; the error reads `percentile rank "<value>" is not a number above 0 and at most 100`, so cobra's wrapper yields the message of [contracts/cli.md](contracts/cli.md) and exit 2 with nothing on standard output; an empty list is refused; ranks reach `runReport` through `reportOptions.Percentiles` and the summary through `report.Options`, reported once each, ascending. Flag help names the default. Tests through `runCLI`: `99.9,90` prints `p90` then `p99.9` and no other rank; `50,50,99` prints two; `0`, `101`, `abc`, `50,abc` and the empty value each exit 2 quoting the value, with and without `--help`; the default prints the four; a non-integer rank labels itself without a trailing zero → commit `feat(report): add --percentiles (#51)`
- [ ] T011 [US3] Add `--bounds`. `cmd/galaxio/report.go`: a second `pflag.Value` validating in `Set`: `LOW,HIGH`, two whole non-negative numbers of milliseconds, the second greater than the first, default `800,1200`, no shorthand; the error reads `boundaries are two whole numbers of milliseconds, the second greater than the first`; the pair reaches the summary through `reportOptions.Bounds` and `report.Options`. Flag help names the default. Tests through `runCLI`: `5,1000` on 3.13.1 prints the labels `ok under 5 ms`, `ok 5 to 1000 ms`, `ok 1000 ms and over`, keeps `failed` at 18 (17.65 %), and its four counts add up to 102 — the three ok counts asserted against a count of the log's own response times made in the test; `1200,800`, `800,800`, `-1,5`, `800`, `1,2,3`, `a,b` and `1.5,2` each exit 2 quoting the value; the two flags together; the default unchanged → commit `feat(report): add --bounds (#51)`

**Checkpoint**: US2 and US3 complete.

---

## Phase 6: A run of any size (US4)

- [ ] T012 [P] [US4] Hold the summary to the memory goal of v0.13.0. `internal/report/scan_test.go`: `TestSummaryMemoryDoesNotGrowWithTheLog` replaces `TestScanMemoryDoesNotGrowWithTheLog` — same 16 MiB and 256 MiB replays through `reporttest.Replay`, same 32 MiB goal and 8 MiB slack, now with the digests and accumulators of T005–T007 in the measured heap; add a second test that walks one million samples with one million distinct request names through the fake reader and asserts the heap a summary holds afterwards is what it holds for one name, because nothing is kept per name (FR-033). `internal/report/scan_bench_test.go`: `BenchmarkScan` keeps reporting throughput and allocations at the new cost of one digest insertion per request; record the figure in quickstart.md §4 when T020 runs, not here → commit `test(report): hold the summary to the memory goal (#51)`

---

## Phase 7: Failures a script can act on (US5)

- [ ] T013 [US5] Fail, by exit code and by name, on a run that cannot be summarised completely. `cmd/galaxio/report.go`: a function of the `report.Summary` and the log path — testable without a log — that returns the runtime failures of [contracts/cli.md](contracts/cli.md) § "Exit codes": `<log>: the run spans no time (<start> .. <end>): no request rate can be computed` for a span of exactly zero, and `<log>: <N> requests have an outcome the source lost: they are neither ok nor failed and the summary does not add up`; `runReport` prints the summary first — every rate `-` for the zero span — then returns them as one `RuntimeError`, joined with `errors.Join` to each other and to a log cut short when several hold, keeping the error chain so `errors.As` still finds the truncation. Bounds the library cannot resolve leave every rate `-` and exit 0. After the scan and before the report, write `report: warning: <N> requests have no recorded end and take part in no timing figure` to standard error when the count is above zero — also under `--quiet`, because it qualifies the numbers. Tests: the function against hand-built summaries for each case and each combination; a command test on a hand-built text-format log under `cmd/galaxio/testdata/report/zero-span/simulation.log` — the header lines of the 3.12.0 recording and a single request whose start and end are the same instant — asserting the summary, `-` rates, the message and exit 1; a cut-short log still printing the summary of what it held with exit 1, and a damaged log printing none; `-o` with any value still exit 2 exactly as in v0.13.0 with the two new flags present. The commit body names the two exit codes that change from 0 to 1 and that no Gatling log is known to produce either → commit `feat(report): fail on a run that cannot be summarised completely (#51)`

**Checkpoint**: US5 complete; every failure scenario of the spec exits with its documented code.

---

## Phase 8: Progress while reading (US6)

- [ ] T014 [US6] Let a read say how far it has got. `internal/report/source.go`: `Open` wraps the file in a counting reader **after** format detection and records the size the file had when it was opened (`Stat`; zero or not a regular file means unknown); `Source` exposes the bytes read so far and that size. `internal/report/scan.go`: `Scan` accepts a tick — nil for none — and calls it with the summary so far at the existing every-1024-items check, before the context check returns; no goroutine, no timer, and nothing retained. Tests: the bytes read never pass the size and reach it at the end for every corpus run; an unknown size is reported as unknown; the tick fires every 1024 items and sees counts that only grow; a nil tick changes nothing; the figures read from the summary inside a tick equal what a full read of the same prefix gives → commit `feat(report): measure how far a read has got (#51)`
- [ ] T015 [US6] Show the progress block. New `cmd/galaxio/report_progress.go`: the six lines of [contracts/cli.md](contracts/cli.md) § "The progress block" — spinner frame advancing per redraw, the log's base name through `printable` and cut to 24 columns, a 30-cell bar of `━` and `─`, the percentage right-aligned in 3 and never above 100, the time left as `m:ss` or `h:mm:ss` from the second draw; the faint second line `figures so far · times in ms · percentiles are t-digest estimates, interpolated`; then the header and the rows `all requests`, `  ✓ ok`, `  ✗ failed` with a 20-wide label, `count` and `share` in 8, and `min`, `mean`, `p50`, `p95`, `p99`, `max` in 7, counts exact below 10 000 and abbreviated `12.3k`/`40.9M`/`1.2G` above, shares with one decimal, `-` for an outcome with no timed request, no rate, fixed columns whatever `--percentiles` asks, every line cut at 79 columns; an unknown size draws the spinner and the name alone on the first line. Drawn from the tick of T014: first at 500 ms into the read, then when 200 ms have passed since the last draw. Redraw is `ESC [ 6 A` then `ESC [ 2 K` and the line, six times; removal is `ESC [ 6 A` then `ESC [ J`; no other sequence except the SGR codes of T009 under the same colour rule applied to standard error; no terminal mode is changed. `cmd/galaxio/report.go`: `reportOptions` gains the two things a test must control — whether standard error can redraw in place (character device, `TERM` set and not `dumb`, not `--quiet`) and the clock — and `runReport` erases the block before the no-end warning, before any error, before the report and before returning, an interrupt included, since it arrives as the cancelled context the walk already returns on. Tests in `cmd/galaxio/report_progress_test.go` with both injected and a clock that advances on every reading: the frames asserted as text, control sequences included; nothing drawn for a read that ends within 500 ms, for a standard error that cannot redraw, for `TERM=dumb`, under `--quiet`; the block erased before the error of a cut-short and of a damaged log and on a cancelled context; the percentage capped at 100 on a log that grew; standard output and the exit code identical with and without the block for every corpus run; abbreviations at 9 999, 10 000, 999 950 and 1e9. Benchmark `BenchmarkRunReport/progress={off,on}` over a 64 MiB replay in the same file, for the 5 % of SC-010 — recorded in quickstart.md §6 by T020, not gated → commit `feat(report): show a progress block while reading (#51)`

**Checkpoint**: all six stories complete; `go test -race ./...` green.

---

## Phase 9: Integration, documentation and evidence

- [ ] T016 Extend `cmd/galaxio/report_integration_test.go` (behind `//go:build integration`) on the built binary: one hundred summaries of the 3.15.1 run are byte-identical (SC-003); piped output carries no escape byte and standard error is empty for a corpus run; the replayed run of 720 000 requests reports its counts in the headline and the description alike; the run directory and an empty working directory are left as they were; exit codes for a bad `--percentiles`, a bad `--bounds`, `-o stats`, a cut-short log and the zero-span fixture of T013 → commit `test(report): integration tests for the summary (#51)`
- [ ] T017 [P] Document the summary in `README.md` § "Reporting Ecosystem": retitle "Read a finished run" so it covers the summary, replace the example with the block of [contracts/cli.md](contracts/cli.md), and add what each part says and how every figure is defined — count, share, rate as the count over the run's span in whole seconds rounded up, minimum and maximum, the half-up mean, the population deviation, the bands and their boundaries' inclusivity, `-` for a figure that does not exist; `--percentiles` and `--bounds` with their defaults and refusals; colour, `--no-color` and `NO_COLOR`; the progress block, where it shows and where it never does; **the percentile limitation** — the default quantile interpolates, 1427 ms printed for the recorded 3.13.1 run where the request at the rank took 1502 ms, [caio/go-tdigest#42](https://github.com/caio/go-tdigest/pull/42) as the change that removes it, and that percentiles are this tool's estimates and are never Gatling's (FR-020); that the command writes no file and prints no row per request or group, both being what the reserved `-o` products are for; the two new exit-1 cases in § "Exit Codes"; and the one-line description under § "Usage" → commit `docs(readme): document the report summary (#51)`
- [ ] T018 With the maintainer's go-ahead for the text, amend issue #51 so that the issue this PR closes describes what was built: a dated "Amended" section recording that `-o json` is superseded by constitution v2.1.0 Principle I, that figures per request and per group and the selectable wall clock of a group move to the `-o` products (#52), that percentiles come from `caio/go-tdigest` at its default quantile with the known divergence and the upstream pull request instead of the exact histogram of the 2026-09-16 amendment, that the console layout is this tool's own with `ok` and `failed`, and restated acceptance criteria. Then change the Assumptions bullet "Issue #51 needs amending to match" in `specs/005-report-summary/spec.md` to say when it was amended and link the comment → commit `docs(speckit): record the amendment of issue #51 (#51)`
- [ ] T019 Run and record the gates in `specs/005-report-summary/quickstart.md` §8 at the branch's final commit: `gofmt -l .`, `go vet ./...` with and without the integration tag, `go test -race -coverprofile=coverage.out ./...` with the total coverage figure (≥ 80 %), `go mod tidy && git diff --exit-code -- go.mod go.sum`, `go mod verify`, `go test -tags=integration -race -count=1 ./...`, `govulncheck ./...` run by hand → commit `docs(speckit): record 005-report-summary gate evidence (#51)`
- [ ] T020 Walk `specs/005-report-summary/quickstart.md` §1–§7 with the built binary and record each observed output next to its expected value, with the date — the memory and throughput figures of §4 and the benchmark pair of §6 included, and §6's terminal steps (the block seen, `TERM=dumb`, `--quiet`, Ctrl-C leaving the terminal as it was) observed in a real terminal. Tick this box, push the branch and open or update the milestone PR against `main`, assigned to milestone `v0.14.0 Report summary` with `Closes #51` in the body and the two published changes named there (the `failed` wording, the two exit codes), and leave it open for maintainer review → commit `docs(speckit): record 005-report-summary quickstart evidence (#51)`

---

## Stories

| Story | Tasks | Independent test |
|---|---|---|
| US1 — the numbers of a run without the HTML report (P1) | T005, T006, T008, T009 | every non-percentile figure of all five corpus runs equals what Gatling recorded; the 3.13.1 golden equals the contract's block; no file is written, no `ko`, no row per request |
| US2 — percentiles that say what they are (P2) | T007, T010 | percentiles pinned per corpus run, 1427 included; identical between two reads; the closing line says whose they are; `--percentiles 99.9,90`; bad ranks exit 2 |
| US3 — bands at the engineer's own boundaries (P2) | T011 (over T006) | `--bounds 5,1000` counted against the log's own response times; bad boundaries exit 2 |
| US4 — a run of any size (P2) | T012 | 16 MiB and 256 MiB replays under 32 MiB; a million names cost what one name costs |
| US5 — failures a script can act on (P2) | T013 | zero span, lost outcome, no recorded end, cut short, damaged: each its code and its message |
| US6 — a long read is seen to progress (P3) | T014, T015 | frames under a fake clock; silence off a terminal; standard output byte-identical with and without the block |

## Dependencies & Execution Order

- **T001** first (spec-first). **T002** needs only the parsec module cache and runs beside it.
- **The Principle II amendment** merges between T002 and T003. It is its own issue and pull
  request; nothing in this list belongs to it.
- **T003** is the first task after it and blocks everything below. **T004** after T003.
- **T005 → T006 → T007** in that order: each adds to `internal/report/summary.go`. T007 is
  US2's and still precedes the printing of US1, so that the summary is printed once, in its
  final layout, and no golden file is rewritten inside this pull request.
- **T008** needs only T004 and may land any time after it; **T009** after T005–T008.
- **T010 → T011** after T009; both edit `cmd/galaxio/report.go`.
- **T012** after T007; it touches only tests in `internal/report/` and runs beside T008–T011.
- **T013** after T011. **T014** after T007; **T015** after T013 and T014.
- **T016** after T015. **T017** after T015, beside T016. **T018** after T017. **T019** after
  T016–T018. **T020** last.

```text
Phase 1:  T001 | T002
          ── Principle II amendment merged (outside this PR) ──
Phase 2:  T003 → T004
Phase 3:  T005 → T006 → T007
Phase 4:  T008 → T009                  (MVP)
Phase 5:  T010 → T011
Phase 6:  T012                         (beside phases 4 and 5)
Phase 7:  T013
Phase 8:  T014 → T015                  (T014 beside phases 5 to 7; T015 after T013)
Phase 9:  T016 | T017 → T018 → T019 → T020
```

Every task is its own commit on one linear branch; parallel means the work does not wait,
not that two tasks share a commit.

## Parallel opportunities

- T001 and T002 (documents; test data).
- T012 beside T008–T011 (tests in `internal/report/` only; the command files are not touched).
- T014 beside T010–T013 (`internal/report/source.go` and `scan.go`; the command files are
  not touched until T015).
- T016 and T017 (an integration test file; the README).

## Implementation Strategy

1. T001–T002: spec on the branch, Gatling's own figures on disk. Wait for the amendment.
2. T003–T004: the plan re-checked, the walk returning a summary; nothing visible changes.
3. T005–T007: every figure computed and proved against the corpus, percentiles pinned.
4. T008–T009: **MVP** — the summary printed in its final layout at the defaults. Stop here
   and validate US1 with quickstart §1 before going on.
5. T010–T013: the two options, the memory goal, the failures.
6. T014–T015: the progress block, last, because the summary is complete without it.
7. T016–T020: the binary exercised end to end, README, issue #51 amended, gates and
   quickstart evidence, the milestone PR.

## Notes

- Each task is one commit, message as given, with the session's attribution trailer;
  `gofmt -w .` before each commit; `go build ./... && go test ./...` green at every commit.
- The console carries the whole-run summary only. No task prints, stores or tests a figure
  per request or group, and no task writes a file or adds a flag that names one: both belong
  to the `-o` products of later milestones (FR-002, FR-016). A reviewer who finds either in
  this branch has found a defect.
- No test reads a percentile Gatling recorded (FR-019); quickstart §2 greps for it.
- Taking the read by rank of caio/go-tdigest#42 is not in this list (FR-022). When a release
  carries it, it is its own change, and T007's pinned 1427 is what makes it visible.
- `go get` for the digest library happens in T007 and nowhere earlier, so `go.mod` never
  names a module no code imports.
- The PR is left open for maintainer review; only the maintainer merges it.
