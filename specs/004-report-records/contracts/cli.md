# Contract: `galaxio report`

**Feature**: [spec.md](../spec.md) | **Plan**: [plan.md](../plan.md)

## Synopsis

```text
galaxio report <tool> [PATH] [-o FORMAT[,FORMAT…]] [--quiet] [--verbose] [--no-color]
```

Turn a finished run of the named tool into a stream of records on standard output. Computes
nothing. There is no subcommand. `galaxio report` with no arguments prints help and exits 0,
as in v0.12.0.

## Tool

| `<tool>` | Behaviour |
|---|---|
| `gatling` | the only accepted value in this milestone |
| anything else | usage error, exit 2, listing the accepted tools |
| omitted with a path (`galaxio report <dir>`) | the path is taken as the tool name and rejected the same way |

## Path

| `PATH` | Behaviour |
|---|---|
| omitted | search `target/gatling` (the Maven and sbt results root) relative to the working directory |
| a `simulation.log` file | read that log; nothing is searched |
| a directory holding `simulation.log` | read it as the run, even if it also contains run directories |
| any other directory | a results root: the run named by `lastRun.txt` if that file exists and the run is still there, otherwise the most recently modified run directory |
| a third positional argument | usage error, exit 2 |

## Flags

| Flag | Values | Default | Meaning |
|---|---|---|---|
| `-o`, `--output` | comma-separated list of `stats`, `global_stats`, `yml` | none (record stream) | report format(s) to produce; every known name is reserved in this milestone and rejected with exit 2 naming the milestone that delivers it; an unknown name is rejected listing the known ones |
| `--quiet`, `-q` (root) | — | off | suppress the informational diagnostic naming the chosen run; errors still print |
| `--verbose`, `-v` (root) | — | off | add the detected format, version, and a per-kind count summary on stderr |
| `--no-color` (root) | — | off | no effect on this command's output, which is never coloured |

Reserved report formats: `stats` and `global_stats` (Gatling's legacy `js/stats.json` and
`js/global_stats.json`, milestones v0.14.0 and v0.15.0) and `yml` (the postponed
OpenNFR-style YAML report). There is no `-o json` and no `-o text`: the record stream is
what the command writes when no format is requested (constitution deviation recorded in the
plan).

## Standard output

Only the record stream as JSON Lines, header first, then records in log order. Never a
summary, never a diagnostic, never a partial line.

## Standard error

| When | Line |
|---|---|
| a run is opened | `report: reading <dir> (found by path\|lastRun.txt\|newest)` — suppressed by `--quiet` |
| the log's version is newer than the verified range | `report: warning: <version>: <reason>` (also in the header's `warnings`) — printed even under `--quiet` |
| `--verbose`, at the end | `report: <format> log, Gatling <version>: N requests, N groups, N user events, N errors` |
| any failure | `Error: <message>` (written by the root `execute`) |

## Exit codes

| Code | Meaning | Examples of the message |
|---|---|---|
| 0 | the whole log was read and written | — |
| 1 | runtime failure while executing a valid invocation | `no Gatling run under <dir>`; `cannot read <path>: <reason>`; `not a Gatling simulation.log: <path>`; `version 3.10.0 is below the supported range 3.11.5 through 3.12.0`; `log cut short at byte 1991: 9 trailing bytes could not be decoded; the 62 records emitted are what the run recorded`; `<parsec detail>; the N records already emitted do not form a complete run`; `writing output: <reason>` |
| 2 | usage failure: unsupported tool, third argument, unknown flag, any `-o` value | `unsupported tool "jmeter": accepted tools: gatling`; `report format "stats" is not available yet: it arrives with milestone v0.15.0`; `unknown report format "json": known formats: stats, global_stats, yml`; `accepts at most 2 arg(s), received 3` |

A truncated log yields every record it held **and** exit 1, so a script cannot mistake a
partial run for a complete one. When the reader closes the pipe, the process ends on
`SIGPIPE` as any Unix filter does; no diagnostics are printed for the failed writes.

## Behavioural guarantees

- Streaming: the header is written before the second item is read; memory does not grow
  with the log (goal: < 32 MiB heap in use, ≤ 4 allocations per record).
- Determinism: two runs of the command on the same log are byte-identical.
- Version range: taken from the library at run time (`simlog.Supported()`), never
  hard-coded; the error for an unsupported version quotes that range.
- Format detection: by the log's leading bytes, never by file name.
- Absence: a value the source did not record is omitted; a value the source can never
  record is named in the header's `absent` list.

## Test seam

`runReport(ctx, opts reportOptions) (reportOutput, error)` where `opts` carries
`Tool`, `Path`, `Quiet`, `Verbose`, `Stdout`, `Stderr` (the tool name and the `-o` list are
validated in the cobra layer, where every value is a usage decision); the output carries the run
location, the rule that chose it, per-kind counts and the ending (clean, truncated,
failed). The cobra layer only parses flags and maps the returned error to the exit code.
