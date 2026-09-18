# Contract: `galaxio report` — the summary

**Feature**: [spec.md](../spec.md) | **Plan**: [plan.md](../plan.md) | Extends
[`specs/004-report-records/contracts/cli.md`](../../004-report-records/contracts/cli.md),
which still holds for the tool, the path, `-o`, the run description and its failures.

## Synopsis

```text
galaxio report <tool> [PATH] [--percentiles RANKS] [--bounds LOW,HIGH]
                             [-o FORMAT[,FORMAT…]] [--quiet | --verbose] [--no-color]
```

Read a finished run, describe it as milestone v0.13.0 does, and summarise it as a whole on
standard output. Needs the log only; the HTML report may be absent. Writes no file.

## Flags added by this milestone

| Flag | Value | Default | Meaning |
|---|---|---|---|
| `--percentiles` | comma-separated ranks, each a number above 0 and at most 100 | `50,75,95,99` | the percentiles to report. Ranks are reported once each, in increasing order, whatever order they were given in. Checked while the flag is parsed |
| `--bounds` | `LOW,HIGH`: two whole, non-negative numbers of milliseconds, `HIGH` greater than `LOW` | `800,1200` | the two boundaries of the response-time bands. Checked while the flag is parsed |

Neither has a shorthand, and no flag names an output file. `-o` is unchanged: `stats`,
`global_stats` and `yml` stay reserved and every value is refused. Those products are the
only files this command is ever to write; figures for a single request or group, and every
form a program reads, arrive with them and not before.

`--quiet` (root) silences standard output and the progress block; warnings and errors are
still written. A CI job that needs only the exit code runs the command that way.

`--no-color` (root) and the `NO_COLOR` environment variable, which did nothing on this
command until now, switch the colours below off.

## Standard output

The run description of v0.13.0, then a blank line and the summary. Recorded 3.13.1 run,
default flags:

```text
run         io.galaxio.parsec.corpus.CorpusSimulation
id          io.galaxio.parsec.corpus.CorpusSimulation
started     2026-09-06T04:47:41.110Z
tool        gatling 3.13.1
log         binary, internal/report/testdata/corpus/gatling/3.13.1/simulation.log
found by    path
span        2026-09-06T04:47:41.603Z .. 2026-09-06T04:47:44.829Z (3.226s)
requests    102 (84 ok, 18 failed)
groups      12 traversals
users       12 events
errors      6
assertions  10 payloads the run declared

102 requests · 25.5 req/s      ✓ 84 ok · 82.35 % · 21/s      ✗ 18 failed · 17.65 % · 4.5/s

response time, ms       min   mean    std    p50    p75    p95    p99    max
all                       0     89    353      1      1   1427   1502   1503
✓ ok                      0    108    387      1      1   1502   1502   1503
✗ failed                  0      1      1      1      2      4      4      4

███████████████░░░░░  76.47 %    78  ok under 800 ms
░░░░░░░░░░░░░░░░░░░░      0 %     0  ok 800 to 1200 ms
█░░░░░░░░░░░░░░░░░░░   5.88 %     6  ok 1200 ms and over
████░░░░░░░░░░░░░░░░  17.65 %    18  failed

times in ms · percentiles are galaxio's t-digest estimates, interpolated
```

Every non-percentile figure above is what Gatling recorded for this run, and every
percentile is what Gatling 3.11's digest gives for its log. **`p95` of all requests is 1427
although no request took between 8 and 1501 ms**: the default quantile interpolates, as
Gatling 3.11's does, which the README documents with this run as its example; Gatling 3.13.1
printed 1072 for it because of
[tdunning/t-digest#230](https://github.com/tdunning/t-digest/issues/230), and the read by
rank of [caio/go-tdigest#42](https://github.com/caio/go-tdigest/pull/42) would print 1502.

**The one change to the run description**: its requests line said `102 (84 ok, 18 ko)` in
v0.13.0. `ko` is Gatling's word; the two outcomes are `ok` and `failed` on every line this
command writes.

### The layout

Nothing in it is Gatling's: the same four parts summarise a run of any tool the command
comes to read.

1. **Headline** — three segments, six spaces apart: `<N> requests · <rate> req/s`, then
   `✓ <N> ok · <share> % · <rate>/s` and `✗ <N> failed · <share> % · <rate>/s`. The share is
   of all requests. The three stand on one line while that line fits 100 columns — it is 90
   for the run above — and one per line beyond that; the rule looks at the text, never at
   the terminal.
2. **Response-time table** — a heading line `response time, ms` with the statistics as
   columns, in this order: `min`, `mean`, `std`, one `pNN` per requested rank in increasing
   order (a rank such as 99.9 is `p99.9`), `max`. Then one row each for `all`, `✓ ok` and
   `✗ failed`. The label column is 20 wide; every figure is right-aligned in 7 columns, and a
   column grows to keep two spaces before a longer value.
3. **Bands** — one line each: a bar of 20 cells, the share of all requests, the count, the
   label. `█` fills `floor((40·count + total) / (2·total))` cells — the share rounded half up
   to a twentieth — `░` the rest; a band that holds any request fills at least one cell, and
   a band that does not hold them all never fills the twentieth. Labels carry the boundaries
   in force: `ok under <LOW> ms` is `t < LOW`, `ok <LOW> to <HIGH> ms` is
   `LOW <= t < HIGH`, `ok <HIGH> ms and over` is `t >= HIGH`, and `failed` is every failed
   request whatever its time. With `--bounds 5,1000` they read `ok under 5 ms`,
   `ok 5 to 1000 ms`, `ok 1000 ms and over`.
4. **Closing line** — the unit of every time, and the percentile statement: whose estimate a
   percentile is and how it is defined.

**Numbers**: times are whole milliseconds. Rates and shares have at most two decimals,
trailing zeros removed, a point as the decimal separator and no thousands separator. The
mean and the deviation are rounded half up (0.5 ms prints as 1).

**Absence**: a figure that does not exist is `-`, never `0`. An outcome that holds no
request prints `0` for its count, `0 %` for its share, `0` for its rate, and `-` for every
time and percentile. A run with no request at all prints `-` for every share. When the run
cannot be timed, every rate is `-`.

**Colour**, only when standard output is a terminal whose `TERM` is set and is not `dumb`,
and neither `--no-color` nor `NO_COLOR` is given: `✓ … ok` green (SGR 32); `✗ … failed` and
the filled part of the failed bar red (SGR 31), and only when anything failed; the table
heading, the unfilled part of every bar and the closing line faint (SGR 2). Everywhere else
the same text is written with no escape sequence in it. Colour says nothing the text does
not.

The text is UTF-8 and is for people. It carries no row for a single request or group,
however many the run has, and `--quiet` suppresses the whole of it.

## Standard error

### The progress block

While the log is read — **only when standard error is a terminal whose `TERM` is set and is
not `dumb`, and `--quiet` is not given** — a block of six lines is redrawn in place:

```text
⠹ reading simulation.log  ━━━━━━━━━━━───────────────────   37 %  0:12 left
figures so far · times in ms · percentiles are t-digest estimates, interpolated
                       count   share    min   mean    p50    p95    p99    max
all requests           41.2M              0     47     38    212    840  60000
  ✓ ok                 40.9M  99.3 %      0     44     38    205    610   9800
  ✗ failed              291k   0.7 %      1    480     12   2100  60000  60000
```

- **First line**: a spinner that advances on every redraw; the log's file name, cut to 24
  columns; a bar of 30 cells, `━` for the share of the log's bytes read against the size it
  had when opened and `─` for the rest; that share as a percentage, never above 100; and,
  from the second draw, the time left as `m:ss` or `h:mm:ss`, estimated from the bytes still
  to read. When the size is unknown the line is the spinner and the name alone.
- **Second line**: `figures so far · times in ms · percentiles are t-digest estimates, interpolated`, faint, whatever the read has reached — the statement
  Principle II asks of every output that carries a percentile.
- **Figures**: the requests read so far — exact below 10 000, then abbreviated (`12.3k`,
  `40.9M`, `1.2G`); the share of ok and failed with one decimal; and the minimum, the mean,
  the median, the 95th and 99th percentile and the maximum so far, in whole milliseconds.
  The columns are these whatever `--percentiles` asks of the report: a label of 20, `count`
  and `share` of 8, six figures of 7 — 78 columns. No rate: it needs the span of the whole
  run. An outcome with no request yet prints `-` for its times.
- **When**: first drawn 500 ms into the read, then at most five times a second and at least
  once a second while records arrive. A read that ends sooner draws nothing — the corpus
  runs never show it.
- **How**: each redraw moves the cursor up six lines (`ESC [ 6 A`) and rewrites every line
  after erasing it (`ESC [ 2 K`); removal moves up and erases to the end of the screen
  (`ESC [ J`). No terminal mode is changed — no hidden cursor, no alternate screen, no
  wrapping switched off — so a command that is killed leaves the block behind and nothing
  else. Colours follow the rule of standard output, applied to standard error.
- **Width**: every line is at most 79 columns; a longer one is cut, never wrapped. The
  terminal's width is never asked for, so a terminal narrower than 80 columns wraps the
  block and may keep remains of it.
- **Removal**: before any warning or error is written, before the report is printed and
  before the command returns — after a complete read, a log cut short, a damaged log and an
  interrupt alike.
- Standard output and the exit code are identical with and without it.

### Warnings and errors

| When | Line |
|---|---|
| requests with no recorded end were counted but took part in no timing figure | `report: warning: <N> requests have no recorded end and take part in no timing figure` — printed even under `--quiet`, because it qualifies the numbers |
| the log's version is newer than the verified range | as in v0.13.0 |
| any failure | `Error: <message>`, written by the root command |

## Exit codes

| Code | Meaning | New with this milestone |
|---|---|---|
| 0 | the run was read to the end and summarised | — |
| 1 | runtime failure while executing a valid invocation | `<log>: the run spans no time (<start> .. <end>): no request rate can be computed`, after the summary with every rate `-`; `<log>: <N> requests have an outcome the source lost: they are neither ok nor failed and the summary does not add up`, after the summary. Both at once, or one of them and a log cut short, are joined into one error. **Both were exit 0 in v0.13.0**; no Gatling log is known to produce either |
| 2 | usage failure, nothing on standard output | `invalid argument "0" for "--percentiles" flag: percentile rank "0" is not a number above 0 and at most 100`; `invalid argument "1200,800" for "--bounds" flag: boundaries are two whole numbers of milliseconds, the second greater than the first` |

A log cut short keeps the behaviour of v0.13.0 and extends it: the summary of what was read
is printed and the exit code is 1. A damaged log prints no summary, as in v0.13.0.

## Behavioural guarantees

- One forward pass; no sample is retained and no figure is kept per request name. Heap in
  use stays under the 32 MiB goal of v0.13.0 for a log of any size and any number of
  distinct names — the summary adds two digests, about 60 KiB — and that is a test the
  ordinary suite runs.
- The command creates, changes and removes no file.
- Counts, minimum, maximum, mean and standard deviation are exact; they equal what Gatling
  recorded for every run of the corpus.
- Percentiles are estimates from a t-digest at the library's defaults, labelled as this
  tool's; for a Gatling run each equals the one Gatling 3.11's digest gives for the same log
  (research.md §18). The same log always gives the same percentiles. Up
  to 200 requests an outcome's percentile lies between the two recorded response times
  around its position; at any size it misplaces its rank by at most 4·q·(1−q)/100 of the
  requests plus one (research.md §2).
- Two summaries of the same run are identical, whether or not a progress block was shown
  while reading.
- A run whose author changed Gatling's percentile ranks or band boundaries is reported at
  this command's own settings; the log does not record theirs, and nothing claims a match.

## Test seam

`runReport(ctx, opts reportOptions) (reportOutput, error)`: `opts` gains `Percentiles` and
`Bounds`; the output gains the `report.Summary`. The console text is a pure function of that
output and of one boolean — colour or not — so a test pins both renderings without a
terminal. `opts` also carries what the progress block depends on and a test must control:
whether standard error can redraw in place, and the clock. With both injected and a clock
that advances on every reading, the frames drawn are deterministic and are asserted as
text, control sequences included.
