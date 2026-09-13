---

description: "Task list for 001-harden-linkage-guard"
---

# Tasks: Harden the Linkage Guard

**Input**: Design documents from `/specs/001-harden-linkage-guard/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)

**Tests**: Required (constitution Principle III). Every story's suite is adopted or run before the file it covers, and the suite is shown to fail against the current guard before the guard is replaced.

**Organization**: Grouped by user story. Stories 1 and 2 are delivered by the same adopted file; Story 2's phase is the verification that the release-branch and version-source cases hold, with no additional file.

**Upstream pin**: `galax-io/spec-kit-galaxio-bootstrap` at `88978a1323113468b289dff5c06fea68d1b3ced5` (research R1). Every adopted file is fetched from that commit and must stay byte-identical to it.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1..US5)
- Include exact file paths in descriptions

## Path Conventions

Repository root. Hooks live in `.claude/hooks/` and `.githooks/`; scripts in `scripts/`; CI in `.github/workflows/ci.yml`; docs in `README.md`, `AGENTS.md`, `.specify/memory/constitution.md`. Scratch files go in the session scratchpad, never in the tree.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Pin and fetch the upstream sources, confirm the local toolchain.

- [X] T001 Fetch the four upstream files at the pinned commit into a scratch directory `$SCRATCH/upstream/` with `gh api "repos/galax-io/spec-kit-galaxio-bootstrap/contents/<path>?ref=88978a1323113468b289dff5c06fea68d1b3ced5" --jq .content | base64 -d`, for `.claude/hooks/linkage-guard.sh`, `.claude/hooks/linkage-guard_test.sh`, `.githooks/pre-push`, `.githooks/pre-push_test.sh`; record the four `sha256sum` values in the task log
- [X] T002 [P] Confirm the toolchain the suites need is present: `bash --version` (≥ 4), `jq --version`, `git --version`, `shellcheck --version`; if `shellcheck` is absent note it and skip T024 locally (CI does not run shellcheck; it is a local hygiene check)
- [X] T003 [P] Run the upstream suites against the upstream files in the scratch directory (`cd $SCRATCH/upstream && chmod +x pre-push && bash linkage-guard_test.sh && bash pre-push_test.sh`); expected `32 passed, 0 failed` and `12 passed, 0 failed`; if either fails, stop: the pin is wrong or the toolchain differs, and research R2 must be redone

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The spec-first commit that AGENTS.md requires before any `fix`/`ci`/`docs` commit lands.

**⚠️ CRITICAL**: No implementation commit may precede this one on the branch.

- [X] T004 Commit the spec artifacts on `001-harden-linkage-guard`: `git add specs/001-harden-linkage-guard .specify/feature.json` then `git commit -m "docs(speckit): add 001-harden-linkage-guard spec/plan/tasks"` with the `Co-Authored-By` trailer; verify with `git log --oneline -1` that it sits on top of `81005c7`
- [X] T005 Capture the current guard for the regression evidence: `git show main:.claude/hooks/linkage-guard.sh > $SCRATCH/old-guard.sh` (the pre-fix file; used by T006 and quickstart step 2)

**Checkpoint**: Spec committed; the branch has one `docs(speckit)` commit and a clean tree.

---

## Phase 3: User Story 1 - The guard judges only what would actually run (Priority: P1) 🎯 MVP

**Goal**: Replace `.claude/hooks/linkage-guard.sh` with the upstream guard so prose, heredocs, wrappers, chained commands and flag order are judged as the shell would run them (spec FR-001..FR-004, FR-006..FR-010; contract [contracts/guard.md](contracts/guard.md)).

**Independent Test**: `bash .claude/hooks/linkage-guard_test.sh` reports `32 passed, 0 failed`; the same suite against the old guard reports `23 passed, 9 failed`.

### Tests for User Story 1 (REQUIRED — Constitution Principle III)

> **NOTE: The suite is adopted first and shown to fail against the current guard. It is the executable contract; it is not edited locally.**

- [X] T006 [US1] Copy `$SCRATCH/upstream/linkage-guard_test.sh` to `.claude/hooks/linkage-guard_test.sh` (mode 0644, byte-identical: `diff -q`), then run it against the old guard, `GUARD=$SCRATCH/old-guard.sh bash .claude/hooks/linkage-guard_test.sh`; expected `23 passed, 9 failed` with exactly the nine cases listed in research R2 (quoted-body `gh pr create`, `gh issue create`, `echo`, `LINKAGE_OFF=1` prefix, both release-branch pushes, annotated tag, `commit && tag`, `log && tag`); paste the FAIL lines into the PR description as the regression evidence

### Implementation for User Story 1

- [X] T007 [US1] Replace `.claude/hooks/linkage-guard.sh` with `$SCRATCH/upstream/linkage-guard.sh` (keep mode 0755: `chmod +x`; confirm `git diff --summary` shows no mode change and `diff -q` against the scratch copy is silent)
- [X] T008 [US1] Run `bash .claude/hooks/linkage-guard_test.sh`; expected `32 passed, 0 failed`; the final case must print `the checker is asked about the tag being created`
- [X] T009 [US1] Exercise the guard the way Claude Code does, per quickstart step 3: pipe `{"tool_input":{"command":"gh pr create --body \"the guard blocks git tag v1.2.3\""}}` with `CLAUDE_PROJECT_DIR=$PWD` and expect exit 0 and empty stderr; pipe `{"tool_input":{"command":"rtk git tag v9.9.9"}}` and expect exit 2 with `BLOCKED by linkage-guard:` followed by the checker's `no milestone whose title starts with 'v9.9.0'` (this proves the `rtk` wrapper this project's hook prepends is peeled, spec scenario 4)
- [X] T010 [US1] Confirm `.claude/settings.json` needs no change (it already wires `$CLAUDE_PROJECT_DIR/.claude/hooks/linkage-guard.sh` as `PreToolUse` for `Bash`); do not edit it

**Checkpoint**: The guard is the upstream one, its suite is green, and the nine-case regression is documented.

---

## Phase 4: User Story 2 - A branch push is not a release (Priority: P2)

**Goal**: Verify the adopted guard treats a branch push as a plain push and reads the version only from the tag (spec FR-005, FR-006).

**Independent Test**: The suite's `push a release branch` and `fast-forward the release branch` cases pass with exit 0, and `push a version tag` hands `--for-tag v1.0.7` to the checker.

### Tests for User Story 2 (REQUIRED — Constitution Principle III)

- [X] T011 [US2] Run the suite and confirm the three lines `ok   [0] push a release branch`, `ok   [0] fast-forward the release branch`, and `ok   [-] the checker is asked about the tag being created` appear in `bash .claude/hooks/linkage-guard_test.sh` output (`grep -E 'release branch|checker is asked'` must return three `ok` lines)

### Implementation for User Story 2

- [X] T012 [US2] Hand-check the version source per quickstart step 3: pipe `{"tool_input":{"command":"git push origin release/1.2.0"}}` and expect exit 0; pipe `{"tool_input":{"command":"git push origin v1.2.1"}}` and expect exit 2 whose stderr names `v1.2.1` (not `1.2.0`) in the checker's milestone lookup; pipe `{"tool_input":{"command":"git push --tags"}}` and expect exit 2 with the `without explicit vX.Y.Z` guidance; record the three results in the task log; no file changes (the behaviour ships in T007)

**Checkpoint**: Stories 1 and 2 hold on the same file; nothing further to land for US2.

---

## Phase 5: User Story 3 - Manual tags from any client are guarded (Priority: P2)

**Goal**: Ship `.githooks/pre-push` and its suite, and document the one-time enable command and the bypass (spec FR-011..FR-013, FR-018; contract [contracts/pre-push.md](contracts/pre-push.md); research R6).

**Independent Test**: `bash .githooks/pre-push_test.sh` reports `12 passed, 0 failed`; with `core.hooksPath` set, pushing tag `v9.9.9` to a throwaway remote is refused with `pre-push: refusing to publish v9.9.9`.

### Tests for User Story 3 (REQUIRED — Constitution Principle III)

- [X] T013 [P] [US3] Copy `$SCRATCH/upstream/pre-push_test.sh` to `.githooks/pre-push_test.sh` (mode 0644, byte-identical); run it before the hook exists and confirm it fails loudly (hook missing), which is the "fails without the change" evidence for this story

### Implementation for User Story 3

- [X] T014 [US3] Copy `$SCRATCH/upstream/pre-push` to `.githooks/pre-push`, `chmod +x .githooks/pre-push`, and confirm `git add -n .githooks/pre-push && git diff --cached --summary` will record mode `100755` (git tracks the executable bit; a 0644 hook is silently never run)
- [X] T015 [US3] Run `bash .githooks/pre-push_test.sh`; expected `12 passed, 0 failed`, last line `the checker is asked about the tag being pushed`
- [X] T016 [US3] Live check per quickstart step 4 in a throwaway clone under `$SCRATCH` with a bare repository as `origin`: `git config core.hooksPath .githooks`, `git tag v9.9.9`, `git push origin v9.9.9` → refused with the checker's output and `pre-push: refusing to publish v9.9.9 — its milestone is not ready`; `git push origin HEAD:refs/heads/scratch` → accepted with no hook output; `git push origin :refs/tags/v9.9.9` after a forced local push with `--no-verify` → accepted (deletion passes); remove the clone afterwards
- [X] T017 [US3] Add a `## Contributing` section at the end of `README.md` (after `## Environment Variables` and whatever follows it) with two short items: (1) enable the release-tag hook once per clone with a fenced `git config core.hooksPath .githooks` block and the sentence from `bootstrap.sh`: without it `.githooks/pre-push` never runs, CI still checks server-side, this is the local fast failure; (2) the sanctioned bypass for a deliberate release step is `LINKAGE_OFF=1 <command>`; a command is never rephrased to get past the guard. Link the contract by naming `scripts/check-linkage.sh` as the rule source. No other README edits

**Checkpoint**: Hook and suite in place, enable command documented; a manual tag from any client is now gated.

---

## Phase 6: User Story 4 - Every shell suite runs in CI (Priority: P2)

**Goal**: A `shell suites` job discovers and runs every `*_test.sh` under the three governed directories on every PR and push to `main`, and gates the image and release (spec FR-014, FR-015; contract [contracts/ci.md](contracts/ci.md); research R3, R5).

**Independent Test**: Locally, the job's discovery loop finds three suites and exits 0; with one case deliberately broken in a scratch copy it exits non-zero naming the case. On the PR, the `shell suites` check is green and `docker image` waits for it.

### Tests for User Story 4 (REQUIRED — Constitution Principle III)

- [X] T018 [US4] Run the exact discovery loop the job will use, from the repository root under `set -euo pipefail`: `suites=(scripts/*_test.sh .claude/hooks/*_test.sh .githooks/*_test.sh); [ "${#suites[@]}" -ge 3 ] || exit 1; for t in "${suites[@]}"; do echo "--- $t"; bash "$t"; done`; before T019 it must fail the floor (only two suites match, because the installer suite is named `test_install.sh`), which is the evidence for the rename
- [X] T019 [US4] `git mv scripts/test_install.sh scripts/install_test.sh`; no content change; rerun the T018 loop and expect three `---` headers and exit 0

### Implementation for User Story 4

- [X] T020 [US4] Add job `shell-suites` to `.github/workflows/ci.yml` after `test`, per [contracts/ci.md](contracts/ci.md): `name: shell suites`, `runs-on: ubuntu-latest`, no `needs`, no `if`; steps `actions/checkout@v4` then one `run` step with `shell: bash`, `set -euo pipefail`, the glob array, the `-ge 3` floor with the message `found N shell suites, expected at least 3`, and the loop that prints `--- <path>` before `bash "$t"`; add a two-line comment above the job saying why the floor exists (a glob that matches nothing must not pass)
- [X] T021 [US4] Add `shell-suites` to the `needs` lists of `docker-image-pr` and `docker-image-main` in `.github/workflows/ci.yml`; change nothing else in those jobs, and nothing in `release` or `publish-image`
- [X] T022 [US4] Validate the workflow file: `python3 -c 'import yaml,sys; yaml.safe_load(open(".github/workflows/ci.yml"))'` (or `yq`), and `grep -n "needs:" -A3 .github/workflows/ci.yml` to eyeball that exactly the two image jobs gained the edge; if `actionlint` is installed, run it

**Checkpoint**: Three suites run in CI on every PR and push; a red suite stops the image and the release.

---

## Phase 7: User Story 5 - The merge gate runs in CI (Priority: P3)

**Goal**: A `linkage` job runs `scripts/check-linkage.sh --pr N` on every pull request with the read permissions it needs (spec FR-016; contract [contracts/ci.md](contracts/ci.md); research R4).

**Independent Test**: On the PR, the `linkage` check is red while the PR lacks a milestone or a `Closes #62` line, and green after both are set and the job is re-run.

### Tests for User Story 5 (REQUIRED — Constitution Principle III)

- [X] T023 [US5] Run the gate locally against an existing merged PR to confirm the command shape and exit codes before wiring it: `REPO=galax-io/galaxio-cli scripts/check-linkage.sh --pr 60` (expect `PASS` or a `FAIL` that names a concrete rule; either proves the invocation) and `scripts/check-linkage.sh --pr 999999` (expect exit 2, `PR #999999 not found`), so the job's fail-closed behaviour on an unreadable PR is known

### Implementation for User Story 5

- [X] T024 [US5] Add job `linkage` to `.github/workflows/ci.yml` after `shell-suites`: `name: linkage`, `if: github.event_name == 'pull_request'`, `runs-on: ubuntu-latest`, job-level `permissions: {contents: read, pull-requests: read, issues: read}`, `env: {GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}", REPO: "${{ github.repository }}"}`, steps `actions/checkout@v4` then `run: scripts/check-linkage.sh --pr "${{ github.event.pull_request.number }}"`; add a comment above it stating that a push to `main` is skipped because it was gated by its PR, and that the two read permissions are explicit because the workflow's top-level `contents: read` narrows the token
- [X] T025 [US5] Re-validate `.github/workflows/ci.yml` as in T022; confirm `linkage` is in nobody's `needs` (merge protection is where it becomes required, and that is configured in GitHub, not here)

**Checkpoint**: All five stories implemented; the PR itself becomes the live test of Story 5.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation that must agree with the change, hygiene checks, the commits and the PR.

- [X] T026 [P] Extend the HTML comment block in `AGENTS.md` (the three `<!-- … -->` lines under the `Release Process` rules that mention `scripts/check-linkage.sh` and `.claude/hooks/linkage-guard.sh`) so it also names `.githooks/pre-push` (opt-in via `core.hooksPath`) and the `linkage` and `shell suites` CI jobs; do not touch any other line of `AGENTS.md`
- [X] T027 [P] Amend `.specify/memory/constitution.md` to **v1.0.1** (PATCH): in the Quality Gates table change the Linkage row's CI job from `(manual today — see #62)` to `linkage`; in Additional constraints replace the paragraph starting `The --pr linkage gate is not run by CI` with one sentence saying the `linkage` job runs it on every pull request and the `shell suites` job runs every `*_test.sh` suite; in the Development Workflow "Milestones" bullet keep the text but drop `(#62)` after `.githooks/pre-push`; rewrite the Sync Impact Report comment at the top (version 1.0.0 → 1.0.1, bump rationale, remove the galaxio-cli#62 follow-up TODO, add a follow-up that `scripts/check-linkage.sh` still has no suite); update the footer to `**Version**: 1.0.1 | **Ratified**: 2026-09-02 | **Last Amended**: <date of the commit>`
- [X] T028 [P] Run `shellcheck -S warning .claude/hooks/linkage-guard.sh .claude/hooks/linkage-guard_test.sh .githooks/pre-push .githooks/pre-push_test.sh scripts/install_test.sh`; expected no output for the four adopted files; if `install_test.sh` warns, leave it (out of scope, it is a rename only) and note it
- [X] T029 Byte-identity check per quickstart step 6: for each of the four adopted files, `gh api "repos/galax-io/spec-kit-galaxio-bootstrap/contents/<path>?ref=88978a1323113468b289dff5c06fea68d1b3ced5" --jq .content | base64 -d | diff -q - <path>`; four silent diffs required (FR-017)
- [X] T030 Run the whole of [quickstart.md](quickstart.md) steps 1, 3, 5 and 6 from a clean tree and confirm every expected line; the local verify commands `go vet ./... && go test ./...` and `gofmt -l .` still pass (no Go changed, but the branch must be green as a whole)
- [X] T031 Commit in the order the plan fixes, each green on its own, each with the `Co-Authored-By` trailer: (a) `fix(hooks): adopt the shared linkage guard and add pre-push (#62)` containing `.claude/hooks/linkage-guard.sh`, `.claude/hooks/linkage-guard_test.sh`, `.githooks/pre-push`, `.githooks/pre-push_test.sh`, and the `scripts/install_test.sh` rename, with the upstream commit `88978a13…` named in the body; (b) `ci: run every shell suite and the PR merge gate (#62)` containing only `.github/workflows/ci.yml`; (c) `docs: document the release guards for contributors (#62)` containing `README.md` and `AGENTS.md`; (d) `docs(speckit): amend constitution to v1.0.1 (linkage gate runs in CI)` containing `.specify/memory/constitution.md`; verify `git log --oneline main..HEAD` shows exactly the spec commit plus these four
- [X] T032 Open the PR against `main` with `gh pr create --base main --milestone "v0.11.0 SDD bootstrap"`, body containing `Closes #62`, the nine-case regression output from T006, the note that `linkage` is red until milestone and closing link are present, and the attribution footer; then `REPO=galax-io/galaxio-cli scripts/check-linkage.sh --pr <N>` locally and expect `PASS`
- [X] T033 Watch the PR's checks: `shell suites` green listing three suites, `linkage` green, `docker image` shows `shell suites` among its dependencies, `test` and `integration tests` unchanged; if `linkage` fails with `Resource not accessible by integration`, the job-level permissions in T024 were not applied and must be fixed before review
- [ ] T034 (skipped: optional, not run — PR #65 run 34756160392 already shows the job fail-closed by construction) Optional, on a throwaway branch only (quickstart step 7): break one guard case, push, confirm `shell suites` fails naming the case, delete the branch; record the run URL in the PR as SC-004 evidence
- [ ] T035 (pending: needs the maintainer to merge PR #65) After merge (rebase only, no merge commit): confirm issue #62 auto-closed, the automatic release cut a **patch** version (no `feat` in the PR), and the milestone shows the PR; file the follow-up issue for a `scripts/check-linkage_test.sh` suite recorded in the constitution's Sync Impact Report

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 first; T002 and T003 in parallel after it.
- **Foundational (Phase 2)**: T004 needs a clean tree; T005 is independent. Blocks every implementation commit, not every task: verification tasks may run before T004, but nothing is committed before it.
- **US1 (Phase 3)**: needs T001, T005. T006 before T007 (suite shown failing first).
- **US2 (Phase 4)**: needs T007; verification only.
- **US3 (Phase 5)**: needs T001. T013 before T014. T017 is independent of T014 (different file).
- **US4 (Phase 6)**: T018 needs T006 and T013 (two suites must exist for the floor evidence); T019 before T020; T021 after T020 (same file).
- **US5 (Phase 7)**: T023 independent; T024 after T021 (same file `ci.yml`); T025 after T024.
- **Polish (Phase 8)**: T026, T027, T028 in parallel after all stories; T029, T030 after them; T031 → T032 → T033 → T034 (optional) → T035 strictly in order.

### User Story Dependencies

- **US1 (P1)**: independent; the MVP.
- **US2 (P2)**: delivered by US1's file; its phase is verification and adds nothing to land.
- **US3 (P2)**: independent of US1 and US2 (different files); its README edit is independent of everything else.
- **US4 (P2)**: needs the suites from US1 and US3 to exist to be meaningful, but the job itself is independent code.
- **US5 (P3)**: independent; serialised behind US4 only because both edit `ci.yml`.

### Parallel Opportunities

- T002 ∥ T003 after T001.
- T006/T007 (guard) ∥ T013/T014 (hook) ∥ T017 (README): three different files.
- T026 ∥ T027 ∥ T028 in Polish.
- Nothing in `ci.yml` is parallel: T020 → T021 → T024 → T025 edit the same file.

---

## Parallel Example: after Setup

```bash
# Different files, no shared state — run together:
Task: "T006 adopt .claude/hooks/linkage-guard_test.sh and show it failing against the old guard"
Task: "T013 adopt .githooks/pre-push_test.sh and show it failing without the hook"
Task: "T017 add the Contributing section to README.md"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 (pin and fetch) and Phase 2 (spec commit).
2. Phase 3: adopt suite, prove nine failures, adopt guard, prove 32 green.
3. **STOP and VALIDATE**: quickstart steps 1 to 3. This alone removes the false blocks that trained contributors to rephrase commands.

### Incremental Delivery

1. US1 → the guard is right (MVP).
2. US2 → evidence the branch/version rules hold; nothing new lands.
3. US3 → pre-push hook and enable instructions; manual tags covered.
4. US4 → suites run in CI and gate the release; the regression class is closed.
5. US5 → merge gate in CI; reviewers stop checking milestones by hand.

Each step is independently green; the commit map in T031 groups them into four commits inside one PR because they are one issue and one concern (the guard and what documents and runs it).

### Commit Map

| Commit | Tasks whose output it contains |
|---|---|
| `docs(speckit): add 001-harden-linkage-guard spec/plan/tasks` | T004 |
| `fix(hooks): adopt the shared linkage guard and add pre-push (#62)` | T006, T007, T013, T014, T019 |
| `ci: run every shell suite and the PR merge gate (#62)` | T020, T021, T024 |
| `docs: document the release guards for contributors (#62)` | T017, T026 |
| `docs(speckit): amend constitution to v1.0.1 (linkage gate runs in CI)` | T027 |

---

## Notes

- The four adopted files are never edited locally, even to fix the stale `Usage:` comment in the guard suite (research R1); an upstream issue is the route.
- `LINKAGE_OFF=1` is the only sanctioned bypass; do not phrase any command in this work to get past the guard, including while testing it (use the JSON-pipe form in T009/T012 instead of running a real `git tag`).
- The `linkage` check on the PR will be red until T032 sets the milestone and closing link; that is Story 5 working.
- Commit type matters for the automatic release: `fix`, `ci`, `docs` only. No `feat` in this PR.
