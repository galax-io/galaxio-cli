# Data Model: Serve the Live Gatling Runs from a Service That Does Real Work

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

Nothing here is persisted and nothing reaches the binary.

## Mock

`reporttest.Mock()`, an `http.Handler`. It holds no state: no counter, no clock, no generator.

| Endpoint | Work per request | Answer |
|---|---|---|
| `GET /` | 2 000 times a fixed 4 KiB block into one SHA-256 | `200 {"digest":"<first 8 bytes of the sum, hex>"}` |
| `GET /report` | 128 000 blocks | the same |
| `GET /export` | 640 000 blocks | the same |
| any other path | none | `404` |
| another method on an endpoint | none | `405` |

The block is the 4 096 bytes `byte(i * 31)` for `i` from 0. Its sums are fixed, so each
endpoint's answer is the same for every request, and a test can hold it to a constant. The
request's headers and body are not read. Answers are `application/json`.

## The scenario the live test writes

| Request | Who sends it |
|---|---|
| `GET /` | every virtual user |
| `GET /report` | users whose number modulo 100 is below 7 |
| `GET /export` | users whose number modulo 100 is 97 or more |

Every request is checked for `status is 200`, as the template checks its own. The requests live
in `internal/report/testdata/live/scenario/cases/HttpActions.scala` and the flow in
`scenarios/HttpScenario.scala` beside it; the test copies both over the rendered files of the
same names.

## What the live test reads of Gatling's console

| Read | From | Fails when |
|---|---|---|
| failed requests | the Global Information block | any |
| successful requests of `GET /`, `GET /report`, `GET /export` | the last progress block: `(OK=n KO=m)` up to 3.13, `\| total \| ok \| failed` from 3.14 | none for a name |
