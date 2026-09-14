# Quickstart: prove `galaxio report` end to end

**Feature**: [spec.md](spec.md) | **Contracts**: [cli.md](contracts/cli.md),
[records.md](contracts/records.md)

## Prerequisites

- Go 1.27.1 (`go.mod`), `jq` for the checks below.
- `github.com/galax-io/parsec v0.1.0` (approved 2026-09-15) added by the first implementation task: `go get github.com/galax-io/parsec@v0.1.0 && go mod tidy`.
- The corpus copied to `internal/report/testdata/corpus/gatling/` with its `PROVENANCE.md`
  (source: parsec v0.1.0 `testdata/corpus/gatling`, MIT, recordings never edited).

```bash
go build -o dist/galaxio ./cmd/galaxio
```

## 1. Named run, both formats (US1, SC-001)

```bash
C=internal/report/testdata/corpus/gatling
dist/galaxio report gatling $C/3.12.0 | head -1 | jq .kind        # "run"
dist/galaxio report gatling $C/3.12.0 | jq -c 'select(.kind=="request")' | wc -l   # 36
dist/galaxio report gatling $C/3.15.1 | jq -c 'select(.kind=="request")' | wc -l   # 102
dist/galaxio report gatling $C/3.15.1/simulation.log | cmp - <(dist/galaxio report gatling $C/3.15.1) && echo identical
```

Expected: the counts equal Gatling's console summary for each recording (36 for the text
runs 3.11.5/3.12.0; 102 for the binary runs 3.13.1/3.14.9/3.15.1), exit code 0, and the
log path and its directory produce identical output.

Observed 2026-09-15: first kind `run`; 3.12.0 → 36 request records; 3.15.1 → 102; the log
path and its directory → `cmp` reports identical output.

## 2. Latest run without naming it (US2)

```bash
dist/galaxio report gatling $C/lastrun/results 2>&1 >/dev/null
# report: reading …/corpussimulation-20260909022708912 (found by lastRun.txt)
tmp=$(mktemp -d); cp -R $C/lastrun/results $tmp/ && rm $tmp/results/lastRun.txt
dist/galaxio report gatling $tmp/results 2>&1 >/dev/null     # … (found by newest)
( cd $tmp && mkdir -p target && cp -R results target/gatling && dist_abs=$OLDPWD/dist/galaxio && $dist_abs report gatling >/dev/null )   # no path → target/gatling
dist/galaxio report gatling $tmp; echo "exit=$?"              # Error: … no Gatling run under …; exit=1
```

Observed 2026-09-15:

```text
report: reading internal/report/testdata/corpus/gatling/lastrun/results/corpussimulation-20260909022708912 (found by lastRun.txt)
report: reading <tmp>/results/corpussimulation-20260909022727230 (found by newest)
report: reading target/gatling/corpussimulation-20260909022727230 (found by newest)   # no path, from the project directory; exit 0
Error: gatling: no Gatling run under <tmp>                                             # exit 1
```

## 3. Streaming and a closed pipe (US3, SC-003)

```bash
dist/galaxio report gatling $C/3.15.1 | head -n 1      # header appears immediately, no stderr flood
go test ./internal/report -run TestWriteAllocations -v     # ≤ 4 allocs/record
go test ./internal/report -bench BenchmarkWrite -benchmem -run ^$
```

Measured 2026-09-15 on an Apple M2 Pro (darwin/arm64, Go 1.27.1), `-benchtime=1x`, the
synthetic log replaying the 3.12.0 body through an `io.Reader` so no file is written:

| Replay size | Records | Throughput | Peak heap in use | Allocations (whole run) |
|---|---|---|---|---|
| 64 MiB | 0.94 M | 121 MB/s, 1.69 M records/s | 2.0 MiB | 475 |
| 2 GiB | 30.0 M | 124 MB/s, 1.74 M records/s, 17.3 s | 2.0 MiB | 87 |

`TestWriteAllocations` under the race detector: 1.79 allocations per record (goal ≤ 4);
without it the benchmark shows none per record. The memory goal is heap in use < 32 MiB for
any log size, met with a margin of sixteen times.

Observed 2026-09-15, closed pipe: a 14 MB replayed run piped into `head -n 1` delivered the
header line, the pipeline exited 0 within the second, and the process wrote nothing to
standard error.

## 4. Failures name what was at fault (US4)

```bash
html=$(mktemp -d); printf '<!DOCTYPE html>\n' > $html/simulation.log
dist/galaxio report gatling $html; echo "exit=$?"                  # not a Gatling simulation.log …; 1
dist/galaxio report gatling /nonexistent; echo "exit=$?"           # cannot read /nonexistent …; 1
cut=$(mktemp -d); head -c 2000 $C/3.15.1/simulation.log > $cut/simulation.log   # a run is a simulation.log
dist/galaxio report gatling $cut | wc -l; echo "exit=${PIPESTATUS[0]}"   # 63 lines (header + 62), exit 1, stderr: the log is cut short …
dist/galaxio report jmeter $C/3.15.1; echo "exit=$?"               # unsupported tool "jmeter": accepted tools: gatling; 2
dist/galaxio report $C/3.15.1; echo "exit=$?"                      # the path is taken as a tool name → unsupported tool; 2
dist/galaxio report gatling $C/3.15.1 extra; echo "exit=$?"        # accepts at most 2 arg(s); 2
dist/galaxio report gatling -o stats,global_stats $C/3.15.1; echo "exit=$?"   # not available yet, arrives with v0.15.0; 2
dist/galaxio report gatling -o yml $C/3.15.1; echo "exit=$?"       # not available yet (postponed YAML); 2
dist/galaxio report gatling -o json $C/3.15.1; echo "exit=$?"      # unknown report format "json": known formats: stats, global_stats, yml; 2
dist/galaxio report; echo "exit=$?"                                # help, 0
```

Observed 2026-09-15 (bash; zsh has no `PIPESTATUS`):

```text
Error: <tmp>/html/simulation.log: gatling: not a Gatling simulation.log: found "<!DOCTYPE " at the start of the stream   # exit 1
Error: cannot read /nonexistent: no such file or directory                                                                 # exit 1
lines=63; Error: gatling: byte 1991: the log is cut short: 9 trailing bytes could not be decoded, and a request name was still to come; the 62 records already written are what the run recorded   # exit 1
Error: unsupported tool "jmeter": accepted tools: gatling                                                                  # exit 2
Error: unsupported tool "internal/report/testdata/corpus/gatling/3.15.1": accepted tools: gatling                          # exit 2 (path where the tool should be)
Error: accepts at most 2 arg(s), received 3                                                                                # exit 2
Error: report format "stats" is not available yet: it arrives with milestone v0.15.0 Legacy stats.json                    # exit 2 (-o stats,global_stats)
Error: report format "yml" is not available yet: the OpenNFR YAML report is postponed                                      # exit 2
Error: unknown report format "json": known formats: stats, global_stats, yml                                               # exit 2
help printed                                                                                                                # exit 0
```

Version-gate cases (below range 3.10.0, refused 3.13.0, newer-than-range warning) are
synthetic and live in `internal/report` tests; run them with:

```bash
go test ./internal/report -run 'TestOpen|TestVersion' -v
```

## 5. Help is unchanged

```bash
dist/galaxio report --help | grep -E 'report <tool> \[PATH\]|-o, --output'
```

Observed 2026-09-15:

```text
  galaxio report <tool> [PATH] [flags]
  -o, --output string   report format(s) to produce, comma-separated: stats, global_stats, yml (reserved for later releases)
```

## 6. Gates

```bash
gofmt -l . && go vet ./... && go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1
go mod tidy && git diff --exit-code -- go.mod go.sum
go test -tags=integration -race -count=1 ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Expected: no formatting diff, vet clean, all tests pass with the race detector, total
coverage ≥ 80%, tidy leaves no diff, integration suite passes (the pipe test needs `head`),
no known vulnerabilities.

Observed 2026-09-15 (T016, commit `80f45e1` and its predecessors, Go 1.27.1, darwin/arm64):

```text
gofmt:      no diff
vet:        clean
tests:      10 packages ok (race on)
coverage:   84.6% total
report pkg: 95.1% mean over 33 functions
mod tidy:   no diff
mod verify: all modules verified
integration: 10 packages ok
build:      ok ( 27M)
govulncheck: No vulnerabilities found.
```

## 7. Documentation

`README.md` § Reporting Ecosystem documents `galaxio report`, its argument, `-o`, exit
codes and the record keys, and links to [contracts/records.md](contracts/records.md) for the
full schema. `galaxio report --help` shows `<tool> [PATH]` and `-o`.
