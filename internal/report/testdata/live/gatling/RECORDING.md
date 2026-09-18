# Live Gatling recordings

What five live Gatling runs left, one directory a version, for
`TestSummaryMatchesLiveGatlingRuns` and `TestLiveGatlingRunsWereServedTheSameResponses`
(galaxio-cli#51; `specs/005-report-summary/research.md` §17). Each run is galaxio's own
`gatling/scala-sbt` template sending `GET /` at 50 requests a second for four minutes to a
stub that serves every version the same responses. A recording is made once and is never
edited or re-made. `TestReportLiveGatling` makes a new set in the same form, but since
galaxio-cli#117 it serves its runs from `reporttest.Mock`, a service that does real work, and
not from the stub below.

| Version | Log format | sbt ran (UTC, 2026-09-17) | Requests (console summary) |
|---|---|---|---|
| `3.11.5` | text | 11:58:08–12:02:32 | 12 250 (OK 12 010, KO 240) |
| `3.12.0` | text | 12:02:33–12:07:00 | 12 250 (OK 12 010, KO 240) |
| `3.13.1` | binary | 12:07:00–12:11:25 | 12 250 (OK 12 010, KO 240) |
| `3.14.9` | binary | 12:11:25–12:15:49 | 12 250 (OK 12 010, KO 240) |
| `3.15.1` | binary | 12:15:49–12:20:14 | 12 250 (OK 12 010, KO 240) |

## How they were made

One version after another on one machine: macOS 26.6.2, Apple M2 Pro, OpenJDK 17.0.10
(Homebrew), sbt 1.12.13 with launcher 2.0.6.

1. **The project.** `galaxio template init gatling/scala-sbt`, with galaxio built from branch
   `113-report-summary` at 0ee4866 (its `template` command is unchanged from `main`),
   rendered galax-io/templates-gatling from a checkout whose `scala-sbt/` and
   `galaxio-pack.yaml` are those of 564da8bfe54182c653662f86de1350192ddfd142 on its `main` —
   pack 0.15.1, never tagged, template 0.3.1 — with these inputs and every other at its
   default:

   ```text
   Name=live NameWord=live GatlingVersion=<version> GatlingPicatinnyVersion=1.27.0
   SbtGatlingVersion=4.19.1 BaseUrl=http://<stub> Intensity=3000 rpm
   RampDuration=10 seconds StageDuration=4 minutes TestDuration=5 minutes
   StartupBannerEnabled=false
   ```

   `TestReportLiveGatling` renders the published pack instead — 0.16.0 on the day, whose
   `scala-sbt` differs only in input defaults, a comment in `build.sbt` and how the
   picatinny banner these runs turn off is logged — and also sets `SbtVersion=1.12.13` and
   `SbtScalafmtVersion=2.6.1`, the 0.15.1 defaults, so that it builds the same project.

   The template's `Stability` simulation starts users at a rate ramping from 0 to the
   intensity over the ramp, then holds it for the stage; each user sends one `GET /` and
   checks `status is 200`, so a run holds 250 + 12 000 = 12 250 requests.
2. **The stub.** An HTTP server on 127.0.0.1 answering request i, counted from zero in the
   order requests arrive, by `livePlan` in `internal/report/live_integration_test.go` as tagged
   `v0.14.0` (`git show v0.14.0:internal/report/live_integration_test.go`): a
   generator seeded with 20260917 and i picks 85 % waiting 5–40 ms, 8 % 80–400 ms, 3 %
   800–1199 ms, 2 % 1200–2000 ms, and 2 % answering 500 within 5 ms, which the check fails.
3. **The run.** `sbt -batch "Gatling/testOnly org.galaxio.performance.live.Stability"` in the
   project.

## What is kept

| File | What it holds |
|---|---|
| `simulation.log.gz` | the `simulation.log` Gatling wrote, byte for byte, compressed |
| `console.txt` | the Global Information block Gatling printed at the end of the run, byte for byte from the rule that opens it to the rule that closes it (`reporttest.GlobalInformation`); the progress blocks before it and sbt's lines around it, which name paths on the recording machine, are not kept |
| `js/global_stats.json` | the whole-run figures Gatling wrote beside its HTML report, byte for byte; 3.11.5, 3.12.0 and 3.13.1 only, because from 3.13.5 Gatling writes none |
| `etalon.tsv` | not Gatling's output: what `../../etalon/Etalon.java` gives for the log — every percentile t-digest 3.1's `AVLTreeDigest(100)`, Gatling 3.11's digest, gives over the seeds 1 to 200, and t-digest 3.3's `MergingDigest(100)` — written by `TestEtalonRecordings` on 2026-09-17 and required byte for byte when it reruns |

The HTML report, `stats.json` and the rest of the console are not kept: nothing reads them.

## What the tests hold them to

Every non-percentile whole-run figure the summary computes equals what Gatling printed and
wrote. Every percentile it estimates keeps the rank rule of research.md §2 over the run's own
log and equals Gatling 3.11's: a value `etalon.tsv` says Gatling 3.11's digest gives for the
log, and on 3.11.5 and 3.12.0 the value Gatling printed and wrote. Those of 3.13.1, 3.14.9 and
3.15.1 come from the digest defect of
[tdunning/t-digest#230](https://github.com/tdunning/t-digest/issues/230), so the test log
describes them and nothing is asserted on them; it describes `MergingDigest`'s values the
same way (research.md §18).

Every version was served the same responses in the same order of arrival, so every run holds
12 250 requests and fails 240 of them; the response times differ by the milliseconds each
JVM, scheduler and HTTP client adds.
