# Research: Read a Finished Gatling Run

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Date**: 2026-09-16

Every decision below was checked against the published `github.com/galax-io/parsec v0.1.0`
API and its recorded corpus, run through `simlog.NewRunReader`. Quoted counts and error
texts are what those runs printed, not what documentation promised.

## 1. Dependency: `github.com/galax-io/parsec v0.1.0` (approved)

**Decision**: Add `github.com/galax-io/parsec v0.1.0` as a direct dependency, pinned to that
tag. Approved by the maintainer on 2026-09-15. It provides `gatling/run.Find` (run
discovery), `gatling/simlog.NewRunReader` (format detection by leading bytes, version gate,
canonical item stream), the `model` types, and the error types the command maps to exit
codes.

**Rationale**: The standard library has no Gatling codec. The text log is a tab-separated
grammar with its own escaping; the binary log is an undocumented stream with a string cache,
JVM-compact strings and big-endian framing. parsec spent five milestones and a recorded
corpus of five versions getting both right, and Principle II says this repository builds on
its definitions rather than re-deriving them. Licence: MIT, GPL-compatible (milestone
v0.12.0). Footprint: `go 1.25`, zero transitive modules, pure Go, so `CGO_ENABLED=0` and the
distroless image are unaffected. Stability: the exported surface is frozen at v0.1.0.

**Alternatives considered**: a hand-written text-only parser (rejected: leaves every run from
3.13 on unreadable, the very gap the issue names); vendoring (rejected: the proxy and
checksum database already give reproducibility); copying parsec's source into `internal/`
(rejected: forks the definitions and loses the corpus-bound version gate).

## 2. Scope: reading only, and what was withdrawn

**Decision** (maintainer, 2026-09-16): this milestone reads a run and reports what it holds.
The line-delimited JSON record stream built in the first revision is **withdrawn and
deleted**, along with the record vocabulary it needed.

**Rationale**: a record-per-line dump restates in a second vocabulary what parsec already
yields, and it is not an output this ecosystem wants. The output wanted is Gatling's own
`stats.json` and `global_stats.json`, in Gatling's schema and with Gatling's numbers,
verified against a run whose files Gatling still wrote or against its HTML report. That
needs the statistics of #51 and the writer of #52 and belongs to the milestones that own
them. The milestone boundary does not move; this milestone delivers the reading those two
stand on.

**What the withdrawal removed**: `record.go` (a record-kind enum, six record structs, a
generic `Opt[T]`, a `Warning`, a `Failure` and a `model.Field`-to-identifier table),
`convert.go`, `write.go` and their tests and golden files. Every one of those types either
restated a parsec definition or existed only to serialise it.

## 3. Build on parsec's primitives; declare nothing that restates them

**Decision**: `internal/report/` declares two types of its own — `Tally`, because the
library exports no count, and `Source`, the open run pairing the library's reader with the
format detected for the report — and uses parsec's types everywhere else: `run.Location` for where the run is, `model.Run` for what
the source said about it, `model.Bounds` for the span, `model.Item` and `model.ItemKind` for
the stream, `model.Outcome` for the split, `model.Field` and `Capabilities` for what the
source cannot record, `gatling.Format` for the log format.

**Rationale**: the library's own `model/example_fold_test.go` states the intended consumer
shape — bucket by `model.Position`, extend one `model.Bounds`, do your own arithmetic — and
the library exports no count, which is why `Tally` exists. Anything else this package
declared would fork a definition the ecosystem shares, which is what the first revision did
and what Principle VI forbids.

**Not used yet, deliberately**: `model.Position` is the bucketing key the statistics of #51
need. This milestone reports totals only, so bucketing by position would be structure
without a use (Principle VI). #51 adds it over the same pass.

## 4. Package layout: a flat `internal/report/`

**Decision**: one package holding `source.go` (locate and open), `scan.go` (the pass and the
tally) and `doc.go`, beside one subpackage, `reporttest/`, holding the synthetic-log builder
the benchmark and the damaged-log test share.

**Rationale**: the constitution names `internal/report/` as the home of report code. Three
report files do not need dividing; the fixture is kept apart because it is a test builder
rather than report code, and its own package is what stops it being importable from the
command. the statistics of #51 and the writer of #52 are added beside
them, and a split is worth making when the package stops reading comfortably.

## 5. Run selection, the default root and what the report says

**Decision**: an empty path becomes `run.DefaultResultsRoot` (`target/gatling`) resolved
against the working directory; otherwise the argument goes to `run.Find` unchanged, which
treats a path holding `simulation.log` (or the log itself) as the run and anything else as a
results root searched by `lastRun.txt` then by newest. The report names the run directory
and the rule that chose it.

**Evidence**: `run.Find` on the corpus `lastrun/results` returns the run `lastRun.txt` names
with `Found=lastRun.txt`; a missing root returns `*run.NotFoundError` reading `gatling: no
Gatling run under <dir>`; an unreadable root wraps `*fs.PathError`. parsec's own
documentation asks a caller that reports which run it read to report the rule too, because
newest is a guess from modification times.

**Alternatives considered**: also searching Gradle's `build/reports/gatling` when no path is
given (rejected: a guessed root can answer about a run nobody asked about); a
`--results-root` flag (rejected: the positional argument already accepts a root).

## 6. Failure semantics and exit codes

All wrap into `RuntimeError` (exit 1) unless noted.

| Condition | parsec signal | Behaviour |
|---|---|---|
| No run under the path | `*run.NotFoundError` | message names the directory searched |
| A named path that does not exist | `*run.NotFoundError` plus our own `os.Stat` | `cannot read <path>: <reason>`; never "no run". The stat is `os.Stat`, the call parsec makes, so that the two layers agree about what exists: `os.Lstat` would call a dangling symlink present and report it as an empty root |
| The default root absent | `*run.NotFoundError` | passed through: the caller never typed the path, so an absent default is an absence of runs, not a path they got wrong |
| A directory that cannot be read | `*fs.PathError` inside parsec's own error | passed through unchanged: parsec's message names the directory it was reading, where a rewrite would name a `simulation.log` whose existence was never established |
| Not a Gatling log | `*gatling.FormatError` | parsec's detail, prefixed with the log path |
| Version below range, or 3.13.0 | `*gatling.VersionError` | parsec's text names the version and the range |
| Version above range | `model.Run.Warnings` | the run is read, exit 0, warning on stderr even under `--quiet` |
| Log cut short | `*gatling.TruncationError` | the counts of what it held are reported, then exit 1 |
| Damaged log | `*gatling.SyntaxError` | **no counts**, exit 1 |
| Write to stdout fails | the writer's error | one `RuntimeError`, reported once |
| Unsupported tool, third argument, unknown flag, any `-o` | — | `UsageError`, exit 2, empty stdout |

**What no reading can catch.** Neither log format carries an end marker, so parsec ends a
run killed exactly on a record boundary with a clean `io.EOF` and this command reports it as
complete. That is a property of the format, not a gap in the reader, and the documentation
says so rather than implying every killed run is caught.

**Why a truncation reports counts and a syntax error does not**: parsec documents them as
different endings. A truncation means the bytes ran out inside a record and "the records
already delivered are the ones it recorded"; a syntax error means "there is no partial
result beside it — records delivered before it are not a result, and no total may be derived
from them". The command follows that distinction rather than inventing one.

**Distinguishing a missing path from an empty root**: `run.Find` reports a path that does
not exist as a directory holding no run. A `os.Lstat` after a `*run.NotFoundError` is what
tells a mistyped path from an empty results root, and only then.

## 7. One pass, and the memory goal

**Decision**: `Scan` loops `rd.Next`, extends one `model.Bounds` and increments counters.
Nothing is retained: parsec's reused `Groups` slice is never kept, and no record outlives
the iteration. `ctx.Err()` is checked every 1024 items. **Goal: heap in use under 32 MiB
regardless of log size.**

**Measurement**: the goal is a test, `TestScanMemoryDoesNotGrowWithTheLog`, which the
ordinary suite runs and therefore CI runs. It scans a 16 MiB and a 256 MiB replay served
through an `io.Reader` without writing a file, measures the heap each scan adds after a
`runtime.GC()` baseline rather than the whole process's, and fails if either exceeds the
goal or if sixteen times the log costs meaningfully more. `BenchmarkScan` keeps the
throughput figures and no longer pretends to gate memory: a benchmark is an instrument for
comparing, and a gate needs a runner.

**The first version measured the wrong thing and leaked.** It took a maximum over
process-wide `HeapInuse` from a 10 ms ticker, accumulated across `b.Loop` iterations, so the
number compared against the goal was a function of `-benchtime`: 1.883 MiB at 1x against
4.914 MiB at 5x for identical code. Its goroutine also outlived any `b.Fatalf`, because
`Fatalf` exits through `runtime.Goexit` and the `close` that stopped it never ran, leaving a
stop-the-world call running a hundred times a second for the rest of the test binary's life.

## 8. Testing the version gate without a 3.13.0 recording

**Decision**: the below-range case uses a synthetic text header claiming `3.10.0`; the
3.13.0 refusal patches the six version bytes at offset 5 of the 3.13.1 corpus log in memory;
the newer-than-range warning uses `3.99.0` for text and the same byte patch to `3.99.9` for
binary.

**Rationale**: parsec's gate reads the version from the RUN record before anything else, so
patching only that string exercises exactly the gate. No recording is edited on disk.

## 9. What the next milestones inherit

#51 folds the same pass into per-position accumulators keyed by `model.Position`, with
successes and failures accumulated apart, exact count/min/max/mean/standard deviation, and
percentiles as estimates. #52 writes those numbers as Gatling's `stats.json` and
`global_stats.json`: the same schema and the same numbers, with every field where a
difference is possible saying so and naming its size. The corpus already carries Gatling's
own `stats.json` and `global_stats.json` for 3.11.5, 3.12.0 and 3.13.1, which is the ground
truth those milestones verify against; from 3.14.0 Gatling writes none, which is the gap
they fill. Those files are not copied into this repository yet, because nothing in this
milestone reads them.

## 10. Skills classification

Required before code (constitution table): `golang-cli`, `golang-spf13-cobra`
(`cmd/galaxio/report.go`), `golang-error-handling` (the error mapping), `golang-testing`
(all tests), `golang-naming` and `golang-documentation` (the exported surface and the
README). Consulted: `golang-dependency-management` and `golang-pkg-go-dev` (§1),
`golang-project-layout` (§4), `golang-benchmark` (§7), `golang-safety` and `golang-security`
(every log is untrusted input; parsec bounds line and string length and this package adds no
buffer of its own), `golang-context` (§7). No skill contradicted the constitution.
