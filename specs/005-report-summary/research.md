# Research: Summarise a Finished Gatling Run

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Date**: 2026-09-17

Every figure below was measured on the recorded corpus of `github.com/galax-io/parsec
v0.1.0`, read through `simlog.NewRunReader`, or read from the source or bytecode it names.
Nothing is quoted from documentation alone.

## 1. Dependency: `github.com/caio/go-tdigest/v5 v5.0.0` (named by the maintainer)

**Decision**: Add `github.com/caio/go-tdigest/v5 v5.0.0` as a direct dependency, pinned to
that tag, and use it at its defaults: compression 100 — the value Gatling itself uses — and
its own random number generator with the library's fixed default seed. Named by the
maintainer on 2026-09-17 (spec, Percentiles in this milestone). `go get` runs only after the
constitution amendment of §13 is merged.

**Rationale**: percentiles over a log of any size in fixed memory need a sketch; the
standard library has none, and the two existing dependencies that touch numbers do not
either. The case Principle IV asks for:

| Question | Finding |
|---|---|
| Licence | MIT — compatible with GPL-2.0-only |
| Maintenance | v5.0.0 tagged 2025-11-29; external pull requests merged the day they were opened (August 2025); one open issue |
| What is linked | `go version -m` on a binary using it lists **one** module, `github.com/caio/go-tdigest/v5`; its non-test code imports the standard library only (`bytes`, `encoding/binary`, `errors`, `fmt`, `math`, `math/rand`, `sort`) |
| What `go mod tidy` writes | one `require` line in `go.mod`; six lines in `go.sum` — the module itself, plus checksums of its two **test-only** dependencies, `gonum.org/v1/gonum v0.11.0` (BSD-3-Clause) and `github.com/leesper/go_rng` (Apache-2.0). Neither is required in `go.mod`, compiled or distributed: `go list -deps` shows neither. The Apache-2.0 one is named here so that nobody finds it in `go.sum` and wonders |
| Static build | builds with `CGO_ENABLED=0`; pure Go |
| Known vulnerabilities | none in the GitHub Advisory Database for the module path, with or without `/v5` (queried 2026-09-17). `govulncheck` is not installed here and no CI job runs it; it is run by hand once the module is in, and the result recorded in quickstart.md, as milestone v0.13.0 did |
| Determinism | the default generator is local and seeded with a constant, so the same input in the same order gives the same digest; measured: two reads of every corpus run give identical quantiles |

**Alternatives considered**:

- `github.com/influxdata/tdigest` — on the measurements the best behaved of the three Go
  libraries (it returns the recorded value even on the 102-request run and uses no random
  numbers), but it is Apache-2.0, which Principle IV excludes for this GPL-2.0-only
  repository. Its last code change is from February 2021 and its `go.mod` carries no `go`
  directive, so its test dependencies would enter this module's graph.
- `github.com/spenczar/tdigest` — archived in January 2023, reads the global `math/rand`
  source so it does not reproduce its own output, imports `go-spew` in non-test code, and on
  the 100-sample set of §2 returns 134 where every definition of a percentile gives 1000.
- An exact one-millisecond histogram with no dependency at all — what issue #51's amendment
  of 2026-09-16 described. It keeps percentiles exact and needs no sketch. The maintainer
  chose the digest on 2026-09-17; the histogram stays the documented way back if the
  decision is ever revisited.

## 2. The percentile read: the library's default quantile, and what is known about it

**Decision**: read every percentile with `TDigest.Quantile(rank/100)` and round half up to a
whole millisecond. Pin the results for the corpus in tests (FR-021). Document the known
divergence in the README with its example and the upstream pull request (FR-020).

**Evidence**: `Quantile` interpolates linearly between the centres of neighbouring centroids
at the index `q·(n − 1)`. While a digest holds fewer samples than twice its compression no
centroid is ever merged — the merge bound is `4·n·q(1 − q)/compression`, which stays under 2
— so the digest holds the run itself and the read is definition R-7, the default of numpy
and of Excel's `PERCENTILE.INC`. On the recorded 3.13.1 run (96 requests at most 7 ms, six at
1502–1503 ms) the index 95.95 falls between the 96th value and the 97th:
`7 + 0.95 · 1495 = 1427.25`.

Percentiles this command will print for the corpus, p50/p75/p95/p99, measured:

| Run | all | ok | failed |
|---|---|---|---|
| 3.11.5 | 2 / 7 / 1503 / 1504 | 7 / 1503 / 1504 / 1504 | 2 / 2 / 4 / 4 |
| 3.12.0 | 2 / 5 / 1503 / 1504 | 5 / 1502 / 1504 / 1504 | 1 / 2 / 2 / 3 |
| 3.13.1 | 1 / 1 / **1427** / 1502 | 1 / 1 / 1502 / 1502 | 1 / 2 / 4 / 4 |
| 3.14.9 | 1 / 1 / **1427** / 1502 | 1 / 1 / 1502 / 1502 | 1 / 2 / 12 / 12 |
| 3.15.1 | 0 / 1 / **1427** / 1502 | 0 / 1 / 1502 / 1502 | 2 / 2 / 3 / 3 |

The three values in bold lie between 7 ms and 1502 ms, where those runs hold no request; the
request at the 95th percentile's rank took 1502, 1501 and 1502 ms.

**The upstream change**: [caio/go-tdigest#42](https://github.com/caio/go-tdigest/pull/42),
opened 2026-09-17, adds `ValueAtRank` and `QuantileDiscrete`, which return the mean of the
centroid holding the rank and never a value between two centroids. That read, measured over
`ForEachCentroid` on the same digests before the pull request was opened, returns the
recorded value in all 60 percentiles of the five runs above. It is open and unreleased; this
milestone does not depend on it and does not copy it (FR-022).

**Why Gatling's percentiles are not a reference**: for the 3.13.1 run Gatling printed 1072.
That comes from a defect in `com.tdunning:t-digest:3.3`, reported as
[tdunning/t-digest#230](https://github.com/tdunning/t-digest/issues/230); two thousand
rebuilds of that digest from the same samples gave 1072, 916 or 1061.

**Alternatives considered**: carrying this repository's own read by rank over
`ForEachCentroid` until the upstream change ships (measured, exact on the corpus; not taken —
maintainer decision of 2026-09-17: use what the library gives by default and note the pull
request); a higher compression (not taken — the defaults are the decision, and compression
does not change a read between two singleton centroids).

## 3. All requests: merge the ok and failed digests when the figures are read

**Decision**: the summary keeps two digests, one per outcome, created on the first request
of that outcome. The percentiles of all requests are read from a clone of the ok digest
merged with the failed one at the moment the summary is produced; no third digest is fed.

**Evidence**: on every corpus run and on a synthetic run of one million samples the merged
digest gives the same p50, p75, p95 and p99 as a digest fed every sample directly — for the
3.13.1 run 1.00 / 1.00 / 1427.25 / 1502.00 both ways — and merging twice gives the same
values. While both digests hold singletons, the merge is their sorted union, so equality is
structural there rather than lucky.

**Alternatives considered**: a third digest fed with every request (rejected: half as much
memory again and one more insertion per request, for the same numbers).

## 4. Exact figures: integer accumulators, arithmetic done when the figures are read

**Decision**: per outcome keep the count, the count of requests that carry a duration, the
minimum and maximum in whole milliseconds, the sum as a 128-bit pair and the sum of squares
as three 64-bit words, both maintained with `math/bits`. A negative duration, which the
library promises never to yield, counts as no recorded end, as the library's own `Bounds`
treats one. Mean and standard deviation are computed with
`math/big` when the summary is produced, never in floating point:

- mean = `floor((2·sum + n) / (2·n))` — half up;
- variance numerator `n·Σx² − (Σx)²`, denominator `n²`; the deviation is
  `k = floor(sqrt(numerator / denominator))`, raised by one when
  `4·numerator ≥ (2k + 1)²·denominator` — half up, decided in integers.

**Rationale**: issue #51 settled each formula against the `stats.json` Gatling wrote for the
3.11.5, 3.12.0 and 3.13.1 recordings, with negative controls: half-up mean (banker's rounding
mismatches 8 fields, truncation 49), population deviation about the unrounded mean (the
sample deviation mismatches 12). The widths are what makes them exact for any input: a
duration is at most 2⁶³−1 ns, under 2⁴³ ms, and a count is at most 2⁶³, so the sum stays
under 2¹⁰⁶ and the sum of squares under 2¹⁴⁹, and neither can wrap for any run the command
can read — Principle II's refusal is never reached. The first design kept a 64-bit sum,
which would wrap after about two million requests of the longest duration a log can
hold; the log is untrusted input, so the re-check of 2026-09-17 (T003) widened both. The
sum of squares of ten million requests of an hour each is already about 1.3·10²⁰, past
64 bits.

**Alternatives considered**: Gatling's own float form `sqrt(Σx²/n − mean²)` (rejected: it
cancels catastrophically on large, tight data; both forms agree on every corpus field, and
the integer one cannot drift); Welford's streaming update (rejected: floating point for what
integers do exactly).

## 5. The request rate and how numbers are printed

**Decision**: `S = ceil(span / 1 s)` from the library's `model.Bounds`, computed on whole
milliseconds as `(ms + 999) / 1000`; all three rates divide by the same `S`. A span of
zero is an error naming the span (exit 1); bounds the library cannot resolve leave every rate
absent. Rates and percentages are printed with at most two decimals, trailing zeros removed,
a point as the decimal separator and no thousands separator; an absent figure is `-`.

**Evidence**: `ResultsHolder.<init>` in `gatling-charts-3.15.1.jar` computes the run-wide
span exactly so (issue #51); spans of 3214–3232 ms give `S = 4`, and 102/4, 84/4 and 18/4
reproduce 25.5, 21 and 4.5. Gatling's console formats every number with
`new DecimalFormat("###,###.##", DecimalFormatSymbols.getInstance(Locale.ENGLISH))` — read
from the bytecode of `io.gatling.shared.util.NumberHelper$` in
`gatling-shared-util_2.13-0.0.14.jar` — and prints `-` for an empty column
(`ConsoleStatsFormat.formatNumber`). At most two decimals with trailing zeros dropped is
therefore Gatling's own rule; the thousands separator is left out because it is the one
part of that rule that belongs to a locale, and because until an `-o` product exists these
numbers are what a script will grep. `strconv.FormatFloat(x, 'f', 2, 64)` rounds the binary
value to nearest even as `DecimalFormat` does by default, so the two agree wherever a
recording can show it.

**Alternatives considered**: counting one-second buckets (rejected by the evidence: it gives
5 seconds and 20.4 where both consoles print 25.5).

## 6. Response-time bands

**Decision**: three counters over successful responses — under the lower boundary, from it
to under the upper one, at or above the upper one — and the failed count; the percentage is
`float64(count) / float64(total) * 100` in exactly that association. Each band is drawn as a
bar of 20 cells, `floor((40·count + total) / (2·total))` of them filled — the share rounded
half up to a twentieth, in integers — with at least one cell for a band that holds a
request and never all twenty for a band that does not hold them all.

**Evidence**: `ResponseTimeRangeBuffer.update` sends every KO into the failed band before it
looks at the time; letting KO into the timing bands mismatches 36 corpus fields. `12/36*100`
is `33.33333333333333` as Gatling recorded it; `100*12/36` is `33.333333333333336`.

## 7. Whole run only: no figure per request or group

**Decision** (clarified 2026-09-17, and repeated by the maintainer the same day — "только
сводная стата; запросы и группы только в -o"): the fold keeps one set of figures, the whole
run, fed by request samples only. A group traversal is counted by the tally as before and
takes part in no figure. Nothing is keyed by `model.Position`.

**Rationale**: the console carries the whole-run summary only, and requests and groups go
to the products `-o` names — `stats`, `global_stats`, `yml` — which later milestones deliver.
Nothing in this milestone would show a figure per request, so computing one would be code
without a consumer (Principle VI), a second digest insertion per request for no output, and
memory that grows with the number of names. The outcome figures of
[data-model.md](data-model.md) are what such a row will be made of, so #52 can key them by
position without reshaping them.

**Evidence that the whole run is requests only**: Gatling's `global_stats.json` for the
3.13.1 run reports 102 requests, not 102 plus its 12 group traversals.

**Alternatives considered**: computing every row now and proving it against `stats.json` in
tests alone (rejected: a requirement nobody can observe, and dead code until #52); printing
the rows on the console (rejected by the maintainer); a text file of rows named by its own
option (rejected by the maintainer — §10).

## 8. One pass, the memory goal, and the refactor it needs

**Decision**: the walk of milestone v0.13.0 stays the only loop. `Scan` gains the options and
returns a `Summary` that carries the `Tally` it already counted; the restructuring is its own
commit, before any statistic. **Goal: heap in use stays under 32 MiB for a log of any size
and any number of distinct request names** — the goal of v0.13.0, unchanged, because the
summary adds two digests and a handful of integers and keeps nothing per name.

**Measurement** (the two digests at the library defaults, a log-normal body around 40 ms
with 0.5 % at 60 s and 5 % failed, heap after `runtime.GC()`):

| Requests | Heap held | Centroids, both digests / the larger | Per insertion |
|---|---|---|---|
| 1 000 000 | below what `HeapAlloc` resolves (0.00 MiB) | 1774 / 981 | 290 ns |
| 10 000 000 | 0.05 MiB | 2045 / 1112 | 286 ns |
| 100 000 000 | 0.06 MiB | 2318 / 1246 | 311 ns |

The goal stays a test in the ordinary suite, as `TestScanMemoryDoesNotGrowWithTheLog` is:
replay a 16 MiB and a 256 MiB log through `reporttest.Replay`, fail if either passes the
goal or if sixteen times the log costs meaningfully more. Throughput is kept by a benchmark
and not gated.

**For #52, measured on the way**: two digests per request or group position cost about
42 KiB — 10.46 MiB at 250 positions, 41.81 MiB at 1000 — flat from one to ten million
requests. Rows per position therefore do not fit the 32 MiB goal past roughly 700 positions,
which is that milestone's question and not this one's.

## 9. Command surface: two new flags

**Decision**:

| Flag | Value | Default | Refused as a usage error |
|---|---|---|---|
| `--percentiles` | comma-separated ranks | `50,75,95,99` | anything that is not a number above 0 and at most 100; an empty list |
| `--bounds` | `LOW,HIGH` in whole milliseconds | `800,1200` | anything but two whole, non-negative numbers, the second greater than the first |

Both are `pflag.Value`s that validate in `Set`, as `-o` does, so that a value the command
cannot honour is refused while it is parsed — before help and before any work. Ranks are
reported once each, in increasing order. `-o` is untouched, and no flag names a file (§10).

**Rationale**: the names are what a user searches for and what Gatling's own configuration
calls them (`percentile1…4`, `lowerBound`/`higherBound`). Under Principle V both are
permanent, so neither gets a shorthand.

## 10. No file in this milestone

**Decision** (maintainer, 2026-09-17): the command creates, changes and removes no file, and
no option names one. The only files `galaxio report` is ever to write are the products `-o`
already names — `stats`, `global_stats` and `yml` — each in full whenever it is asked for,
and each delivered by its own milestone (`stats` and `global_stats` by v0.15.0, #52).

**Rationale**: `-o` is where this command's files were decided to live (constitution v2.1.0,
Principle I: a command that chooses among several products names them). A second way to get
a file would be a second published surface for the same need, permanent under Principle V.

**Alternatives considered**: a human-readable text file of the whole run and every request
and group, named by a `--summary-file` option — drafted from a misreading of the
clarification and rejected by the maintainer ("никаких файлов не надо кроме тех что сейчас
перечислены в -o").

## 11. What is reported on standard error, and the two new failures

| Condition | Behaviour |
|---|---|
| samples with no recorded end | counted; in no timing figure; `report: warning: N requests have no recorded end…` on standard error, even under `--quiet`, because it qualifies the numbers |
| samples whose outcome the source lost | counted apart; after the summary, exit 1 naming the count |
| a span of zero | every rate `-`; after the summary, exit 1 naming both instants |
| bounds the library cannot resolve | every rate `-`; exit 0, as the span line is already omitted in v0.13.0 |
| several of these and a log cut short | one `RuntimeError` joining them, as v0.13.0 joins a write failure with a scan failure |

The two exits of 1 change what v0.13.0 returns for such runs; no Gatling log is known to
produce either (spec, Assumptions), and the change is listed under gate V.

## 12. Reference data for the tests

**Decision**: copy what Gatling itself recorded into
`internal/report/testdata/corpus/gatling/<version>/`, unchanged and with provenance:
`global_stats.json` for 3.11.5 and 3.12.0, `js/global_stats.json` for 3.13.1, and
`console.txt` for 3.13.1, 3.14.9 and 3.15.1. Whole-run figures are compared with the JSON
files by reading them; for 3.14.9 and 3.15.1, which have a console only, they are a table in
the test citing the console line. No percentile in any of those files is ever read by a test
(FR-019). `stats.json`, which holds the rows per request, is not copied: nothing reads it
before #52.

**Rationale**: milestone v0.13.0 left them out because nothing read them (its research §9).
They are MIT with the rest of the corpus; `PROVENANCE.md` gains the rows.

## 13. The constitution amendment this feature waited for

**Decision** (clarified 2026-09-17): Principle II's percentile clause was amended first — its
own issue and pull request inside milestone `v0.14.0`, as #112/#113 were — and no
implementation task started before it was merged: issue
[#114](https://github.com/galax-io/galaxio-cli/issues/114), pull request
[#115](https://github.com/galax-io/galaxio-cli/pull/115), merged 2026-09-17, constitution
v2.2.0 (MINOR). The ratified bullet:

> A percentile MAY be an estimate read from a bounded-memory sketch rather than an exact
> order statistic: an exact one needs every sample or a histogram as wide as the range of
> values, and a log of any size bounds neither. An estimate MUST be deterministic — the same
> log yields the same percentiles — and every output carrying one MUST say that it is an
> estimate and how it is computed. Its estimator, the estimator's parameters and how it is
> read MUST be recorded in the feature's `research.md`, and its values for the recorded
> corpus MUST be pinned in tests, so that changing any of the three fails a test instead of
> moving printed numbers in silence. Where an estimate is known to differ from the value
> recorded at the percentile's rank, the documentation MUST say so with an example.

Counts, minimum, maximum, mean and standard deviation stay exact, with a refusal where one
cannot be kept exact, and the bullet forbidding parity with another tool's percentiles is
unchanged. The ratified text differs from the wording first proposed here: it admits an
estimate rather than requiring one, so an exact percentile stays compliant, and it adds the
recorded estimator and the pinned corpus values. Gate II of
`.specify/templates/plan-template.md` changed with it.

**How this feature meets it**: the estimator, its parameters and its read are §1 and §2; the
values for the five corpus runs are pinned by T007; the summary's closing line and the
progress block's second line say that the percentiles are t-digest estimates, interpolated
(§15); the README documents the 1427-for-1502 case (FR-020).

**Also outside this pull request**: issue #51's text needs amending to match the
specification — `-o json`, per-request figures in the text output, the exact histogram with
its capped pool and refusal, and the selectable wall clock of a group are all superseded or
deferred to the `-o` products.

## 14. Skills classification

Required before code (constitution table): `golang-cli` and `golang-spf13-cobra` (two
flags, `pflag.Value` validation, exit paths), `golang-error-handling` (the joined runtime
errors), `golang-testing` (every task), `golang-naming` and `golang-documentation` (the
exported surface of `internal/report` and the README). Consulted for this plan:
`golang-dependency-management` (§1 — its checklist is the table there; its rule that an agent
asks before `go get` is met by the maintainer naming the module, and `go get` still waits
for §13). To consult when the code is written: `golang-pkg-go-dev` (§1, before pinning),
`golang-benchmark` then `golang-performance` (§8), `golang-safety` and `golang-security`
(every log is untrusted input: the log's file name reaches the progress block through the
existing `printable`, and this feature writes no file), `golang-refactoring` (§8, the `Scan`
restructuring), `golang-design-patterns` (the options struct and the tick). No skill
contradicted the constitution. §15 and §16 were added after this classification and change
none of it: the progress block and the layout are command surface (`golang-cli`) and test
work (`golang-testing`).

## 15. Progress while reading

**Decision** (maintainer, 2026-09-17, chosen from two animated mock-ups): a block of six
lines on standard error, shown when standard error is a character device, `TERM` is set and
is not `dumb`, and `--quiet` is not given; redrawn in place and erased before the report.
Its layout is in [contracts/cli.md](contracts/cli.md).

- **Bar, percentage, time left**: bytes read over the size seen at opening. `Open` wraps the
  file in a counting reader after format detection; the library reads ahead in blocks, so
  the figure leads the records by one block, which is tens of kilobytes. A size of zero or a
  file that is not regular leaves all three out. The time left is the elapsed time scaled by
  the bytes still to read, from the second draw.
- **The second line** says, faintly, `figures so far · times in ms · percentiles are t-digest
  estimates, interpolated`. Principle II (v2.2.0) asks every output carrying a percentile to
  say that it is an estimate and how it is computed, and the block is such an output; the
  first design left the line blank, and the re-check of 2026-09-17 (T003) filled it.
- **Figures**: for all, ok and failed requests — the count, the share, the minimum, the
  half-up mean, the 50th, 95th and 99th percentile and the maximum. All requests are read
  from a clone of the ok digest merged with the failed one, the same read the final figures
  use. Counts are exact below 10 000 and abbreviated above. No rate: it needs the final
  span. The columns are fixed, whatever `--percentiles` asks, so the block is 78 columns
  wide by construction.
- **When it is drawn**: the walk already stops every 1024 items to look at the context. It
  calls a tick there; the command's tick looks at the clock and draws if 500 ms have passed
  since the start and 200 ms since the last draw. A read that ends within 500 ms draws
  nothing. No goroutine and no timer, so the race detector has nothing to find and a
  `Fatalf` has nothing to leak — the lesson of milestone v0.13.0's first memory benchmark.
- **How it is redrawn**: cursor up six lines (`ESC [ 6 A`), then each line erased
  (`ESC [ 2 K`) and rewritten; removal is cursor up and erase to the end of the screen
  (`ESC [ J`). **No terminal mode is changed** — the cursor is not hidden, wrapping is not
  switched off, the alternate screen is not used — so there is nothing to restore, and a
  killed command leaves the block behind and nothing else.
- **When it goes**: before any other write to standard error, before the report, and before
  returning. An interrupt arrives as the cancelled context of `signal.NotifyContext` in
  `main`, which the walk already returns on, so it takes the path of any other ending.
- **Width**: a line that would pass 79 columns is cut. The terminal's width is never asked
  for: the standard library cannot, and `golang.org/x/term` would be a new module for it. A
  terminal narrower than 80 columns wraps the block and may keep remains of it — accepted
  and documented.

**Cost**: a clock read every 1024 items and, five times a second, one digest clone and merge
of at most a few thousand centroids and nine quantile reads. Not measured yet; SC-010 caps
it at 5 % and quickstart §6 records the pair of benchmark figures.

**Alternatives considered**: a single status line redrawn with a carriage return (specified
first; rejected by the maintainer on sight, and it could not show ok and failed side by
side); switching line wrapping off while the block is up, which would make a narrow
terminal cut the block instead of wrapping it (rejected: a killed command would leave the
terminal with wrapping off), and hiding the cursor (rejected for the same reason);
`golang.org/x/term` for the width and for the classic Windows console mode (rejected: a new
module; that console simply gets no block); a `--progress` flag with periodic lines for CI
logs (out of scope: new published surface nobody has asked for); drawing on standard output
(rejected: it is the report's, and scripts read it); a ticker goroutine (rejected: a second
goroutine touching the accumulators needs locking the walk does not otherwise need).

## 16. The look of the summary

**Decision** (maintainer, 2026-09-17, chosen from three mock-ups drawn with the figures of
the 3.13.1 run): the panel — a headline with the requests, ok and failed and their shares
and rates; a response-time table with one row per outcome and the statistics in columns;
the bands as bars with share and count; one closing line. The outcomes are `ok` and
`failed`. Colour on a capable terminal only. Its layout is in
[contracts/cli.md](contracts/cli.md).

**Rationale**: the first draft reproduced Gatling's console — statistics as rows; `total`,
`OK` and `KO` as columns; the bands written `t < 800 ms` — and the maintainer rejected it as
too much like Gatling: the same command is to summarise JMeter, k6, Locust and Yandex.Tank
runs (milestones v0.19.0 to v0.22.0). One row per series with the statistics in columns is
how JMeter's summariser, Locust's table, k6's trend lines, vegeta and wrk all print;
Gatling's orientation is the exception. `ko` is Gatling's word and stands on the released
requests line of v0.13.0, so that line changes, with the maintainer's approval (gate V).

- **Widths**: a label column of 20 and figures right-aligned in 7, a column growing for a
  longer value. With the default four ranks the table is 76 columns wide; every further rank
  adds 7.
- **Headline**: one line while it fits 100 columns — 90 for the 3.13.1 run — and one segment
  per line beyond that, so that a soak test's counts do not push it into wrapping. The rule
  depends on the text alone, never on the terminal.
- **Bars**: 20 cells, so a cell is 5 %; the rule of §6 keeps a small band visible and a
  nearly full one honest.
- **Colour**: three SGR codes — green, red, faint — when standard output is a character
  device, `TERM` is set and is not `dumb`, and neither `--no-color` nor `NO_COLOR` is given.
  The root command already merges the last two into one option; this command is the first to
  read it. `failed` is red only when something failed, so a clean run shows no red.
- **UTF-8**: `✓`, `✗`, `█`, `░`, `·`, and in the progress block `━`, `─` and the braille
  spinner. The text is for people, and k6 and other current tools print the same symbols; no
  ASCII fallback is added.

**Alternatives considered**: one table carrying the counts, shares and rates as further
columns of the same three rows (90 columns wide; offered as the default and not chosen);
k6-style metric lines, `min=0 mean=89 …` (numbers do not line up, so ok and failed are hard
to compare); the Gatling-like table of the first draft (rejected by the maintainer); a
thousands separator (not taken: §5).
