# Tasks: Define Ecosystem Naming and Licensing

**Input**: Design documents from `/specs/003-define-naming-licence/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)

**Tests**: Required by the feature specification and Constitution Principle III. Command
tests are written first through `runCLI`; documentation/licence stories use reproducible
pre-change and post-change audits from `quickstart.md`.

**Organization**: Tasks are grouped by user story. Delivery boundaries are explicit because
#47 and #48 require separate one-commit PRs and #48 depends on #47.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel after the preceding gate because it touches different files
  and does not depend on another incomplete task in the group.
- **[Story]**: Maps the task to US1, US2, or US3 from `spec.md`.
- Every task names the exact file or directory it changes or validates.

## Phase 1: Setup (Spec-First Delivery)

**Purpose**: Isolate the feature artifacts from the already-dirty Spec Kit installation and
land the required planning record before implementation.

- [X] T001 Confirm `.specify/feature.json` selects `specs/003-define-naming-licence/`; classify only `.specify/feature.json` and `specs/003-define-naming-licence/` as feature-planning changes, leaving `.agents/`, `.specify/extensions/.registry`, `.specify/init-options.json`, `.specify/integration.json`, and `.specify/integrations/codex.manifest.json` outside the feature commit
- [X] T002 Stage `.specify/feature.json` and the complete `specs/003-define-naming-licence/` directory, create `docs(speckit): add 003-define-naming-licence spec/plan/tasks`, and land a spec-only PR assigned to milestone `v0.12.0 Naming and licence` that references but does not close #47 or #48

**Checkpoint**: The specification, plan, contracts, research, quickstart, and this task list
are on `main` before any `feat` or issue-fix commit begins.

---

## Phase 2: Foundational (Approval and Evidence Gates)

**Purpose**: Revalidate external facts and obtain the approvals required before changing an
observable command or the project licence notice.

**⚠️ CRITICAL**: No user-story implementation begins until both tasks complete.

- [X] T003 Re-run the read-only baseline from `specs/003-define-naming-licence/quickstart.md` against `cmd/galaxio/root.go`, `README.md`, `LICENSE`, `Dockerfile`, and `github.com/galax-io/parsec`; stop and amend `specs/003-define-naming-licence/research.md` in a spec-only change if repository ownership, visibility, licence metadata, or command-name availability has drifted
- [X] T004 Obtain explicit maintainer approval for the additive public surface planned in `cmd/galaxio/report.go` and for the `GPL-2.0-only` project notice planned in `LICENSE`; keep both files unchanged until the corresponding approvals are recorded on #47 and #48

**Checkpoint**: Current evidence still supports the design and both ask-first boundaries are
satisfied.

---

## Phase 3: User Story 1 — Use One Canonical Component Vocabulary (Priority: P1) 🎯 MVP

**Goal**: Make `parsec` and `report` the sole active names and expose
`galaxio report` from root help without defining report operations.

**Independent Test**: Root help lists `report`; `galaxio report` and
`galaxio report --help` return help on stdout with exit 0; an unexpected report argument
returns exit 2 on stderr; the decision record and README map the two repositories and CLI
namespace to one canonical name each.

### Tests for User Story 1

> Write the command tests first and run them before implementation; they must fail because
> `report` is not yet registered.

- [X] T005 [US1] Extend `cmd/galaxio/root_test.go` with a failing `report` expectation in `TestHelpPrintsMinimalUsage`, table-driven `report` and `report --help` cases asserting exit 0/stdout/stderr, and an unexpected-argument case asserting `UsageError` exit 2 and empty stdout

### Implementation for User Story 1

- [X] T006 [P] [US1] Create `cmd/galaxio/report.go` with `newReportCommand()`, `Use: "report"`, short text `Report on finished load-test runs.`, a long description that defers operational subcommands, `cobra.NoArgs` wrapped in `UsageError`, and help-only `RunE`; add no flags, `runReport`, JSON output, feature gate, or runtime package
- [X] T007 [US1] Register `newReportCommand()` in `cmd/galaxio/root.go` without changing or removing any existing root command, global flag, error normalization, or exit behavior
- [X] T008 [P] [US1] Add a concise ecosystem-naming and report-namespace entry to `README.md` that uses only `parsec` and `galaxio report`, identifies `parsec` as public, and links `specs/003-define-naming-licence/research.md`
- [X] T009 [US1] Run the US1 command tests and Section 1–2 checks in `specs/003-define-naming-licence/quickstart.md`; confirm all earlier root commands remain visible, no active placeholder alias exists, no input/file is touched, and no `internal/report/` or `parsec` import was introduced
- [X] T010 [US1] On a fresh `galaxio/`-prefixed branch from updated `main`, commit only `cmd/galaxio/report.go`, `cmd/galaxio/root.go`, `cmd/galaxio/root_test.go`, and the US1 portion of `README.md` as `feat(cli): reserve report command group (#47)`; open a milestone-1 PR with `Closes #47`, run `scripts/check-linkage.sh --pr <PR_NUMBER>`, and land it before starting US2

**Checkpoint**: User Story 1 is independently complete; #47 is closed on `main`, both
canonical identities are discoverable, and no report behavior has been invented.

---

## Phase 4: User Story 2 — Consume the Shared Library Without a Licence Conflict (Priority: P1)

**Goal**: Make the retained `GPL-2.0-only` CLI posture explicit and preserve the audited MIT
boundary for future `parsec` consumption without editing or importing that repository.

**Independent Test**: `LICENSE`, README, the OCI label, and documented GitHub classifier all
resolve to GPL version 2 only; `parsec` remains public/MIT with its notice; the decision
record cites the FSF MIT/Expat compatibility and Apache-2.0 boundary; no external repository
or dependency graph is changed.

### Tests for User Story 2

> Run the audit before editing; it must identify the current README wording and missing
> project-specific `only` notice as failures while confirming the existing Docker label.

- [X] T011 [US2] Execute Section 3 of `specs/003-define-naming-licence/quickstart.md` before implementation and compare every result with `specs/003-define-naming-licence/contracts/licence-posture.md`, including the public MIT licence/notice for `github.com/galax-io/parsec` and GitHub's legacy `gpl-2.0` classifier

### Implementation for User Story 2

- [X] T012 [P] [US2] Prepend the approved project-selection notice naming SPDX `GPL-2.0-only` to `LICENSE`, clearly separate it from the licence body, and leave every existing GNU GPL Version 2 term byte-for-byte unchanged
- [X] T013 [P] [US2] Rewrite the `README.md` licence section to state `GPL-2.0-only` explicitly, identify public `parsec` as MIT for the reviewed dependency relationship, and link `LICENSE` and `specs/003-define-naming-licence/research.md`
- [X] T014 [US2] Re-run Section 3 of `specs/003-define-naming-licence/quickstart.md`, inspect `git diff -- LICENSE` to prove the change is notice-only, and verify all repository-owned surfaces plus the documented GitHub classifier satisfy `specs/003-define-naming-licence/contracts/licence-posture.md`
- [X] T015 [US2] Verify `go.mod`, `go.sum`, `Dockerfile`, `.goreleaser.yaml`, and `.github/workflows/release.yml` are unchanged by US2; confirm no `parsec` import, dependency addition/upgrade, external-repository edit, or whole-module compatibility claim entered the change
- [X] T016 [US2] After #47 is on `main`, create a fresh `galaxio/`-prefixed branch and commit only `LICENSE` plus the US2 portion of `README.md` as `docs(licence): record compatible ecosystem boundary (#48)`; open a milestone-1 PR with `Closes #48`, run `scripts/check-linkage.sh --pr <PR_NUMBER>`, and land it before starting US3

**Checkpoint**: User Story 2 is independently auditable; #48 is closed on `main`, the
CLI/parsec relationship has no unresolved licence conflict, and the pre-existing Cobra risk
remains accurately scoped to a separate review.

---

## Phase 5: User Story 3 — Close the Milestone on Auditable Evidence (Priority: P2)

**Goal**: Leave a versioned verification record showing why #47, #48, and #107 can remain
closed and why milestone 1 has no outstanding work.

**Independent Test**: The decision record links the selected/rejected choices, merged PRs
and commits, repository ownership, root-help evidence, licence audit, authoritative sources,
closed #47/#48/#107 states, and a milestone response with zero open issues.

### Tests for User Story 3

- [ ] T017 [US3] Run Section 5 of `specs/003-define-naming-licence/quickstart.md` after both implementation PRs and the #107 scope correction merge, and identify every missing PR URL, commit SHA, issue state, milestone association, root-help result, or licence-audit result that prevents the decision record from reaching `verified`

### Implementation for User Story 3

- [ ] T018 [US3] Append an implementation-verification section to `specs/003-define-naming-licence/research.md` with status `verified`, verification date, merged #47/#48/#107 PR URLs and commit SHAs, command/repository/licence evidence, all closed issue states, milestone `open_issues: 0`, and an explicit statement that release tagging was not performed
- [ ] T019 [US3] Re-run the independent test from `specs/003-define-naming-licence/quickstart.md` and cross-check `specs/003-define-naming-licence/research.md` against FR-013–FR-015 and SC-006–SC-007 in `specs/003-define-naming-licence/spec.md`; leave the record unverified if any evidence is missing or inconsistent
- [ ] T020 [US3] Commit the verified `specs/003-define-naming-licence/research.md` as `docs(speckit): record 003 naming and licence verification`, and land the verification-only PR on milestone `v0.12.0 Naming and licence` before any release audit or tag action

**Checkpoint**: All three stories are complete, #47, #48, and #107 are closed, and the local
decision record contains reproducible evidence rather than an inferred milestone state.

---

## Phase 6: Polish & Cross-Cutting Validation

**Purpose**: Prove the final repository state is green, scoped, and ready for the separate
release process without starting it.

- [ ] T021 Run every repository gate listed in Section 4 of `specs/003-define-naming-licence/quickstart.md`: formatting, module hygiene, vet, race/coverage tests, build, and integration tests; confirm CI coverage remains at least 80%
- [ ] T022 Audit `go.mod`, `go.sum`, `internal/`, `.github/workflows/release.yml`, `Dockerfile`, and `.goreleaser.yaml` against the exclusions in `specs/003-define-naming-licence/plan.md`; remove any report arithmetic, source parsing, dependency change, external-repository promise, or release/publish change that entered accidentally
- [ ] T023 Review `git status --short`, all milestone PR histories, and `scripts/check-linkage.sh` results against the commit boundaries in `specs/003-define-naming-licence/plan.md`; ensure Spec Kit installation changes remain separate and hand off milestone release readiness without creating or pushing a tag

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: starts immediately; T002 depends on T001.
- **Phase 2 (Foundational)**: depends on the spec-only PR from T002; T004 depends on the
  refreshed evidence from T003 and blocks all observable/licence edits.
- **Phase 3 (US1)**: depends on T004. T005 is test-first; T006 and T008 may then run in
  parallel; T007 depends on T006; T009 depends on T007 and T008; T010 depends on T009.
- **Phase 4 (US2)**: depends on #47 landing in T010. T011 is the pre-change audit; T012 and
  T013 may then run in parallel; T014 depends on both; T015 follows the audit; T016 depends
  on T015.
- **Phase 5 (US3)**: depends on #48 landing in T016. T017–T020 are sequential because each
  consumes the evidence produced by the preceding task.
- **Phase 6 (Polish)**: depends on the verification PR from T020. T021–T023 are sequential
  to avoid auditing while module hygiene or other gates may still alter the worktree.

### User Story Dependency Graph

```text
Spec-first setup -> approvals -> US1 / #47 -> US2 / #48 -> US3 verification -> polish
```

- **US1 (P1)**: first deliverable and suggested MVP; no dependency on another user story.
- **US2 (P1)**: depends on US1 because #48 names the library identity selected by #47.
- **US3 (P2)**: depends on both implementation stories and the #107 scope correction because
  it verifies their merged commits, issue closures, and milestone state.

### Within Each User Story

- Run the failing test/audit before implementation.
- Apply the smallest file-scoped changes that make it pass.
- Re-run the independent test before creating the story's commit/PR.
- Keep the issue commit green and include its README change in the same PR when the public
  surface changes.

## Parallel Opportunities

### User Story 1

After T005 establishes the failing contract, these tasks can run together:

```text
T006: create cmd/galaxio/report.go
T008: document the canonical names and report namespace in README.md
```

T007 waits for T006 because root registration must compile against the new constructor.

### User Story 2

After T011 captures the failing audit, these tasks can run together:

```text
T012: add the approved GPL-2.0-only notice to LICENSE
T013: make the README.md licence boundary explicit
```

T014 waits for both so the complete surface audit observes one consistent state.

### User Story 3

No safe parallel task is identified. The verification record requires merged #47 evidence,
then merged #48 evidence, then one consistent milestone snapshot.

## Implementation Strategy

### MVP First — User Story 1

1. Complete Phase 1 and land the spec-only PR.
2. Complete Phase 2 and record both approvals.
3. Write the failing US1 command tests.
4. Add the help-only `report` group and README naming summary.
5. Validate and land the one-commit #47 PR.
6. Stop: users and contributors now have the canonical vocabulary and discoverable command
   namespace without any report implementation.

### Incremental Delivery

1. **US1 / #47**: canonical names and root-help namespace.
2. **US2 / #48**: explicit, audited `GPL-2.0-only`/MIT boundary after #47 lands.
3. **Scope correction / #107**: remove the unrelated component from CLI documentation and
   governance rationale.
4. **US3**: durable merged-evidence record and zero-open-issue milestone verification.
5. **Polish**: all gates and scope audit; hand off to the separate release procedure.

### Commit and PR Boundaries

```text
docs(speckit): add 003-define-naming-licence spec/plan/tasks
feat(cli): reserve report command group (#47)
docs(licence): record compatible ecosystem boundary (#48)
docs(speckit): amend constitution to v1.1.1 (remove unrelated coupling)
docs(speckit): record 003 naming and licence verification
```

Each PR is assigned to milestone `v0.12.0 Naming and licence`. The #47, #48, and #107 PRs
carry their own closing keyword, one issue per commit. No task here authorizes an external
repository edit, release-workflow change, release tag, or publication.

## Notes

- `[P]` tasks touch different files and have no dependency on an incomplete parallel task.
- Every story task carries its `[USn]` label; setup, foundational, and polish tasks do not.
- Tests and audits are mandatory and precede the change they verify.
- `parsec` is named and audited but not imported.
- The installed `golang-spf13-cobra` skill is applied together with the existing
  parent-command pattern and the constitution as recorded in `research.md`.
