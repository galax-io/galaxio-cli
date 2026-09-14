# Feature Specification: Report a Gatling Run as Records

**Feature Branch**: `110-report-records`

**Created**: 2026-09-15

**Status**: Draft

**Input**: User description: "https://github.com/galax-io/galaxio-cli/issues/50"

**Tracking**: [milestone `v0.13.0 Report dump`](https://github.com/galax-io/galaxio-cli/milestone/3);
[galax-io/galaxio-cli#50](https://github.com/galax-io/galaxio-cli/issues/50)

## Background

A finished Gatling run is a directory holding an HTML report and a `simulation.log`. The
log is written in one of two formats: a text format up to Gatling 3.12 and a binary format
from 3.13 on. Only the matching Gatling version can read either, and from 3.13.5 on Gatling
no longer exports a file a script can consume. Anyone who wants the raw requests of a run —
to load them into a database, diff two runs, or feed a notebook — has to write a parser
first, once per Gatling version.

This feature makes `galaxio report`, reserved by milestone `v0.12.0` as a help-only
namespace, an operational command. `galaxio report gatling [PATH]` names the tool that
produced the run, locates the run, and turns it into a stream of records: a run header,
then one record per request, group traversal, virtual-user event and run-level error, in
the order the log recorded them. The stream is line-delimited JSON, one object per line,
which is the encoding the live sidecar will emit, so a consumer written for one reads the
other. It is what the command writes when no report format is requested; there is no
`-o json` and no `-o text`. The `-o` flag selects report formats that later milestones
deliver and is reserved now: `stats` and `global_stats` (Gatling's legacy `stats.json` and
`global_stats.json`, milestones v0.14.0 and v0.15.0) and `yml` (the OpenNFR-style YAML
report, postponed). Each is rejected until it exists. `galaxio report` with no arguments
keeps printing help, as in v0.12.0.

The command reads runs through the shared result-primitives library `parsec`, whose licence
compatibility was settled in milestone `v0.12.0`. Statistics, summaries and report formats
are later milestones; this feature emits records and computes nothing.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Report a named run for any supported Gatling version (Priority: P1)

An engineer names the tool (`gatling`) and points the command at a run — either the run
directory or the `simulation.log` inside it — and receives line-delimited JSON on standard
output. It works the same whether
Gatling wrote the log as text (3.11.5 through 3.12.0) or binary (3.13.1 through 3.15.1).
The engineer never has to know or state which Gatling version produced the run.

**Why this priority**: This is the whole value of the milestone. Without it there is no way
to get a run's requests out of Gatling's private format, and every other story builds on
the same output.

**Independent Test**: Run the command against one recorded 3.12.0 run and one recorded
3.15.1 run from the corpus; every output line parses as a standalone JSON object and the
count of request records equals the total Gatling printed in its console summary for that
run.

**Acceptance Scenarios**:

1. **Given** a Gatling 3.12.0 run directory, **When** the engineer runs `galaxio report gatling`
   with that directory as the path, **Then** standard output holds one JSON object per
   line, the first is the run header, and the number of request records matches Gatling's
   console summary for the run; the exit code is 0.
2. **Given** a Gatling 3.15.1 run directory, **When** the engineer runs the command with that
   directory, **Then** the same holds: header first, request count equals the console
   summary, exit code 0.
3. **Given** the path of a `simulation.log` file rather than its directory, **When** the
   engineer passes that path, **Then** the output is identical to passing the directory.
4. **Given** either run, **When** the engineer passes the directory without naming the tool
   first, **Then** the exit code is 2 and the error says the first argument must name the
   tool and lists `gatling`.
5. **Given** a run whose log was written by a Gatling version newer than any the tooling has
   verified, **When** it is read, **Then** the records are emitted, the header carries a
   warning naming the version and why it is unverified, and the exit code is 0.

---

### User Story 2 - Report the latest run without naming it (Priority: P1)

An engineer who has just finished a run inside a project runs `galaxio report gatling` with
no path.
The most recent run under the build tool's usual results directory is found and written, and
a diagnostic on standard error names the run directory chosen and the rule that chose it.

**Why this priority**: The common case is "the run I just did". Requiring the timestamped
directory name Gatling generates would make every script compute it first.

**Independent Test**: In a working directory holding a results root with two run
directories, run `galaxio report gatling` with no path and confirm the newer run is the one
written and named on standard error.

**Acceptance Scenarios**:

1. **Given** a results root containing several run directories and no marker of the last
   run, **When** `galaxio report gatling` runs with no path from that project, **Then** the most
   recently modified run is written and standard error names its directory and states it was
   chosen as the newest.
2. **Given** a results root where the build tool left a marker naming the last run and that
   run still exists, **When** `galaxio report gatling` runs with no path, **Then** the run the marker
   names is written and standard error says it was chosen by the marker.
3. **Given** a directory that holds no run at all, **When** the command runs against it,
   **Then** the exit code is 1 and the error names the directory that was searched.
4. **Given** a path that is a results root rather than a run, **When** it is passed as the
   argument, **Then** it is searched exactly as the default root would be.

---

### User Story 3 - Stream a multi-gigabyte log (Priority: P2)

An engineer reports a run whose log is larger than the memory available and pipes the output
into another tool. Records begin to appear immediately, the command's memory does not grow
with the size of the log, and when the downstream tool stops reading the command stops
quietly.

**Why this priority**: Real soak-test logs run to gigabytes. A command that has to hold the
run in memory first would be unusable on exactly the runs people most need to inspect.

**Independent Test**: Write a log of several gigabytes into a consumer that reads the first
line and then closes; the first record arrives within a second, peak memory stays under the
fixed bound the plan states, and the command exits without a flood of write errors.

**Acceptance Scenarios**:

1. **Given** a log of several gigabytes, **When** its records are written to a pipe, **Then** the first
   record is available before the whole log has been read and peak memory does not exceed a
   fixed bound independent of the log's size.
2. **Given** a downstream consumer that closes the pipe after a few lines, **When** the
   command's next write fails because the reader is gone, **Then** the command stops
   without printing a diagnostic for every subsequent record.
3. **Given** the same log written twice, **When** the outputs are compared, **Then** they are
   byte-for-byte identical.

---

### User Story 4 - Fail with a message the engineer can act on (Priority: P2)

When a run cannot be read, the engineer learns what was refused, read or searched, and
why, and a script can tell a usage mistake from a runtime failure by the exit code alone.

**Why this priority**: The command will run unattended in CI. A bare non-zero exit or a
generic "cannot read log" costs a person a debugging session.

**Independent Test**: Exercise each failure below and assert exit code, that standard
output is empty or holds only complete records, and that standard error names the thing at
fault.

**Acceptance Scenarios**:

1. **Given** a log written by an unsupported Gatling version, including 3.13.0, **When** it is
   read, **Then** the exit code is 1 and the error names the version found and the range
   supported.
2. **Given** a log that was cut short because the run was killed, **When** it is read,
   **Then** every record the log did contain is emitted, standard error says the log ended
   early and where, and the exit code is 1 so that a script does not mistake a partial run
   for a complete one.
3. **Given** a path that does not exist or cannot be read, **When** it is passed, **Then** the
   exit code is 1 and the error names that path and the reason, never claiming the directory
   held no run.
4. **Given** a tool name other than `gatling`, a third positional argument, or an unknown
   flag, **When** the command is invoked, **Then** the exit code is 2, nothing is written to
   standard output, and for the tool the error lists the accepted tool.
5. **Given** a log whose bytes cannot be decoded part-way through, **When** it is read,
   **Then** the exit code is 1 and the error says the records already emitted are not a
   complete run.
6. **Given** `-o` with a reserved format (`stats`, `global_stats` or `yml`, alone or
   comma-separated) or with an unknown name, **When** the command is invoked, **Then** the
   exit code is 2, nothing is written to standard output, and the error names the milestone
   that delivers a reserved format or lists the known names for an unknown one.

### Edge Cases

- A directory is itself named `simulation.log`: it is still examined for a log inside it,
  because what a run directory is called is not the command's to judge.
- A directory holds a `simulation.log` and also contains run directories beneath it: it is
  treated as a run, never as a results root, so the engineer is answered about the place
  they named.
- The marker of the last run names a directory that no longer exists: the marker is
  ignored, the newest run is chosen instead, and standard error reports the rule as
  newest.
- A run has zero requests (every user failed before its first request): the header and any
  user or error records are emitted, the exit code is 0, and the request count is honestly
  zero.
- A run's log contains a value the source can never record, such as a response code for a
  Gatling request: the field is absent from every record and the header states that the
  source cannot provide it, so a consumer does not mistake an empty column for a bug.
- A record lacks a value the source usually records: the field is omitted from that record
  only; it is never filled with zero, an average or a guess.
- Standard output is a terminal rather than a pipe: the output is the same records; no
  paging, colouring or summarising is applied.
- The `--quiet` root flag is set: the informational diagnostic naming the chosen run is
  suppressed; errors are still printed.
- `galaxio report` is invoked with no arguments at all: help is printed and the exit code is
  0, exactly as in milestone v0.12.0; nothing is searched.
- The results root exists but is unreadable because of permissions: this is reported as a
  read failure on that path, not as "no run found".

## Requirements *(mandatory)*

### Functional Requirements

**Input and run selection**

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
- **FR-004**: The command MUST report on standard error which run directory was chosen and
  by which rule (named explicitly, marker, or newest). This diagnostic MUST be suppressed by
  `--quiet`. Under `--verbose` the command MUST additionally report on standard error, after
  the stream, the log format, the Gatling version and the number of records of each kind.
- **FR-005**: When no run is found, the command MUST exit 1 and name the directory searched.
  A directory that cannot be read MUST be reported as a read failure naming that path, never
  as an absence of runs.

**Output**

- **FR-006**: Standard output MUST consist solely of the record stream as line-delimited
  JSON: exactly one JSON object per line, each parseable on its own, with no wrapping
  array, preamble or trailer.
- **FR-007**: The first record MUST be the run header. It MUST carry the run's identity,
  name, description, start time, tool name and tool version, what the source can never
  provide, any warning raised while reading the run, and the run's declared assertions as
  opaque values.
- **FR-008**: After the header the command MUST emit one record per request, group
  traversal, virtual-user event and run-level error, in the order the log recorded them.
  Every record MUST carry a field naming its kind. A declared assertion that a source
  yields among its events rather than ahead of them MUST be emitted as an assertion
  record carrying the payload opaquely, so the stream is lossless.
- **FR-009**: A request record MUST carry the request name, its group path as an ordered
  list, its start time, its duration, its outcome (success or failure) exactly as the source
  recorded it and never inferred, and, on failure, the failure type and message. Scenario,
  response code and byte counts MUST be present when the source recorded them.
- **FR-010**: A group record MUST carry the group path, start time, duration, cumulated
  response time and outcome. A user event record MUST carry the scenario, the event kind
  and its time. A run error record MUST carry its message and time.
- **FR-011**: Every instant MUST be encoded as milliseconds since the Unix epoch and every
  duration as milliseconds.
- **FR-012**: A value the source did not record MUST be omitted from the record; it MUST
  never be rendered as zero, null-as-zero, an average or a guess.
- **FR-013**: With no `-o` the command MUST write the record stream defined above. The
  `-o` flag MUST take a comma-separated list of report format names whose known values are
  `stats`, `global_stats` and `yml`. The command MUST NOT offer `-o json` or `-o text`
  (maintainer decision, see Assumptions).
- **FR-013a**: `stats` and `global_stats` are the reserved names for Gatling's legacy
  `stats.json` and `global_stats.json`: milestone v0.14.0 computes the statistics and
  milestone v0.15.0 writes them in Gatling's legacy shape. This feature MUST NOT compute
  either and MUST reject both names as not yet available, naming milestone v0.15.0 as the
  one that delivers them.
- **FR-013b**: `yml` is the reserved name for the OpenNFR-style YAML report, postponed to a
  later decision. This feature MUST NOT add a YAML output and MUST reject `yml` as not yet
  available.
- **FR-014**: The record schema MUST be documented in the README in the same change, and
  the documented schema is a published surface: a later change that removes or renames a
  field, or changes a field's encoding, is a breaking change.

**Format and version support**

- **FR-015**: The command MUST read both the text and the binary log format and MUST
  identify the format from the log's content, never from its name or extension.
- **FR-016**: The command MUST read every Gatling version the shared library supports, MUST
  take that range from the library rather than a hard-coded list, and MUST refuse a version
  outside it with an exit code of 1 and a message naming the version found and the range
  supported.
- **FR-017**: A version newer than the verified range MUST be read, and the resulting warning
  MUST appear in the run header and on standard error. The warning is not informational and
  MUST be printed even under `--quiet`.

**Streaming and failure**

- **FR-018**: The command MUST emit records as they are read. Peak memory MUST NOT grow with
  the size of the log, and the plan MUST state the peak-memory bound it commits to.
- **FR-019**: When the log ends early because the run did not finish, the command MUST emit
  every record it did contain, report on standard error that the log was cut short and
  where, and exit 1.
- **FR-020**: When a record cannot be decoded, the command MUST stop, exit 1, and say that
  the records already emitted do not form a complete run.
- **FR-021**: When standard output is closed by the reader, the command MUST stop writing
  without emitting a diagnostic per failed write.
- **FR-022**: Usage failures — a tool name other than `gatling`, more than two arguments,
  an unknown flag, any `-o` value (every known name is reserved in this milestone, and an
  unknown name is unknown) — MUST exit 2 with nothing on standard output; every failure
  while executing a valid invocation MUST exit 1. Every error message MUST name what was
  searched, read or refused.
- **FR-023**: The output for a given run MUST be deterministic: running the command twice
  on the same log MUST produce identical output.
- **FR-024**: `galaxio report` with no arguments MUST keep printing help and exiting 0, as
  in milestone v0.12.0; the help MUST show `<tool> [PATH]` and `-o`.

### Key Entities

- **Run location**: Where a run's artefacts sit and which rule chose them — an explicit
  path, the build tool's marker of the last run, or the newest directory in a results root.
- **Run header**: The first record of a stream: run identity, name, description, start,
  tool and version, the capabilities the source lacks, warnings raised while reading, and
  declared assertions carried opaquely.
- **Request record**: One sampled request: name, ordered group path, start, duration,
  outcome, failure detail when failed, and the optional scenario, response code and byte
  counts.
- **Group record**: One traversal of a group: path, start, duration, cumulated response time
  and outcome.
- **User event record**: A virtual user starting or ending a scenario, with the time.
- **Run error record**: A run-level error message with its time.
- **Assertion record**: An opaque declared-assertion payload a source yields among its
  events, carried verbatim; never produced for a Gatling log, whose payloads sit on the
  header.
- **Record stream**: The ordered sequence of header then records that standard output
  carries, one JSON object per line, sharing its schema with the live sidecar.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For every recorded run in the corpus, including at least one 3.12.0 and one
  3.15.1 run, the number of request records emitted equals the total in Gatling's console
  summary for that run — a 100% match with zero exceptions.
- **SC-002**: A consumer can load the output into a JSON-aware tool, a notebook or a database
  loader with no custom parsing: every line of every output parses as a standalone JSON
  object.
- **SC-003**: On a log of at least two gigabytes, the first record reaches the reader within
  one second of invocation and peak memory stays under the bound stated in the plan; the
  bound is the same for a log a hundred times smaller.
- **SC-004**: Every failure scenario listed in this specification exits with its documented
  code and names the path, directory or version at fault; a script can distinguish usage
  from runtime failure by the exit code alone in 100% of cases.
- **SC-005**: An engineer who has never used the command can get the records of the run they
  just finished from the project directory with `galaxio report gatling` and no path on the
  first attempt.
- **SC-006**: The record schema is documented in the README before the milestone ships, and
  the live sidecar can adopt it without a field being renamed or re-encoded.

## Assumptions

- The shared library `parsec` (MIT, compatible with this repository's GPL-2.0-only licence
  per milestone `v0.12.0`) supplies run discovery, format detection, the version gate and
  the canonical records. Adding it is the first new dependency since the constitution was
  ratified and is the plan's ask-first decision; this specification depends on the
  capabilities it already publishes, not on their shape.
- Supported versions at the time of writing are text logs from 3.11.5 through 3.12.0 and
  binary logs from 3.13.1 through 3.15.1, with 3.13.0 refused because no run of it can
  produce the report its verification needs. The command takes the range from the library
  so this list is descriptive, not a contract of this feature.
- The default results root with no path is the Maven and sbt location, `target/gatling`,
  relative to the working directory. Gradle writes elsewhere; a Gradle user passes the
  path. No search of the Gradle location is attempted, because a guessed root can return a
  plausible output for a run nobody asked about.
- The run's declared assertions are carried in the header as opaque values so the stream is
  lossless. Interpreting them is milestone `v0.17.0 Assertion artefacts` and is out of
  scope here.
- The maintainer's decision (2026-09-15) on the `report` surface: the tool is the first
  positional argument (`galaxio report gatling`); the JSON Lines record stream is what the
  command writes when no report format is requested and has no `-o` name of its own; there
  is no `-o json` and no `-o text`; `-o` takes a comma-separated list of report formats, of
  which `stats` and `global_stats` (Gatling's legacy files, milestones v0.14.0 and v0.15.0)
  and `yml` (OpenNFR-style YAML, postponed) are reserved and rejected until built. The
  plan's Constitution Check records the deviation from Principle I's `-o text|json` with
  its justification.
- A truncated log yields the records it holds and a non-zero exit. Whether a partial run
  is usable is the consumer's decision; the exit code makes the partiality impossible to
  miss in a script, and the emitted records make it possible to use anyway.
- No subcommand: `report` itself is the operational command (maintainer decision,
  2026-09-15), taking `<tool> [PATH]`. Because the tool argument is required, `galaxio
  report` with no arguments keeps printing help exactly as the v0.12.0 placeholder did, so
  no published behaviour changes; `galaxio report <something>` still exits 2, now as an
  unsupported tool rather than an unknown command.
- Which tool produced the run is stated by the user as the first argument, never read from
  a file name. Within the named tool, detection is by content: a Gatling run is what holds a
  `simulation.log`, and the log's format (text or binary) is identified from its leading
  bytes. The header's `tool` field still reports what the source stated. A path that is not
  a run of the named tool is a runtime error naming the path. When JMeter, k6, Locust and
  Yandex.Tank arrive (milestones v0.19.0 to v0.22.0) they become further accepted tool
  names.
- Group path, request name and scenario are emitted as the source recorded them, without
  normalisation. Bucketing and comparison across runs are the summary milestone's concern.
- Out of scope: any statistic (count, mean, percentile, range, series), any report format
  other than the record stream, CSV output (rejected in the issue because the nested group
  path has no honest CSV encoding), compressed or archived inputs, and reading a log that
  is still being written.
- Dependencies: milestone `v0.12.0 Naming and licence` (issues #47 and #48) established the
  `report` namespace and the licence boundary; milestone `v0.11.0 SDD bootstrap` (#49)
  established this workflow; the library's canonical model (parsec#4) defines the records
  this command emits.
