# Implementation Plan: Define Ecosystem Naming and Licensing

**Branch**: `galaxio/109-restore-review-workflow` | **Date**: 2026-09-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/003-define-naming-licence/spec.md`
(milestone [`v0.12.0 Naming and licence`](https://github.com/galax-io/galaxio-cli/milestone/1),
issues [#47](https://github.com/galax-io/galaxio-cli/issues/47) and
[#48](https://github.com/galax-io/galaxio-cli/issues/48), plus the
[scope correction #107](https://github.com/galax-io/galaxio-cli/issues/107))

## Summary

Make `parsec` and `report` the durable vocabulary for the public result-primitives library
and CLI reporting namespace. Add a help-only
`galaxio report` parent command that is visible from root help, document the names and the
`GPL-2.0-only`/`MIT` boundary, and record the compatibility evidence needed to close #47
and #48. This milestone deliberately adds no report subcommand, calculation, input/output
contract, `parsec` import, or external-repository change.

## Technical Context

**Language/Version**: Go 1.27.1 (`go.mod`, `Dockerfile`).

**Primary Dependencies**: `github.com/spf13/cobra v1.10.2` for the existing command tree.
No new or upgraded dependency; `parsec` is named but is not imported in this feature.

**Storage**: N/A. The durable state is versioned Markdown under
`specs/003-define-naming-licence/`, the user-facing `README.md`, and the canonical GPL v2
text in `LICENSE`.

**Testing**: Standard-library `testing`; command-level assertions through `runCLI` for root
help, `galaxio report`, `galaxio report --help`, and invalid arguments. Repository gates are
`go vet ./...`, `go test -race ./...`, `go build ./...`, and the existing 80% coverage gate.

**Target Platform**: The existing statically linked, cross-platform CLI targets: macOS,
Linux, and Windows on amd64/arm64, plus the distroless Linux container.

**Project Type**: Go CLI and versioned decision documentation.

**Performance Goals**: N/A for the help-only namespace and documentation. No result stream
is opened and no report arithmetic is introduced.

**Constraints**: Preserve `GPL-2.0-only`; do not edit `galax-io/parsec`;
do not change a release workflow or publish a tag; do not add `internal/report/`, report
flags, JSON output, or a module dependency; keep `LICENSE` byte-for-byte canonical so
GitHub can classify it, and express the project's `GPL-2.0-only` selection in README and
container metadata.

**Scale/Scope**: One new command-group file, one root registration, command-level tests,
two focused README additions, and the spec/plan/research/contracts/quickstart artifacts.

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

| # | Gate (constitution principle) | Status |
|---|---|---|
| I | Command Contract — operational commands use `runX(ctx, opts)` and `-o text\|json`; help-only namespace parents may route to help without either; all commands honour the root flags and 0/1/2 exit contract. | PASS — `report` is a help-only namespace like `generate` and `template`: it validates `NoArgs`, maps misuse to `UsageError`, inherits `--verbose`/`--quiet`/`--no-color`, and returns help with exit 0. It has no result to send through `runX` or `-o`; the first operational report subcommand must define both in its own specification. |
| II | Report Arithmetic Lives Here — statistics are computed in `internal/report/` over `parsec` primitives; one pass and bounded memory; absence stays absent; sources are detected by content. | PASS — this milestone reserves the public name only. It performs no arithmetic, reads no source, imports no `parsec`, and creates no `internal/report/`; Principle II becomes actionable in the later operational report feature. |
| III | Tests Land With The Change — stdlib `testing`, table-driven, command tests through `runCLI`, integration tags, race on, coverage ≥ 80%, regression tests mandatory. | PASS — root visibility and both direct help paths are asserted through `runCLI`; invalid arguments assert exit 2 and stderr. Substring assertions match the repository's help-test convention; no generated output or integration path exists, so no golden or integration fixture is warranted. Full race and coverage gates still run. |
| IV | Minimal, Explicit Dependencies — no new module without rationale, compatibility review, and approval. | PASS — no dependency changes. Cobra is already direct dependency truth; the pre-existing Apache-2.0/GPL-2.0-only tension is recorded in [research.md](research.md) rather than expanded by this feature. |
| V | Published Surfaces — list command/flag/output changes; approve breaking changes; update README in the same PR; preserve deprecations. | PASS — the only binary surface is the additive, non-breaking `report` parent name and help text. No existing command, flag, default, exit code, JSON structure, schema, or generated output changes. README documents both the namespace and licence posture. The completed milestone remains pending maintainer review in PR #110. |
| VI | Idiomatic, Simple Go — gofmt/vet clean; errors as values; no panic flow, duplication, speculative abstraction, or out-of-scope refactor. | PASS — the design mirrors the existing parent-command constructor, adds no fake runner or premature abstraction, and confines code changes to `cmd/galaxio/report.go`, root registration, and behavior tests. |

**Post-design re-check**: all gates remain PASS. The CLI contract and licence-posture
contracts narrow the feature to observable help and auditable metadata; they do not add
report behavior or a new dependency. Constitution v2.0.0 now states the existing help-only
namespace convention directly, so there is no `runX` or `-o` exception hidden in the plan.

## Project Structure

### Documentation (this feature)

```text
specs/003-define-naming-licence/
├── plan.md                       # This implementation plan
├── research.md                   # Authoritative decisions, evidence, alternatives, risks
├── data-model.md                 # Conceptual identities, licence posture, decision record
├── quickstart.md                 # Reproducible CLI, licence, issue, and gate verification
├── contracts/
│   ├── cli-help.md               # `galaxio report` discoverability and exit contract
│   └── licence-posture.md        # Surface audit and compatibility boundary
├── checklists/
│   └── requirements.md           # Specification-quality checklist
├── spec.md                       # Normative feature requirements
└── tasks.md                      # Phase 2 output from /speckit-tasks; not created here
```

### Source Code (repository root)

```text
cmd/galaxio/
├── report.go                     # NEW: help-only `report` parent command
├── root.go                       # Register `newReportCommand()`
└── root_test.go                  # Root visibility, report help, argument/exit tests

README.md                         # #47 canonical names/report usage; #48 explicit licence boundary
LICENSE                           # #48: canonical, unmodified GNU GPL Version 2 text
scripts/license_surface_test.sh  # Regression guard for licence text and exact metadata
Dockerfile                        # audit only: already labels `GPL-2.0-only`
go.mod                            # unchanged: no parsec import or dependency update
.goreleaser.yaml                  # audit only; no publishing/release change
```

**Structure Decision**: Keep command discovery beside the other Cobra constructors and keep
the milestone decision where this repository already versions feature contracts. No new
`docs/decisions/` hierarchy or runtime package is justified. `research.md` is the
authoritative decision record; README contains the concise user-facing summary and links
back to it.

## Complexity Tracking

No constitution violations; table intentionally empty.

## Implementation Notes for /speckit-tasks

The dirty worktree contains Spec Kit installation changes. They remain a separate concern
and MUST NOT be swept into the feature's spec or implementation commits.

1. Keep specification, implementation, correction, and validation work in milestone PR
   #110. Do not open a replacement, documentation-only, or stacked PR.
2. Preserve one task per green commit. The already-merged milestone commits are historical
   evidence; corrective tasks T024–T027 each receive exactly one commit in PR #110.
3. Keep `Closes #47`, `Closes #48`, `Closes #104`, `Closes #107`, and `Closes #109` on the
   milestone PR so GitHub closes them only when the reviewed PR lands on `main`. Until then,
   the issues and the milestone remain open.
4. The agent may update PR #110 and its evidence but MUST leave it open. Only explicit
   maintainer instruction after review authorizes merge or PR closure.
5. Do not import or modify `parsec`, change a release workflow, close the milestone, run a
   release audit, or create/push `v0.12.0` while preparing this PR.
