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
milestone does not depend on it and does not copy it, and taking it would part from Gatling
3.11's numbers, which the maintainer made the target (FR-022, §18).

**The rule of how a percentile may differ from what the run recorded** (maintainer
request, 2026-09-17: the difference must be proved by tests, not only described):

- **While an outcome holds at most 200 requests**, a percentile is the linear interpolation
  between the two recorded response times around the position rank/100·(n−1), rounded half
  up: it lies between them, and it differs from the request at the percentile's rank by at
  most their gap. A digest merges no centroid before it holds 200 values, so it holds the
  run itself and reads it as definition R-7 does. On the 3.13.1 run the neighbours of the
  95th percentile of all 102 requests are 7 ms and 1502 ms: 1427 is printed, 1502 is the
  request at the rank.
- **At any size**, a percentile misplaces its rank among the recorded requests by at most
  4·q·(1−q)/100 of them plus one request: the share of requests below the printed value and
  the share at or below it bracket q = rank/100 to within that tolerance. 4·n·q·(1−q)/100 is
  the most the library merges into one centroid around q, and one request is the step
  between two recorded values. At the 95th percentile the tolerance is 0.19 percentage
  points plus one request; at the median, one point; at the 99th, 0.04. The rule bounds the
  rank, not the value: where response times have a gap, the value can be anywhere across it.

Measured before it was stated: five distributions — log-normal around 40 ms with 0.5 % at
60 s, 95 % at most 10 ms and 5 % near 1500 ms, six distinct values, uniform to 5 s, and two
modes at 50 and 800 ms — at 201, 1 000, 12 000, 200 000 and 1 000 000 requests and three seeds
each, ranks 50 to 99.9. No read passed the tolerance; the worst came to 0.79 of it, and
at 12 000 requests and more to 0.40. `TestPercentilesRule` holds every corpus run to both
halves of the rule and the synthetic runs to the second; the live Gatling runs of T010 hold
real runs of about 12 000 requests to the second.

**Which of Gatling's percentiles are the reference**: those of 3.11.x and 3.12.x, which use
`com.tdunning:t-digest` 3.1. Every percentile this tool prints equals the one their digest
gives for the same log, and the tests assert it (§18). From 3.13.0 Gatling uses t-digest 3.3, whose
`AVLTreeDigest` loses counts: for the 3.13.1 run it printed 1072 where the request at the
rank took 1502, a defect reported as
[tdunning/t-digest#230](https://github.com/tdunning/t-digest/issues/230), and two thousand
rebuilds of that digest from the same samples gave 1072, 916 or 1061. Those numbers
describe the defect, not the run, and the tests only describe them.

**Alternatives considered**: carrying this repository's own read by rank over
`ForEachCentroid` until the upstream change ships (measured, exact on the corpus; not taken —
maintainer decision of 2026-09-17: use what the library gives by default and note the pull
request); a higher compression (not taken — the defaults are the decision, and compression
does not change a read between two singleton centroids).

## 3. All requests: a digest of their own, fed in log order

**Decision**: the summary keeps three digests, one for each
outcome and one for all requests, each created on its first request. The walk feeds every
recorded response time of a successful or failed request to its outcome's digest and to the
digest of all requests, in log order, which is how Gatling 3.11 feeds its own three (§18).
Nothing is merged and nothing is cloned when the figures are read, so a read in the middle
of a walk changes nothing that follows; a test reads a long walk every hundred requests and
holds every quantile of all three digests to the unread walk's.

**Why not two digests merged when they are read**, the first design: it kept two digests and read all
requests from a fresh digest into which the ok digest and then the failed one were merged.
Its evidence was the corpus runs and a synthetic run of a million requests whose failures
were as fast as its successes; on those the merged digest gave the numbers of a digest fed
directly. But two compressed digests merged are not one digest fed the same requests in
order, and the two agree only while failures sit inside the dense part of the successes. On
the synthetic runs of `internal/report/synthetic_test.go` — 12 000 requests, whole numbers
only so that every architecture draws the same run — merging gave:

| Run | Rank | Merged | Fed in log order | t-digest 3.1 |
|---|---|---|---|---|
| every 20th request fails slowly | p95 of all | 1215 | 1214 | 1214 |
| every 20th request fails slowly | p99 of all | 4241 | 4244 | 4244 or 4245 |
| 5 % fail slowly | p95 of all | 358 | 366 | 366 |
| 30 % fail slowly | p75 of all | 1691 | 1693 | 1693 |
| 5 % fail as fast as the rest | every rank | equal | equal | equal |

On a run of 12 000 requests with 5 % of failures at 1000–5000 ms it was 401 ms away. `TestSyntheticRunsEqualGatling311` holds
all three rows of every such run to the `etalon.tsv` t-digest 3.1 wrote for it, with no
exception, and fails on three of the four against the merged read.

**Cost**, measured with a log-normal body around 40 ms, 0.5 % at 60 s and 5 % failed: the
three digests hold 0.04 MiB of heap after a million requests and 0.08 MiB after ten million
(1146, 1148 and 947 centroids), and a request costs one insertion more — 64 MiB of the
replayed corpus log scan at 96 MB/s where two digests scanned at 138 MB/s.

**Alternatives considered**: merging the two outcomes' digests when all requests are read
(the first design; rejected above); a clone of the ok digest merged with the failed one
(rejected earlier: `TDigest.Clone` seeds the clone's generator by drawing from the
original's, which changes every later insertion into the original, so the final percentiles
would depend on how often the progress block happened to draw).

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
duration is at most 2⁶³−1 ns, under 2⁴⁴ ms, and a count is at most 2⁶³, so the sum stays
under 2¹⁰⁷ and the sum of squares under 2¹⁵¹, and neither can wrap for any run the command
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
summary adds three digests and a handful of integers and keeps nothing per name.

**Measurement** (the two digests of the first design at the library defaults — §3 has the
figures for three — a log-normal body around 40 ms
with 0.5 % at 60 s and 5 % failed, heap after `runtime.GC()`):

| Requests | Heap held | Centroids, both digests / the larger | Per insertion |
|---|---|---|---|
| 1 000 000 | below what `HeapAlloc` resolves (0.00 MiB) | 1774 / 981 | 290 ns |
| 10 000 000 | 0.05 MiB | 2045 / 1112 | 286 ns |
| 100 000 000 | 0.06 MiB | 2318 / 1246 | 311 ns |

The goal stays a test in the ordinary suite, as `TestScanMemoryDoesNotGrowWithTheLog` is:
replay a 16 MiB and a 64 MiB log through `reporttest.Replay`, fail if either passes the goal
or if four times the log costs meaningfully more. Throughput is kept by a benchmark and not
gated.

**What the gates cost**. Both collect first and measure the live heap rather than the heap in
use on either side of the walk, which carries the allocator's own spans: 0.06 MiB for a
64 MiB log, where the looser measure reads megabytes of noise and would need 256 MiB and a
million names to see past it. The tighter number needs a megabyte of slack rather than eight,
and 64 MiB and 100 000 names are then enough — which matters, because these two tests are the
longest in the package and CI runs them under the race detector and coverage. Checked by
retaining one word and one name per request: both gates fail, at 4.2 MiB and 5.9 MiB. Under
`-race -coverprofile` the package takes 38 s against the 18 s of `main`; the difference is
the third digest of §3, one insertion a request.

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
the test citing the console line. The only percentiles read from those files are those of
3.11.5 and 3.12.0, which this tool's must equal (FR-019, §18); the later versions' are only
described. `stats.json`, which holds the rows per request, is not copied: nothing reads it
before #52. The live runs of §17 bring their own recordings, under
`internal/report/testdata/live/gatling/`.

**Rationale**: milestone v0.13.0 left them out because nothing read them (its research §9).
They are MIT with the rest of the corpus; `PROVENANCE.md` gains the rows.

**One reader of Gatling's figures**. `reporttest` is the only place that decodes a
`global_stats.json` or a console and compares a summary with what Gatling recorded:
`ReadGlobalStats`, `ParseConsole` and `Compare` serve the corpus runs and the live ones
alike, so a second decoder cannot drift from the first over what equal means — two of them
would have to agree on whether a share is compared as a two-decimal string or within 0.005,
and a rate exactly or rounded. `reporttest.Versions` and `IsReference` are likewise written
once, so the rule about which Gatling versions are a reference cannot be spelled two ways,
one of which would stop treating a 3.11.6 recording as one. `reporttest.Items` is the one
reader stub, and `reporttest.Samples` the one reading of what a run holds, which
`EtalonSamples`, `Durations` and any test that feeds its own digest all go through, so that
none of them can lose the check for a negative duration the others make.

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

**Superseded in part, 2026-09-17**: constitution v3.0.0 (T011) replaced the parity bullet
quoted as unchanged above. A Gatling run's percentiles must equal Gatling 3.11's, and the
tests must assert it (§18).

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
  from a fresh digest the ok and failed digests are merged into, the same read the final
  figures use, which leaves both digests as they were (§3). Counts are exact below 10 000 and abbreviated above. No rate: it needs the final
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

**Cost**: a clock read every 1024 items and, five times a second, one merge of at most a few
thousand centroids into a fresh digest and nine quantile reads. Not measured yet; SC-010 caps
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

**Taking a terminal for a terminal**. The device is asked whether it is one, in the standard
library and with no new module: on Linux and macOS the terminal attributes `isatty(3)` reads,
through `syscall.Syscall6` with `TCGETS` and `TIOCGETA`; on Windows the console's own mode,
read and never changed, where the `ENABLE_VIRTUAL_TERMINAL_PROCESSING` flag says the console
interprets escape sequences rather than printing them — Windows sets no TERM, and goreleaser
ships windows/amd64 and windows/arm64, which TERM alone says nothing about. TERM still has to
name a terminal that is not dumb where a terminal sets it. Asking `os.ModeCharDevice` and
TERM instead would take `/dev/null` for a terminal, and `2>/dev/null` would then cost the
walk a redraw five times a second and write control sequences where FR-037 allows none.
Verified on a real pty: colour and the block at `TERM=xterm-256color`, nothing at
`TERM=dumb`, at an empty TERM, under `NO_COLOR`, or when the stream is a file, a pipe or
`/dev/null`.

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

**Deciding once what may be drawn**. Quiet, no-colour and the two terminal checks decide
three things between them — the summary's colour, whether the block is drawn, and the
block's colour — and `modesFor` reads them once from the command into `outputModes`, which
the command carries. Reading them in two places is how the summary on standard output and
the block on standard error come to disagree about a terminal or about `--no-color`, and how
a field named for one rule comes to fold a second into itself. `streams.modes` is the rule
alone, a pure function a test holds to every combination, since a test has no terminal to
reach it through. Each of the smaller rules is written once beside it: `failedStyle`, the
label painter that pads outside the paint, one renderer for a share or a rate, and one list
of the block's ranks from which its heading and its rows are both built, so that a column
cannot end up under the wrong name. Nothing derivable is stored: the block counts its draws
rather than keeping a flag, takes `*report.Source` rather than an interface with one
implementer, and reads an unknown size as 0.

## 17. Live Gatling runs

**Decision** (maintainer, 2026-09-17: a live test with Gatling running 3 to 5 minutes at
50 rps, the same data on every version, galaxio's own template, 3.11 as the reference): the
summary and the percentile rule are proved on real Gatling runs as well as on the corpus.

- **The load** comes from galaxio's own `gatling/scala-sbt` template, rendered by the galaxio
  binary's `template init` and changed only through its inputs: the Gatling version,
  gatling-picatinny 1.27.0 — which builds and runs on every version from 3.11.5 to 3.15.1,
  measured — gatling-sbt 4.19.1, the stub's URL, `Intensity=3000 rpm`, a 10-second ramp and
  4 minutes steady. Its `Stability` simulation sends `GET /` with a `status is 200` check:
  12 250 requests a run.
- **The same data on every version**: the stub answers request i by a plan seeded with i
  alone (`livePlan`, seed 20260917) — 85 % wait 5–40 ms, 8 % 80–400 ms, 3 % 800–1199 ms, 2 %
  1200–2000 ms, and 2 % answer 500 within 5 ms. Every version is served the same waits and
  fails the same requests; what differs is the milliseconds each JVM, scheduler and HTTP
  client adds, which the logs record as they are.
- **What is held**: every non-percentile whole-run figure equal to the console's Global
  Information block and, up to 3.13.x, to `global_stats.json`; every percentile within the
  rank rule of §2 over the run's own log and equal to Gatling 3.11's (§18); later versions'
  described in the test log and not held, because of tdunning/t-digest#230.
- **Two layers**: `TestReportLiveGatling` (integration tag, `GALAXIO_LIVE_GATLING=1`) runs
  Gatling itself; `TestSummaryMatchesLiveGatlingRuns` holds the committed recordings under
  `internal/report/testdata/live/gatling/` to the same assertions in the ordinary suite,
  without a JDK. `GALAXIO_LIVE_RECORD` writes a new set, and `GALAXIO_LIVE_RECORDINGS` points
  the ordinary test at one.
- **The recordings** (`RECORDING.md` beside them): five runs of 2026-09-17, one version after
  another on one machine, each keeping its `simulation.log` byte for byte (compressed), the
  Global Information block of its console — the rest of the console names paths on the
  recording machine — and `js/global_stats.json` up to 3.13.1. They were rendered from
  templates-gatling as merged at 564da8b (pack 0.15.1). The harness renders the published
  pack and sets the two sbt versions the published pack's defaults changed back to 0.15.1's,
  because a registry cannot pin a pack to a commit: galaxio appends the pack's version to a
  GitHub source as its ref.

**Measured** on the recordings, all requests; a misplacement is a fraction of the run's
12 250 requests, and the rule allows 0.00198 at p95 and 0.00048 at p99:

| Gatling | Requests (ok, failed) | Request at the p95 / p99 rank | This tool p95 / p99 | Gatling printed p95 / p99 | Gatling's p99 misplaced by | Gatling's percentiles |
|---|---|---|---|---|---|---|
| 3.11.5 | 12 250 (12 010, 240) | 815 / 1578 | 813 / 1575 | 813 / 1575 | 0.00012 | held, the reference |
| 3.12.0 | 12 250 (12 010, 240) | 814 / 1578 | 814 / 1576 | 814 / 1576 | 0.00012 | held, the reference |
| 3.13.1 | 12 250 (12 010, 240) | 815 / 1578 | 813 / 1580 | 723 / 1670 | 0.00167 | described |
| 3.14.9 | 12 250 (12 010, 240) | 816 / 1577 | 813 / 1575 | 721 / 1670 | 0.00167 | described |
| 3.15.1 | 12 250 (12 010, 240) | 815 / 1577 | 813 / 1575 | 704 / 1671 | 0.00167 | described |

Every other whole-run figure equals Gatling's on every version, the rates and shares to the
two decimals of the console and exactly to `global_stats.json`. This tool's percentiles
misplace their rank by at most 0.00012 on every run, and every version's p50 and p75, Gatling's
included, by nothing. Gatling 3.11.5 and 3.12.0 printed the same four percentiles as this
tool on their own runs, which T012 asserts (§18). From 3.13.1 Gatling's p95 lands between
400 and 800 ms, where the stub sends no response, so its rank is off by only 0.00078; its p99
misplaces the rank by 0.00167 — about 20 of the 12 250 requests where the rule allows 6 —
which is tdunning/t-digest#230 on a real run.

**Alternatives considered**: parsec's hand-written corpus probe (not taken: the maintainer
asked for galaxio's own template, which exercises `template init` on the way); replaying the
corpus at a larger scale (not taken: synthetic, and it shows no real JVM's timing); a random
stub (rejected: the versions would not see the same data); running Gatling in every CI
build (not taken: minutes a version and the network, so it is gated).

## 18. Equal to Gatling 3.11

**Decision** (maintainer, 2026-09-17: `galaxio report` replaces the report Gatling builds, so
its numbers must match Gatling's as in 3.11, "as in t-digest 3.1"; constitution v3.0.0,
T011): every percentile of a Gatling run equals the one Gatling 3.11.x and 3.12.x compute
for the same log, and the tests assert it.

**What Gatling 3.11 and 3.12 compute**, read from `io.gatling.charts.stats.buffers.GeneralStatsBuffer`
in gatling-charts 3.11.5 and 3.12.0, both of which depend on `com.tdunning:t-digest` 3.1: one
`new AVLTreeDigest(100.0)` for all requests and one for each outcome, fed every response
time in log order, and a percentile of `Math.round(digest.quantile(rank / 100.0))`.
`AVLTreeDigest` draws on an unseeded `java.util.Random`, so two runs of Gatling over one log
can print two values for a percentile.

**Why this tool's numbers are the same**: caio/go-tdigest v5 at compression 100 bounds a
centroid by 4·n·q·(1−q)/δ, adds a value to its nearest centroid and interpolates between
centroids, which are the three choices t-digest 3.1's `AVLTreeDigest` makes.

**The read, and the half**. The library's `Quantile` is `AVLTreeDigest.quantile`
operation for operation, and on amd64 it returns t-digest 3.1's doubles bit for bit. On
arm64 the Go compiler fuses each product of the interpolation into the addition after it,
which moves the last bit, and a percentile at a half then rounds the other way: on a live
run of 1 222 successful requests the 95th percentile is 398.5, which the fused read gave as
398.49999999999994 where t-digest 3.1 gives 398.5000000000001 and `Math.round` 399. The
obvious answer, rounding up a value within a slack below the half, is wrong: values one bit
below a half are real in Java too — 48.49999999999999 for the 95th percentile of [1, 51],
which Gatling prints as 48 — and a slack of 1e-14 makes 49 of them, 84 times in 31 400 small
sets on amd64, where no slack disagrees never. So `percentiles.go` reads the
digest itself, from `ForEachCentroid`, in t-digest 3.1's order of operations with every
product converted to `float64` before it is used, which the language guarantees is never
fused; and `roundHalfUp` compares the exact fraction with a half, as `Math.round` does. Over
the same 31 400 sets that read never disagreed with the jar on either architecture.
`TestPercentileEqualsTDigest31` pins eight such sets to what the jar gave, and fails on
arm64 against the library's own read.

**The etalon**: `internal/report/testdata/etalon/Etalon.java`, run by the JDK's source
launcher with both jars, each in a class loader of its own, and a run's requests on standard
input in log order (`reporttest.EtalonSamples`). It builds the AVL digest of t-digest 3.1 once
for each seed from 1 to 200 and prints every value it gave, and builds `MergingDigest(100)` of
t-digest 3.3 once beside it. Its output is kept as `etalon.tsv` beside each of the ten
recordings and each of the four synthetic runs of §3. `TestEtalonRecordings` (integration tag)
reruns it and requires those files byte for byte; the jars come from `GALAXIO_TDIGEST_JARS`,
the local Maven repository or the Coursier cache a Gatling build fills, and are pinned by
SHA-256.

**What ties an etalon to its run**. A test that skips proves nothing, and CI provides neither a
JDK nor the jars, so `TestEtalonRecordings` skipped there and the ordinary suite trusted
fourteen files nothing checked. Two things answer it. The etalon now prints, in its header,
how many sample lines it read and their SHA-256, `ReadEtalon` requires both and refuses a
column and rank given twice, and `TestEtalonsAreOfTheirRuns` — in the ordinary suite, with no
JDK — recomputes them from each run: an etalon.tsv copied from another recording fails.
And the integration job of `ci.yml` and `release.yml` sets up a Temurin JDK and fetches the
two jars from Maven Central through `.github/actions/tdigest-jars`, which checks each against
the SHA-256 the test pins and sets `GALAXIO_ETALON_REQUIRED=1`, under which a missing JDK or
jar fails the test instead of skipping it.

**What the tests hold**, through `reporttest.ComparePercentiles`: every percentile at ranks
50, 75, 95 and 99, for all, ok and failed requests, is a value the AVL digest gives for the
log; on 3.11.5 and 3.12.0 every percentile Gatling printed equals this tool's, or is another
value the digest gives, which its generator can draw; what later versions printed and what
`MergingDigest` gives are described in the test log. `TestPercentilesEqualGatling311` does it
on the five corpus runs, `TestSummaryMatchesLiveGatlingRuns` on the five live recordings, and
`TestReportLiveGatling` on a fresh run, with the etalon run over its log.

**Measured** on the ten recordings (2026-09-17):

| Recordings | Percentiles | A value Gatling 3.11's digest gives | Equal to what Gatling 3.11.5 and 3.12.0 printed | Equal to `MergingDigest` 3.3 | Equal to the request at the rank |
|---|---|---|---|---|---|
| corpus, five runs | 60 | 60 | 24 of 24, in `global_stats.json` | 53 | 52 |
| live, five runs | 60 | 60 | 24 of 24, on the console and in `global_stats.json` | 40 | 42 |

Over 200 seeds the digest gives two values for 2 of the 120: the 99th percentile of successful
requests on the live 3.12.0 run, 1585 or 1586 (Gatling printed 1586, this tool gives 1586),
and the 75th on the live 3.15.1 run, 37 or 38.

| All requests, p95 | This tool | Gatling 3.11's digest | Gatling printed | `MergingDigest` 3.3 | Request at the rank |
|---|---|---|---|---|---|
| corpus 3.13.1 | 1427 | 1427 | 1072, by 3.13.1 | 1502 | 1502 |
| live 3.11.5 | 813 | 813 | 813, by 3.11.5 | 682 | 815 |

**Where they differ**: caio's `AddWeighted` merges into a centroid in place, and a centroid
whose mean does not change stays where it is; t-digest 3.1's `AVLTreeDigest` removes the
centroid from its tree and adds it back, after every centroid of an equal mean. Both hold the
same centroids, in another order inside a run of equal means, and `Quantile` interpolates
between the centres of neighbouring centroids — so at the edge of such a run the two can
round to the two sides of a boundary. A fresh live run of 3.11.5, made for quickstart §8,
showed it: 240 failed requests, 43 of 5 ms merged partly into pairs, and the 75th
percentile read as 5.25 by caio (5) and 5.5 by t-digest 3.1 (6, every seed; Gatling printed
6). Nothing was recorded between 5 and 6. Across 105 recorded and synthetic sets and six
ranks it is 1 value in 630, and none of the 120 on the committed recordings. A copy of the
library that moves a merged centroid as t-digest 3.1 does matched all 630 at 11 % more per
insertion; the maintainer chose to keep the library unmodified (constitution v3.1.0), so
`ComparePercentiles` describes such a value and reports any other. What it describes is held
to the amendment's words (`acrossEqualRuns`): the two recorded response times around
the two values have nothing recorded between them and were each recorded at least twice, the
position rank/100·(n−1) lies inside those two runs, and the rank rule holds. Its first form
Asking only that nothing be recorded between the two values is far too wide: it lets through
900 and 1502 for the 1427 of the 3.13.1 run, whose response times leave a gap there, and any
difference of one wherever values are distinct. The tests of the ten committed recordings
refuse the description altogether, since none of them needs it.

**Where two builds differ** (maintainer, 2026-09-17: keep the library, describe it).
The fusion of "The read, and the half" also happens inside the library, where a value joins
a centroid: `(x1*w1 + x2*w2) / (w1 + w2)` is one multiply-add and a division on arm64, and
two products, a sum and a division on amd64 and in Java. The means then differ in their last
bit; a later search for the nearest centroid can fall the other way; and the two builds end
with different centroids. Measured by running one test natively and as `GOARCH=amd64`: 22 of
1 500 synthetic digests hold a different number of centroids, and at the default ranks 2 of
7 500 percentiles differ, each by 1 ms. For 12 000 requests uniform over 0–2599 ms the 99th
percentile is 2572 on arm64 and 2571 on amd64, and t-digest 3.1 gives 2572 for every seed;
for 6 000 over 0–3799 ms the 95th is 3610 and 3609. Neither build is the one that is always
right. All ten recordings and the four synthetic runs give the same numbers on both. It
cannot be removed without changing the library's source, which the maintainer ruled out for
the order of equal centroids as well, so it is described: in the README, in the contract's
guarantees and here. FR-018's "the same log always yields the same percentiles" holds for
one build.

**Why `MergingDigest` is not the reference**: it has no defect, and t-digest 3.3 recommends
it, but it is another algorithm. It merges neighbours in sorted order under another scale
function and reads a quantile another way. On the run of 102 requests it returns the recorded
value, 1502. On the live runs one of its centroids spans the gap between 400 and 800 ms the
stub leaves, and its 95th percentile falls into that gap, at 682–719 ms, where the requests at
the rank took 815–829 ms. It matches neither Gatling 3.11 nor this tool, so the tests describe
it and hold it to nothing.

**Alternatives considered**: `MergingDigest` as the reference (rejected, above); the exact
percentile over every response time (rejected: Gatling 3.11 does not print it, 813 against
815, and it needs every sample); the read by rank of caio/go-tdigest#42 (rejected: 1502
where Gatling 3.11 prints 1427); a Go copy of t-digest 3.1's `AVLTreeDigest` (not taken: the
library already gives its numbers, and a copy is one more thing to prove).
