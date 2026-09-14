# Implementation Plan: Define Ecosystem Naming and Licensing

**Branch**: `003-define-naming-licence` | **Date**: 2026-09-14 | **Spec**: [spec.md](spec.md)

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
`specs/003-define-naming-licence/` plus user-facing `README.md` and `LICENSE` notices.

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
flags, JSON output, or a module dependency; add a short `GPL-2.0-only` project notice before
the full GPL v2 terms without changing those verbatim terms, and only after the required
licence-sensitive approval.

**Scale/Scope**: One new command-group file, one root registration, command-level tests,
two focused README additions, and the spec/plan/research/contracts/quickstart artifacts.

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

| # | Gate (constitution principle) | Status |
|---|---|---|
| I | Command Contract — every new command is a thin cobra wrapper over `runX(ctx, opts)`; offers `-o text\|json` with a documented JSON structure; exits 0/1/2 via `UsageError`/`RuntimeError`; honours `--verbose`/`--quiet`/`--no-color`; experimental commands sit behind `internal/featureflags`. | PASS — `report` is a presentation-only namespace, not an operation: like existing `generate` and `template` parents, it validates `NoArgs`, maps misuse to `UsageError`, inherits root flags, and returns help with exit 0. It has no result to send through `runX` or `-o`; the first operational report subcommand must define both in its own specification. |
| II | Report Arithmetic Lives Here — statistics are computed in `internal/report/` over `parsec` primitives; one pass and bounded memory; absence stays absent; sources are detected by content. | PASS — this milestone reserves the public name only. It performs no arithmetic, reads no source, imports no `parsec`, and creates no `internal/report/`; Principle II becomes actionable in the later operational report feature. |
| III | Tests Land With The Change — stdlib `testing`, table-driven, command tests through `runCLI`, integration tags, race on, coverage ≥ 80%, regression tests mandatory. | PASS — root visibility and both direct help paths are asserted through `runCLI`; invalid arguments assert exit 2 and stderr. Substring assertions match the repository's help-test convention; no generated output or integration path exists, so no golden or integration fixture is warranted. Full race and coverage gates still run. |
| IV | Minimal, Explicit Dependencies — no new module without rationale, compatibility review, and approval. | PASS — no dependency changes. Cobra is already direct dependency truth; the pre-existing Apache-2.0/GPL-2.0-only tension is recorded in [research.md](research.md) rather than expanded by this feature. |
| V | Published Surfaces — list command/flag/output changes; approve breaking changes; update README in the same PR; preserve deprecations. | PASS — the only binary surface is the additive, non-breaking `report` parent name and help text. No existing command, flag, default, exit code, JSON structure, schema, or generated output changes. README documentation lands with #47; licence wording lands with #48. The observable command addition still requires maintainer approval before implementation under the project ask-first rule. |
| VI | Idiomatic, Simple Go — gofmt/vet clean; errors as values; no panic flow, duplication, speculative abstraction, or out-of-scope refactor. | PASS — the design mirrors the existing parent-command constructor, adds no fake runner or premature abstraction, and confines code changes to `cmd/galaxio/report.go`, root registration, and behavior tests. |

**Post-design re-check**: all gates remain PASS. The CLI contract and licence-posture
contracts narrow the feature to observable help and auditable metadata; they do not add
report behavior or a new dependency. The apparent global `-o` mismatch is an existing
root/parent-command convention and is explicitly deferred rather than silently broadened.

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
LICENSE                           # #48: add approved GPL-2.0-only project notice before the
                                  # unchanged verbatim GPL v2 terms
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

1. Land the complete `specs/003-define-naming-licence/` artifact set first as
   `docs(speckit): add 003-define-naming-licence spec/plan/tasks`, assigned to milestone
   `v0.12.0`. Reference #47 and #48 but do not close either issue with the planning PR.
2. After explicit approval of the additive public command, land one #47 PR/commit such as
   `feat(cli): reserve report command group (#47)`. Include `report.go`, root registration,
   all command-level tests, and the required README naming/usage entry; the PR body carries
   `Closes #47` and milestone 1.
3. After #47 lands, obtain the licence-sensitive approval and land one #48 PR/commit such
   as `docs(licence): record compatible ecosystem boundary (#48)`. Make README state
   `GPL-2.0-only` explicitly and add the approved project-selection notice to `LICENSE`
   without editing the verbatim GPL terms. Document GitHub's legacy
   `GPL-2.0` classifier as the platform representation of the v2-only file; the PR body
   carries `Closes #48` and milestone 1.
4. Do not import or modify `parsec`, change a release workflow, close the milestone, or
   create/push `v0.12.0` in these PRs. Once both fixes and the #107 scope correction are on
   `main` and their issues are closed, the separate release procedure begins with the
   mandated linkage audit.
