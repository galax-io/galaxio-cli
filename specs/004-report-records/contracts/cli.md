# Contract: `galaxio report`

**Feature**: [spec.md](../spec.md) | **Plan**: [plan.md](../plan.md)

## Synopsis

```text
galaxio report <tool> [PATH] [-o FORMAT[,FORMAT…]] [--quiet | --verbose] [--no-color]
```

Read a finished run of the named tool and report what it holds. Computes nothing and writes
no data format. `galaxio report` with no arguments prints help and exits 0, as in v0.12.0.

## Tool

| `<tool>` | Behaviour |
|---|---|
| `gatling` | the only accepted value in this milestone |
| anything else | usage error, exit 2, listing the accepted tools |
| omitted with a path (`galaxio report <dir>`) | the path is taken as the tool name and rejected the same way |

## Path

| `PATH` | Behaviour |
|---|---|
| omitted | search `target/gatling`, the Maven and sbt results root, relative to the working directory |
| a `simulation.log` file | read that log; nothing is searched |
| a directory holding `simulation.log` | read it as the run, even if it also contains run directories |
| any other directory | a results root: the run named by `lastRun.txt` if that file exists and the run is still there, otherwise the most recently modified run directory |
| a path that does not exist | runtime error, exit 1, naming it as a path that cannot be read — never as a directory holding no run |
| a file that is not a `simulation.log` | runtime error, exit 1, saying the path is not a Gatling run; `no run under <file>` would send the reader to look inside a file |
| given but empty (`report gatling ""`) | usage error, exit 2: the empty string is an unset variable, not a request to search the default |
| a third positional argument | usage error, exit 2 |

## Flags

| Flag | Values | Default | Meaning |
|---|---|---|---|
| `-o`, `--output` | comma-separated `stats`, `global_stats`, `yml` | none | report format(s) to produce. Checked as the flag is parsed, which is before everything else the command does — including any help, since `--help` is answered after flags and a check stated later would accept the value and discard it. Every name in the list is checked. An unknown name is rejected first, listing the known ones, because a typo outranks a milestone; otherwise the first name is rejected naming the milestone that delivers it. An empty or blank value names no format and is rejected as unknown |
| `--quiet`, `-q` (root) | — | off | suppress the report; errors and the unverified-version warning still print |
| `--verbose`, `-v` (root) | — | off | add what the source can never record |
| `--no-color` (root) | — | off | no effect; this command's output is never coloured |

There is no `-o json` and no `-o text`. The first machine-readable output this command will
publish is `stats.json`, identical to Gatling's own, in the milestone that owns it.

## Standard output

An aligned key-value block, one fact per line, in this order. Absent facts are omitted
rather than shown empty.

```text
run         io.galaxio.parsec.corpus.CorpusSimulation
id          io.galaxio.parsec.corpus.CorpusSimulation
started     2026-09-06T04:48:14.356Z
tool        gatling 3.15.1
log         binary, internal/report/testdata/corpus/gatling/3.15.1/simulation.log
found by    path
span        2026-09-06T04:48:14.885Z .. 2026-09-06T04:48:18.117Z (3.232s)
requests    102 (84 ok, 18 ko)
groups      12 traversals
users       12 events
errors      6
assertions  10 payloads the run declared
```

- `started` is the run's own recorded start; `span` is the library's bounds over the run,
  which is a different quantity and is omitted when the library cannot bound the run.
- `requests` splits by the outcome the source recorded. A request whose outcome the source
  lost is counted in the total and in neither `ok` nor `ko`, and is then named on its own
  line as `unknown`.
- `assertions` counts the opaque payloads the run declared, wherever the source put them:
  ahead of the events, where the library lands them on the run description, or among the
  events, where it yields them in the stream. Both are the run's, so both are counted. The
  line is omitted when the run declared none.
- `other` counts records of a kind this release does not know, so a later library version
  cannot make the command under-report a log in silence. It is omitted when there are none.
- `run`, `id` and every other value taken from the log are quoted when they are not
  printable, so a log cannot colour the output or erase the line it is on.
- `absent` is added under `--verbose`, listing what this source can never record, in the
  library's own words.
- `--quiet` suppresses the whole block.

## Standard error

| When | Line |
|---|---|
| the log's version is newer than the verified range | `report: warning: <version>: <reason>` — printed even under `--quiet` |
| any failure | `Error: <message>`, written by the root command |

## Exit codes

| Code | Meaning | Examples of the message |
|---|---|---|
| 0 | the run was read to the end | — |
| 1 | runtime failure while executing a valid invocation | `gatling: no Gatling run under <dir>`; `cannot read <path>: <reason>` for a path the caller named, on the cleaned spelling the library searched; `<path> is not a Gatling run: pass a simulation.log, a run directory, or a results root` for a path that is there but is a file; `gatling: reading results root <dir>: <reason>` for one it could not read; `<path>: gatling: not a Gatling simulation.log: found "…" at the start of the stream`; `<path>: gatling: version 3.10.0 is below the supported range 3.11.5 through 3.12.0`; `<path>: gatling: byte 1991: the log is cut short: …; the run was not read to the end`; `<path>: … ; the run was not read completely`; `writing report: <reason>`, joined with the scan's own error when both happen |
| 2 | usage failure | `unsupported tool "jmeter": accepted tools: gatling`; `invalid argument "stats" for "-o, --output" flag: report format "stats" is not available yet: it arrives with milestone v0.15.0 Legacy stats.json`; `invalid argument "json" for "-o, --output" flag: unknown report format "json": known formats: stats, global_stats, yml` — the flag names itself because the value is refused while it is parsed; `no path given: pass a run directory, a simulation.log or a results root, or omit the argument to search target/gatling`; `accepts at most 2 arg(s), received 3`; `unknown command "extra" for "galaxio completion bash"`, for the commands cobra writes as much as the ones here; `--verbose and --quiet cannot be used together`, which the root command rejects before this one runs |

A truncated log reports the counts of what was read **and** exits 1, so a script cannot
mistake a partial run for a complete one: those records are what the run recorded before it
was killed.

A **damaged** log is different and reports nothing. The library states that the records
before the damage are not a result and that no total may be derived from them, so the
command prints no counts and exits 1 rather than standing behind a number it cannot.

A run killed **exactly on a record boundary** is not detectable at all: neither log format
carries an end marker, so the library ends such a log with a clean end-of-file and the
command reports it as complete. The command does not claim to catch every killed run.

## Behavioural guarantees

- One forward pass; no record is retained; memory does not grow with the log. The goal is
  under 32 MiB of heap in use for a log of any size, and it is enforced by a test the
  ordinary suite runs, not by a benchmark nothing runs.
- Determinism: two reads of the same run produce identical output.
- Version range taken from the library at run time, never hard-coded; the error for an
  unsupported version quotes that range.
- Format detection by the log's leading bytes, never by file name.
- Absence is the source's own statement, reported as absent and never as a zero.

## Test seam

`runReport(ctx, opts reportOptions) (reportOutput, error)` where `opts` carries `Tool`,
`Path` with the `PathSet` that says whether an argument was given at all, `Quiet`,
`Verbose`, `Stdout`, `Stderr`; the output carries the run location, the run description
parsec returned, the log format detected for the report, and the tally. Every rule a test
must pin lives behind this call, including the refusal of a path that is present and empty
— which is why `PathSet` is there, since the empty string alone cannot say whether an
argument was given.

`-o` is not among them, and deliberately: every name it accepts is reserved, so the flag
refuses the value as it parses it and no format ever reaches the work. `ctx` is the one the
CLI runs under; cancelling it stops the read where it is.
