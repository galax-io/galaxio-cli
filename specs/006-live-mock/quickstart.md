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

## 5. Gates

```bash
test -z "$(gofmt -l .)" && echo fmt-clean
go vet ./... && go vet -tags=integration ./... && echo vet-clean
go test -race -count=1 -coverprofile=/tmp/cover.out ./... && go tool cover -func=/tmp/cover.out | tail -1
go mod tidy && git diff --exit-code -- go.mod go.sum && echo tidy-clean
```

Expected: `fmt-clean`, `vet-clean`, green with total coverage at least 80 %, `tidy-clean`.
