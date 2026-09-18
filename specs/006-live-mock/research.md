# Research: Serve the Live Gatling Runs from a Service That Does Real Work

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Date**: 2026-09-18

Measured on the machine the live recordings were made on: macOS 26.6.2, Apple M2 Pro (8
performance and 4 efficiency cores), Go 1.27.1.

## 1. The mock

**Decision**: `reporttest.Mock()`, an `http.Handler` in the test-support package. For each request
it feeds a fixed 4 KiB block to one SHA-256 a fixed number of times, and answers `200` with
`{"digest":"<the first 8 bytes of the sum, in hex>"}`. The number of times depends only on the
endpoint:

| Endpoint | Blocks | Hashed | Measured alone |
|---|---|---|---|
| `GET /` | 2 000 | 7.8 MiB | 3–4 ms |
| `GET /report` | 128 000 | 500 MiB | 202 ms |
| `GET /export` | 640 000 | 2.5 GiB | about 1 s (512 000 blocks took 811–817 ms) |

Any other path is answered `404` by the standard library's `ServeMux`. Any other method, `HEAD`
included, is answered `405` by the handler before any work. A `GET` pattern would also match
`HEAD`, and do the whole work for it.

**Rationale**: a request takes as long as its hashing and the requests ahead of it for a core,
so the response times of a run come from the machine and the load, not from a table. `/export`
is sized to land between the default band boundaries of 800 and 1 200 ms, and above them when
a core is busy or slow. `internal/report/reporttest` is imported only by tests, so the mock never
reaches the binary.

**Alternatives considered**:

- **The stub as it is.** Every response time in it is a number drawn from `livePlan`.
- **Long endpoints that sleep**, as a mock of a slow dependency. The maintainer chose real work.
- **The service of the hand-made run unchanged**: one endpoint, and a refusal with `503` past 512
  requests in flight. That limit acts only past the knee, which a live run at 50 users a second
  never reaches, so it would be code no run exercises.

## 2. Who calls the long endpoints

**Decision**: every virtual user calls `GET /`. Users whose number modulo 100 is below 7 then
call `GET /report`, and those whose number modulo 100 is 97 or more call `GET /export`.

**Rationale**: the live test starts 50 users a second. If every user called both long
endpoints, the mock would need 50 × 1.2 s of hashing each second, 60 cores. With 7 % and 3 %:

- `/`: 50 × 3.5 ms.
- `/report`: 3.5 × 202 ms.
- `/export`: 1.5 × 1 s.

That is about 2.4 cores. A user's number is given by Gatling in the order users start, so every
version sends the same requests. `TestLiveGatlingRunsWereServedTheSameResponses` then holds a new
set of recordings as it holds the committed one.

**Alternatives considered**:

- **Gatling's `randomSwitch`.** Its generator is unseeded, so two versions would send different
  numbers of long requests, and a new set of recordings would fail that test.
- **The mock choosing the work by arrival order**, as `livePlan` chose a wait. That would keep
  one request name, where the maintainer asked for endpoints. It would also make the mock choose
  which request is long.

## 3. The scenario

**Decision**: after rendering the template, the test copies two committed Scala files from
`internal/report/testdata/live/scenario/` over the rendered ones, at the same paths under
`src/test/scala/org/galaxio/performance/live/`:

- `cases/HttpActions.scala` keeps the template's `getMainPage` for `GET /` and adds `getReport`
  and `getExport`, named `GET /report` and `GET /export` and checked for `status is 200` as the
  template checks its own;
- `scenarios/HttpScenario.scala` keeps the template's object and class and sends them.

If the template no longer renders either file, the test fails and names it.

**Rationale**: the template's scenario sends one request, and the endpoints need a scenario that
calls them. Requests belong in `cases/` and the flow in `scenarios/`, as the template lays them
out. Only these two files are committed; the rest of the project is rendered by `galaxio` from
the template, as a user gets it. Real `.scala` files are readable and editable as Scala, which
strings inside the Go test were not, and replacing whole files does not depend on the template's
formatting, which splicing a line into its file would. The same two files compile and run on
3.11.5, 3.12.0, 3.13.1, 3.14.9 and 3.15.1, so no version needs its own.

**Alternatives considered**: committing the whole Gatling project — its build and every
version pin would drift from the template users get. Rendering the template itself is tested
where the template lives: the CI of galax-io/templates-gatling builds `galaxio` from this
repository, runs `galaxio template validate` on the pack, renders every template and runs its
`Debug` simulation.

## 4. The recordings

**Decision**: the five committed live recordings, their etalons and the tests that read them do
not change. `RECORDING.md` says that the stub which served them is `livePlan` in
`internal/report/live_integration_test.go` as tagged `v0.14.0`. It is removed from the tree,
since nothing else calls it.

## 5. How the mock is tested

**Decision**: with Gatling, by the live run it serves. There is no Go test of the mock
(maintainer, 2026-09-18). `TestReportLiveGatling` fails when Gatling prints a failed request, or
no successful request to one of the three endpoints.

**Rationale**: a summary equal to Gatling's proves nothing about the mock. Gatling records a
broken endpoint as failed requests, which the summary counts just as faithfully. So the test
reads what Gatling printed. The failed count comes from its Global Information block, which the
test already parses. The count of each request name comes from the last progress block, which
holds the whole run:

- Gatling 3.11 to 3.13 print `> GET /report  (OK=230  KO=0 )`;
- 3.14 and later print `> GET /report  |  230 |  230 |  0`, in columns of the total, ok and
  failed.

A mock with `/export` broken failed the test with exactly two messages: 51 failed requests, and
no successful `GET /export` (quickstart §3).

**Alternatives considered**: a table-driven Go test calling the mock through `httptest`, which
the maintainer rejected: it tests the mock without the load generator it exists for.

## 6. What a new live run no longer holds

The mock fails no request at this load, so a new live run has no failed outcome. Its failed
column is absent, and the summary reports it as absent, which Gatling's console and
`global_stats.json` agree with. Failed requests stay covered elsewhere:

- five corpus runs with failures;
- the five committed live recordings, with 240 failures each;
- the synthetic runs, whose failures are slower than their successes.

## 7. Skills read

- `golang-testing`, `golang-naming` and `golang-documentation` for the mock and the check.
- `galaxio-gatling-pro` for the scenario. Requests go in `cases/` and the flow in `scenarios/`,
  in two files and not one, and the checks are the template's own `status is 200`.

No skill asked for a dependency, a layout change or a weaker gate.
