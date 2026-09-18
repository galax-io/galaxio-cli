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


Observed 2026-09-17 at 72a8d25, `dist/galaxio` built from it, Go 1.27.1, darwin/arm64 —
the summary part of the first two commands, then the rest:

```text
102 requests · 25.5 req/s      ✓ 84 ok · 82.35 % · 21/s      ✗ 18 failed · 17.65 % · 4.5/s

response time, ms       min   mean    std    p50    p75    p95    p99    max
all                       0     89    353      1      1   1427   1502   1503
✓ ok                      0    108    387      1      1   1502   1502   1503
✗ failed                  0      1      1      1      2      4      4      4

███████████████░░░░░  76.47 %    78  ok under 800 ms
░░░░░░░░░░░░░░░░░░░░      0 %     0  ok 800 to 1200 ms
█░░░░░░░░░░░░░░░░░░░   5.88 %     6  ok 1200 ms and over
████░░░░░░░░░░░░░░░░  17.65 %    18  failed

times in ms · percentiles are galaxio's t-digest estimates, interpolated

36 requests · 9 req/s      ✓ 18 ok · 50 % · 4.5/s      ✗ 18 failed · 50 % · 4.5/s

response time, ms       min   mean    std    p50    p75    p95    p99    max
all                       0    252    559      2      5   1503   1504   1504
✓ ok                      0    503    707      5   1502   1504   1504   1504
✗ failed                  0      1      1      1      2      2      3      3

███████░░░░░░░░░░░░░  33.33 %    12  ok under 800 ms
░░░░░░░░░░░░░░░░░░░░      0 %     0  ok 800 to 1200 ms
███░░░░░░░░░░░░░░░░░  16.67 %     6  ok 1200 ms and over
██████████░░░░░░░░░░     50 %    18  failed

times in ms · percentiles are galaxio's t-digest estimates, interpolated

bare 3.15.1 directory: same summary
ko lines: 0
escape lines: 0
working directory entries: 0
run directory untouched
quiet bytes: 0
```

The 3.13.1 description reads `requests    102 (84 ok, 18 failed)`, and its whole output is
the block of [contracts/cli.md](contracts/cli.md) byte for byte. Every 3.12.0 figure equals
`global_stats.json`: count 36/18/18, min 0/0/0, max 1504/1504/3, mean 252/503/1, std
559/707/1, rate 9/4.5/4.5, bands 12/0/6/18 at 33.33/0/16.67/50 % — and so do its
percentiles, 2/5/1503/1504, 5/1502/1504/1504 and 1/2/2/3, which are Gatling 3.12.0's.

## 2. Percentiles that say what they are (US2, SC-005, SC-006)

```bash
dist/galaxio report gatling $C/3.13.1 | grep -E '^(response time|all |times in ms)'
dist/galaxio report gatling $C/3.13.1 --percentiles 99.9,90 | grep -E '^response time'
dist/galaxio report gatling $C/3.13.1 --percentiles 0; echo "exit=$?"
dist/galaxio report gatling $C/3.13.1 --percentiles 50,abc; echo "exit=$?"
grep -n -E 'go-tdigest/pull/42|1427' README.md
go test -run 'TestPercentilesEqualGatling311|TestSummaryMatchesLiveGatlingRuns' -v ./internal/report/
```

Expected: the `all` row carries `1427` under `p95` and `1502` under `p99`, and the closing
line says the percentiles are galaxio's t-digest estimates, interpolated; the heading
line of the second command carries `p90` then `p99.9` and no other rank; exit 2 with nothing
on standard output for both bad values, the error quoting the value; the README names the
1427/1502 example, the upstream pull request and the rule; the last command passes: every
percentile of the ten recordings is a value Gatling 3.11's digest gives for the log and
equals what 3.11.5 and 3.12.0 printed, and its log only describes what 3.13.1, 3.14.9 and
3.15.1 printed and what `MergingDigest` gives (research.md §18).


Observed 2026-09-17 at 72a8d25:

```text
response time, ms       min   mean    std    p50    p75    p95    p99    max
all                       0     89    353      1      1   1427   1502   1503
times in ms · percentiles are galaxio's t-digest estimates, interpolated
response time, ms       min   mean    std    p90  p99.9    max
Error: invalid argument "0" for "--percentiles" flag: percentile rank "0" is not a number above 0 and at most 100
exit=2
Error: invalid argument "50,abc" for "--percentiles" flag: percentile rank "abc" is not a number above 0 and at most 100
exit=2
--- PASS: TestPercentilesEqualGatling311 (0.00s)
--- PASS: TestSummaryMatchesLiveGatlingRuns (0.00s)
ok  	github.com/galax-io/galaxio-cli/internal/report	0.507s
```

Nothing reached standard output for either bad value. The README names the 1427/1502
example, caio/go-tdigest#42 and the rank rule; the walk found the rule missing from it, and
T025 adds the sentence. The test log holds "Gatling printed" notes only under 3.13.1
(12), 3.14.9 (6) and 3.15.1 (6) — none under 3.11.5 or 3.12.0, whose every percentile equals
this tool's — and 39 notes on `MergingDigest`.

## 3. Bands at other boundaries (US3)

```bash
dist/galaxio report gatling $C/3.13.1 --bounds 5,1000 | grep -E '(ok under|ok [0-9]+ to|and over|  failed)'
dist/galaxio report gatling $C/3.13.1 --bounds 1200,800; echo "exit=$?"
```

Expected: the labels `ok under 5 ms`, `ok 5 to 1000 ms` and `ok 1000 ms and over`, `failed`
still 18 (17.65 %), and the four shares adding up to 100; exit 2 for boundaries that do not
increase.


Observed 2026-09-17 at 72a8d25 (the `grep` above had a `$` that kept only the lines ending
in `and over` and `failed`; the walk dropped it):

```text
██████████████░░░░░░  71.57 %    73  ok under 5 ms
█░░░░░░░░░░░░░░░░░░░    4.9 %     5  ok 5 to 1000 ms
█░░░░░░░░░░░░░░░░░░░   5.88 %     6  ok 1000 ms and over
████░░░░░░░░░░░░░░░░  17.65 %    18  failed
Error: invalid argument "1200,800" for "--bounds" flag: boundaries are two whole, non-negative numbers of milliseconds, the second greater than the first
exit=2
```

71.57 + 4.9 + 5.88 + 17.65 = 100 and 73 + 5 + 6 + 18 = 102.

## 4. A run of any size (US4, SC-004)

```bash
go test -run 'TestSummaryMemoryDoesNotGrowWithTheLog' -v ./internal/report/
go test -run '^$' -bench 'BenchmarkScan' -benchmem ./internal/report/
```

Expected: the test passes — a 16 MiB and a 256 MiB replay each stay under the 32 MiB goal,
and the larger costs no meaningful amount more. The benchmark's throughput is recorded here,
not gated.


Observed 2026-09-17 at 72a8d25, on an Apple M2 Pro with nothing else running:

```text
scan_test.go:356: heap in use: 1.07 MiB over 16 MiB, 1.07 MiB over 256 MiB
--- PASS: TestSummaryMemoryDoesNotGrowWithTheLog (2.18s)
BenchmarkScan/size=64MiB-12     3    472958653 ns/op   141.88 MB/s   1983941 records/s   1107712 B/op   68 allocs/op
BenchmarkScan/size=2048MiB-12   1  15880613958 ns/op   135.23 MB/s   1890881 records/s   1107712 B/op   68 allocs/op
```

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


Observed 2026-09-17 at 72a8d25:

```text
$ dist/galaxio report gatling $out/cut
(the description and the summary of the 84 requests the cut log held: 22 lines)
Error: $out/cut/simulation.log: gatling: byte 2961: the log is cut short: 39 trailing bytes could not be decoded, and a request message was still to come; the run was not read to the end
exit=1
Error: invalid argument "json" for "-o, --output" flag: unknown report format "json": known formats: stats, global_stats, yml
exit=2
Error: invalid argument "stats" for "-o, --output" flag: report format "stats" is not available yet: it arrives with milestone v0.15.0 Legacy stats.json
exit=2
```

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
dist/galaxio report gatling $C/3.13.1 2> $out/corpus.err > /dev/null; wc -c < $out/corpus.err
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


Observed 2026-09-17 at 72a8d25, on the 1 208 320 195-byte log. The terminal steps were run
with standard error attached to a pseudo-terminal (`pty.openpty`, `TERM=xterm-256color`),
recording every byte and when it arrived, rather than watched by eye:

```text
terminal:   exit 0 in 9.2 s; 44 draws, the first 0.54 s in, at most 0.20 s apart;
            43 redraws each opening with ESC[6A ESC[2K; percentages 0..99, never above 100;
            time left shown from the second draw; ESC[6A ESC[J last; no ESC[? sequence
            first frame:
            ⠋ reading simulation.log  ━─────────────────────────────    5 %
            figures so far · times in ms · percentiles are t-digest estimates, interpolated
                                   count   share    min   mean    p50    p95    p99    max
            all requests            539k              0    252      2   1504   1504   1504
              ✓ ok                  270k  50.0 %      0    503      5   1504   1504   1504
              ✗ failed              269k  50.0 %      0      1      1      3      3      3
TERM=dumb:  exit 0, 0 bytes on standard error
--quiet:    exit 0, 0 bytes on standard error, 0 on standard output
Ctrl-C 3 s: exit 1; 13 draws, then ESC[6A ESC[J and, at the start of a line,
            "Error: $out/big/simulation.log: context canceled"; no ESC[? sequence
redirected: 0 bytes on standard error; stdout identical
corpus run: 0 bytes on standard error
BenchmarkRunReport/progress=off-12          3   478351986 ns/op   140.28 MB/s
BenchmarkRunReport/progress=on-12           3   481712486 ns/op   139.30 MB/s
BenchmarkRunReport/progress=every-tick-12   2   751918334 ns/op    89.24 MB/s
```

The pair differs by 0.7 %: a 64 MiB read ends before the first draw is due and pays for
reading the clock. Drawing at every tick, the upper bound, costs about 0.3 ms a draw, so five
draws a second cost about 0.15 % of a read. The corpus command is now written with standard
error to a file: under zsh, `2>&1 >/dev/null | wc -c` counted standard output as well, 1342
bytes, because zsh copies a redirected stream to every target. The report of the big log
puts its headline on three lines, as it is past 100 columns:

```text
9216000 requests · 2304000 req/s
✓ 4608000 ok · 50 % · 1152000/s
✗ 4608000 failed · 50 % · 1152000/s
```

## 7. One hundred identical summaries (SC-003)

```bash
dist/galaxio report gatling $C/3.15.1 > $out/ref.out
for i in $(seq 100); do dist/galaxio report gatling $C/3.15.1 | cmp -s - $out/ref.out || echo "differs at $i"; done; echo done
```

Expected: `done` and no `differs` line.


Observed 2026-09-17 at 72a8d25: `done`, and no `differs` line.

## 8. Live Gatling runs (SC-012)

```bash
rec=$(mktemp -d)
GALAXIO_LIVE_GATLING=1 GALAXIO_LIVE_RECORD=$rec go test -tags=integration -run TestReportLiveGatling -timeout 60m -v ./internal/report/
GALAXIO_LIVE_RECORDINGS=$rec go test -run 'TestSummaryMatchesLiveGatlingRuns|TestLiveGatlingRunsWereServedTheSameResponses' -v ./internal/report/
go test -run 'TestSummaryMatchesLiveGatlingRuns|TestLiveGatlingRunsWereServedTheSameResponses' -v ./internal/report/
go test -tags=integration -run TestEtalonRecordings -v ./internal/report/
```

Expected: the first command takes about five minutes a version and passes for 3.11.5, 3.12.0,
3.13.1, 3.14.9 and 3.15.1 — the template rendered from the published registry, every
non-percentile figure equal to what Gatling printed, every percentile within the rank rule and
a value Gatling 3.11's digest gives for the fresh log, run by the etalon, and on 3.11.5 and
3.12.0 equal to what Gatling printed or another value its generator draws; the log describes
what the later versions printed and what `MergingDigest` gives. It needs the t-digest 3.1 and
3.3 jars in `GALAXIO_TDIGEST_JARS`, the local Maven repository or the Coursier cache, and
skips before running Gatling without them. `GALAXIO_LIVE_VERSIONS=3.11.5` runs one
version, `GALAXIO_LIVE_STEADY=20s` shortens the steady stage, and
`GALAXIO_LIVE_TEMPLATES=local:<checkout>` renders a local templates-gatling checkout instead.
The second command holds the fresh recordings to the same, and logs the same requests, ok and
failed for every version. The third does it on the committed recordings, without a JDK: 12 250
requests (12 010 ok, 240 failed) a version, and the figures of research.md §17 and §18. The
fourth reruns the etalon over the ten recordings and passes when every `etalon.tsv` is
reproduced byte for byte; `GALAXIO_ETALON_RECORD=1` rewrites them.

Observed 2026-09-17. The first command ran from 13:47 to 14:10 UTC at 72a8d25, rendering the
published pack 0.16.0, and drove Gatling 3.11.5, 3.12.0, 3.13.1, 3.14.9 and 3.15.1 through
12 250 requests each. Every non-percentile figure matched on all five, and four versions
passed. 3.11.5 failed on one value: the 75th percentile of failed requests was 5, where
Gatling printed 6 and t-digest 3.1 gives 6 for every seed. That is the order of equal
centroids of research.md §18; T024 and T025 then admitted it and described it. The first
command was not run again. The second command replays those same fresh recordings through
the same comparison: at 72a8d25 it failed the same way, and at 0f8340a it passes, with the
difference described in 4 notes and 12 250 requests (12 010 ok, 240 failed) logged for every
version. The third command passes on the committed recordings, and the fourth reproduces all
ten `etalon.tsv` byte for byte.

## 9. Gates

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

Observed 2026-09-18 at the branch's last commit that changes code, Go 1.27.1, darwin/arm64:

```text
gofmt:        no diff
vet:          clean
vet (tags):   clean
tests:        10 packages ok (race on)
coverage:     87.9% total, floor 80.0%
report pkgs:  internal/report 97.6%, reporttest 88.9%, cmd/galaxio 90.5%
mod tidy:     no diff
mod verify:   all modules verified
integration:  10 packages ok (race on), TestEtalonRecordings run with a JDK, not skipped
shell suites: 8 ok
govulncheck:  No vulnerabilities found. (govulncheck v1.8.0, DB updated 2026-09-16)
binary:       18.4 MiB, 0.10 MiB over main; no testdata byte in it
```

The binary was checked for test data, since this milestone adds 0.9 MiB of recordings:
`go list -deps ./cmd/galaxio` links neither `reporttest` nor any test file, the only
`go:embed` in the repository is the Scala templates, and the first 64 bytes of a corpus
`global_stats.json`, a console, an `etalon.tsv` and a golden appear nowhere in a
`-trimpath -ldflags="-s -w"` build. Go excludes a directory named `testdata` from a build,
and the 0.10 MiB the binary gains over `main` is the digest library.

Every commit on the branch that changes code was exported on its own with `git archive`,
built, and tested with `go test -count=1 ./...`; all of them pass.
