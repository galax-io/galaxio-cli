# Gatling corpus recordings

Copied unchanged from `github.com/galax-io/parsec` **v0.1.0**, `testdata/corpus/gatling/`
(MIT licence; notice below). Each `simulation.log` is exactly what Gatling wrote; a
recording is captured once and is never edited or re-made. The logs, the `lastRun.txt`
marker and the figures Gatling itself recorded for the runs (below) are copied; the HTML
reports, `stats.json` and parsec's own golden files are not.

| Entry | Format | Recorded | Requests (console summary) | Groups | User events | Errors |
|---|---|---|---|---|---|---|
| `3.11.5` | text | 2026-09-03 | 36 (OK 18, KO 18) | 12 | 12 | 6 |
| `3.12.0` | text | 2026-09-03 | 36 (OK 18, KO 18) | 12 | 12 | 6 |
| `3.13.1` | binary | 2026-09-06 | 102 (OK 84, KO 18) | 12 | 12 | 6 |
| `3.14.9` | binary | 2026-09-06 | 102 (OK 84, KO 18) | 12 | 12 | 6 |
| `3.15.1` | binary | 2026-09-06 | 102 (OK 84, KO 18) | 12 | 12 | 6 |
| `lastrun/results` | binary ×3 | 2026-09-09 | 102 each | 12 | 12 | 6 |

`lastrun/results/lastRun.txt` names `corpussimulation-20260909022708912`, which is not
the newest of the three directories, so a reader that honours the marker and one that
picks the newest choose different runs.

The 3.11.5 and 3.12.0 runs were made by an earlier version of parsec's probe simulation
(36 requests); the binary runs by the current one (102 requests). All are the same
simulation otherwise: one request outside any group, the rest under `outer` and
`outer / inner, with comma`; six virtual users; one run-level error per user.

`.gitattributes` marks every `simulation.log`, `global_stats.json`, `console.txt` and
`etalon.tsv` as `-text` so no checkout rewrites the binary logs' bytes, the text logs' line
endings or the recorded figures.

## What Gatling recorded

The figures Gatling computed for a run are what the report summary is tested against
(galaxio-cli#51). Each file is Gatling's own output, copied byte for byte from the same
parsec release, and is read by tests only: `galaxio report` writes no such file.

| File | Run | What it holds |
|---|---|---|
| `3.11.5/global_stats.json` | 3.11.5 | the whole-run figures Gatling wrote beside its HTML report |
| `3.12.0/global_stats.json` | 3.12.0 | the same |
| `3.13.1/js/global_stats.json` | 3.13.1 | the same, in the `js/` directory 3.13.1 wrote it to |
| `3.13.1/console.txt` | 3.13.1 | the standard output of the run, captured while it was recorded, its `Global Information` summary included |
| `3.14.9/console.txt` | 3.14.9 | the same; from 3.13.5 Gatling writes no `global_stats.json` |
| `3.15.1/console.txt` | 3.15.1 | the same |

`stats.json`, which holds the figures of every request and group, is not copied: nothing
reads it before galaxio-cli#52. The percentiles in `3.11.5/global_stats.json` and
`3.12.0/global_stats.json` are the ones this tool's must equal; those 3.13.1, 3.14.9 and
3.15.1 printed come from tdunning/t-digest#230 and are only described (galaxio-cli#51). The
live runs under `../../live/gatling/` have their own record, `RECORDING.md`.

## What the real t-digest gives

`<version>/etalon.tsv` is not Gatling's output: it is what `../../etalon/Etalon.java` printed
for the run's `simulation.log` — every percentile t-digest 3.1's `AVLTreeDigest(100)`, the
digest Gatling 3.11 and 3.12 use, gives over the seeds 1 to 200 of its generator, and what
t-digest 3.3's `MergingDigest(100)` gives, at ranks 50, 75, 95 and 99 for all, ok and failed
requests. `TestEtalonRecordings` reruns it and requires these files byte for byte.

## Licence notice

MIT License  Copyright (c) 2026 Galaxio 

Permission is hereby granted, free of charge, to any person obtaining a copy of this
software and associated documentation files (the "Software"), to deal in the Software
without restriction, including without limitation the rights to use, copy, modify, merge,
publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons
to whom the Software is furnished to do so, subject to the following conditions: the above
copyright notice and this permission notice shall be included in all copies or substantial
portions of the Software. THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND.
