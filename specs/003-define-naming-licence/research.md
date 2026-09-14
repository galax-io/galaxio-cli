# Research: Define Ecosystem Naming and Licensing

Evidence was checked on 2026-09-14 against milestone
[`v0.12.0 Naming and licence`](https://github.com/galax-io/galaxio-cli/milestone/1),
issues [#47](https://github.com/galax-io/galaxio-cli/issues/47) and
[#48](https://github.com/galax-io/galaxio-cli/issues/48), the current repository tree, and
the organization repositories visible to the authenticated maintainer. Reproduction
commands are collected in [quickstart.md](quickstart.md).

This file is the authoritative decision record required by FR-013. The normative
requirements remain in [spec.md](spec.md).

## R1. Canonical component identities

**Decision**: Use exactly these canonical identities:

| Role | Canonical identity | Repository/module or invocation | Visibility |
|---|---|---|---|
| Result-primitives library | `parsec` | `github.com/galax-io/parsec` | Public |
| Live-metrics sidecar | `comet` | `galax-io/comet` | Private |
| Finished-run CLI namespace | `report` | `galaxio report` | Public command help |

**Rationale**: The organization already controls both repository paths. GitHub reports
`galax-io/parsec` as public and `galax-io/comet` as private, which resolves the collision
edge case within the only namespace that matters. `report` is the noun a CLI user is most
likely to try for finished-run output, and it does not collide with an existing root
command. The three names are distinct enough to keep the library, live sidecar, and user
action separate in later specifications.

**Alternatives considered**:

- `galaxio-results` and `galaxio-tail`: descriptive, but longer and inconsistent with the
  established organization register; rejected after the organization secured the concise
  repository paths.
- A descriptive public-library name with `comet` for the private component: rejected
  because it creates two naming systems for one ecosystem.
- A command name derived from either repository: rejected because users seek an action,
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

**Decision**: Treat the repository's current GPL v2 terms, exact container label, and
GitHub classifier as evidence of the retained licence, then make the project selection
unambiguous in README and—only after licence-sensitive maintainer approval—in a short
project-specific `GPL-2.0-only` notice placed before the unchanged GPL text.
Do not rewrite the verbatim licence terms or mutate repository settings solely to chase a
different GitHub API spelling.

| Surface | Observed 2026-09-14 | Required outcome |
|---|---|---|
| `LICENSE` | Verbatim GNU GPL Version 2 terms; no project-specific `only` notice | After approval, prepend an explicit `GPL-2.0-only` project notice and retain the licence body verbatim |
| `Dockerfile` | `org.opencontainers.image.licenses="GPL-2.0-only"` | No change |
| `README.md` | Links to `LICENSE` but does not name the expression | State `GPL-2.0-only` explicitly and link the decision record |
| GitHub repository metadata | `gpl-2.0` / “GNU General Public License v2.0” | Document as GitHub's classifier for the recognized v2 file; verify it still resolves after any notice change |
| `galax-io/parsec` | Public; `MIT` | Audit only; no external edit |
| `galax-io/comet` | Private; GitHub reports another/unspecified licence | Make no licence commitment; visibility only is in scope |

**Rationale**: README is currently the only repository-owned surface that is too implicit.
The same GPL v2 text is used for both `only` and `or-later` expressions; SPDX notes that a
project notice distinguishes them. Adding such a notice is a licensing-sensitive act even
when it preserves the intended policy, so it must be reviewed before implementation. The
GitHub API's `gpl-2.0` key is not itself a repository-controlled SPDX declaration and cannot
be normalized through a code-only change; the displayed licence name and detected file are
the auditable evidence.

**Alternatives considered**:

- Replace or edit clauses in the GPL text: rejected because the licence permits verbatim
  copies and says changing the document is not allowed.
- Treat the README link alone as exact metadata: rejected because it does not distinguish
  `GPL-2.0-only` from `GPL-2.0-or-later`.
- Edit `parsec` or `comet` from this repository: rejected by scope and the cross-repository
  ask-first boundary.

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

**Decision**: Land the Spec Kit artifacts first without closing either issue. Then land a
separate one-commit #47 PR for the public namespace and its README entry. After #47, land a
separate one-commit #48 PR for explicit licence posture, with any `LICENSE` notice subject
to approval. Assign every PR to milestone 1 and use `Closes #47` / `Closes #48` only on the
corresponding implementation PR.

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
