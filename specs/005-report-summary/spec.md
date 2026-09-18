# Feature Specification: Summarise a Finished Gatling Run

**Feature Branch**: `113-report-summary`

**Created**: 2026-09-17

**Status**: Draft

**Input**: User description: "только пометь что есть pr который выправит данные но сейчас мы
используем по умолчанию тот что дает caio" — given for
[galax-io/galaxio-cli#51](https://github.com/galax-io/galaxio-cli/issues/51), the feature this
specification describes. In English: only note that a pull request exists that will
straighten the figures out, and that for now the command uses by default what the `caio`
library gives.

**Tracking**: [milestone `v0.14.0 Report summary`](https://github.com/galax-io/galaxio-cli/milestone/4);
[galax-io/galaxio-cli#51](https://github.com/galax-io/galaxio-cli/issues/51)

## Background

The numbers of a finished run — request counts, response-time extremes, mean, deviation,
percentiles, the request rate and the response-time bands — live in an HTML report meant for
a browser. Gatling prints them once to the console and never again. A CI job that wants to
print the outcome of a load test, or an engineer holding an archived run, cannot see them
without a browser and a downloaded artefact.

Milestone `v0.13.0` made `galaxio report gatling [PATH]` read a run and say what it holds. This
milestone makes the same command **summarise** it: below the run description it prints the
figures of Gatling's console summary for the run as a whole, computed from the log in the
same single pass and without the HTML report being present. While a long log is being
read, a terminal shows how far the read has got and what the figures are so far.

The command writes no file in this milestone. Figures for every request and every group are
not printed on the console either: they belong to the report products `-o` already names —
`stats`, `global_stats` and `yml`, the only files this command is ever to write — and
arrive with the milestones that deliver those products.

The arithmetic is this repository's. The shared library `parsec` yields the primitives — the
position of a sample, the bounds of a run, the outcome the source recorded — and exports no
statistic. How each non-percentile figure is computed was settled against the files Gatling
itself wrote for the recorded corpus (issue #51, amendment of 2026-09-16), and this
specification adopts those definitions as requirements.

### Percentiles in this milestone

**Maintainer decision, 2026-09-17.** Percentiles are taken from a t-digest,
`github.com/caio/go-tdigest` (MIT), and read with **the quantile that library gives by
default**.

That quantile interpolates between neighbouring samples or clusters, so on a run whose
response times have a gap it can print a value no request had. The recorded 3.13.1 run is
the example: 96 requests took at most 7 ms, six took 1502–1503 ms, and nothing fell in
between. The 95th percentile of those 102 requests is the 97th smallest one, 1502 ms; the
library's default quantile gives 1427 ms, a point on the line between the 96th request and
the 97th.

**A pull request that straightens this out exists.**
[caio/go-tdigest#42](https://github.com/caio/go-tdigest/pull/42) adds a read by rank that
does not interpolate; on that run it returns 1502, and on every recorded run it returns the
value of the request standing at the percentile's rank. It is open and unreleased. Until a
release of the library carries it, this command prints what the library gives by default and
says so; taking the new read is a follow-up change, not part of this milestone.

Gatling's own percentile for that run, 1072 ms, is not a target either: it comes from a
defect in the digest library Gatling uses, reported as
[tdunning/t-digest#230](https://github.com/tdunning/t-digest/issues/230), and Gatling does not
reproduce it between two readings of the same log.

## Clarifications

### Session 2026-09-17

- Q: The digest decision does not meet the exactness clause of constitution Principle II —
  amend the clause first, or carry a justified deviation in the plan? → A: Amend the clause
  first, as its own issue and pull request inside milestone `v0.14.0`, the way #112 was
  handled; implementation of #51 starts after that amendment is merged.
- Q: Are the rows for every request and group always printed, or only on request? → A: The
  console carries the whole-run summary only. Figures for every request and group go to
  files, and the only files this command writes are the report products `-o` already names —
  `stats`, `global_stats` and `yml` — each written in full whenever it is asked for. Those
  products arrive with their own milestones, so this milestone writes no file and adds no
  option that names one. `--quiet` silences the console; that is the mode for CI.
- Q: Is the per-message error table of Gatling's console summary part of this milestone? → A:
  No. The error count the run description already prints stays; a breakdown by message is a
  separate, later task.
- Q: Does the option that computes group rows from the wall-clock duration of a traversal
  stay in this milestone? → A: No, it is deferred. With the answer above no group figure is
  shown in this milestone at all; when group rows arrive with the `-o` products they are
  computed from the cumulated response time, which is what Gatling reports and what can be
  checked against it.
- Q: Should the command show that a long read is progressing, and keep its figures moving
  while it reads? → A: Yes, in this milestone, as a P3 story: a live block on standard
  error, redrawn in place while the log is read — a progress bar with the share of the log
  read and the time left, and beneath it the summary so far for all, ok and failed
  requests — only when standard error is a terminal that can redraw in place and `--quiet`
  is not set. When the read ends the block is erased and the report is printed as always.
  Standard output does not change by a byte.
- Q: What should the summary look like on the console? → A: Not like Gatling's console
  table. The layout is this tool's own and universal — the same for a run of any tool the
  command comes to read: a headline with the requests and their rate, then ok and failed
  with their share and rate; a response-time table with one row for all, ok and failed
  requests and the statistics in columns; and the response-time bands as bars with their
  share and count. Chosen by the maintainer from three mock-ups; the exact layout is in the
  plan's contract.
- Q: What are the two outcomes called? → A: `ok` and `failed`, everywhere in this command's
  output; `ko` is Gatling's word. The requests line of the run description, published with
  `v0.13.0` as `102 (84 ok, 18 ko)`, becomes `102 (84 ok, 18 failed)`: the maintainer
  approved that change of published text in this clarification.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See the numbers of a run without the HTML report (Priority: P1)

An engineer, or a CI job, points the command at a finished run and gets the figures of
Gatling's console summary for the run as a whole: how many requests, how many of them
succeeded and failed, the fastest and slowest response, the mean, the deviation, the
percentiles, the request rate, and how many responses fell into each response-time band.
Every figure is given for all requests, for the successful ones and for the failed ones.
They are laid out the way this tool lays out a run of any tool — a headline, a
response-time table and the bands as bars — and not the way Gatling's console does.

**Why this priority**: It is the whole reason for the milestone. Without it the outcome of a
load test cannot be printed by a pipeline or read from an archive.

**Independent Test**: Run the command against every recorded run in the corpus and compare
each non-percentile figure with what Gatling itself recorded for that run — its console
summary, or its `global_stats.json` where it wrote one.

**Acceptance Scenarios**:

1. **Given** the recorded 3.13.1 run, **When** the engineer runs `galaxio report gatling` with
   that directory, **Then** the exit code is 0, the run description of milestone `v0.13.0`
   is still printed, and below it the summary reports 102 requests (84 ok, 18 failed); a
   minimum of 0 ms (0 ok, 0 failed); a maximum of 1503 ms (1503 ok, 4 failed); a mean of
   89 ms (108 ok, 1 failed); a standard deviation of 353 ms (387 ok, 1 failed); and a mean
   rate of 25.5 requests per second (21 ok, 4.5 failed) — the figures Gatling printed for
   that run.
2. **Given** the same run, **When** it is summarised, **Then** the response-time bands report
   78 responses under 800 ms (76.47 %), none from 800 ms to under 1200 ms (0 %), 6 at
   1200 ms or more (5.88 %) and 18 failed (17.65 %).
3. **Given** a text-format run (3.11.5 or 3.12.0), **When** it is summarised, **Then** every
   non-percentile figure equals the one in the `global_stats.json` Gatling wrote for it.
4. **Given** a run directory that holds a `simulation.log` and no HTML report, **When** it is
   summarised, **Then** the output is the same as for the complete directory.
5. **Given** the same run summarised twice, **When** the two outputs are compared, **Then**
   they are identical, percentiles included.
6. **Given** `--quiet`, **When** the command runs, **Then** the summary is suppressed like the
   rest of the report, and errors and warnings are still printed.
7. **Given** a run with several requests and groups, **When** it is summarised, **Then** the
   console carries no row for a single request or group, and the command creates, changes
   and removes no file.
8. **Given** any recorded run, **When** it is summarised, **Then** the two outcomes are
   called `ok` and `failed` on every line of the output — the requests line of the run
   description included, which said `ko` in `v0.13.0` — each outcome is a row of the
   response-time table, each band is a bar with its share and its count, and nothing in the
   layout exists only for Gatling.
9. **Given** standard output is a terminal that says it understands colour, **When** a run
   is summarised, **Then** `ok` is shown in green and `failed`, when anything failed, in
   red; **and given** standard output is anything else — a pipe, a file, a terminal whose
   `TERM` is unset or `dumb` — or `--no-color` is passed, or `NO_COLOR` is set, **then** the
   same text is written with no escape sequence in it.

---

### User Story 2 - Percentiles that say what they are (Priority: P2)

The engineer reads the 50th, 75th, 95th and 99th percentile for all, successful and failed
requests of the run, and the output states that these are this tool's estimates and not
Gatling's. The engineer can ask for other ranks.

**Why this priority**: Percentiles are what thresholds are written against, so a percentile
printed without saying whose it is invites a comparison that cannot hold. The non-percentile
figures of the first story are usable without them.

**Independent Test**: Summarise the recorded runs twice each and confirm the percentiles are
identical between readings, labelled, and — for the 3.13.1 run — the documented values.

**Acceptance Scenarios**:

1. **Given** any recorded run, **When** it is summarised, **Then** the four default
   percentiles are given in whole milliseconds for all, ok and failed requests, and the output
   says they are estimates computed by this tool and are not Gatling's.
2. **Given** the recorded 3.13.1 run, **When** it is summarised, **Then** the 95th percentile
   of all requests is printed as 1427 ms. The README names this run as the known case where
   the default quantile differs from the request standing at the percentile's rank (1502 ms)
   and names the upstream pull request that removes the difference.
3. **Given** the engineer asks for the ranks 90 and 99.9, **When** the run is summarised,
   **Then** those percentiles are printed instead of the defaults.
4. **Given** a rank that is not a number, is not above 0, or is above 100, **When** the
   command is invoked, **Then** the exit code is 2, nothing is written to standard output,
   and the error quotes the value refused.
5. **Given** an outcome that holds no request — the failed requests of a run in which
   nothing failed — **When** it is summarised, **Then** its percentiles are shown as
   absent, never as 0.

---

### User Story 3 - Response-time bands at the engineer's own boundaries (Priority: P2)

The engineer whose service has its own notion of fast and slow sets the two boundaries of
the response-time bands instead of Gatling's 800 ms and 1200 ms.

**Why this priority**: The default bands answer Gatling's question, not the service's. It is
a small addition once the bands exist, and the issue requires it.

**Independent Test**: Summarise one recorded run at the default boundaries and at a custom
pair, and check the band counts against a count of the log's own response times.

**Acceptance Scenarios**:

1. **Given** the boundaries 5 ms and 1000 ms, **When** the recorded 3.13.1 run is summarised,
   **Then** the three timing bands count its successful responses against those boundaries,
   failed requests stay in the failed band, and the four percentages add up to 100 %.
2. **Given** boundaries that are not two increasing, non-negative numbers, **When** the command
   is invoked, **Then** the exit code is 2 and the error quotes the value refused.

---

### User Story 4 - Summarise a run of any size (Priority: P2)

The engineer summarises a soak test of many millions of requests. The command walks the log
once, its memory does not grow with the number of requests, and it finishes.

**Why this priority**: The runs people most need to summarise are the long ones. Milestone
`v0.13.0` already reads them within a fixed bound; the summary must not lose that.

**Independent Test**: Summarise a run of one million requests and one of ten million and
observe the same peak-memory bound.

**Acceptance Scenarios**:

1. **Given** a run of one million requests, **When** it is summarised, **Then** peak memory
   stays under the bound the plan states, and a run ten times larger stays under the same
   bound, however many distinct request names either holds.

---

### User Story 5 - Fail with a message the engineer can act on (Priority: P2)

When a run cannot be summarised completely, the engineer learns which figures were withheld
and why, and a script can tell a partial summary from a complete one by the exit code.

**Why this priority**: The command runs unattended. A summary that silently leaves a figure
out, or fills it with a zero, is worse than none.

**Independent Test**: Exercise each case below and assert the exit code, what is printed and
what the message names.

**Acceptance Scenarios**:

1. **Given** a log that was cut short because the run was killed, **When** it is summarised,
   **Then** the summary of what it did hold is printed, the error says the log ended early
   and where, and the exit code is 1 — as the counts of milestone `v0.13.0` already behave.
2. **Given** a log whose bytes cannot be decoded part-way through, **When** it is read,
   **Then** no summary is printed, because the records before the damage are not a result,
   and the exit code is 1.
3. **Given** a run whose span is zero, **When** it is summarised, **Then** every rate is shown
   as absent, the error names the span, and the exit code is 1; a rate is never printed as
   infinite and the span is never replaced by one second.
4. **Given** a run the library cannot bound in time, **When** it is summarised, **Then** every
   rate is shown as absent and every other figure is printed.
5. **Given** a request whose end the source did not record, **When** the run is summarised,
   **Then** the request is counted, takes part in no timing figure, and the number of such
   requests is reported on standard error.
6. **Given** a request whose outcome the source lost, **When** the run is summarised, **Then**
   it is counted apart, as neither ok nor failed, its count is reported, and the exit code
   is 1, because a total that exceeds the two outcomes it is made of is not a summary.
7. **Given** `-o` with any value, **When** the command is invoked, **Then** the exit code is 2
   exactly as in milestone `v0.13.0`: the summary adds no machine-readable form.

---

### User Story 6 - See that a long read is progressing (Priority: P3)

An engineer summarising a log of gigabytes at a terminal sees, while the command reads, a
small block that is redrawn in place: a progress bar with how much of the log has been read
and roughly how long is left, and beneath it the summary so far — count, share, fastest,
mean, median, 95th and 99th percentile and slowest, for all, ok and failed requests — with
the numbers moving as the read goes on. When the read ends the block disappears and the
report is printed as always. A pipeline sees none of it.

**Why this priority**: The summary is complete without it. But a read that takes a minute
and prints nothing looks like a hang, and the figures are already there to be shown: the
pass accumulates them as it goes.

**Independent Test**: Read a replayed log of a gigabyte with standard error attached to a
terminal and again with it redirected; compare standard output of the two and look at what
standard error received.

**Acceptance Scenarios**:

1. **Given** standard error is a terminal that can redraw in place and a log that takes
   several seconds to read, **When** it is summarised, **Then** a block of fixed height on
   standard error is redrawn in place at least once a second with a progress bar, the
   percentage of the log read, an estimate of the time left, and the figures so far for
   all, ok and failed requests; it never scrolls, never exceeds 100 %, and is gone before
   the report appears.
2. **Given** standard error is not a terminal — a pipeline, a pipe, a file — **When** the same
   log is summarised, **Then** standard error receives nothing but the warnings and errors it
   received before this story existed.
3. **Given** `--quiet` on a terminal, **When** the log is summarised, **Then** no progress
   block is drawn.
4. **Given** the same run summarised with the progress block and without it, **When** the two
   standard outputs are compared, **Then** they are identical byte for byte, and so is the
   exit code.
5. **Given** a log that is read faster than the progress block is first due, **When** it is
   summarised on a terminal, **Then** nothing is drawn at all.
6. **Given** a log that is cut short or damaged, **When** the read stops, **Then** the block
   is erased before the error is written, so the error stands on a clean screen.
7. **Given** a terminal that does not say it can redraw in place — `TERM` is unset or is
   `dumb` — **When** a log is summarised, **Then** no block is drawn and no control sequence
   is written.
8. **Given** the read is interrupted from the keyboard, **When** the command stops, **Then**
   the block is erased before the command exits and the terminal is left as it was found.

### Edge Cases

- A run has zero requests: the summary reports zero requests, a rate of 0 where the run can
  be timed, every time and percentile as absent, and the exit code is 0.
- The run has one request: minimum, maximum, mean and every percentile are that request's
  response time, and the standard deviation is 0.
- A mean that falls exactly between two whole milliseconds is rounded up, as Gatling does:
  0.5 ms is printed as 1.
- All response times of the run are equal: the standard deviation is 0 and every percentile
  is that value.
- The author of the run overrode Gatling's percentile ranks or band boundaries in
  `gatling.conf`: the log does not record that. The command reports at its own defaults or at
  the values it was given, and never claims to match such a run's HTML report.
- A percentile rank is given twice, or the ranks are given out of order: each is reported
  once, in increasing order.
- The log grows while it is being read, or its size cannot be known: the percentage is taken
  against the size seen when the log was opened and never exceeds 100, or is left out of the
  progress block altogether, the bar and the time left with it; reading a log that is still
  being written remains unsupported.
- The terminal is narrower than 80 columns: every line of the progress block fits 80 columns
  by construction; a narrower terminal wraps it and may keep remains of it on screen. That
  is accepted: a terminal's width cannot be asked for without a new dependency.
- Nothing failed: the failed outcome shows a count of 0, a share of 0 %, a rate of 0 where
  the run can be timed and every time as absent, and `failed` is not shown in red.
- The run holds many thousands of distinct request names: the console output is no longer
  than for a run with one name, and memory does not grow with the number of names, because
  no figure is kept per name in this milestone.

## Requirements *(mandatory)*

### Functional Requirements

**What is reported**

- **FR-001**: `galaxio report gatling [PATH]` MUST print, after the run description of
  milestone `v0.13.0` and without removing any line of it, the summary of the run as a
  whole, and MUST NOT print a row for a single request or group. The summary MUST NOT
  require the HTML report, or any file other than the log, to be present. The one change to
  the run description is the wording of FR-005.
- **FR-002**: The command MUST NOT create, change or remove any file, and MUST NOT add an
  option that names an output file. The only files this command is to write are the report
  products `-o` names, and those stay reserved in this milestone (FR-031).
- **FR-003**: The summary MUST report, for all requests, for successful ones and for failed
  ones: the count, the minimum, maximum and mean response time, the standard deviation, the
  percentiles, and the mean number of requests per second.
- **FR-004**: The whole-run summary MUST report the response-time bands: how many successful
  responses fell under the lower boundary, between the two boundaries, and at or above the
  upper one, how many requests failed, and each as a percentage of all requests.

**Wording and layout**

- **FR-005**: The two outcomes MUST be called `ok` and `failed` on every line this command
  writes. Gatling's word `ko` MUST NOT appear; the requests line of the run description,
  published with `v0.13.0` as `<N> (<N> ok, <N> ko)`, MUST read `<N> (<N> ok, <N> failed)`.
- **FR-006**: The summary MUST be laid out in this tool's own, tool-independent form, so that
  a run of any tool the command comes to read is summarised in the same layout: a headline
  with the requests and their rate, then ok and failed, each with its count, its share of
  all requests and its rate; a response-time table with one row for all, ok and failed
  requests and, in columns, the minimum, the mean, the standard deviation, the percentiles
  in increasing order and the maximum; the response-time bands, each as a bar with its
  share and its count; and one closing line giving the unit of every time and the
  percentile statement of FR-019. It MUST NOT reproduce Gatling's console table.
- **FR-007**: On a terminal that says it understands colour the summary MUST show `ok` in
  green and `failed` in red when anything failed, and MAY dim headings and the unfilled part
  of a bar. It MUST write no escape sequence when standard output is anything else — a pipe,
  a file, a terminal whose `TERM` is unset or `dumb` — when `--no-color` is passed or when
  `NO_COLOR` is set, and colour MUST NOT carry anything the text does not say.

**Arithmetic**

- **FR-008**: Counts MUST be exact. `ok` counts the samples the source recorded as
  successful, `failed` those it recorded as failed, and the count of all requests is their
  sum.
- **FR-009**: Minimum and maximum MUST be the exact extremes in whole milliseconds.
- **FR-010**: The mean MUST be the exact arithmetic mean rounded half up to a whole
  millisecond.
- **FR-011**: The standard deviation MUST be the population deviation, taken about the
  unrounded mean and rounded half up to a whole millisecond, and MUST be computed without
  loss of precision for any run the command can read.
- **FR-012**: The mean number of requests per second MUST be the count divided by the run's
  span in whole seconds, rounded up, where the span is the library's own definition of where
  the run begins and ends. One span MUST serve every rate the summary prints.
- **FR-013**: The three timing bands MUST count successful responses only; a failed request
  belongs to the failed band whatever its response time. The boundaries are lower-inclusive:
  under the first, from the first to under the second, at or above the second. A percentage
  MUST be the band's count over all requests of the run.
- **FR-014**: Successful and failed samples MUST be accumulated apart. A failure MUST NOT
  contribute to any figure of ok requests; it MUST contribute to every timing figure of all
  requests.
- **FR-015**: The summary MUST aggregate requests only. A group traversal MUST NOT be counted
  as a request and MUST NOT take part in any figure.
- **FR-016**: No figure for a single request or a single group is part of this milestone —
  neither from the cumulated response time of a traversal nor from its wall-clock duration.
  Those figures arrive with the `-o` products that carry them and are a separate, later
  task.

**Percentiles**

- **FR-017**: The summary MUST report the 50th, 75th, 95th and 99th percentile by default,
  in whole milliseconds rounded half up, for all, successful and failed requests. The
  percentiles of all requests MUST be computed over the successful and failed samples
  together.
- **FR-018**: Percentiles MUST be computed by a bounded-memory digest and read with the
  default quantile of the digest library named in Assumptions. The same log MUST always
  yield the same percentiles.
- **FR-019**: Every output carrying a percentile MUST say that it is this tool's estimate and
  how it is defined. A percentile MUST NOT be presented as reproducing Gatling's, MUST NOT be
  asserted against a percentile Gatling recorded in any test, and MUST NOT be offered as a
  difference from one.
- **FR-020**: The README MUST state the known limitation in the same change: the default
  quantile interpolates, so where response times have a gap a printed percentile can be a
  value no request had. It MUST give the recorded 3.13.1 run as the example — 1427 ms printed
  where the request at the 95th percentile's rank took 1502 ms — and MUST name
  [caio/go-tdigest#42](https://github.com/caio/go-tdigest/pull/42) as the pull request that
  removes it.
- **FR-021**: The tests MUST pin the percentiles of the recorded runs as this tool computes
  them, the 3.13.1 example included, so that taking the upstream read later is a visible,
  deliberate change of those expectations rather than a silent one.
- **FR-022**: Taking the read by rank once a release of the library carries it is NOT part of
  this milestone. When it is done it MUST be its own change, MUST update the pinned
  expectations and the README together, and MUST be called out in the changelog, because it
  alters numbers users have seen.
- **FR-023**: The percentile ranks MUST be configurable on the command line, defaulting to
  Gatling's. A rank MUST be a number above 0 and not above 100; anything else MUST be a usage
  error quoting the value. Ranks MUST be reported once each, in increasing order.
- **FR-024**: The two band boundaries MUST be configurable on the command line, defaulting to
  Gatling's 800 ms and 1200 ms. They MUST be two non-negative, strictly increasing numbers of
  milliseconds; anything else MUST be a usage error quoting the value.

**Absence and failure**

- **FR-025**: An outcome that holds no sample MUST show every timing figure and percentile as
  absent. Absence MUST NOT be rendered as a zero, which is a real response time.
- **FR-026**: A sample with no recorded end MUST be counted, MUST take part in no timing
  figure, and the number of such samples MUST be reported on standard error.
- **FR-027**: A sample whose outcome the source lost MUST be counted apart, as neither ok nor
  failed and in no figure; its count MUST be reported and the command MUST exit 1.
- **FR-028**: When the run's span is zero the command MUST show every rate as absent, name
  the span in the error and exit 1. It MUST NOT print an infinite rate and MUST NOT
  substitute a span. When the library cannot bound the run, every rate MUST be shown as
  absent and the span MUST NOT be shortened to what could be placed in time.
- **FR-029**: A log cut short MUST be summarised as far as it was read, reported as cut short
  and exit 1; a log that cannot be decoded MUST produce no summary and exit 1. Both are the
  behaviour of milestone `v0.13.0`, extended to the summary.
- **FR-030**: The command MUST NOT claim to match a run whose author changed Gatling's
  percentile ranks or band boundaries: the log does not record them.

**Surface and process**

- **FR-031**: `-o` MUST keep the behaviour of milestone `v0.13.0`: its known values stay
  reserved and any value is a usage error. The summary MUST NOT add `-o json`, `-o text` or
  any other machine-readable form; that form is the `stats.json` of
  [#52](https://github.com/galax-io/galaxio-cli/issues/52).
- **FR-032**: `--quiet` MUST suppress the summary with the rest of the report; errors and
  warnings MUST still be printed. This is the mode a CI job uses.
- **FR-033**: The log MUST be read once, in a single forward pass, with no sample retained.
  Memory MUST NOT grow with the number of samples or with the number of distinct request
  names, and the plan MUST state the peak-memory goal.
- **FR-034**: Summarising the same run twice MUST produce identical output.
- **FR-035**: The arithmetic MUST live in this repository's report package and MUST use the
  library's own definitions of a run's bounds and an outcome rather than re-derive them.
- **FR-036**: The summary, its options, the definition of every figure and the percentile
  limitation MUST be documented in the README in the same change.

**Progress while reading**

- **FR-037**: While the log is read the command MUST show a progress block on standard
  error when, and only when, standard error is a terminal that says it can redraw in place
  and `--quiet` is not set. When standard error is anything else — a pipe, a file, a
  terminal whose `TERM` is unset or `dumb` — nothing of it MUST be written, not even a
  control sequence.
- **FR-038**: The block MUST carry a progress bar with the share of the log read so far, as
  a percentage of its bytes that never exceeds 100, and, once it can be estimated, the time
  left; a line saying that the figures are so far, that times are in milliseconds and that
  the percentiles are estimates and how they are computed (FR-019); and beneath it the
  figures so far for all, ok and failed requests: the count, the share, the minimum, the
  mean, the median, the 95th and 99th percentile and the maximum. It MUST NOT carry a
  request rate, which needs the span of the whole run.
- **FR-039**: The block MUST have a fixed height and every line of it MUST fit 80 columns. It
  MUST be redrawn in place without scrolling, no more often than five times a second and at
  least once a second while records keep arriving, and MUST be erased before anything else
  is written to standard error, before the report is printed and before the command exits,
  whatever the ending — an interrupt included. It MUST NOT change any mode of the terminal,
  so that a command that is killed leaves nothing behind but the block itself.
- **FR-040**: A read that ends before the first update is due MUST draw nothing.
- **FR-041**: Standard output and the exit code MUST be identical byte for byte whether or
  not the progress block is shown.
- **FR-042**: Showing the progress block MUST NOT add a second pass, retain a sample, or add
  a dependency; recognising a terminal MUST use the standard library.

### Key Entities

- **Summary**: The figures of the run as a whole, each given for all, successful and failed
  requests.
- **Run span**: Where the run begins and ends by the library's definition, and the number of
  whole seconds, rounded up, that every rate divides by.
- **Response-time bands**: Three timing bands over successful responses, split by two
  boundaries, plus the failed band; counts and percentages of all requests.
- **Read progress**: How many bytes of the log have been read against its size when it was
  opened, and how long that took; what the progress block's bar, percentage and time left
  are made of. Never part of the report.
- **Percentile estimate**: A rank, the value printed for it, and the statement of how it was
  computed and that it is not Gatling's.
- **Known divergence**: The documented case where the default quantile prints a value between
  two recorded response times, with the upstream pull request that removes it.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For every recorded run in the corpus, every non-percentile figure of the whole
  run — counts, minimum, maximum, mean, standard deviation, mean requests per second, band
  counts and percentages — equals what Gatling itself recorded for that run: a 100 % match
  with zero exceptions.
- **SC-002**: An engineer or a CI job obtains all of these figures from the log alone, with
  no browser, no HTML report and no Gatling installation.
- **SC-003**: Summarising the same run one hundred times gives one hundred identical outputs,
  percentiles included.
- **SC-004**: A run of one million requests is summarised within the fixed memory bound the
  plan states, and a run ten times larger stays within the same bound.
- **SC-005**: Every percentile the command prints is accompanied by the statement of whose
  estimate it is, and no test in the repository compares a percentile with one Gatling
  recorded: both are verifiable by reading the output and the test suite.
- **SC-006**: The documented divergence is reproducible: the shipped command prints 1427 ms
  as the 95th percentile of the recorded 3.13.1 run, the README explains it and links the
  upstream pull request, and taking the upstream read later changes that expectation in one
  reviewed change.
- **SC-007**: Every failure scenario in this specification exits with its documented code and
  names the figure withheld, the span, the value refused or the place the log ended.
- **SC-008**: The console output has the same number of lines for a run with seven request
  names and for one with seven hundred; summarising a run leaves every file as it was and
  creates none; and a CI job run with `--quiet` gets an empty standard output and the exit
  code.
- **SC-009**: On a terminal, a read of a gigabyte shows the progress block within one second
  and redraws it at least once a second until it ends; with standard error redirected, the
  same read writes nothing to it.
- **SC-010**: Standard output of a run is byte-identical with and without the progress block,
  for every run of the corpus and for the replayed gigabyte, and the read with the block
  takes at most 5 % longer than without it.
- **SC-011**: Nothing in the output is Gatling's own: for each of the five recorded runs, no
  line names an outcome `ko` or `KO`, and the summary has no element that exists only for
  Gatling — verifiable by reading the output.

## Assumptions

- **Maintainer decision, 2026-09-17.** Percentiles come from `github.com/caio/go-tdigest`
  (v5, MIT) read with its default quantile. The read by rank proposed upstream in
  [caio/go-tdigest#42](https://github.com/caio/go-tdigest/pull/42) corrects the known
  divergence but is unreleased; this milestone only records that it exists (FR-020, FR-022)
  and does not carry its own copy of that read.
- **This decision did not meet one clause of the constitution, so that clause was amended
  first (clarified 2026-09-17).** Principle II (v2.1.0) required percentiles to be exact over
  the samples recorded and required the command to refuse where a bounded-memory accumulator
  could not keep them exact. A digest's quantile is an estimate — on the 3.13.1 run it is not
  the recorded value, and on a large continuous run it stays an estimate even with the
  upstream read by rank — so the conflict was permanent, not a gap to be waited out. The
  clause was therefore amended before any implementation, by the procedure
  [#112](https://github.com/galax-io/galaxio-cli/issues/112) followed: its own issue,
  [#114](https://github.com/galax-io/galaxio-cli/issues/114), and its own pull request,
  [#115](https://github.com/galax-io/galaxio-cli/pull/115), merged on 2026-09-17 inside
  milestone `v0.14.0`, as the maintainer requested explicitly in this clarification and as
  the one-milestone-one-PR rule requires for a split. Constitution v2.2.0 admits a percentile
  estimated by a bounded-memory sketch if it is deterministic, says wherever it is printed
  that it is an estimate and how it is computed, has its estimator recorded in the feature's
  research and its values for the corpus pinned in tests, and has a known divergence
  documented with an example; counts, minimum, maximum, mean and standard deviation stay
  exact. The plan's Constitution Check is written against that text. The other clauses of
  Principle II are met as they were: percentiles are never presented as Gatling's, every
  output says whose definition it is, accumulation is one pass in bounded memory, and
  absence is reported as absent.
- **Principle IV.** The digest library is a new direct dependency, named by the maintainer in
  the decision above. It is MIT, which is compatible with this repository's GPL-2.0-only
  licence, and its non-test code imports the standard library only. The plan records the
  case in `research.md`, including why the two alternatives measured were not taken:
  `influxdata/tdigest` is Apache-2.0, which Principle IV excludes, and `spenczar/tdigest` is
  archived and does not reproduce its own output.
- **Console only, and no file (clarified 2026-09-17).** The console carries the whole-run
  summary and this milestone writes no file. Issue #51's requirement that the text output
  carry the figures per request is superseded by that clarification: figures per request
  and per group arrive with the first `-o` product that carries them — `stats`, the
  `stats.json` of [#52](https://github.com/galax-io/galaxio-cli/issues/52) — and the
  products `-o` names are the only files this command is to write, each always in full.
  `--quiet`, which already exists as a root flag, is what a CI job uses.
- **Progress while reading (maintainer decision, 2026-09-17).** Included in this milestone
  as a P3 story, and as a live block: a single status line was specified first and the
  maintainer rejected it on sight. The block is drawn with the two things every current
  terminal understands — move up, erase — and changes no mode of the terminal, so a killed
  command leaves nothing behind but the block. A terminal that does not say it understands
  them — `TERM` unset or `dumb`, which includes the classic Windows console — gets no
  progress at all rather than a degraded one. No flag is added: a terminal gets the block,
  everything else gets nothing, and `--quiet` silences it; a `--progress` option that
  prints periodic lines into a CI log is out of scope. A terminal is recognised with the
  standard library alone, and its width is never asked for: the block fits 80 columns by
  construction.
- **Issue #51 needs amending to match.** Its text still asks for `-o json`, which
  constitution v2.1.0 Principle I no longer permits on this command (the machine-readable
  form is the `stats.json` of #52), and its amendment of 2026-09-16 describes exact
  percentiles from a millisecond histogram with a capped pool and a refusal on overflow,
  which this decision supersedes. Its requirement that the text output carry the figures per
  request, and its first amendment's requirement that the wall clock of a group traversal be
  selectable, are deferred to the `-o` products (clarified 2026-09-17). Its definitions of
  every non-percentile figure stand and are the requirements above.
- The figures Gatling recorded for the corpus are the acceptance reference for non-percentile
  figures: a console summary for 3.13.1, 3.14.9 and 3.15.1, a `global_stats.json` for 3.11.5,
  3.12.0 and 3.13.1. They are recorded in the shared library's corpus; this repository's
  copy holds the logs only, so the plan brings the recorded figures in as test data with
  their provenance.
- Gatling's percentiles are not a reference for anything. For the 3.13.1 run Gatling printed
  1072 ms, which comes from the defect reported as
  [tdunning/t-digest#230](https://github.com/tdunning/t-digest/issues/230) and changes between
  two readings of the same log.
- Every response time in every recorded log is a whole, non-negative number of milliseconds,
  and figures are printed in whole milliseconds as Gatling prints them.
- The defaults are Gatling's own: ranks 50, 75, 95 and 99, and boundaries 800 ms and 1200 ms,
  unchanged from 3.11.5 to 3.15.1. The names of the options that change them are chosen in
  the plan and are new published surface under Principle V.
- **Two exit codes change for runs no Gatling log is known to produce, and Principle V asks
  that this be said.** Milestone `v0.13.0` exits 0 for a run whose span is zero and for a run
  holding a request whose outcome the source lost; with the summary both exit 1 (FR-027,
  FR-028), as issue #51 decides, because a rate cannot be computed for the first and the
  outcomes of the second do not add up to its total. The shared library states that no
  adapter produces a lost outcome, and a zero span needs a log with a single instant in it,
  so no existing invocation on a real run changes its exit code.
- The per-message error table of Gatling's console summary is out of scope (clarified
  2026-09-17): the run description already reports how many errors the log holds, issue
  #51 settles no figure for a breakdown by message, and it is
  the one place where memory would grow with the data, because the number of distinct
  messages is not bounded. It is a separate, later task.
- **The layout is this tool's own and universal (maintainer decision, 2026-09-17).** It
  carries Gatling's figures, not Gatling's words or Gatling's table, because the same
  command is to summarise JMeter, k6, Locust and Yandex.Tank runs (milestones `v0.19.0` to
  `v0.22.0`) in the same layout. The maintainer chose it from three mock-ups. It uses UTF-8
  symbols — a check mark, a cross, block characters — and, on a terminal, colour. The text
  is for people; the forms a program reads are the `-o` products.
- **One line of published output changes, approved in clarification.** The requests line of
  the run description said `18 ko` in `v0.13.0` and says `18 failed` from this milestone
  (FR-005). Principle V asks that it be listed: the plan lists it, and the README and the
  tests that quote the line change with it.
- Out of scope: any file output and any machine-readable output, including `stats.json` and
  `global_stats.json` (#52); figures per request and per group, which arrive with those
  products; charts; comparing two runs; a rendered HTML page; checking figures against
  requirements (milestone `v0.16.0`); tools other than Gatling; taking the upstream read by
  rank; the per-message error table; periodic progress lines for CI logs; asking the
  terminal for its width.
- Dependencies: the amendment of Principle II's percentile clause (#114, merged as #115 on
  2026-09-17) precedes implementation; milestone `v0.13.0 Read a run` (#50) delivers the
  single pass this feature folds over; `parsec` v0.1.0 supplies the bounds and outcome
  definitions (parsec#8); constitution v2.1.0 (#112) governs the `-o` surface and v2.2.0
  (#115) the percentile wording.
