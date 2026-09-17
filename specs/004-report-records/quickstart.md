# Quickstart: prove `galaxio report` end to end

**Feature**: [spec.md](spec.md) | **Contract**: [contracts/cli.md](contracts/cli.md)

## Prerequisites

Go 1.27.1 and the corpus already in the tree at
`internal/report/testdata/corpus/gatling/` (copied from parsec v0.1.0, MIT, never edited).

```bash
go build -o dist/galaxio ./cmd/galaxio
C=internal/report/testdata/corpus/gatling
```

## 1. Read a named run, both formats (US1, SC-001)

```bash
dist/galaxio report gatling $C/3.12.0
dist/galaxio report gatling $C/3.15.1
dist/galaxio report gatling $C/3.15.1/simulation.log | cmp - <(dist/galaxio report gatling $C/3.15.1) && echo identical
dist/galaxio report gatling $C/3.12.0 --verbose | grep absent
```

Expected: the request counts equal each recording's own Gatling console summary — 36
(18 ok, 18 ko) for the text runs, 102 (84 ok, 18 ko) for the binary ones — the log line
names the detected format, and the log path and its directory produce identical output.

Observed 2026-09-16:

```text
tool        gatling 3.12.0
log         text, …/3.12.0/simulation.log
requests    36 (18 ok, 18 ko)
groups      12 traversals
users       12 events
tool        gatling 3.15.1
requests    102 (84 ok, 18 ko)
log path vs directory: identical
absent      sample scenario, sample response code, sample bytes sent, …
```

## 2. Read the latest run without naming it (US2)

```bash
dist/galaxio report gatling $C/lastrun/results | grep 'found by'
tmp=$(mktemp -d); cp -R $C/lastrun/results $tmp/ && rm $tmp/results/lastRun.txt
dist/galaxio report gatling $tmp/results | grep 'found by'
( cd $tmp && mkdir -p project/target && cp -R results project/target/gatling && cd project && "$OLDPWD/dist/galaxio" report gatling | grep 'found by' )
dist/galaxio report gatling $tmp; echo "exit=$?"
```

Expected: `lastRun.txt`, then `newest` once the marker is gone, then `newest` from the
project directory with no path, then exit 1 naming the directory searched.

Observed 2026-09-16: `found by    lastRun.txt`, then `found by    newest` once the marker is
gone, then `found by    newest` with `requests    102 (84 ok, 18 ko)` from the project
directory with no path, and finally `Error: gatling: no Gatling run under target/gatling`
with exit 1 — an absent default root is an absence of runs, not a path the caller got wrong.

## 3. Read a multi-gigabyte log (US3, SC-003)

```bash
go test ./internal/report -run '^$' -bench BenchmarkScan -benchtime=1x -benchmem
go test -tags=integration -race -run TestReportIntegrationReadsALargeRun ./cmd/galaxio
```

## 4. Failures name what was at fault (US4, SC-004)

```bash
html=$(mktemp -d); printf '<!DOCTYPE html>\n' > $html/simulation.log
dist/galaxio report gatling $html; echo "exit=$?"                    # not a Gatling simulation.log; 1
dist/galaxio report gatling /nonexistent; echo "exit=$?"             # cannot read /nonexistent; 1
cut=$(mktemp -d); head -c 2000 $C/3.15.1/simulation.log > $cut/simulation.log
dist/galaxio report gatling $cut; echo "exit=$?"                     # the counts it held, then cut short; 1
dist/galaxio report jmeter $C/3.15.1; echo "exit=$?"                 # unsupported tool; 2
dist/galaxio report $C/3.15.1; echo "exit=$?"                        # the path read as a tool name; 2
dist/galaxio report gatling $C/3.15.1 extra; echo "exit=$?"          # accepts at most 2 args; 2
dist/galaxio report gatling -o stats $C/3.15.1; echo "exit=$?"       # reserved for v0.15.0; 2
dist/galaxio report gatling -o json $C/3.15.1; echo "exit=$?"        # unknown report format; 2
dist/galaxio report; echo "exit=$?"                                  # help; 0
```

Observed 2026-09-16 (bash; zsh has no `PIPESTATUS`):

```text
Error: <tmp>/h/simulation.log: gatling: not a Gatling simulation.log: found "<!DOCTYPE " at the start of the stream   # 1
Error: cannot read /nonexistent: no such file or directory                                                            # 1
Error: cannot read <tmp>/dl: no such file or directory                                                                # 1, a dangling symlink, not an empty root
requests    57 (57 ok, 0 ko)                                                                                          # the counts the cut log held
Error: <tmp>/c/simulation.log: gatling: byte 1991: the log is cut short: …; the run was not read to the end           # 1
Error: no path given: pass a run directory, a simulation.log or a results root, …                                     # 2, an argument given but empty
Error: report format "stats" is not available yet: it arrives with milestone v0.15.0 Legacy stats.json                # 2, with no tool given at all
Error: unsupported tool "jmeter": accepted tools: gatling                                                             # 2
help printed                                                                                                          # 0
```

A path whose own name contains the words an earlier release matched on — `.../unknown
command/x` — exits 1 like any other missing path, because the code now comes from the
error's type.

Version-gate cases (below range, 3.13.0, newer-than-range warning) are synthetic and live in
the package tests:

```bash
go test ./internal/report -run 'TestOpen' -v
```

## 5. Help is unchanged

```bash
dist/galaxio report --help | grep -E 'report <tool> \[PATH\]|-o, --output'
```

Observed 2026-09-16:

```text
  galaxio report <tool> [PATH] [flags]
  -o, --output formats   report format(s) to produce, comma-separated: stats, global_stats, yml (reserved for later releases)
```

## 6. Gates

```bash
test -z "$(gofmt -l .)" && go vet ./... && go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1
go mod tidy && git diff --exit-code -- go.mod go.sum
go mod verify
go test -tags=integration -race -count=1 ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Expected: no formatting diff, vet clean, all tests pass with the race detector, total
coverage at or above 80%, tidy leaves no diff, integration suite passes, no known
vulnerabilities.

## 7. Milestone and issue

```bash
gh api repos/galax-io/galaxio-cli/milestones/3 --jq '.title + " — " + .description'
gh issue view 50 --repo galax-io/galaxio-cli --json title,milestone --jq '.title + " [" + .milestone.title + "]"'
```

Observed 2026-09-16, after the record stream was withdrawn:

```text
v0.13.0 Read a run — S2 — A finished Gatling run is a directory of HTML and a log only the
matching Gatling version can read, and from 3.13.5 on Gatling exports nothing at all; this
CLI cannot open one. Reading a run is the first result a user can see from the whole
initiative, and what every later report milestone stands on.
A finished Gatling run cannot be read at all [v0.13.0 Read a run]
```

Issue #50 carries an `## Amended` section recording the withdrawal, what the milestone now
delivers, and the restated acceptance criteria.

## 8. Documentation

`README.md` § Reporting Ecosystem documents the command, its arguments, the reserved `-o`
formats and the exit codes. `galaxio report --help` shows `<tool> [PATH]` and `-o`.
