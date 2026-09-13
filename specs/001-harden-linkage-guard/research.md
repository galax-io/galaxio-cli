# Research: Harden the Linkage Guard

All findings verified on 2026-09-13 against the live repositories; commands to reproduce are in [quickstart.md](quickstart.md).

## R1. Source of the guard and hook

**Decision**: Copy four files byte-for-byte from `galax-io/spec-kit-galaxio-bootstrap` at commit `88978a1323113468b289dff5c06fea68d1b3ced5` (the tree that contains merged PR #6, 2026-09-06): `.claude/hooks/linkage-guard.sh`, `.claude/hooks/linkage-guard_test.sh`, `.githooks/pre-push`, `.githooks/pre-push_test.sh`. Record this commit in the `fix(hooks)` commit message.

**Rationale**: The bootstrap's own AGENTS.md declares these files "the verbatim sources", copied "byte-for-byte — never re-implemented", and keeps three internal copies identical. `galax-io/parsec#51` adopted them the same way and its CI has been green since. A local fork would be replaced silently by the next `copier update` and would lose the 32-case contract the copies share. FR-017 in the spec requires this.

**Alternatives considered**:
- *Run `copier update` to pull the template forward.* Rejected for this issue: `.copier-answers.yml` is at template commit `d8b7cd1` (pre-#6), and an update re-renders `AGENTS.md`, `CLAUDE.md` and `.gitignore` from Jinja, which is a broader change than #62 and collides with the constitution's noted disagreement about the AGENTS.md release section. A separate chore can do it later; the files will then already match.
- *Rewrite the guard locally with the same behaviour.* Rejected: three repositories already converged on this shape by being wrong repeatedly; a fourth implementation restarts that process.

**Known upstream nit, left as-is**: the guard suite's header comment says `Usage: bash scripts/test-linkage-guard.sh`, a stale path from before the suite moved beside the hook. Byte-identity wins; the nit is worth an upstream issue, not a local edit.

## R2. Regression evidence and hygiene of the adopted files

**Finding**: Run locally with bash 5.3 and jq 1.7:

| Suite | Against | Result |
|---|---|---|
| `linkage-guard_test.sh` (32) | upstream guard | 32 passed, 0 failed |
| `linkage-guard_test.sh` (32) | **current** `.claude/hooks/linkage-guard.sh` | 23 passed, **9 failed** |
| `pre-push_test.sh` (12) | upstream hook | 12 passed, 0 failed |
| `scripts/test_install.sh` | current `install.sh` | passed |
| `shellcheck -S warning` on the four files | | clean |

The nine failures on the current guard are exactly the issue's complaints: four false blocks (`gh pr create` with a quoted body mentioning a tag, `gh issue create` quoting a tag command, `echo "git tag v9.9.9"`, the `LINKAGE_OFF=1` prefix), two release-branch pushes wrongly gated, and three misses (`git tag -a v1.0.7 -m …`, `git commit … && git tag`, `git log … && git tag`). This satisfies Principle III's "fails without the fix" for a bug fix, and it is the corpus SC-001 and SC-002 are measured against.

## R3. Suite discovery in CI and the installer suite's name

**Decision**: The CI step globs `scripts/*_test.sh .claude/hooks/*_test.sh .githooks/*_test.sh`, refuses to run if fewer than 3 suites match, and runs each with `bash "$t"` under `set -euo pipefail`. Rename `scripts/test_install.sh` to `scripts/install_test.sh` so it matches.

**Rationale**: The constitution names the three directories and the `*_test.sh` convention; parsec's `verify.yml` uses the same glob with a floor, and the floor is what stops a glob typo passing on an empty match. The installer suite is the only existing suite and it would be the one the glob misses; renaming it is one `git mv` and is in scope because FR-015 says the existing installer suite must run. `install_test.sh` follows the Go and parsec convention (`name_test`).

**Alternatives considered**:
- *Glob both `test_*.sh` and `*_test.sh`.* Rejected: two conventions is how the next suite ends up in neither.
- *An explicit list of suites in the job.* Rejected: the issue's third point is precisely that suites existed and were "run by nothing"; a list needs maintaining and fails silently when it is not.

**Gap recorded, not in scope**: `scripts/check-linkage.sh` has no suite, and the constitution says every script under `scripts/` ships one. Testing it means stubbing `gh`; that is its own issue. Adding it later needs no change to the CI job.

## R4. Running the merge gate in CI

**Decision**: A job `linkage` (display name `linkage`) with `if: github.event_name == 'pull_request'`, job-level `permissions: {contents: read, pull-requests: read, issues: read}`, `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}`, `REPO: ${{ github.repository }}`, running `scripts/check-linkage.sh --pr ${{ github.event.pull_request.number }}`. Not in the `needs` chain of the release jobs (they run only on `push` to `main`, where the job is skipped).

**Rationale**: The constitution assigns this to #62 and states that in this repository the merge is the release, so this gate is the one that protects a version. The checker already implements `--pr` and needs only `gh` and `jq`, both preinstalled. The repository is public, but the workflow's top-level `permissions: contents: read` narrows `GITHUB_TOKEN`, and a narrowed token gets `403 Resource not accessible by integration` on `pulls`/`issues` reads even for public data; the two read permissions are therefore explicit. `REPO` is passed because `check-linkage.sh` otherwise calls `gh repo view`, which needs a checkout with a remote and one more API call.

**Consequence to state on the PR**: the job is red until the PR carries the milestone and a `Closes #62` line. That is the intended behaviour (spec Story 5, scenario 1) and is why the job runs on `pull_request` only.

**Alternatives considered**:
- *Run it on `push` to `main` too.* Rejected: there is no PR number on a push, and the merge already happened; the constitution says a push to `main` was gated by its PR.
- *Fail closed on forks.* Not needed: `pull_request` from a fork still receives a read-only `GITHUB_TOKEN`, which is all the gate needs. If GitHub ever withholds it, `gh` exits non-zero and the job fails with the API error rather than passing, which is the fail-closed behaviour FR-016 wants.

## R5. Where the shell-suite job sits in the workflow

**Decision**: Add job `shell-suites` (display name `shell suites`) that runs on both `pull_request` and `push` to `main`, with no `needs`; add `shell-suites` to the `needs` of `docker-image-pr` and `docker-image-main`. `release` and `publish-image` are untouched and inherit the gate through `docker-image-main`.

**Rationale**: FR-015 wants the suites on every PR and every push to `main`; SC-003 wants them "required for merge". Branch protection lives outside the repository and cannot be verified here, whereas a `needs` edge is in-repo and makes a red suite stop the image build and therefore the release. Adding an edge does not change any step of the release job, which keeps this inside the ask-first line the spec's assumptions draw: the CI addition is proposed and approved through this plan's review.

**Alternatives considered**:
- *Put the suites as a step inside the `test` job.* Rejected: `test` is the Go job with Go setup and coverage; a shell failure there would read as a Go failure, and the job would need Go even when only a hook changed.
- *Make `test` depend on `shell-suites`.* Rejected: adds latency to the longest job for no gain; the suites run in seconds in parallel.

## R6. Documenting the hook and the bypass

**Decision**: README.md gains a short `## Contributing` section with two items: enable the hook once per clone (`git config core.hooksPath .githooks`, wording taken from what `bootstrap.sh` prints), and the sanctioned bypass (`LINKAGE_OFF=1 <cmd>`) for a deliberate release step, with the rule that a command is never rephrased to get past the guard. AGENTS.md's HTML comment under the milestone rules is extended to name `.githooks/pre-push`. The AGENTS.md release-process section that describes release branches is not edited here; the constitution's Sync Impact Report already tracks it as shared boilerplate to fix upstream.

**Rationale**: README is the file Principle V says is updated in the same PR, and it is the only file a new contributor is sure to read. AGENTS.md above the `---` is copier-rendered; keeping the edit to the comment avoids fighting the template. FR-013 and FR-018.

## R7. Bash portability of the guard

**Finding**: The guard uses `[[ =~ ]]` with `BASH_REMATCH`, `${var//pat/rep}` with `$'\n'`, and `<<<`; all are bash 3.2 compatible, so even macOS' system bash runs it. Claude Code invokes the hook through the shebang (`/usr/bin/env bash`), which on this machine resolves to Homebrew bash 5.3. The pre-push hook is run by git via the same shebang. No portability work needed.

## R8. Cost of the guard on the non-release path

**Finding**: ~30 ms per invocation for `go test ./... && git status` on the reference guard (20-run mean, this machine). The path is bash string operations plus a handful of `grep -E` calls; the checker is only invoked when a release is detected. FR-010 holds.
