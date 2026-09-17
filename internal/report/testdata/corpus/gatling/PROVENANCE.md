# Gatling corpus recordings

Copied unchanged from `github.com/galax-io/parsec` **v0.1.0**, `testdata/corpus/gatling/`
(MIT licence; notice below). Each `simulation.log` is exactly what Gatling wrote; a
recording is captured once and is never edited or re-made. Only the logs and the
`lastRun.txt` marker are copied; the HTML reports, console captures and parsec's own
golden files are not.

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

`.gitattributes` marks every `simulation.log` as `-text` so no checkout rewrites the
binary logs' bytes or the text logs' line endings.

## Licence notice

MIT License  Copyright (c) 2026 Galaxio 

Permission is hereby granted, free of charge, to any person obtaining a copy of this
software and associated documentation files (the "Software"), to deal in the Software
without restriction, including without limitation the rights to use, copy, modify, merge,
publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons
to whom the Software is furnished to do so, subject to the following conditions: the above
copyright notice and this permission notice shall be included in all copies or substantial
portions of the Software. THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND.
