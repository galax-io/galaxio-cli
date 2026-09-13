# Data Model: Harden the Linkage Guard

No persistent data. The entities below are the values the two guards and the CI jobs reason about, with the rules that decide each outcome.

## Guarded command (PreToolUse guard)

The JSON payload Claude Code pipes to the hook: `{"tool_input":{"command":"<text>"}}`. Only `tool_input.command` is read.

| Stage | Value | Rule |
|---|---|---|
| raw command | the string as typed | empty → pass; contains no `git` substring → pass; `LINKAGE_OFF` is `1` in the environment, or the text contains `LINKAGE_OFF=1` → pass |
| lines | raw split on newlines | a line matching `<<[-]?['"]?WORD['"]?` opens a heredoc; every following line up to the one whose trimmed text equals WORD is dropped |
| segments | lines split on `&&`, `\|\|`, `;`, `\|` | each segment is judged alone; a verdict on one never carries to another |
| peeled segment | segment with leading `NAME=value` words and wrapper words (`rtk`, `proxy`, `sudo`, `env`, `nohup`, `time`, `command`, `builtin`, `exec`, `xargs`) removed | a segment not starting with `git` after peeling is ignored |
| git segment class | `commit`/`log`/`show` → exempt; `tag` → candidate; `push` → candidate; other → ignored | first candidate that matches a release rule stops the scan |

**Release rules** (the segment is a release when either holds):

- `tag` rule: starts with `git tag`, contains `v?X.Y.Z`, and the first token after `tag` is not one of `-l`, `--list`, `-d`, `--delete`, `-v`, `--verify`, `-n…`.
- `push` rule: starts with `git push` and contains `--tags`, or `refs/tags/`, or a whitespace-delimited `vX.Y.Z` (optionally preceded by `origin`).

**Version**: the first `v?X.Y.Z` in the matching segment. Absent → refuse (message names the manual command).

**Outcome**:

| State | Exit | stderr |
|---|---|---|
| not a release | 0 | none |
| release, checker missing or not executable | 2 | `BLOCKED by linkage-guard:` + `checker missing (<path>) — release cannot be verified` |
| release, no version in the segment | 2 | `BLOCKED …` + how to run `check-linkage.sh --for-tag` by hand |
| release, checker fails | 2 | `BLOCKED …` + the checker's own output (which names the milestone and each failing rule) |
| release, checker passes | 0 | none |

## Pushed reference (git pre-push hook)

One line per ref on stdin: `<local ref> <local sha> <remote ref> <remote sha>`.

| Field | Rule |
|---|---|
| remote ref | judged only when it matches `refs/tags/v[0-9]*`; branches and other tags pass |
| local sha | all zeros means deletion → pass |
| version | `remote ref` with `refs/tags/` removed; handed verbatim to `check-linkage.sh --for-tag` |

Every line is judged; the exit status is 1 if any judged tag failed (checker missing → refused with `pre-push: … cannot be verified`; checker failing → its output then `pre-push: refusing to publish <version> — its milestone is not ready`), otherwise 0. The hook exits 0 immediately when run outside a git work tree.

## Milestone readiness (existing, consumed)

Owned by `scripts/check-linkage.sh --for-tag vX.Y.Z`: maps the version to the milestone whose title starts with `vX.Y.0`, and passes only when every issue in it is closed and every PR merged and carrying the milestone. Exit 0 ready, 1 not ready, 2 cannot determine (no such milestone, no `gh`). Both guards treat 1 and 2 as "refuse".

## Pull-request linkage (existing, consumed by the `linkage` CI job)

`scripts/check-linkage.sh --pr N`: PASS when the PR has a milestone, closes at least one issue, and every closed issue carries the same milestone. Exit 0 pass, 1 fail, 2 PR not found or prerequisites missing.

## Shell test suite

A file matching `scripts/*_test.sh`, `.claude/hooks/*_test.sh` or `.githooks/*_test.sh`; run as `bash <file>`; exit 0 all cases pass, non-zero otherwise; prints one line per case. Each hook suite replaces the checker with a stub in a temporary directory (guard: via `CLAUDE_PROJECT_DIR`; hook: via a throwaway repository whose `scripts/check-linkage.sh` records its arguments and answers from `CHECKER_VERDICT`), so no suite touches the network or real milestones.

## State transitions

```text
agent command ──► guard ──► not a release ──► runs
                        └─► release ──► checker ──► ready ──► runs
                                                └─► not ready / unknown ──► BLOCKED (exit 2)

git push ──► pre-push ──► no vX.Y.Z tag ref ──► pushes
                       └─► tag ref(s) ──► checker per tag ──► all ready ──► pushes
                                                            └─► any not ready ──► refused (exit 1)

pull request ──► linkage job ──► --pr passes ──► check green
                              └─► fails ──► check red until milestone + Closes #N
```
