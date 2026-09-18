# Quickstart: prove the live runs are served by the mock

**Feature**: [spec.md](spec.md) | **Data model**: [data-model.md](data-model.md)

Each section states what is expected. "Observed" is filled in, with the date, when the
validation task for that section runs; nothing is recorded here before it was seen.

## Prerequisites

- Go 1.27.1; a JDK (17 or later) and sbt on the path; the network, for the published template
  registry and sbt's dependencies.
- The two t-digest jars `TestReportLiveGatling` already needs, in the Maven or Coursier cache or
  in the directory `GALAXIO_TDIGEST_JARS` names.

## 1. The mock stays out of the binary and does real work (FR-001 to FR-003, SC-003, SC-004)

```bash
go list -deps ./cmd/galaxio | grep -c reporttest
grep -nE '"time"|/rand"|time\.Sleep' internal/report/reporttest/mock.go
go test -race -count=1 ./...
```

Expected: `0` packages the command links are `reporttest`. `grep` finds nothing: the mock neither
sleeps nor draws a random number. The ordinary suite is green, with no test added to it.

Observed 2026-09-18 at 2ef2976, Go 1.27.1, darwin/arm64: `0`; `grep` printed nothing and exited 1;
all ten packages with tests `ok` under the race detector.

## 2. Every Gatling version against the mock (US1, US2, FR-004, FR-005, FR-008, SC-001, SC-002)

```bash
rec=$(mktemp -d)
GALAXIO_LIVE_GATLING=1 GALAXIO_LIVE_STEADY=1m GALAXIO_LIVE_RECORD=$rec \
  go test -tags=integration -race -count=1 -timeout 40m -run TestReportLiveGatling -v ./internal/report/
go build -o /tmp/galaxio ./cmd/galaxio
gunzip -k $rec/3.11.5/simulation.log.gz && /tmp/galaxio report gatling $rec/3.11.5
GALAXIO_LIVE_RECORDINGS=$rec go test -count=1 -run TestLiveGatlingRunsWereServedTheSameResponses -v ./internal/report/
```

Expected:

- The live test passes for 3.11.5, 3.12.0, 3.13.1, 3.14.9 and 3.15.1. Gatling printed no failed
  request, and a successful `GET /`, `GET /report` and `GET /export` in its console, in both
  console formats.
- Every comparison it made before this feature holds. Where a later version prints another
  percentile, a note gives the one tdunning/t-digest#230 accounts for.
- The summary of the 3.11.5 run shows successful requests in the band under 800 ms and in the
  band from 800 to 1 200 ms, and no failed request.
- `TestLiveGatlingRunsWereServedTheSameResponses` passes on the new set: every version holds the
  same number of requests.

Observed 2026-09-18 at 2ef2976, OpenJDK 17.0.10, sbt 1.12.13, from 16:19:48 to 16:27:15 UTC:

- `--- PASS` for 3.11.5 (89.9 s), 3.12.0 (90.8 s), 3.13.1 (88.1 s), 3.14.9 (86.3 s) and 3.15.1
  (87.3 s). `servedEverything` passed in both console formats, `(OK=n KO=m)` up to 3.13.1 and
  columns from 3.14.9.
- No difference was reported. Every note was `MergingDigest gives …`, where it differs from
  Gatling 3.11's digest, or, for 3.13.1, 3.14.9 and 3.15.1, the p95 or p99 Gatling printed, with
  `tdunning/t-digest#230` (3.15.1: printed 237 and 1101, where the digest and this tool give
  235 and 1100).
- The summary of the 3.11.5 run: `3576 requests · 50.37 req/s`, `✓ 3576 ok · 100 %`,
  `✗ 0 failed`. The all row reads min 3, mean 49, std 185, p50 4, p75 6, p95 245, p99 1113,
  max 1189. The bands: 3 480 ok under 800 ms (97.32 %) and 96 ok from 800 to 1 200 ms (2.68 %).
- `TestLiveGatlingRunsWereServedTheSameResponses`: `--- PASS`. Every version holds `3576
  requests (3576 ok, 0 failed)`.

## 3. A broken endpoint fails the Gatling run (FR-008, SC-005)

```bash
cp internal/report/reporttest/mock.go /tmp/mock.go.keep
sed -i '' 's#"/export"#"/exportx"#' internal/report/reporttest/mock.go
GALAXIO_LIVE_GATLING=1 GALAXIO_LIVE_VERSIONS=3.11.5 GALAXIO_LIVE_STEADY=30s \
  go test -tags=integration -count=1 -timeout 20m -run TestReportLiveGatling -v ./internal/report/ | grep -E 'Gatling printed|^--- '
cp /tmp/mock.go.keep internal/report/reporttest/mock.go && git diff --exit-code -- internal/report/reporttest/mock.go
```

Expected: the test fails on two lines, the failed requests Gatling printed and no successful
`GET /export`. The mock is restored unchanged.

Observed 2026-09-18 at 2ef2976, once with 3.11.5 and once with 3.15.1, to cover both console
formats. Each failed with exactly:

```text
    live_integration_test.go:242: Gatling printed 51 failed requests where the mock fails none
    live_integration_test.go:242: Gatling printed no successful GET /export: the mock or the live scenario is broken
--- FAIL: TestReportLiveGatling
```

`git diff --exit-code` found the mock unchanged afterwards, both times.

## 4. The committed recordings (FR-006)

```bash
git diff --stat main -- internal/report/testdata/live/gatling/
go test -race -count=1 -run 'TestSummaryMatchesLiveGatlingRuns|TestLiveGatlingRunsWereServedTheSameResponses|TestEtalonsAreOfTheirRuns' ./internal/report/
git show v0.14.0:internal/report/live_integration_test.go | grep -n 'func livePlan'
```

Expected: only `RECORDING.md` changed under `testdata/live/gatling/` — the recordings themselves
are untouched. The new `testdata/live/scenario/` is the two committed Scala files, not a
recording. The tests of the recordings pass, and `livePlan` is still readable at `v0.14.0`,
where `RECORDING.md` says it is.

Observed 2026-09-18 at 2ef2976: `RECORDING.md | 7 +++++--` alone under `testdata/live/gatling/`;
`ok` for the three tests; `34:func livePlan(i uint64) (time.Duration, int) {`.

## 5. Gates

```bash
test -z "$(gofmt -l .)" && echo fmt-clean
go vet ./... && go vet -tags=integration ./... && echo vet-clean
go test -race -count=1 -coverprofile=/tmp/cover.out ./... && go tool cover -func=/tmp/cover.out | tail -1
go mod tidy && git diff --exit-code -- go.mod go.sum && echo tidy-clean
```

Expected: `fmt-clean`, `vet-clean`, green with total coverage at least 80 %, `tidy-clean`.

Observed 2026-09-18 at 2ef2976: `fmt-clean`, `vet-clean`, ten packages `ok`, `total: 87.5%`,
`tidy-clean`.
