# Contract: PreToolUse linkage guard

**File**: `.claude/hooks/linkage-guard.sh` (byte-identical to `galax-io/spec-kit-galaxio-bootstrap`). **Wired by**: `.claude/settings.json`, `hooks.PreToolUse[matcher=Bash]`. **Suite**: `.claude/hooks/linkage-guard_test.sh`, 32 cases, is the executable form of this contract; a change to either file is a change to both.

## Input

- stdin: Claude Code's PreToolUse JSON; only `.tool_input.command` is read.
- env: `CLAUDE_PROJECT_DIR` (root that holds `scripts/check-linkage.sh`; falls back to `git rev-parse --show-toplevel`, then `pwd`); `LINKAGE_OFF=1` disables the guard.

## Decision table

Judged per segment after heredoc bodies are dropped and wrappers peeled (see [data-model.md](../data-model.md)).

| Command shape | Gated |
|---|---|
| any text with no `git` substring | no |
| `gh pr create …` / `gh issue create …` / `echo …` whose text mentions `git tag vX.Y.Z` | no |
| a heredoc body containing a tag command | no |
| `git status`, `git branch`, `git log …vX.Y.Z…`, `git show vX.Y.Z` | no |
| `git commit -m "… git tag v1.0.7 …"` | no |
| `git tag -l …`, `git tag --list …`, `git tag -d …`, `git tag -v …`, `git tag -n…` | no |
| `git push origin feat/x`, `git push origin release/1.0`, `git push origin origin/main:refs/heads/release/0.0.0` | no |
| `LINKAGE_OFF=1 git tag v1.0.7` | no |
| `git tag v1.0.7`, `git tag -a v1.0.7 -m …`, `git tag -m … -a v1.0.7` | yes |
| `git push origin v1.0.7`, `git push --tags`, `git push origin refs/tags/v1.0.7` | yes |
| `rtk proxy git tag v1.0.7`, `sudo git tag v1.0.7`, `xargs git push --tags`, `GIT_SSH_COMMAND=ssh git push origin v1.0.7` | yes |
| `git commit -m prep && git tag v1.0.7`, `git log -1 && git tag v1.0.7`, `cd /tmp && git tag v1.0.7`, heredoc then `git tag v1.0.7` on the next line | yes |

## Output

| Result | Exit | stdout | stderr |
|---|---|---|---|
| not gated | 0 | empty | empty |
| gated and milestone ready | 0 | empty | empty |
| gated, refused | 2 | empty | `BLOCKED by linkage-guard:` then the reason: checker missing, no explicit `vX.Y.Z`, or the checker's own findings |

The checker is invoked as `scripts/check-linkage.sh --for-tag <version>` with the version taken from the gated segment (the suite asserts `--for-tag v1.0.7` for `git tag v1.0.7`).

## Non-goals

Not an authority: it sees only commands an agent runs; the pre-push hook and the merge gate cover the rest. Never blocks `commit`, `log`, `show`, plain pushes, or non-git commands, and never suggests rephrasing.
