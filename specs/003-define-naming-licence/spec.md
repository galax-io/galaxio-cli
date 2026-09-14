# Feature Specification: Define Ecosystem Naming and Licensing

**Feature Branch**: `003-define-naming-licence`

**Created**: 2026-09-14

**Status**: Draft

**Input**: User description: "https://github.com/galax-io/galaxio-cli/milestone/1"

**Tracking**: [milestone `v0.12.0 Naming and licence`](https://github.com/galax-io/galaxio-cli/milestone/1);
[galax-io/galaxio-cli#47](https://github.com/galax-io/galaxio-cli/issues/47) and
[galax-io/galaxio-cli#48](https://github.com/galax-io/galaxio-cli/issues/48), with the
[scope correction #107](https://github.com/galax-io/galaxio-cli/issues/107)

## Background

The milestone establishes names and licensing boundaries for the two related CLI/reporting
components before later work publishes interfaces that would be expensive to rename:

- `parsec` is the public load-test result-primitives library at
  `github.com/galax-io/parsec`;
- `report` is the user-facing command group, invoked as `galaxio report`.

The selected licensing boundary keeps `galaxio-cli` under `GPL-2.0-only` and the public
`parsec` library under `MIT`. This permits the CLI to consume the shared library while the
library remains usable by permissively licensed and closed-source consumers. The project
relies on the Free Software Foundation's classification of the Expat licence commonly
identified by the SPDX identifier `MIT` as
[GPL-compatible](https://www.gnu.org/licenses/license-list.html#Expat); the same source
classifies [Apache-2.0 as incompatible with GPL version 2 by itself](https://www.gnu.org/licenses/license-list.html#apache2),
so it remains unsuitable for direct inclusion in a GPL-2.0-only program without a separate
licensing decision.

This specification turns those existing choices into one durable milestone contract. It
does not define report calculations or the public library API.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Use one canonical component vocabulary (Priority: P1)

A maintainer or downstream contributor can name every ecosystem component without relying
on placeholders or revisiting alternatives. Repository references, planning documents, and
user-facing command help consistently use `parsec` and `report` for their distinct
roles.

**Why this priority**: Published repository, module, and command names become dependencies
for later milestones. Settling them first prevents redirects, import-path churn, and two
names for the same capability.

**Independent Test**: Review the authoritative decision record, organization repositories,
active planning documents, and root command help; each of the two roles resolves to
exactly one canonical name and no active placeholder remains.

**Acceptance Scenarios**:

1. **Given** a contributor planning result ingestion, **When** they look up the public
   result-primitives library, **Then** the canonical repository and module identity is
   `github.com/galax-io/parsec`.
2. **Given** a CLI user looking for finished-run reporting, **When** they inspect root help,
   **Then** `report` appears as the command group and is invoked as `galaxio report`.
3. **Given** a later issue or specification in this ecosystem, **When** it refers to either
   component, **Then** it uses the corresponding canonical name rather than a
   descriptive placeholder or rejected alternative.

---

### User Story 2 - Consume the shared library without a licence conflict (Priority: P1)

A maintainer can include the public result-primitives library in the CLI while preserving
the CLI's existing copyleft licence. Contributors and consumers can identify both licence
identities and the compatibility rationale from durable project records.

**Why this priority**: An unresolved incompatibility would block the dependency that later
reporting work requires and would put every resulting release under avoidable legal
uncertainty.

**Independent Test**: Audit the CLI licence surfaces, the public library licence identity,
and the recorded compatibility source; all CLI surfaces identify `GPL-2.0-only`, the
library identifies `MIT`, and the review contains no unresolved incompatibility between
them.

**Acceptance Scenarios**:

1. **Given** the CLI repository, **When** a consumer checks its licence file, container
   metadata, readme, or repository metadata, **Then** every surface identifies the same
   `GPL-2.0-only` licence.
2. **Given** the public `parsec` repository, **When** a consumer checks its licence, **Then**
   it identifies the SPDX licence `MIT` and retains the required notice.
3. **Given** the CLI consuming `parsec`, **When** a maintainer reviews licence
   compatibility, **Then** the recorded rationale cites an authoritative classification
   that permits the combination.
4. **Given** a proposed direct dependency under Apache-2.0, **When** it is evaluated for
   the GPL-2.0-only CLI, **Then** it remains blocked until a separate approved licensing
   decision resolves the incompatibility.

---

### User Story 3 - Close the milestone on auditable evidence (Priority: P2)

A maintainer can close the naming and licensing issues using evidence rather than an
implicit convention. Later milestones can cite the decision and proceed without reopening
the same alternatives.

**Why this priority**: The milestone cannot produce its deliberate release while either
decision remains open, and future work needs a stable reference for both choices.

**Independent Test**: Verify that the decision record names the selected alternatives,
explains why they satisfy the milestone, links both tracked issues, and provides acceptance
evidence sufficient to close them when the specification work lands.

**Acceptance Scenarios**:

1. **Given** issue #47, **When** its acceptance evidence is reviewed, **Then** it shows both
   selected repository names and the `report` command name without unresolved alternatives.
2. **Given** issue #48, **When** its acceptance evidence is reviewed, **Then** it shows
   consistent CLI licensing and a compatible permissive licence for the shared library.
3. **Given** both issue fixes and the scope correction are on the milestone's release
   commit, **When** the milestone is audited, **Then** #47, #48, and #107 are closed and the
   milestone has no remaining open work.

### Edge Cases

- A generic or well-known word such as `parsec` exists elsewhere: the canonical
  identity includes the `galax-io` organization, and organization ownership of the exact
  repository path is the deciding evidence.
- The term “MIT licence” is used ambiguously: project records use the SPDX identifier `MIT`,
  preserve its exact notice, and connect it to the FSF's Expat compatibility entry.
- Root help already contains a command with a conflicting name: the `report` name is not
  accepted until the conflict is resolved without silently replacing an existing command.
- A licence surface cannot be inspected or disagrees with the others: milestone acceptance
  fails rather than inferring the intended licence from another surface.
- A later proposal seeks to rename a published component: that is a separate compatibility
  change and does not reopen this milestone implicitly.

## Requirements *(mandatory)*

### Functional Requirements

**Canonical identities**

- **FR-001**: The decision record MUST designate `parsec` as the sole canonical name for
  the public result-primitives library and MUST identify its repository and module path as
  `github.com/galax-io/parsec`.
- **FR-002**: The decision record MUST designate `report` as the sole canonical CLI command
  group name and MUST show its invocation as `galaxio report`.
- **FR-004**: The organization MUST control the selected `parsec` repository path and it
  MUST be publicly discoverable.
- **FR-005**: Root CLI help MUST list `report` as a command group without changing or
  removing an existing published command.
- **FR-006**: Active specifications, contributor guidance, and user-facing documentation
  that refer to these components MUST use the canonical names and MUST NOT introduce a
  second alias or unresolved placeholder.

**Licence boundary**

- **FR-007**: `galaxio-cli` MUST remain licensed as `GPL-2.0-only`; no relicensing of the
  CLI is part of this feature.
- **FR-008**: Every CLI licence surface—including the licence file, container metadata,
  readme, and repository metadata—MUST identify `GPL-2.0-only` consistently.
- **FR-009**: The public `parsec` library MUST use the SPDX licence `MIT`, preserve the
  licence notice required by that licence, and remain usable by consumers outside the CLI.
- **FR-010**: The decision record MUST explain that the selected `MIT`/Expat licence is
  classified as GPL-compatible and MUST cite the authoritative source used for that
  conclusion.
- **FR-011**: The decision record MUST also preserve the boundary that Apache-2.0 is not
  compatible with GPL version 2 by itself; a direct Apache-2.0 dependency MUST require a
  separate, explicit licensing decision before inclusion.
- **FR-012**: This feature MUST NOT relicense any other Galaxio repository.

**Decision traceability**

- **FR-013**: A durable record MUST link milestone `v0.12.0 Naming and licence`, issues #47
  and #48, and scope correction #107; state the chosen alternatives; summarize the rejected
  alternatives; and explain why the selections meet downstream needs.
- **FR-014**: Evidence for issue #47 MUST include repository-path ownership and root-help
  visibility; evidence for issue #48 MUST include the consistent licence-surface audit and
  the compatibility classification.
- **FR-015**: When the corresponding fixes and scope correction land on the milestone's
  release commit, their pull requests MUST close #47, #48, and #107 so the milestone records
  no completed work as open.

### Key Entities

- **Component identity**: A canonical name, role, organization repository path, visibility,
  and—where applicable—user-facing invocation for one ecosystem component.
- **Licence posture**: The SPDX licence identity and visible metadata for a repository,
  together with the obligations relevant to its consumers.
- **Compatibility decision**: The reviewed relationship between the CLI's
  `GPL-2.0-only` licence and the public library's `MIT` licence, including its authoritative
  evidence and the incompatible alternatives kept out of scope.
- **Decision record**: The durable milestone artifact linking selections, rationale,
  rejected alternatives, tracked issues, and acceptance evidence.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Both in-scope component roles have exactly one canonical identity across the
  decision record and active user-facing documentation: `parsec` and `report`.
- **SC-002**: The selected `parsec` repository path resolves under `galax-io` and remains
  publicly discoverable.
- **SC-003**: A user can find `galaxio report` from root help on the first inspection,
  without knowing an internal project or repository name.
- **SC-004**: 100% of the audited CLI licence surfaces agree on `GPL-2.0-only`, and the
  public library is identified as `MIT`; the audit reports zero unresolved mismatches.
- **SC-005**: The compatibility review reports zero unresolved licence conflicts blocking
  `galaxio-cli` from consuming `parsec` under the selected licences.
- **SC-006**: A downstream maintainer can identify both names, both in-scope licence
  identities, and the compatibility rationale from the decision record without requesting
  clarification.
- **SC-007**: Issues #47, #48, and #107 are closed when their changes land, reducing the
  milestone's open issue count to zero and making it eligible for its deliberate release
  audit.

## Assumptions

- The repository's ratified constitution is authoritative: it already uses `parsec` and
  `galaxio report`, keeps the CLI under `GPL-2.0-only`, and identifies `parsec`
  as MIT-licensed.
- The existing `galax-io/parsec` repository is conclusive evidence that the organization
  controls the selected library name; this feature does not create, rename, or edit that
  external repository.
- The selected licence strategy is the fallback described in issue #48: preserve the CLI's
  existing GPL-2.0-only licence and keep the shared public library under MIT. The project
  records an engineering compatibility decision, not individualized legal advice.
- Adding the `report` command group establishes its public name and discoverability only.
  Report inputs, outputs, calculations, flags, and operational behaviour belong to later
  specifications.
- Issue #48 remains ordered after #47 because the licensing record refers to the selected
  library identity.
- The rejected naming alternative (`galaxio-results`) and CLI relicensing alternatives
  (Apache-2.0 or GPL-3.0-or-later) remain historical context, not active aliases or
  requirements.
- Out of scope: package naming inside `parsec`; report calculations; changes to other
  repositories; and milestone release/tag execution.
