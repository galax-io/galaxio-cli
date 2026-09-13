# Quickstart: Harden the Linkage Guard

Runnable checks that prove the feature end to end. Prerequisites: bash ≥ 4, `git`, `jq`, `shellcheck` (optional), `gh` authenticated (merge gate only).

## 1. The suites pass

```bash
for t in scripts/*_test.sh .claude/hooks/*_test.sh .githooks/*_test.sh; do echo "--- $t"; bash "$t"; done
```

Expected: `32 passed, 0 failed` for the guard, `12 passed, 0 failed` for pre-push, `PASSED` for the installer; three suites found.

## 2. The old guard fails the new suite (the regression the fix closes)

Before replacing the file, or from `git show main:.claude/hooks/linkage-guard.sh > /tmp/old-guard.sh`:

```bash
GUARD=/tmp/old-guard.sh bash .claude/hooks/linkage-guard_test.sh
```

Expected: `23 passed, 9 failed`, the failures being the quoted-body `gh pr create`, `gh issue create`, `echo`, the `LINKAGE_OFF=1` prefix, both release-branch pushes, the annotated tag, and the two chained `… && git tag` forms. See [research.md](research.md) R2.

## 3. The guard's decision on a command, by hand

```bash
printf '%s' '{"tool_input":{"command":"gh pr create --body \"the guard blocks git tag v1.2.3\""}}' | CLAUDE_PROJECT_DIR=$PWD bash .claude/hooks/linkage-guard.sh; echo "exit=$?"
```

Expected `exit=0`. Replace the command with `git tag v9.9.9` and expect `exit=2` with `BLOCKED by linkage-guard:` on stderr and the checker's report that no milestone starts with `v9.9.0` (contract: [contracts/guard.md](contracts/guard.md)).

## 4. Enable and exercise the pre-push hook

```bash
git config core.hooksPath .githooks
```

Then, in a throwaway clone with a remote you may push to, `git tag v9.9.9 && git push origin v9.9.9`. Expected: the push is refused with `pre-push: refusing to publish v9.9.9 — its milestone is not ready`, and `git tag -d v9.9.9` afterwards cleans up. A branch push in the same clone goes through with no output from the hook (contract: [contracts/pre-push.md](contracts/pre-push.md)).

## 5. Lint the adopted files

```bash
shellcheck -S warning .claude/hooks/linkage-guard.sh .claude/hooks/linkage-guard_test.sh .githooks/pre-push .githooks/pre-push_test.sh
```

Expected: no output.

## 6. Byte-identity with upstream

```bash
for f in .claude/hooks/linkage-guard.sh .claude/hooks/linkage-guard_test.sh .githooks/pre-push .githooks/pre-push_test.sh; do gh api "repos/galax-io/spec-kit-galaxio-bootstrap/contents/$f" --jq .content | base64 -d | diff -q - "$f" && echo "same: $f"; done
```

Expected: four `same:` lines.

## 7. CI, on the pull request

- `shell suites` job: green, listing the three suites.
- `linkage` job: red until the PR has the milestone `v0.11.0 SDD bootstrap` and `Closes #62` in its body, then green after re-run. That first red is Story 5 scenario 1 observed live.
- On a throwaway branch (never on the PR: AGENTS.md forbids add-then-remove churn), push a commit that breaks one guard case (for example, change `2` to `0` on the `lightweight tag` line of the suite), watch `shell suites` go red naming that case, then delete the branch. That is SC-004.
- `docker image` waits for `shell suites`; on `main` the release therefore cannot run past a red suite.

Contract for the jobs: [contracts/ci.md](contracts/ci.md).
