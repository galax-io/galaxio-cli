# Data Model: Summarise a Finished Gatling Run

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Contract**:
[contracts/cli.md](contracts/cli.md)

Nothing here is persisted. The run location, the run description, the bounds and the
outcome are the shared library's types, used unchanged as in milestone v0.13.0
([004 data model](../004-report-records/data-model.md)). What follows is what this feature
adds in `internal/report/`, because the library computes no statistic.

## Summary options

What the caller chose; validated before any work and never changed by the run.

| Field | Meaning | Rule |
|---|---|---|
| percentile ranks | the percentiles to report | each above 0 and at most 100; reported once each, in increasing order; default 50, 75, 95, 99 |
| band boundaries | lower and upper boundary in whole milliseconds | non-negative, upper greater than lower; default 800 and 1200 |

## Outcome figures

The figures of one outcome — ok or failed — over the whole run. Accumulated in one pass;
nothing else about a sample is kept.

| Field | Meaning |
|---|---|
| count | every request of this outcome, with or without a recorded end |
| timed | those of them that carry a duration; the divisor of the mean and the deviation |
| minimum, maximum | extremes in whole milliseconds; meaningless while `timed` is zero |
| sum | sum of durations in milliseconds, 128 bits kept as two 64-bit words |
| sum of squares | 192 bits, kept as three 64-bit words |
| digest | the library's t-digest at its defaults, created on the first timed request |

A duration is at most 2⁶³−1 ns, under 2⁴³ ms, and a count is at most 2⁶³, so the sum stays
under 2¹⁰⁶ and the sum of squares under 2¹⁴⁹: neither can wrap, for any run the command can
read. A negative duration, which the library promises never to yield, counts as no recorded
end.

Derived when the summary is produced, in integers: the mean `floor((2·sum + timed) /
(2·timed))`; the population deviation about the unrounded mean, rounded half up by comparing
`4·numerator` with `(2k + 1)²·denominator`; each percentile as the digest's default
quantile, rounded half up. While `timed` is zero every one of them is **absent**, never 0.

**All requests**: not stored. Count, timed, sum and sum of squares add; minimum and maximum
combine; its percentiles are read from a fresh digest into which the ok digest and then the
failed one are merged — never from a clone, whose seed the library draws from the original's
generator, so that reading the figures in the middle of a walk changes nothing that follows.
A failure therefore reaches the figures of all requests and never a figure of ok requests.

## What feeds them

Every request sample feeds the figures of its own outcome, and nothing else does. A group
traversal is counted by the tally of v0.13.0 as before and takes part in no figure — neither
its cumulated response time nor its wall-clock duration (FR-015).

## Not in this model

Figures for a single request or a single group (FR-016). Nothing in this milestone shows
them, so nothing is keyed by the library's `model.Position` and memory does not depend on
how many names a run holds. They arrive with the `-o` products; an outcome's figures above
are what such a row will be made of.

## Response-time bands

Four counters: ok responses under the lower boundary, from the lower boundary to under the
upper one, at or above the upper one; and failed requests, whatever their time. Each share
is the counter over all requests of the run, times one hundred, in that order of operations.

## Run span in seconds

The library's bounds give where the run begins and ends. The divisor of every rate is that
span in whole seconds, rounded up, and it is one value for all three rates.

| State | Rates | Exit |
|---|---|---|
| span above zero | count over seconds | — |
| span exactly zero | absent | 1, naming both instants |
| bounds the library cannot resolve | absent | 0 |

A count of zero over a known span is a rate of 0, not an absence.

## Counted apart

| What | Where it is counted | What it takes part in |
|---|---|---|
| a request with no recorded end | its outcome's `count`, not `timed` | the count, the share and the rate only; reported on standard error |
| a request whose outcome the source lost | the summary's own counter, neither ok nor failed | nothing; reported, and the command exits 1 |

## Read progress

How far the read has got: the bytes of the log read so far, the size the log had when it was
opened, and the time since the read began. The bar, the percentage and the time left of the
progress block are made of these three; the figures beneath them are read from the two
outcomes as they stand. It belongs to the open source, not to the summary, and no part of it
reaches the report or the exit code.

| State | Progress block |
|---|---|
| standard error can redraw in place, `--quiet` unset, 500 ms into the read | drawn, then redrawn at most every 200 ms |
| the read ended earlier than that | never drawn |
| standard error is not a terminal, `TERM` is unset or `dumb`, or `--quiet` | never drawn, and no control sequence written |
| the size is zero or the log is not a regular file | drawn without a bar, a percentage or a time left |
| the read ends, by any ending | erased before anything else is written |

## Summary

What one pass produces: the tally of v0.13.0 unchanged, the options in force, the figures
of the two outcomes, the bands, the span in seconds with the reason it is missing when it
is, and the two counters above. The console text is a pure function of it, of the run
description and of whether colour is on.

## Known divergence (documented, not modelled)

The digest's default quantile interpolates between neighbouring recorded values, so where
response times have a gap a percentile can be a value no request had: 1427 ms for the 95th
percentile of the recorded 3.13.1 run, whose request at that rank took 1502 ms. How far a
percentile may differ is a rule the tests hold every percentile to (research.md §2): up to
200 requests it lies between the two recorded values around its position, and at any size
it misplaces its rank by at most 4·q·(1−q)/100 of the requests plus one. The tests pin the
corpus values and hold them to that rule, the README explains both, and
[caio/go-tdigest#42](https://github.com/caio/go-tdigest/pull/42) removes it upstream. No
field or flag of this model exists for it: taking the upstream read is a later change to
one call.
