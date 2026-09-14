# Tasks: Define Ecosystem Naming and Licensing

**Input**: Design documents from `/specs/003-define-naming-licence/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)

**Tests**: Required by the feature specification and Constitution Principle III. Command
tests are written first through `runCLI`; documentation/licence stories use reproducible
pre-change and post-change audits from `quickstart.md`.

**Organization**: Tasks are grouped by user story. The milestone has one review boundary:
PR #110. Within that PR, every active task maps to exactly one green commit.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel after the preceding gate because it touches different files
  and does not depend on another incomplete task in the group.
- **[Story]**: Maps the task to US1, US2, or US3 from `spec.md`.
- Every task names the exact file or directory it changes or validates.

**Delivery record**: T001–T023 are the original execution record. They describe outcomes
that were merged before the v2.0.0 workflow correction and do not certify that maintainer
review or task-per-commit delivery occurred. `[~]` marks a historical instruction invalidated
or superseded by the review findings. T024–T027 are the authoritative remediation plan for
PR #110. The two governance prerequisites are tracked on #109 and already have their own
commits: `docs(agents): require reviewed single-PR milestones (#109)` and
`docs(speckit): amend constitution to v2.0.0 (reviewed task commits) (#109)`.

## Phase 1: Setup (Spec-First Delivery)

**Purpose**: Isolate the feature artifacts from the already-dirty Spec Kit installation and
record the original spec-first delivery.

- [X] T001 Confirm `.specify/feature.json` selects `specs/003-define-naming-licence/`; classify only `.specify/feature.json` and `specs/003-define-naming-licence/` as feature-planning changes, leaving `.agents/`, `.specify/extensions/.registry`, `.specify/init-options.json`, `.specify/integration.json`, and `.specify/integrations/codex.manifest.json` outside the feature commit
- [X] T002 Record the historical spec artifact delivery in commit `43985dc` / PR #103; do not use that already-merged split as the workflow for corrective work

**Checkpoint**: The specification, plan, contracts, research, quickstart, and this task list
exist on `main`; all corrective work stays in PR #110.

---

## Phase 2: Foundational (Approval and Evidence Gates)

**Purpose**: Record the original evidence and the approval assumption later invalidated by
the full review.

**Historical gate**: T004 did not establish maintainer PR review and is not reused.

- [X] T003 Re-run the read-only baseline from `specs/003-define-naming-licence/quickstart.md` against `cmd/galaxio/root.go`, `README.md`, `LICENSE`, `Dockerfile`, and `github.com/galax-io/parsec`; stop and amend `specs/003-define-naming-licence/research.md` in a spec-only change if repository ownership, visibility, licence metadata, or command-name availability has drifted
- [~] T004 Superseded: issue comments created during implementation were not maintainer PR review; PR #110 must remain open until the maintainer reviews it and explicitly instructs merge or closure

**Checkpoint**: Current evidence supports the design; review approval remains pending on
PR #110.

---

## Phase 3: User Story 1 — Use One Canonical Component Vocabulary (Priority: P1) 🎯 MVP

**Goal**: Make `parsec` and `report` the sole active names and expose
`galaxio report` from root help without defining report operations.

**Independent Test**: Root help lists `report`; `galaxio report` and
`galaxio report --help` return help on stdout with exit 0; an unexpected report argument
returns exit 2 on stderr; the decision record and README map the public library repository
and CLI namespace to one canonical name each.

### Tests for User Story 1

> Write the command tests first and run them before implementation; they must fail because
> `report` is not yet registered.

- [X] T005 [US1] Extend `cmd/galaxio/root_test.go` with a failing `report` expectation in `TestHelpPrintsMinimalUsage`, table-driven `report` and `report --help` cases asserting exit 0/stdout/stderr, and an unexpected-argument case asserting `UsageError` exit 2 and empty stdout

### Implementation for User Story 1

- [X] T006 [P] [US1] Create `cmd/galaxio/report.go` with `newReportCommand()`, `Use: "report"`, short text `Report on finished load-test runs.`, a long description that defers operational subcommands, `cobra.NoArgs` wrapped in `UsageError`, and help-only `RunE`; add no flags, `runReport`, JSON output, feature gate, or runtime package
- [X] T007 [US1] Register `newReportCommand()` in `cmd/galaxio/root.go` without changing or removing any existing root command, global flag, error normalization, or exit behavior
- [X] T008 [P] [US1] Add a concise ecosystem-naming and report-namespace entry to `README.md` that uses only `parsec` and `galaxio report`, identifies `parsec` as public, and links `specs/003-define-naming-licence/research.md`
- [X] T009 [US1] Run the US1 command tests and Section 1–2 checks in `specs/003-define-naming-licence/quickstart.md`; confirm all earlier root commands remain visible, no active placeholder alias exists, no input/file is touched, and no `internal/report/` or `parsec` import was introduced
- [X] T010 [US1] Record the historical report implementation in commit `a4320f3` / PR #105; review its behavior as part of the full milestone review and do not open another implementation PR

**Checkpoint**: User Story 1 behavior is implemented and independently testable; #47 remains
open until reviewed PR #110 lands.

---

## Phase 4: User Story 2 — Consume the Shared Library Without a Licence Conflict (Priority: P1)

**Goal**: Make the retained `GPL-2.0-only` CLI posture explicit and preserve the audited MIT
boundary for future `parsec` consumption without editing or importing that repository.

**Independent Test**: `LICENSE`, README, the OCI label, and documented GitHub classifier all
resolve to GPL version 2 only; `parsec` remains public/MIT with its notice; the decision
record cites the FSF MIT/Expat compatibility and Apache-2.0 boundary; no external repository
or dependency graph is changed.

### Tests for User Story 2

> The historical audit preceded the README update. T024 adds the deterministic regression
> guard for the final canonical licence file and exact metadata surfaces.

- [X] T011 [US2] Execute Section 3 of `specs/003-define-naming-licence/quickstart.md` before implementation and compare every result with `specs/003-define-naming-licence/contracts/licence-posture.md`, including the public MIT licence/notice for `github.com/galax-io/parsec` and GitHub's legacy `gpl-2.0` classifier

### Implementation for User Story 2

- [~] T012 Superseded by T024: the prepended project-selection notice broke GitHub licence detection even though the GNU GPL clauses were unchanged
- [X] T013 [P] [US2] Rewrite the `README.md` licence section to state `GPL-2.0-only` explicitly, identify public `parsec` as MIT for the reviewed dependency relationship, and link `LICENSE` and `specs/003-define-naming-licence/research.md`
- [~] T014 Invalidated by review: the post-change GitHub classifier returned `NOASSERTION`; T024 restores the canonical file and T027 verifies the pushed PR branch
- [X] T015 [US2] Verify `go.mod`, `go.sum`, `Dockerfile`, `.goreleaser.yaml`, and `.github/workflows/release.yml` are unchanged by US2; confirm no `parsec` import, dependency addition/upgrade, external-repository edit, or whole-module compatibility claim entered the change
- [X] T016 [US2] Record the historical licence documentation in commit `fae78c5` / PR #106; retain the correct README metadata, use T024 for the classifier fix, and do not open another licence PR

**Checkpoint**: User Story 2 is independently auditable; #48 remains open until reviewed
PR #110 lands, the CLI/parsec relationship has no unresolved licence conflict, and the
pre-existing Cobra risk remains accurately scoped to a separate review.

---

## Phase 5: User Story 3 — Close the Milestone on Auditable Evidence (Priority: P2)

**Goal**: Leave a versioned verification record that distinguishes implemented work from
maintainer-reviewed and merged work.

**Independent Test**: Before merge, the decision record links the selected/rejected choices,
historical commits, PR #110, repository ownership, root-help evidence, licence audit, and
authoritative sources while keeping review, issue closure, and milestone completion pending.

### Tests for User Story 3

- [~] T017 Superseded by T027, which validates the one milestone PR and records open pre-merge issue state

### Implementation for User Story 3

- [~] T018 Superseded by T027: record status `in-review`, not `verified`, until PR #110 is reviewed and merged
- [~] T019 Superseded by T027: cross-check FR-013–FR-015 and SC-006–SC-007 without inferring review or closure from implementation
- [~] T020 Superseded by T027: commit validation evidence in PR #110 and leave that PR open; do not create or land a verification-only PR

**Checkpoint**: All implementation stories are reviewable in one place. The local decision
record contains reproducible pre-merge evidence and makes no false closure claim.

---

## Phase 6: Polish & Cross-Cutting Validation

**Purpose**: Prove the final repository state is green, scoped, and ready for the separate
release process without starting it.

- [~] T021 Superseded by the complete local and CI gate run in T027
- [~] T022 Superseded by the scope and excluded-surface audit in T027
- [~] T023 Superseded by the PR history, linkage, dirty-worktree isolation, and no-tag audit in T027

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phases 1–6**: historical implementation record; superseded instructions are not reused.
- **Phase 7 (Convergence)**: T024–T027 run sequentially in PR #110 because later tasks audit
  and document the files changed by earlier tasks.
- **Maintainer review**: follows T027. It is not an agent-completable checkbox; PR #110 stays
  open until the maintainer explicitly decides its disposition.

### User Story Dependency Graph

```text
Historical implementation -> T024 licence fix -> T025 workflow repair -> T026 scope cleanup -> T027 validation -> maintainer review
```

- **US1 (P1)**: behavior exists and is covered by command tests.
- **US2 (P1)**: licence metadata exists and T024 guards the classifier-sensitive file.
- **US3 (P2)**: T027 records the complete pre-merge evidence; review and closure follow.

### Within Each User Story

- Run the failing test/audit before implementation.
- Apply the smallest file-scoped changes that make it pass.
- Re-run the independent test before creating the task commit.
- Keep every task commit green and push all task commits to PR #110.

## Parallel Opportunities

No Phase 7 task runs in parallel: each one consumes the corrected artifacts from the
preceding task and receives its own commit.

## Implementation Strategy

1. Preserve the implemented `report` behavior and correct only review findings.
2. Complete T024–T027 sequentially, one green commit each, in PR #110.
3. Force-update the existing PR only after local validation; do not create a replacement.
4. Leave PR #110 and all linked issues open for maintainer review.
5. After an explicit post-review merge instruction, GitHub may close the linked issues;
   release preparation remains a separate maintainer action.

### Commit and PR Boundaries

```text
PR #110
├── issue #109 task: agent workflow guidance
├── issue #109 task: constitution and task template
├── T024: machine-detectable GPLv2
├── T025: truthful single-PR workflow artifacts
├── T026: CLI scope cleanup
└── T027: validation evidence
```

PR #110 is assigned to `v0.12.0 Naming and licence` and carries the applicable closing
references. No task authorizes an external-repository edit, PR merge/closure,
release-workflow change, release audit, release tag, or publication.

## Notes

- `[P]` tasks touch different files and have no dependency on an incomplete parallel task.
- Every story task carries its `[USn]` label; setup, foundational, and polish tasks do not.
- Tests and audits are mandatory and precede the change they verify.
- `parsec` is named and audited but not imported.
- The installed `golang-spf13-cobra` skill is applied together with the existing
  parent-command pattern and the constitution as recorded in `research.md`.

---

## Phase 7: Convergence

**Purpose**: Correct review findings without removing the implemented milestone behavior,
then leave one green milestone PR open for maintainer review.

- [X] T024 **CRITICAL** Restore the byte-for-byte canonical GNU GPL Version 2 text in `LICENSE`, keep the exact `GPL-2.0-only` project selection in `README.md` and `Dockerfile`, add a deterministic `scripts/license_surface_test.sh` regression suite, and align `specs/003-define-naming-licence/{plan.md,research.md,data-model.md,contracts/licence-posture.md,quickstart.md}` with machine-detectable licence metadata per FR-008, SC-004, and Constitution III (contradicts)
- [X] T025 **CRITICAL** Reconcile `specs/003-define-naming-licence/{plan.md,research.md,data-model.md,quickstart.md,tasks.md}` with one milestone PR, exactly one task per green commit, maintainer-owned review/merge/closure, the Constitution I help-only namespace exception, and truthful completion states per Constitution I and Development Workflow (contradicts)
- [X] T026 Correct stale repository-count wording in `specs/003-define-naming-licence/{spec.md,tasks.md,research.md}` and the in-scope GitHub issue #104, while retaining only the `galaxio-cli`/`parsec` boundary and recording the issue update in the decision evidence per FR-006 and US3/AC1 (partial)
- [X] T027 Run every local quality gate and external naming/licence/linkage check, append reproducible evidence to `specs/003-define-naming-licence/research.md`, leave review-dependent issue and milestone closure explicitly pending, update completion boxes in `specs/003-define-naming-licence/tasks.md`, and perform no merge, issue closure, release audit, or tag per FR-013–FR-015 and SC-006–SC-007 (partial)

**Checkpoint**: PR #110 contains one commit for each convergence task, is green, and remains
open for maintainer review; issue closure and release work remain pending until that reviewed
PR lands on `main`.
