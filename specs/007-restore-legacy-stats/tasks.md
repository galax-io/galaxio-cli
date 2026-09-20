# Tasks: Restore legacy Gatling statistics files

**Input**: [spec.md](spec.md), [plan.md](plan.md)
**Branch**: `galaxio/120-restore-legacy-stats`
**Milestone**: #5 — v0.15.0 Legacy stats.json; issue #52

**Tests**: Every implementation task includes standard-library tests in the same green
commit. No Jenkins fixture, JVM oracle, operating-system matrix or new dependency is part
of this task list.

**Delivery**: Keep all tasks in one milestone PR. Each task maps to one semantic, green
commit; update its checkbox in that commit.

## Phase 1: Specification baseline

**Purpose**: Establish the approved scope before production changes.

- [x] T001 Commit the restored specification, concise implementation plan, quality checklist and this task list in `specs/007-restore-legacy-stats/`

---

## Phase 2: Shared report foundation

**Purpose**: Compute the whole-run and per-position views once for both legacy products.

- [x] T002 Add one-pass request/group aggregation and validated total/OK/KO export views with focused tests in `internal/report/scan.go`, `internal/report/source.go`, `internal/report/tree.go`, `internal/report/export.go` and adjacent `*_test.go` files

**Checkpoint**: A completed scan exposes bounded-memory root and position statistics without
changing the existing human report.

---

## Phase 3: User Story 1 — Export whole-run statistics (Priority: P1) 🎯 MVP

**Goal**: Write only `js/global_stats.json` for the selected run and remain silent on
success.

**Independent Test**: Run the real CLI against each committed Gatling 3.11.5–3.15.1 corpus
entry with `-o global_stats`; decode the legacy schema and compare whole-run counts.

- [x] T003 [US1] Implement product selection, the 15-field legacy statistics document and complete-file publication with unit tests in `internal/report/legacy/legacy.go`, `internal/report/legacy/publish.go` and adjacent `*_test.go` files
- [x] T004 [US1] Activate `-o global_stats`, four-rank validation, silent export and corpus command tests in `cmd/galaxio/report.go` and `cmd/galaxio/report_export_test.go`

**Checkpoint**: The global product works independently and no-output invocations retain the
existing human report.

---

## Phase 4: User Story 2 — Export request/group hierarchy (Priority: P1)

**Goal**: Write `js/stats.json` with every observed request/group position and permit both
products in one invocation.

**Independent Test**: Export nested and repeated positions with Unicode, quotes and
backslashes; decode the tree and compare its root statistics with `global_stats.json`.

- [x] T005 [US2] Render the legacy group/request hierarchy, stable collision-safe identifiers, original decoded names and combined product selection with tests in `internal/report/legacy/legacy.go`, `internal/report/legacy/legacy_test.go` and `cmd/galaxio/report_export_test.go`

**Checkpoint**: `stats`, `global_stats` and their combined selection are independently
usable; distinct positions are not merged by display name.

---

## Phase 5: User Story 3 — Preserve existing artifacts (Priority: P1)

**Goal**: Refuse collisions by default and replace only explicitly selected regular files.

**Independent Test**: Exercise existing selected files, unselected sentinels, directories,
symlinks, invalid selections and unusable logs; verify exit class, named path and unchanged
bytes.

- [x] T006 [US3] Complete overwrite, all-selected preflight and unsafe-destination regression coverage in `internal/report/legacy/publish.go`, `internal/report/legacy/publish_test.go` and `cmd/galaxio/report_export_test.go`

**Checkpoint**: No selected or unrelated artifact changes without explicit authorization;
newly-created failed outputs are removed.

---

## Phase 6: User Story 4 — Reproducible calculated statistics (Priority: P2)

**Goal**: Produce stable decoded documents using the report command's existing arithmetic
and no explanatory output or companion files.

**Independent Test**: Repeat exports with default and custom ranks/bounds, empty outcomes and
escaped names; compare decoded documents, root/global equality, file inventory and streams.

- [x] T007 [US4] Add repeatability, custom option, empty-outcome and selected-only inventory regressions in `cmd/galaxio/report_export_test.go` and `internal/report/legacy/legacy_test.go`
- [x] T008 [US4] Document the final invocation and run formatting, tidy, vet, race/coverage, build, integration and shell-suite gates in `README.md` and `specs/007-restore-legacy-stats/validation/quality.md`

**Checkpoint**: All four stories and every success criterion are covered without external
test infrastructure.

---

## Dependencies and execution order

```text
T001 → T002 → T003 → T004 → T005 → T006 → T007 → T008
```

- T002 is the shared one-pass foundation.
- T003/T004 deliver the independently usable global MVP.
- T005 adds the hierarchy without changing global semantics.
- T006 hardens publication after both products exist.
- T007 verifies cross-product reproducibility.
- T008 records the final repository gates.

## Parallel opportunities

The implementation is intentionally serialized because the small task set shares the report
scan, renderer and command files. Within T006, publisher unit cases and command-level refusal
cases can be prepared independently before the single task commit. Within T007, renderer and
CLI regressions touch different test files and can be prepared in parallel.

## Requirement traceability

| Coverage | Tasks |
|---|---|
| FR-001–FR-003, SC-001 | T003–T005 |
| FR-004–FR-011, SC-002–SC-003, SC-005 | T002–T005, T007 |
| FR-012–FR-014, FR-018, SC-004 | T006 |
| FR-015–FR-017, SC-006–SC-007 | T002, T004, T007 |
| FR-019–FR-020 | T004–T008 |

## Implementation strategy

1. Deliver T001–T004 as the global-file MVP.
2. Add the hierarchy in T005.
3. Finish collision safety in T006.
4. Pin repeatability and complete repository validation in T007–T008.
5. Leave the milestone PR open for maintainer review; do not merge, close the issue or tag a
   release.
