# Quickstart: prove the report summary end to end

**Feature**: [spec.md](spec.md) | **Contract**: [contracts/cli.md](contracts/cli.md) |
**Data model**: [data-model.md](data-model.md)

Each section states what is expected. "Observed" is filled in, with the date, when the
validation task for that section runs; nothing is recorded here before it was seen.

## Prerequisites

- The amendment of constitution Principle II's percentile clause is merged
  ([research.md](research.md) §13). Nothing below is built before that.
- Go 1.27.1, and the corpus at `internal/report/testdata/corpus/gatling/`, including what
  Gatling recorded for each run (`global_stats.json`, `console.txt`).

```bash
go build -o dist/galaxio ./cmd/galaxio
C=internal/report/testdata/corpus/gatling
out=$(mktemp -d)
```

## 1. The whole-run summary, and nothing else (US1, SC-001, SC-002, SC-008, SC-011)

```bash
dist/galaxio report gatling $C/3.13.1
dist/galaxio report gatling $C/3.12.0
cp $C/3.15.1/simulation.log $out/ && dist/galaxio report gatling $out
dist/galaxio report gatling $C/3.13.1 | grep -c -i -w ko
dist/galaxio report gatling $C/3.13.1 | grep -c $'\x1b'
sum() { find "$1" -type f -exec shasum {} + | sort | shasum; }
b=$(sum $C/3.13.1); w=$(mktemp -d); (cd $w && "$OLDPWD"/dist/galaxio report gatling "$OLDPWD"/$C/3.13.1 >/dev/null; ls -A | wc -l); [ "$b" = "$(sum $C/3.13.1)" ] && echo "run directory untouched"
dist/galaxio report gatling $C/3.13.1 --quiet | wc -c
```

Expected: below the run description — whose requests line now reads `102 (84 ok, 18
failed)` — the headline, the response-time table, the four bars and the closing line of
[contracts/cli.md](contracts/cli.md). For 3.13.1: `102 requests · 25.5 req/s`, `✓ 84 ok ·
82.35 % · 21/s`, `✗ 18 failed · 17.65 % · 4.5/s`; min 0/0/0, mean 89/108/1, std 353/387/1,
max 1503/1503/4 for all/ok/failed; bars of 15, 0, 1 and 4 cells for 78 (76.47 %), 0 (0 %),
6 (5.88 %) and 18 (17.65 %) — Gatling's own figures for that run. For 3.12.0 every
non-percentile figure equals `$C/3.12.0/global_stats.json`. The third command, a directory
holding nothing but the log, prints the same summary as the full 3.15.1 directory. `0` lines
name an outcome `ko`; `0` lines carry an escape sequence, because the output is piped; the
empty working directory still holds `0` entries and `run directory untouched` is printed:
the command wrote no file. `0` bytes under `--quiet`. No row for a single request or group
appears anywhere.

## 2. Percentiles that say what they are (US2, SC-005, SC-006)

```bash
dist/galaxio report gatling $C/3.13.1 | grep -E '^(response time|all |times in ms)'
dist/galaxio report gatling $C/3.13.1 --percentiles 99.9,90 | grep -E '^response time'
dist/galaxio report gatling $C/3.13.1 --percentiles 0; echo "exit=$?"
dist/galaxio report gatling $C/3.13.1 --percentiles 50,abc; echo "exit=$?"
grep -n -E 'go-tdigest/pull/42|1427' README.md
grep -rln -E 'percentiles[1-4]' --include='*_test.go' .
```

Expected: the `all` row carries `1427` under `p95` and `1502` under `p99`, and the closing
line says the percentiles are galaxio's t-digest estimates and not Gatling's; the heading
line of the second command carries `p90` then `p99.9` and no other rank; exit 2 with nothing
on standard output for both bad values, the error quoting the value; the README names the
1427/1502 example, the upstream pull request and the rule; the last `grep` lists only the
tests that hold Gatling 3.11.x and 3.12.x percentiles to the rank rule as a reference —
no test compares a Gatling percentile with one this tool prints.

## 3. Bands at other boundaries (US3)

```bash
dist/galaxio report gatling $C/3.13.1 --bounds 5,1000 | grep -E '(ok under|ok [0-9]+ to|and over|  failed)$'
dist/galaxio report gatling $C/3.13.1 --bounds 1200,800; echo "exit=$?"
```

Expected: the labels `ok under 5 ms`, `ok 5 to 1000 ms` and `ok 1000 ms and over`, `failed`
still 18 (17.65 %), and the four shares adding up to 100; exit 2 for boundaries that do not
increase.

## 4. A run of any size (US4, SC-004)

```bash
go test -run 'TestSummaryMemoryDoesNotGrowWithTheLog' -v ./internal/report/
go test -run '^$' -bench 'BenchmarkScan' -benchmem ./internal/report/
```

Expected: the test passes — a 16 MiB and a 256 MiB replay each stay under the 32 MiB goal,
and the larger costs no meaningful amount more. The benchmark's throughput is recorded here,
not gated.

## 5. Failures a script can act on (US5, SC-007)

```bash
mkdir -p $out/cut && head -c 3000 $C/3.13.1/simulation.log > $out/cut/simulation.log
dist/galaxio report gatling $out/cut; echo "exit=$?"
dist/galaxio report gatling $C/3.13.1 -o json; echo "exit=$?"
dist/galaxio report gatling $C/3.13.1 -o stats; echo "exit=$?"
```

Expected: the summary of what the cut log held, then the error saying it ended early, exit
1; exit 2 for `-o json`, listing the known formats, and exit 2 for `-o stats`, naming the
milestone that delivers it — both exactly as in v0.13.0. The span-of-zero, lost-outcome,
no-recorded-end and damaged-log cases need hand-built input and are proved by the unit and
command tests named in tasks.md.

## 6. Progress on a terminal, silence everywhere else (US6, SC-009, SC-010)

```bash
L=$C/3.12.0/simulation.log; n=$(grep -n -m1 '^RUN' $L | cut -d: -f1); mkdir -p $out/big
tail -n +$((n+1)) $L > $out/body; for i in $(seq 256); do cat $out/body; done > $out/chunk
{ head -n $n $L; for i in $(seq 1000); do cat $out/chunk; done; } > $out/big/simulation.log   # about 1 GiB of text log
dist/galaxio report gatling $out/big > $out/tty.out          # in a real terminal: watch standard error
dist/galaxio report gatling $out/big 2> $out/err.txt > $out/pipe.out; wc -c < $out/err.txt; cmp $out/tty.out $out/pipe.out && echo "stdout identical"
TERM=dumb dist/galaxio report gatling $out/big > /dev/null    # in a real terminal: nothing is drawn
dist/galaxio report gatling $out/big --quiet                  # in a real terminal: nothing is drawn
dist/galaxio report gatling $out/big > /dev/null              # in a real terminal: press Ctrl-C half way
dist/galaxio report gatling $C/3.13.1 2>&1 >/dev/null | wc -c
go test -run '^$' -bench 'BenchmarkRunReport' ./cmd/galaxio/
```

Expected: in the terminal, the six-line block of [contracts/cli.md](contracts/cli.md) — a
spinner, the log's name, a bar, a percentage that never passes 100, the time left, the line
saying the percentiles are estimates, and the figures so far for all, ok and failed
requests — first seen within a second, redrawn at
least once a second, never scrolling, and gone when the report appears. `0` bytes on the
redirected standard error and `stdout identical`. Nothing drawn under `TERM=dumb` or
`--quiet`. After Ctrl-C the block is gone, the error stands on a clean line, and the
terminal behaves as before — the cursor is visible and long lines still wrap. `0` bytes of
standard error for a corpus run, which ends before the first draw is due. The benchmark
pair — with and without the block — is recorded here and differs by at most 5 %.

## 7. One hundred identical summaries (SC-003)

```bash
dist/galaxio report gatling $C/3.15.1 > $out/ref.out
for i in $(seq 100); do dist/galaxio report gatling $C/3.15.1 | cmp -s - $out/ref.out || echo "differs at $i"; done; echo done
```

Expected: `done` and no `differs` line.

## 8. Gates

```bash
test -z "$(gofmt -l .)" && go vet ./... && go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1
go mod tidy && git diff --exit-code -- go.mod go.sum
go test -tags=integration -race -count=1 ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Expected: no formatting diff, vet clean, tests green with the race detector, total coverage
at or above 80 %, `go mod tidy` leaving no diff, integration tests green, and `govulncheck`
reporting no vulnerability — run by hand and its result recorded here, because no CI job
runs it.
