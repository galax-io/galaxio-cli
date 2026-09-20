# Feature Specification: Restore legacy Gatling statistics files

**Feature Branch**: `galaxio/120-restore-legacy-stats`

**Created**: 2026-09-18

**Status**: Clarified and implemented

**Input**: [Milestone v0.15.0 Legacy stats.json](https://github.com/galax-io/galaxio-cli/milestone/5)

**Issue**: [#52 — missing legacy statistics files](https://github.com/galax-io/galaxio-cli/issues/52)

## Clarifications

### Session 2026-09-18

- Q: Should exports create explanatory files or print compatibility notices? → A: No.
  Generate only the selected statistics files. Successful export is silent; failures still
  produce diagnostics.
- Q: May request and group names be limited to ASCII? → A: No. A tool reading Gatling logs
  must accept their names without reducing character fidelity.
- Q: Should display names reproduce Gatling's historical HTML substitutions? → A: No.
  Preserve the original name after JSON decoding and use valid JSON escaping.

## User Scenarios & Testing *(mandatory)*

Gatling no longer produces the legacy `stats.json` and `global_stats.json` files used by
existing integrations. The report command restores these files explicitly from a completed
run. Both products are supported and can be selected separately or together. This feature
does not restore Gatling's HTML report or assertion artifacts.

### User Story 1 - Export whole-run statistics for an existing consumer (Priority: P1)

A CI or dashboard maintainer has a completed run from a Gatling version that no longer
writes `global_stats.json`. The maintainer exports the global product into the run's
legacy `js/` location so an existing consumer can read the familiar schema and figures.

**Why this priority**: The missing whole-run file is the immediate compatibility break.

**Independent Test**: Export the global product from each committed Gatling corpus version,
decode the document as the legacy schema, and compare its total, successful and failed
figures with the report summary for the same run.

**Acceptance Scenarios**:

1. **Given** a complete supported run without a `js/` directory, **When**
   `galaxio report gatling <run> -o global_stats` succeeds, **Then**
   `<run>/js/global_stats.json` is created and `stats.json` is not created.
2. **Given** a log path, run directory, results root or default results location, **When**
   the global product is requested, **Then** the existing run-selection rules choose the
   same run as the human report and place the file under that run.
3. **Given** no output product, **When** the existing report command runs, **Then** its
   summary, flags and exit behavior remain unchanged and no statistics file is created.
4. **Given** `--quiet` and a valid export, **When** the command completes, **Then** the
   selected file is written and successful execution emits no human report or explanation.

### User Story 2 - Export the request and group hierarchy (Priority: P1)

A dashboard maintainer needs statistics for individual requests and nested groups. The
maintainer selects `stats` and receives the legacy-shaped hierarchy without distinct
request positions being merged merely because their display names match.

**Why this priority**: Consumers of `stats.json` require more than a whole-run total.

**Independent Test**: Export a run containing ungrouped requests, nested groups, repeated
names and mixed outcomes; decode the hierarchy and verify every observed position and its
statistics.

**Acceptance Scenarios**:

1. **Given** a complete supported run, **When** `-o stats` is selected, **Then** only
   `<run>/js/stats.json` is created, containing the root, groups and requests.
2. **Given** the same run and options, **When** `-o stats,global_stats` is selected,
   **Then** both products are created and the global document equals the root node's
   statistics semantically.
3. **Given** equal display names at different positions or names containing Unicode,
   quotes, backslashes or angle brackets, **When** the run is exported, **Then** positions
   remain distinct, decoded display names equal the source names and the JSON is valid.
4. **Given** the same input and settings, **When** it is exported repeatedly, **Then** the
   hierarchy and generated identifiers are stable.

### User Story 3 - Preserve existing artifacts (Priority: P1)

A user exports into a run directory that may already contain original Gatling output or a
previous export. Existing files remain intact unless replacement is explicitly requested.

**Why this priority**: Compatibility output must not silently destroy run evidence.

**Independent Test**: Put sentinel contents in selected and unselected destinations,
exercise collision and overwrite cases, and compare every affected path.

**Acceptance Scenarios**:

1. **Given** any selected destination already exists, **When** export is attempted without
   `--overwrite`, **Then** the command exits 1, names the collision, preserves all existing
   files and does not create another selected product.
2. **Given** existing selected files, **When** `--overwrite` is supplied, **Then** only
   the selected regular files are replaced; the log and unselected products remain intact.
3. **Given** an invalid product, an incompatible option combination or a percentile
   selection that cannot fill the legacy schema, **When** invoked, **Then** the command
   exits 2 before publishing an artifact.
4. **Given** a damaged, truncated or otherwise unusable log, **When** export is attempted,
   **Then** the command exits 1 and publishes no new legacy statistics.
5. **Given** a destination that cannot be safely written, **When** publishing, **Then** the
   command exits 1, names the path and leaves no truncated newly-created file.

### User Story 4 - Receive reproducible calculated statistics (Priority: P2)

A performance engineer receives only the calculated statistics in the selected legacy
files. Repeated exports remain comparable and use the same statistical definitions as the
existing report command.

**Why this priority**: Existing consumers need stable fields and numbers, not additional
explanatory artifacts.

**Independent Test**: Run the export twice for each corpus fixture, decode every statistics
object, compare non-percentile figures with the recorded Gatling evidence, and verify
percentiles against the report command's documented estimate rules.

**Acceptance Scenarios**:

1. **Given** a recorded run, **When** it is exported, **Then** counts, response-time
   summaries, rates and response-time bands use the existing report definitions.
2. **Given** an outcome containing no samples, **When** it is exported, **Then** its legacy
   statistics columns use zero placeholders and its zero count distinguishes absence from
   a measured zero-duration response.
3. **Given** custom response-time boundaries and exactly four distinct percentile ranks,
   **When** export succeeds, **Then** the selected settings are used consistently in every
   applicable node.
4. **Given** a successful export, **When** outputs are inspected, **Then** only the selected
   statistics files exist; no companion explanation, notice or extra field was generated.

### Edge Cases

- The entire run or one outcome column contains no samples.
- A sample lacks a duration, has an unknown outcome, or would make a mandatory statistic
  unrepresentable.
- A run has one sample, only failures, equal durations or very low throughput.
- A product list repeats a valid product or mixes valid, unknown and reserved products.
- The caller supplies fewer or more than four distinct percentile ranks.
- Request and group names repeat, differ only after identifier shortening, or contain
  non-ASCII characters, combining characters, quotes, backslashes or control characters.
- A group is implicit because requests name it but no group record exists.
- A destination is a directory, symlink or file created concurrently.
- The run directory is read-only or cancellation occurs while the log is scanned.
- Publishing one selected file succeeds and a later file fails; the error must identify
  the failure without claiming that the set was atomic.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The Gatling report command MUST support `stats` and `global_stats` in its
  existing comma-separated `-o` selection, separately or together. Duplicate valid names
  MUST be treated as one selection. Unknown and reserved products MUST remain usage errors.
- **FR-002**: Run discovery and supported input versions MUST follow the existing report
  contract. Products MUST be written under the selected run's `js/` directory, which is
  created when absent. A direct log path MUST use its containing run directory.
- **FR-003**: Without `-o`, the command MUST preserve the existing human report and create
  no legacy file. With `-o`, success MUST produce only the selected files and no human
  report, compatibility notice or estimate explanation. Errors MUST remain visible.
- **FR-004**: `stats.json` MUST contain an `All Requests` root, nested group nodes and
  request nodes with their legacy node types, names, paths, identifiers, statistics and
  child contents. Request nodes MUST NOT have group-only child contents.
- **FR-005**: Distinct request or group positions MUST NOT be combined on display name
  alone. Implicit ancestor groups required by an observed request path MUST be represented.
- **FR-006**: `global_stats.json` MUST contain the whole-run statistics represented by
  the root node of `stats.json` when both use the same input and options.
- **FR-007**: Every statistics object MUST expose the legacy fields for total, successful
  and failed request counts; minimum, maximum, mean and standard deviation; four
  percentiles; four response-time bands; and mean requests per second.
- **FR-008**: Exported figures MUST use the existing report definitions. Selecting a legacy
  product MUST NOT change the whole-run figures that the human report would calculate.
  Percentiles remain the report command's deterministic estimates.
- **FR-009**: Every product MUST be valid UTF-8 JSON in the legacy-shaped schema. Standard
  JSON escaping and a trailing newline are required. Byte-for-byte reproduction of
  historical whitespace, map order, HTML entity substitution and Java floating-point
  spelling is explicitly not required.
- **FR-010**: Original request and group display names MUST survive JSON decoding unchanged.
  Names accepted from a Gatling log MUST NOT be rejected merely for containing Unicode or
  punctuation. Derived identifiers MUST be deterministic and collisions MUST NOT drop or
  merge distinct nodes.
- **FR-011**: Repeated export of the same input with the same options and program version
  MUST produce semantically equal documents and stable identifiers. No compatibility claim
  is made about historical map ordering where the legacy format did not define it.
- **FR-012**: Existing selected destinations MUST remain untouched unless the caller supplies
  `--overwrite`. A pre-existing selected destination MUST prevent publication of the
  requested selection before any selected file is written.
- **FR-013**: Overwrite permission MUST apply only to selected regular files. Directories,
  symlinks and other non-regular destinations MUST be refused, and unselected products and
  unrelated run artifacts MUST remain unchanged.
- **FR-014**: Input validation, scan and rendering failures MUST prevent publication. A
  successfully published individual file MUST contain a complete JSON document. If a later
  file fails, the command MUST report failure but need not provide cross-file atomicity.
- **FR-015**: Legacy export MUST require exactly four distinct normalized percentile ranks,
  defaulting to 50, 75, 95 and 99, and MUST use the existing configurable response-time
  boundaries. An incompatible selection MUST be a usage error.
- **FR-016**: Missing mandatory measurements in a populated outcome MUST cause a named
  failure rather than an invented value. Empty outcomes MUST use the legacy zero encoding,
  distinguished by their zero sample count.
- **FR-017**: The run MUST be scanned once for an export. Memory MAY grow with the number of
  distinct request and group positions and the rendered documents, but MUST NOT retain
  every sample for a fixed set of positions.
- **FR-018**: Exit codes MUST remain 0 for completed requested work, 1 for input, calculation
  or publication failure, and 2 for invalid usage. The complete product selection and
  export-specific options MUST be validated before the run is published.
- **FR-019**: Automated tests MUST cover product selection, both document shapes, root/global
  equality, nested and repeated positions, Unicode and escaped names, empty outcomes,
  deterministic output, invalid options, collisions, overwrite and destination failures
  across the committed Gatling 3.11.5 through 3.15.1 corpus where applicable.
- **FR-020**: User documentation MUST describe the two product names, their destination,
  the four-rank restriction and `--overwrite`. HTML reports, `stats.js`,
  `assertions.xml`, OpenNFR output and support for another input tool are out of scope.

### Key Entities *(include if feature involves data)*

- **Selected run**: The completed Gatling log and owning directory resolved by the report
  command's current discovery rules.
- **Statistics tree**: The whole-run root plus one statistics node for each observed request
  or group position.
- **Statistics node**: A group or request identity with total, successful and failed
  statistics and, for groups, child nodes.
- **Legacy product selection**: The distinct requested products and explicit replacement
  intent applied to one selected run.
- **Published product**: A complete `stats.json` or `global_stats.json` document under
  the selected run's `js/` directory.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Selecting either product creates exactly that one file; selecting both creates
  exactly two files; omitting `-o` creates none.
- **SC-002**: Every committed corpus run from Gatling 3.11.5 through 3.15.1 exports valid
  legacy-shaped JSON whose total, successful and failed whole-run counts equal the report
  summary for that run.
- **SC-003**: A fixture with ungrouped requests, nested groups, repeated names, Unicode,
  quotes and backslashes retains every distinct position and every decoded display name.
- **SC-004**: Across collision and invalid-input tests, zero existing bytes are changed
  without `--overwrite` and zero unselected products are changed with it.
- **SC-005**: Two exports of the same run with the same settings decode to equal documents;
  when both products are selected, the global document equals the root statistics object.
- **SC-006**: Every successful export writes zero companion files, adds zero explanatory
  fields and emits zero human-report or explanatory output.
- **SC-007**: The export scans each run once and retains one aggregate per distinct request
  or group position rather than one retained value per sample.

## Assumptions

- Scope is the remaining work for issue #52 in milestone v0.15.0. Earlier report parsing and
  summary work is the baseline rather than new scope.
- Existing run discovery, version support and statistical definitions remain authoritative.
  This feature adds product selection, per-position aggregates, rendering and publication.
- Both legacy products are in scope and remain individually selectable.
- The fixed four percentile slots require exactly four distinct ranks; customized historical
  runs can be compared only when the caller supplies their original ranks and boundaries.
- Display names follow the clarification: decoded names equal the source names. Historical
  `&quot;` and `&#92;` substitutions are not reproduced.
- The legacy schema is the compatibility target, not Gatling's exact serializer. JSON
  formatting, key order and floating-point spelling may differ while decoded fields and
  values remain compatible.
- Successful export contains statistics only. Existing report documentation may continue to
  explain how its percentile estimates work; no such explanation is added to the export
  invocation's output or artifacts.
- Validation uses the repository's existing corpus and focused command tests. A dedicated
  Jenkins environment, a JVM differential oracle and an operating-system test matrix are
  not acceptance requirements for this feature.
- Work remains in one milestone pull request. This specification does not close the issue,
  merge the pull request or publish a release.
