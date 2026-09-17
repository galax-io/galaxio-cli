# Feature Specification: Read a Finished Gatling Run

**Feature Branch**: `110-report-records`

**Created**: 2026-09-15

**Status**: Draft (scope reduced 2026-09-16)

**Input**: User description: "https://github.com/galax-io/galaxio-cli/issues/50"

**Tracking**: [milestone `v0.13.0 Read a run`](https://github.com/galax-io/galaxio-cli/milestone/3);
[galax-io/galaxio-cli#50](https://github.com/galax-io/galaxio-cli/issues/50)

## Background

A finished Gatling run is a directory holding an HTML report and a `simulation.log`. The log
is written in one of two formats: a text format up to Gatling 3.12 and a binary format from
3.13 on. Only the matching Gatling version can read either, and from 3.13.5 on Gatling
exports no machine-readable file at all. Nothing in this CLI can open such a run.

This milestone makes `galaxio report`, reserved by milestone `v0.12.0` as a help-only
namespace, an operational command that **reads** a run. `galaxio report gatling [PATH]`
names the tool, locates the run, opens its log whatever Gatling wrote it, walks it once,
and reports what it read: the tool and version, the log format, the run's identity and
start, the span it covers, and how many requests, group traversals, virtual-user events and
run-level errors it holds, with successes and failures counted apart.

Reading is the whole of this milestone. It computes no statistic and emits no data format.
The counts it prints are tallies of what the log holds, which is the proof that the log was
read; a mean, a percentile, a range or a series is the next milestone's work.

The run is read through the shared result-primitives library `parsec`, whose licence
compatibility was settled in milestone `v0.12.0`. This feature builds on the library's own
primitives — the position of a sample, the bounds of a run, the outcome the source
recorded, what the source can never record — and declares no vocabulary of its own beside
them.

### Withdrawn from this milestone

**Maintainer decision, 2026-09-16.** The line-delimited JSON record stream that issue #50
proposed, and that an earlier revision of this specification described, is **withdrawn**. A
record-per-line dump duplicates in a second vocabulary what the library already yields, and
it is not the output this ecosystem wants. Nothing in this feature writes records.

The output a finished run is wanted in is Gatling's own `stats.json` and
`global_stats.json`: **the same schema and the same numbers**, verified against a run whose
files Gatling still produced, or against the figures in its HTML report where it no longer
does. Bit-for-bit parity is not required; what is required is that every field where a
difference is possible says so explicitly and names the size of it. That work is the
statistics of [#51](https://github.com/galax-io/galaxio-cli/issues/51) and the writer of
[#52](https://github.com/galax-io/galaxio-cli/issues/52), and belongs to the milestones that
own them. This milestone delivers the reading those two stand on, and stops there.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Read a run of any supported Gatling version (Priority: P1)

An engineer names the tool and points the command at a run, either the run directory or the
`simulation.log` inside it, and learns whether this tool can read it and what it holds. It
works the same whether Gatling wrote the log as text (3.11.5 through 3.12.0) or binary
(3.13.1 through 3.15.1), and the engineer never has to know which version produced it.

**Why this priority**: Without it there is no way to open a Gatling run at all, and every
later milestone reads through this path.

**Independent Test**: Run the command against one recorded 3.12.0 run and one recorded
3.15.1 run from the corpus; each exits 0, names the version and format it detected, and
reports a request count equal to the total Gatling itself reported for that run — its
console summary where one was captured, its `global_stats.json` otherwise.

**Acceptance Scenarios**:

1. **Given** a Gatling 3.12.0 run directory, **When** the engineer runs `galaxio report gatling`
   with that directory as the path, **Then** the exit code is 0 and standard output names
   the tool and version, the text log format, the run's identity and start, and the counts
   of requests, groups, user events and errors, with the request count matching the total
   Gatling itself reported for the run.
2. **Given** a Gatling 3.15.1 run directory, **When** the command runs with that directory,
   **Then** the same holds for the binary format and that run's counts.
3. **Given** the path of a `simulation.log` file rather than its directory, **When** the
   engineer passes that path, **Then** the output is identical to passing the directory.
4. **Given** either run, **When** the engineer passes the directory without naming the tool
   first, **Then** the exit code is 2 and the error quotes the value it read as a tool and
   lists `gatling`.
5. **Given** a run whose log was written by a Gatling version newer than any the tooling has
   verified, **When** it is read, **Then** the run is read, a warning naming the version and
   why it is unverified is printed, and the exit code is 0.

---

### User Story 2 - Read the latest run without naming it (Priority: P1)

An engineer who has just finished a run inside a project runs `galaxio report gatling` with
no path. The most recent run under the build tool's usual results directory is found and
read, and the command names the run directory chosen and the rule that chose it.

**Why this priority**: The common case is "the run I just did". Requiring the timestamped
directory name Gatling generates would make every script compute it first.

**Independent Test**: In a working directory holding a results root with several run
directories, run `galaxio report gatling` with no path and confirm the expected run is the
one read and named.

**Acceptance Scenarios**:

1. **Given** a results root containing several run directories and no marker of the last
   run, **When** the command runs with no path from that project, **Then** the most recently
   modified run is read and the output names its directory and states it was chosen as the
   newest.
2. **Given** a results root where the build tool left a marker naming the last run and that
   run still exists, **When** the command runs with no path, **Then** the run the marker
   names is read and the output says it was chosen by the marker.
3. **Given** a directory that holds no run at all, **When** the command runs against it,
   **Then** the exit code is 1 and the error names the directory that was searched.
4. **Given** a path that is a results root rather than a run, **When** it is passed as the
   argument, **Then** it is searched exactly as the default root would be.

---

### User Story 3 - Read a multi-gigabyte log (Priority: P2)

An engineer reads a run whose log is larger than the memory available. The command walks it
once, its memory does not grow with the size of the log, and it finishes.

**Why this priority**: Real soak-test logs run to gigabytes. A reader that had to hold the
run in memory would be unusable on exactly the runs people most need to inspect, and the
statistics milestone inherits this path.

**Independent Test**: Read a log of at least two gigabytes and observe that peak memory
stays under the bound the plan states and does not differ from the bound observed on a log
a hundred times smaller.

**Acceptance Scenarios**:

1. **Given** a log of several gigabytes, **When** it is read, **Then** peak memory does not
   exceed a fixed bound independent of the log's size, and the bound is the same one a log
   thirty times smaller reaches.
2. **Given** the same run read twice, **When** the outputs are compared, **Then** they are
   identical.

---

### User Story 4 - Fail with a message the engineer can act on (Priority: P2)

When a run cannot be read, the engineer learns what was refused, read or searched, and why,
and a script can tell a usage mistake from a runtime failure by the exit code alone.

**Why this priority**: The command will run unattended in CI. A bare non-zero exit or a
generic "cannot read log" costs a person a debugging session.

**Independent Test**: Exercise each failure below and assert the exit code and that the
message names the thing at fault.

**Acceptance Scenarios**:

1. **Given** a log written by an unsupported Gatling version, including 3.13.0, **When** it is
   read, **Then** the exit code is 1 and the error names the version found and the range
   supported.
2. **Given** a log that was cut short because the run was killed, **When** it is read,
   **Then** the counts of what it did contain are reported, the error says the log ended
   early and where, and the exit code is 1 so that a script does not mistake a partial run
   for a complete one.
3. **Given** a path that does not exist or cannot be read, **When** it is passed, **Then** the
   exit code is 1 and the error names that path and the reason, never claiming the directory
   held no run.
4. **Given** a tool name other than `gatling`, a third positional argument, or an unknown
   flag, **When** the command is invoked, **Then** the exit code is 2, nothing is written to
   standard output, and for the tool the error lists the accepted tool.
5. **Given** a log whose bytes cannot be decoded part-way through, **When** it is read,
   **Then** the exit code is 1, the error says the run was not read completely, and no
   counts are reported, because the records before the damage are not a result.
6. **Given** `-o` with any value, **When** the command is invoked, **Then** the exit code is 2
   and the error names the milestone that delivers that report format, or lists the known
   names for an unknown one.

### Edge Cases

- A directory is itself named `simulation.log`: it is still examined for a log inside it,
  because what a run directory is called is not the command's to judge.
- A directory holds a `simulation.log` and also contains run directories beneath it: it is
  treated as a run, never as a results root, so the engineer is answered about the place
  they named.
- The marker of the last run names a directory that no longer exists: the marker is
  ignored, the newest run is chosen instead, and the rule reported is newest.
- A run has zero requests: the counts are reported honestly as zero and the exit code is 0.
- A run's log contains no value for something the source can never record: that absence is
  the source's own statement, reported as absent and never as a zero.
- The `--quiet` root flag is set: the report is suppressed and only errors are printed.
- `galaxio report` is invoked with no arguments at all: help is printed and the exit code is
  0, exactly as in milestone v0.12.0; nothing is searched.
- The results root exists but is unreadable because of permissions: this is reported as a
  read failure on that path, not as "no run found".

## Requirements *(mandatory)*

### Functional Requirements

**Input and run selection**

- **FR-000**: An argument that is given but empty MUST be refused as a usage error rather
  than treated as an omitted one. The empty string is the zero value of every unset
  configuration field, and guessing a results root for it would answer confidently about a
  run nobody asked for.
- **FR-001**: The command MUST take the tool name as its first positional argument;
  `gatling` is the only accepted value in this milestone, and any other name MUST be a usage
  error listing the accepted tools. It MUST accept at most one further argument, which may
  be a run directory, a `simulation.log` file, or a results root containing run
  directories.
- **FR-002**: With no path the command MUST search the conventional results root of the
  Maven and sbt build tools, resolved against the working directory.
- **FR-003**: When the argument or default names a results root, the command MUST choose the
  run a build-tool marker names if that marker exists and the run is still present, and
  otherwise the most recently modified run directory in the root.
- **FR-004**: The command MUST report the log it read and the rule that chose the run,
  named as the library names it: `path`, `lastRun.txt`, or `newest`.
- **FR-005**: When no run is found, the command MUST exit 1 and name the directory searched.
  A path the caller named that does not exist MUST be reported as a read failure naming that
  path, never as an absence of runs, and that test MUST follow symlinks so that a dangling
  one is not mistaken for an empty root. The default results root is the exception: the
  caller never typed it, so an absent one is an absence of runs. A directory that cannot be
  read MUST keep the library's own message, which names the directory it was reading.

**Reading and what is reported**

- **FR-006**: The command MUST read the run's log once, in a single forward pass, through
  the shared library, and MUST NOT retain the records it walks.
- **FR-007**: The command MUST read both the text and the binary log format and MUST
  identify the format from the log's content, never from its name or extension.
- **FR-008**: The command MUST read every Gatling version the shared library supports, MUST
  take that range from the library rather than a hard-coded list, and MUST refuse a version
  outside it with an exit code of 1 and a message naming the version found and the range
  supported.
- **FR-009**: A version newer than the verified range MUST be read, and the resulting warning
  MUST be reported. The warning is not informational and MUST be printed even under
  `--quiet`.
- **FR-010**: The command MUST report the tool and the version the run stated, and the log
  format it detected.
- **FR-011**: The command MUST report the run's identity, its simulation name and its
  recorded start.
- **FR-012**: The command MUST report the span the run covers, taken from the library's own
  definition of where a run begins and ends rather than derived here, together with the
  elapsed time between its two instants. When the library cannot bound the run, the span
  MUST be left out of the report rather than shown empty or as a zero.
- **FR-013**: The command MUST report the number of requests, group traversals,
  virtual-user events and run-level errors the log held, naming what each count counts so
  that a traversal is not read as a group or an event as a virtual user. A declared-assertion
  payload the source yielded among its events, and an item of a kind this release does not
  know, MUST each be counted and reported rather than walked past in silence, and MUST report the request count
  split into successes and failures, taking the outcome as the source recorded it and never
  inferring it. A request whose outcome the source lost MUST be counted in the total and in
  neither split, and MUST be reported on its own line, so that successes, failures and lost
  outcomes sum to the total. These counts MUST be exact.
- **FR-014**: Under `--verbose` the command MUST additionally report what the source can
  never record, as the library states it.
- **FR-015**: The command MUST NOT compute any statistic: no mean, percentile, range,
  standard deviation, rate or per-interval series. It MUST NOT write any data format, and in
  particular MUST NOT emit a per-record stream.
- **FR-016**: The command MUST NOT declare a vocabulary that duplicates one the shared
  library already defines. The position of a sample, the bounds of a run, the outcome of a
  sample, an optional value and the statement of what a source cannot record MUST be used
  from the library.

**Flags and failure**

- **FR-017**: `-o` MUST accept a comma-separated list of report format names whose known
  values are `stats`, `global_stats` and `yml`. Every one is reserved for a later milestone,
  so any `-o` value MUST be a usage error naming the milestone that delivers it, or listing
  the known names for an unknown one. The command MUST NOT offer `-o json` or `-o text`.
- **FR-018**: `--quiet` MUST suppress the report; errors and the unverified-version warning
  MUST still be printed.
- **FR-019**: When the log ends early because the run did not finish, the command MUST
  report the counts of what it did read, say that the log was cut short and where, and
  exit 1. One shape cannot be detected and MUST NOT be claimed: neither log format carries
  an end marker, so a run killed exactly on a record boundary is indistinguishable from a
  complete one and is reported as complete. The documentation MUST say so rather than imply
  every killed run is caught.
- **FR-020**: When a record cannot be decoded, the command MUST stop, exit 1, say the run
  was not read completely, and report no counts: the library states that the records before
  a damaged one are not a result and that no total may be derived from them.
- **FR-021**: Usage failures — a tool name other than `gatling`, more than two arguments,
  an empty argument, an unknown flag, any `-o` value — MUST exit 2 with nothing on standard
  output. `-o` MUST be rejected before the help a bare invocation prints, so that a flag the
  command cannot honour is never accepted and ignored. The exit code MUST be decided by the
  kind of failure and never by matching words in the message, which a caller's own path or a
  quoted log byte can otherwise imitate; every
  failure while executing a valid invocation MUST exit 1. Every error message MUST name what
  was searched, read or refused.
- **FR-022**: Reading the same run twice MUST produce identical output.
- **FR-023**: `galaxio report` with no arguments MUST keep printing help and exiting 0, as
  in milestone v0.12.0; the help MUST show `<tool> [PATH]` and `-o`.
- **FR-024**: The command MUST be documented in the README in the same change.

### Key Entities

- **Run location**: Where a run's artefacts sit and which rule chose them — an explicit
  path, the build tool's marker of the last run, or the newest directory in a results root.
- **Run identity**: What the source stated about the run as a whole: its identifier,
  simulation name, recorded start, tool, tool version, any warning its version gate raised,
  and what it can never record. All of it comes from the library's run description.
- **Run span**: Where the run begins and ends, as the library's own bounds define it.
- **Read tally**: How many requests, group traversals, virtual-user events and run-level
  errors the log held, with requests split by the outcome the source recorded. A count of
  what was walked, not a statistic derived from it.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For every recorded run in the corpus, including at least one text-format and
  one binary-format run, the reported request count equals the total Gatling itself reported
  for that run — its console summary where one was captured, its `global_stats.json`
  otherwise — and the success and failure counts equal its OK and KO totals: a 100% match
  with zero exceptions.
- **SC-002**: An engineer can tell, for any Gatling run they hold, whether this tool reads it
  and what it contains, without opening the HTML report or knowing which Gatling produced it.
- **SC-003**: On a log of at least two gigabytes, peak memory stays under the bound stated in
  the plan, and that bound is the same for a log thirty times smaller.
- **SC-004**: Every failure scenario listed in this specification exits with its documented
  code and names the path, directory, version, tool or format at fault; a script can
  distinguish usage from runtime failure by the exit code alone in 100% of cases.
- **SC-005**: An engineer who has never used the command can read the run they just finished
  from the project directory with `galaxio report gatling` and no path on the first attempt.
- **SC-006**: The reading path this milestone delivers is the one the statistics milestone
  folds over: it yields, per sample, the library's position, outcome and duration without
  this repository having redefined any of them.

## Assumptions

- The shared library `parsec` (MIT, compatible with this repository's GPL-2.0-only licence
  per milestone `v0.12.0`) supplies run discovery, format detection, the version gate, the
  canonical records and the definitions this feature reports: the position of a sample, the
  bounds of a run, the outcome recorded, and what a source cannot provide. Adding it is the
  first new dependency since the constitution was ratified and was approved by the
  maintainer on 2026-09-15.
- Supported versions at the time of writing are text logs from 3.11.5 through 3.12.0 and
  binary logs from 3.13.1 through 3.15.1, with 3.13.0 refused because no run of it can
  produce the report its verification needs. The command takes the range from the library,
  so this list is descriptive, not a contract of this feature.
- The default results root with no path is the Maven and sbt location, `target/gatling`,
  relative to the working directory. Gradle writes elsewhere; a Gradle user passes the path.
  No search of the Gradle location is attempted, because a guessed root can return a
  plausible answer about a run nobody asked about.
- **Maintainer decision, 2026-09-16.** The record stream is withdrawn; this milestone
  delivers reading only, and its milestone boundary does not move. `stats.json` and
  `global_stats.json` are the output the ecosystem wants, in Gatling's own schema and with
  Gatling's own numbers, verified against a run whose files Gatling still produced or
  against the figures in its HTML report where it no longer writes them. Bit-for-bit parity
  is not the requirement; naming every possible divergence is. That work is
  [#51](https://github.com/galax-io/galaxio-cli/issues/51) and
  [#52](https://github.com/galax-io/galaxio-cli/issues/52) and is specified there, not here.
  Issue #50 needs amending to match, since its text still proposes the withdrawn record
  dump.
- **That requirement agrees with the ratified constitution and with both issues.**
  Principle II already requires counts, minimum, maximum, mean and standard deviation to be
  exact, and already says percentiles are estimates that must be labelled as such rather
  than claimed identical to Gatling's; #52 already requires the output to state which
  statistics are estimates. No amendment is needed. What the statistics milestone still owes
  is the size of the divergence: which fields can differ, by how much, and measured against
  which recording. Establishing that from the recorded corpus is in progress and its result
  belongs in #51's plan, not in this feature.
- The tool is named by the user as the first argument and is never inferred from a file
  name. Within the named tool, detection is by content: a Gatling run is what holds a
  `simulation.log`, and the log's format is identified from its leading bytes. When JMeter,
  k6, Locust and Yandex.Tank arrive they become further accepted tool names.
- Out of scope: any statistic; any data format, including the withdrawn record stream and
  the later `stats.json`; compressed or archived inputs; and reading a log that is still
  being written.
- Dependencies: milestone `v0.12.0 Naming and licence` (issues #47 and #48) established the
  `report` namespace and the licence boundary; milestone `v0.11.0 SDD bootstrap` (#49)
  established this workflow; the library's canonical model (parsec#4) defines the primitives
  this command reads and reports.
