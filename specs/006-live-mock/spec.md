# Feature Specification: Serve the Live Gatling Runs from a Service That Does Real Work

**Feature Branch**: `115-live-mock`

**Created**: 2026-09-18

**Status**: Draft

**Input**: User description: "давай мы данный мок закоммитим в репозиторий и используем в
тестах интеграции например?", "только надо стабилити тест запускать в интеграции а не в
максперфе", "да не надо там перегружать просто хотелось обновить мок и все" and "мок + пару
долгих эндпоинтов" — given for
[galax-io/galaxio-cli#117](https://github.com/galax-io/galaxio-cli/issues/117), the feature
this specification describes. In English: commit this mock to the repository and use it in the
integration tests; run the Stability simulation there, not MaxPerformance; no need to overload
anything, just update the mock; the mock plus a couple of long endpoints.

**Tracking**: [milestone `v0.15.0 Legacy stats.json`](https://github.com/galax-io/galaxio-cli/milestone/5);
[galax-io/galaxio-cli#117](https://github.com/galax-io/galaxio-cli/issues/117)

## Background

`TestReportLiveGatling` runs Gatling itself and holds `galaxio report gatling` to what Gatling
printed for the same run. The service those runs load is a stub. For every request it sleeps
for a time drawn from a fixed distribution and answers `200`, or `500` for 2 % of requests.
Every response time in a live run is therefore a number the stub chose.

On 2026-09-18 a run was made by hand against a small service that does real work for every
request: it hashes a buffer, and a request takes as long as the hashing and the requests ahead
of it for a core. The summary equalled Gatling's on every figure of that run too. That service
is the mock this feature commits and gives the live test in place of the stub.

## Clarifications

### Session 2026-09-18

- Q: Should the service of the hand-made run be kept and used by the tests? → A: Yes. It
  becomes part of the repository's test support, and the live integration test loads it.
- Q: Which of the template's simulations does the integration test run? → A: `Stability`, as it
  does today, not `MaxPerformance`.
- Q: Should the test push the service past its knee? → A: No. The load stays what the live test
  runs today; only the mock changes.
- Q: What does the mock serve? → A: A fast endpoint and a couple of long ones.
- Q: Where do the long endpoints get their time, when a second of work for each of 50 requests
  a second is more than a machine has? → A: From real work as well, and only a small share of
  the virtual users calls them.
- Q: How is the mock itself tested? → A: With Gatling: by the live run it serves, not by a Go
  test of its own.
- Q: Where does the work belong? → A: In milestone `v0.15.0 Legacy stats.json`, beside #52.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Hold the summary to Gatling on response times nobody chose (Priority: P1)

A maintainer runs the live test. It renders galaxio's own template, runs `Stability` against the
mock and holds every figure of the summary to what Gatling printed and wrote, exactly as today.
The difference is the service: every response time in the run comes from work the mock did,
not from a table.

**Why this priority**: it is the change asked for.

**Independent Test**: run the live test for one Gatling version and read its verdict.

**Acceptance Scenarios**:

1. **Given** a machine with a JDK, sbt and the variable the live test already asks for, **When**
   the live test runs, **Then** Gatling loads the mock, and the summary's figures are held to
   Gatling's console, `global_stats.json` and the etalon as they are today.
2. **Given** that run, **When** every figure equals Gatling's, **Then** the test passes.
3. **Given** the mock, **When** it serves a request, **Then** nothing in it waits, sleeps or draws
   a random number.
4. **Given** a run in which Gatling records a failed request, or no successful request to one of
   the mock's endpoints, **When** the live test reads what Gatling printed, **Then** it fails and
   names the endpoint.

---

### User Story 2 - Long requests beside short ones (Priority: P2)

Every virtual user calls the fast endpoint. A small share of them also calls one of two long
endpoints, so the run holds three request names and response times from a few milliseconds to
about a second, across the response-time bands.

**Why this priority**: a run of one request name whose every response takes a few milliseconds
would hold the summary to Gatling on almost nothing.

**Independent Test**: run the live test for one Gatling version and read the summary of its
run.

**Acceptance Scenarios**:

1. **Given** a live run, **When** its log is read, **Then** it holds requests to the fast
   endpoint and to both long ones.
2. **Given** two live runs of different Gatling versions, **When** their logs are read, **Then**
   both hold the same number of requests to each endpoint.

---

### Edge Cases

- **The template changes the scenario the test replaces.** The test fails and names the file it
  expected, rather than running a scenario it did not write.
- **A long endpoint is called on a slow machine.** The request simply takes longer; nothing is
  asserted about response times.
- **The committed recordings.** They were served by the old stub and stay as they are, with the
  tests that read them. Their provenance says where that stub lives.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The repository's test support MUST provide a mock service that does real work for
  every request it serves and chooses no response time. How long a request takes MUST come from
  the work and from the requests ahead of it for a core.
- **FR-002**: The mock MUST serve one fast endpoint and two long ones, each doing a fixed amount
  of work.
- **FR-003**: The mock MUST be test support only and MUST NOT reach the command a user installs.
- **FR-004**: `TestReportLiveGatling` MUST load the mock instead of the stub, with the same
  simulation, intensity, versions and comparisons as today.
- **FR-005**: Every virtual user MUST call the fast endpoint. A fixed share of users MUST also
  call each long endpoint, chosen by the user's number, so that every Gatling version sends the
  same requests.
- **FR-006**: The committed recordings and the tests that read them MUST stay as they are, and
  their provenance MUST say where the stub that served them can still be read.
- **FR-007**: The feature MUST add no environment variable and no change to what the command
  does.
- **FR-008**: The mock MUST be tested by the Gatling run it serves. The live test MUST fail when
  Gatling prints a failed request, or no successful request to one of the endpoints.

### Key Entities

- **Mock**: the service the live runs load. It has three endpoints, and for each the amount of
  work it does per request, and nothing else.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A live run of Gatling 3.11.5 against the mock passes every comparison the live test
  makes today.
- **SC-002**: That run holds three request names and response times in at least two of the
  response-time bands.
- **SC-003**: The mock's code holds no sleep and no random number.
- **SC-004**: The ordinary suite and the integration suite run without the live variable are
  unchanged.
- **SC-005**: A mock with one endpoint broken fails the live test on that endpoint.

## Assumptions

- The mock is not a mock under Principle III: the system under test is the command, and Gatling
  and the reference digest are real. The service stands in for whatever a user loads, which no
  test can reach.
- New live runs have no failed requests, since the mock fails none at this load. Failed requests
  stay covered by the committed recordings, the corpus and the synthetic runs.
- A long endpoint doing a second of work for every user at 50 users a second would need 50
  cores, so the long endpoints are called by a small share of users.
