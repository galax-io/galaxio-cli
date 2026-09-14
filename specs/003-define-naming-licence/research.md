# Research: Define Ecosystem Naming and Licensing

Evidence was checked on 2026-09-14 against milestone
[`v0.12.0 Naming and licence`](https://github.com/galax-io/galaxio-cli/milestone/1),
issues [#47](https://github.com/galax-io/galaxio-cli/issues/47) and
[#48](https://github.com/galax-io/galaxio-cli/issues/48), with the scope correction tracked
in [#107](https://github.com/galax-io/galaxio-cli/issues/107), the current repository tree,
and the public `parsec` repository. Reproduction commands are collected in
[quickstart.md](quickstart.md).

This file is the authoritative decision record required by FR-013. The normative
requirements remain in [spec.md](spec.md).

## R1. Canonical component identities

**Decision**: Use exactly these canonical identities:

| Role | Canonical identity | Repository/module or invocation | Visibility |
|---|---|---|---|
| Result-primitives library | `parsec` | `github.com/galax-io/parsec` | Public |
| Finished-run CLI namespace | `report` | `galaxio report` | Public command help |

**Rationale**: The organization controls the public `galax-io/parsec` repository, which
resolves the library-name collision edge case within the only namespace that matters.
`report` is the noun a CLI user is most likely to try for finished-run output, and it does
not collide with an existing root command. The two names keep the shared primitives and
the user action distinct in later specifications.

**Alternatives considered**:

- `galaxio-results`: descriptive, but longer and inconsistent with the established
  organization register; rejected after the organization secured the concise repository
  path.
- A command name derived from the library repository: rejected because users seek an action,
  not an internal component.

## R2. Minimal `report` command surface

**Decision**: Add `cmd/galaxio/report.go` with `newReportCommand() *cobra.Command`, register
it in `newRootCommand()`, and make both `galaxio report` and `galaxio report --help` print
the namespace help with exit 0. Reject unexpected arguments through `cobra.NoArgs` wrapped
as `UsageError`, producing exit 2. Use `Report on finished load-test runs.` as the short
description; the long description must also say that this milestone reserves the command
group and that operational subcommands are introduced separately.

**Rationale**: Existing namespace commands `generate` and `template` keep help routing in
their Cobra constructors. `report` has no operational behavior in this milestone, so a
fake `runReport`, output struct, or JSON renderer would define an API the specification
explicitly leaves for later. The group is not experimental: hiding it would contradict
FR-005 and SC-003.

The repository currently declares `-o` on operational leaf commands, while root and parent
namespaces expose help without an output-format flag. The constitution describes `-o` as a
global flag, but broadening it is not part of #47. This plan follows the actual parent-group
contract and requires the first operational `report` subcommand to define its text/JSON
result and `runX` seam.

**Alternatives considered**:

- Add an empty report operation with `runReport` and `-o text|json`: rejected because an
  empty result is invented behavior and would become a compatibility-sensitive contract.
- Add `internal/report/` now: rejected because there is no source, calculation, or output
  requirement in this milestone.
- Feature-gate the group: rejected because the acceptance test requires first-inspection
  discoverability.

## R3. Command tests and documentation

**Decision**: Extend `TestHelpPrintsMinimalUsage` with `report`; add table-driven direct-help
coverage for `report` and `report --help`; and add an invalid-argument command test. All
tests use `runCLI` and assert exit code, stdout, and stderr. README gets the canonical names
and a short `galaxio report` usage entry with the #47 change.

**Rationale**: These assertions test the only behavior introduced. Existing parent-help
tests use stable substring checks rather than golden files, so matching that style avoids
a new convention and Cobra-version churn. There is no real external integration path or
generated artifact to justify an integration test or golden file.

**Alternatives considered**:

- Test `newReportCommand()` directly: rejected because it bypasses root registration and
  the public exit/output contract.
- Snapshot all help text: rejected because ordering and inherited Cobra help are not the
  requirement; presence, invocation, exit code, and stream placement are.

## R4. CLI and library licence boundary

**Decision**: Keep `galaxio-cli` under `GPL-2.0-only` and keep the public `parsec` library
under `MIT`. Do not relicense the CLI or import `parsec` in this feature. Preserve this
engineering compatibility decision with links to the FSF's
[Expat/MIT classification](https://www.gnu.org/licenses/license-list.html#Expat) and
[Apache-2.0 classification](https://www.gnu.org/licenses/license-list.html#apache2).

**Rationale**: The FSF classifies the Expat licence commonly identified by SPDX `MIT` as a
GPL-compatible free-software licence. It classifies Apache-2.0 as incompatible with GPLv2
by itself. Therefore an MIT `parsec` can be consumed by the GPL-2.0-only CLI without
relicensing either project, while future direct Apache-2.0 additions remain subject to a
separate approved decision. SPDX identifies the intended CLI expression as
[`GPL-2.0-only`](https://spdx.org/licenses/GPL-2.0-only.html).

**Alternatives considered**:

- Relicense the CLI to Apache-2.0: rejected because the milestone chose to retain the
  existing copyleft boundary; it would also be a separate licence change requiring explicit
  approval.
- Relicense the CLI to GPL-3.0-or-later: rejected because changing the public licensing
  promise is unnecessary when the shared library can use MIT.
- Give `parsec` Apache-2.0: rejected for this consumer because Apache-2.0 is not compatible
  with GPL version 2 by itself.

## R5. Licence-surface audit and normalization

**Decision**: Keep `LICENSE` byte-for-byte equal to the canonical GNU GPL Version 2 text so
GitHub's licence classifier can recognize it. Express the project's version-2-only
selection as `GPL-2.0-only` in README and the OCI image label. Guard these surfaces with a
deterministic repository test; do not mutate repository settings to compensate for a
non-canonical licence file.

| Surface | Observed 2026-09-14 | Required outcome |
|---|---|---|
| `LICENSE` | Canonical GNU GPL Version 2 text, SHA-256 `8177f97513213526df2cf6184d8ff986c675afb514d4e68a404010521b880643` | Keep byte-for-byte canonical; do not prepend project metadata |
| `Dockerfile` | `org.opencontainers.image.licenses="GPL-2.0-only"` | No change |
| `README.md` | States `GPL-2.0-only` and links `LICENSE` and this record | Keep the exact expression |
| GitHub repository metadata | Canonical file classifies as `GPL-2.0`; prepending the project notice regressed it to `NOASSERTION` | Require the GPL v2 classification and verify the PR branch before merge |
| `galax-io/parsec` | Public; `MIT` | Audit only; no external edit |

**Rationale**: The six-line project preamble preserved every GPL clause but changed GitHub's
classifier from `GPL-2.0` to `NOASSERTION`. The exact project selection does not need to be
embedded in the licence text: README and the OCI label are explicit repository-owned
metadata, while the canonical file gives tools a stable, recognized licence body. The
regression test checks all three without network access.

**Alternatives considered**:

- Prepend SPDX/project metadata to `LICENSE`: rejected because it breaks GitHub detection;
  use README and the OCI label for the exact project selection.
- Replace or edit clauses in the GPL text: rejected because the canonical file is both the
  licence text and the classifier input.
- Treat the README link alone as exact metadata: rejected because the exact SPDX expression
  and the canonical licence text are both required.
- Edit `parsec` from this repository: rejected by scope and the cross-repository ask-first
  boundary.

## R6. Durable decision location

**Decision**: Keep the normative contract in `spec.md`, decisions and dated evidence in
this `research.md`, executable validation in `quickstart.md`, and public summaries in
README. No separate ADR directory is introduced.

**Rationale**: This repository already treats `specs/NNN-*` as the versioned source for a
milestone. The record links both issues, the milestone, selected alternatives, rejected
alternatives, and evidence. README makes it discoverable without duplicating the full legal
and design analysis.

**Alternatives considered**:

- Create `docs/decisions/`: rejected because no ADR convention exists and two authorities
  would drift.
- Put the full decision only in GitHub issue comments: rejected because issue state is not
  versioned with the code or available offline.

## R7. Delivery and closure sequence

**Decision**: Land the Spec Kit artifacts first without closing either implementation issue.
Then land a separate one-commit #47 PR for the public namespace and its README entry. After
#47, land a separate one-commit #48 PR for explicit licence posture, with any `LICENSE`
notice subject to approval. Remove the unrelated component references through #107 before
final verification. Assign every PR to milestone 1 and use each closing reference only on
the corresponding implementation or correction PR.

**Rationale**: This satisfies the repository's spec-first, one-issue/one-commit, dependency
ordering, milestone, and closure rules. It also keeps the already-uncommitted Spec Kit
installation files out of feature commits.

**Alternatives considered**:

- Close both issues from the spec PR: rejected because root help still lacks `report` and
  README still lacks an explicit licence expression.
- Combine #47 and #48 in one implementation PR: rejected because they are separately
  tracked concerns and #48 depends on #47.
- Tag the release from either implementation PR: rejected because release execution starts
  only after both issues are merged/closed and the linkage audit passes.

## R8. Known pre-existing risk: Apache-2.0 dependencies

**Finding**: `go.mod` already directly depends on Cobra, which is Apache-2.0 licensed. That
dependency predates this feature, but it means the project must not generalize the
`parsec`-specific result into a claim that every dependency in the current binary has been
cleared for GPL-2.0-only distribution.

**Decision**: Do not add, upgrade, remove, or replace Cobra in this milestone. Record the
risk for a separate dependency/licensing review. SC-005 is evaluated narrowly as written:
there must be no unresolved conflict between `galaxio-cli` and the MIT `parsec` library.

**Rationale**: Replacing the CLI framework is unrelated, risky observable work. Ignoring
the existing dependency would make the decision record overclaim; expanding #48 into a
whole-module audit would make the naming milestone unbounded.

## R9. Skill guidance and project authority

**Finding**: The required `golang-spf13-cobra` skill named by the constitution is installed
after the Go skill-set refresh and was applied together with `golang-cli`,
`golang-error-handling`, `golang-testing`, `golang-naming`, and `golang-documentation`.
Where general testing guidance suggests third-party assertions or goroutine-leak tooling,
the repository constitution wins: this feature uses standard-library tests and adds no
dependency. No goroutine is introduced.
