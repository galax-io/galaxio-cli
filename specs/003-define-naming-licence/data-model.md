# Data Model: Define Ecosystem Naming and Licensing

This feature introduces no runtime persistence. The model below defines the conceptual
records that the versioned specification, research, README, CLI help, and repository
metadata must agree on.

## 1. Component Identity

Represents one stable ecosystem name.

| Field | Type | Rules |
|---|---|---|
| `canonicalName` | string | Required; exactly one active name per role |
| `role` | enum | `result-primitives-library` or `finished-run-cli-namespace` |
| `organization` | string | `galax-io` for repository-backed components |
| `repositoryPath` | string, optional | Organization-qualified path when the component has a repository |
| `modulePath` | string, optional | Required for the public Go library; omitted for the CLI namespace |
| `visibility` | enum | `public` or `not-applicable` |
| `invocation` | string, optional | Required for the CLI namespace |
| `aliases` | list of string | Must be empty for active documentation; rejected alternatives live in the decision record |

### Canonical instances

| `canonicalName` | `role` | `repositoryPath` / `modulePath` | `visibility` | `invocation` |
|---|---|---|---|---|
| `parsec` | result-primitives-library | `github.com/galax-io/parsec` | public | — |
| `report` | finished-run-cli-namespace | — | not-applicable | `galaxio report` |

### Validation

- No two instances may own the same role.
- `parsec` must resolve as a public repository controlled by `galax-io`.
- `report` must be present in root CLI help and must not replace an existing command.
- Active documentation must use `canonicalName`; an alternative is never an alias.

## 2. Licence Posture

Represents a repository's selected licence identity and visible evidence.

| Field | Type | Rules |
|---|---|---|
| `subject` | repository path | Required |
| `spdxExpression` | string or absent | Exact for in-scope public repositories |
| `visibility` | enum | `public` |
| `surfaces` | list of evidence | File, README, container, and repository metadata as applicable |
| `noticeRequired` | boolean | Whether redistribution must retain a notice |
| `scopeStatus` | enum | `selected`, `audit-only`, or `out-of-scope` |

### In-scope instances

| `subject` | `spdxExpression` | `noticeRequired` | `scopeStatus` |
|---|---|---:|---|
| `github.com/galax-io/galaxio-cli` | `GPL-2.0-only` | true | selected |
| `github.com/galax-io/parsec` | `MIT` | true | selected; external surface is audit-only in this repository |

### Validation

- Every audited CLI surface must identify GPL version 2 only or a documented platform
  classifier for that same selection.
- README must contain the exact `GPL-2.0-only` expression.
- The OCI label must equal `GPL-2.0-only`.
- `LICENSE` remains byte-for-byte canonical GPL v2 text so repository classifiers can
  recognize it; README and the OCI label carry the exact `GPL-2.0-only` project selection.
- `parsec` must retain its MIT notice; this feature only audits it.

## 3. Compatibility Decision

Captures whether a library may be consumed by the CLI under the selected licences.

| Field | Type | Rules |
|---|---|---|
| `consumer` | Licence Posture reference | `galaxio-cli` / `GPL-2.0-only` |
| `dependency` | Licence Posture reference | Proposed library licence |
| `classification` | enum | `compatible`, `blocked`, or `requires-separate-review` |
| `authority` | URL | Required primary classification source |
| `scope` | string | Must identify the exact dependency relationship reviewed |
| `notes` | string | Obligations, limits, and pre-existing risks |

### Decisions

| Dependency licence | Classification | Authority |
|---|---|---|
| `MIT` for `parsec` | compatible | FSF Expat entry |
| `Apache-2.0` for a future direct dependency | requires-separate-review / blocked pending approval | FSF Apache-2.0 entry |

The decision does not certify the entire existing dependency graph. In particular, Cobra's
pre-existing Apache-2.0 licence is tracked as a separate audit risk.

## 4. Decision Record

Links selections to delivery evidence.

| Field | Type | Rules |
|---|---|---|
| `milestone` | URL + title | Must be milestone 1, `v0.12.0 Naming and licence` |
| `issues` | ordered list | #47, then dependent #48 |
| `selectedAlternatives` | Component Identity / Licence Posture references | Required |
| `rejectedAlternatives` | list with rationale | Required; never promoted as aliases |
| `evidence` | list | Repository ownership, CLI help, licence surfaces, compatibility sources |
| `status` | enum | `draft`, `implemented`, `in-review`, `verified` |
| `verifiedAt` | date, optional | Set only after the reviewed milestone PR lands and all linked issues close |

### State transitions

```text
draft -> implemented -> in-review -> verified
```

- `draft -> implemented`: task commits implement the command namespace, licence posture,
  and scope corrections without claiming review approval.
- `implemented -> in-review`: one complete milestone PR has current evidence and green
  checks and is left open for maintainer review.
- `in-review -> verified`: the maintainer approves and explicitly merges the PR; root help,
  licence audit, issue closure, and milestone state then satisfy
  [quickstart.md](quickstart.md).
- Any missing/inconsistent surface keeps the record at its current state; no value is
  inferred from a different surface.

## Relationships

```text
Decision Record
├── selects 2 Component Identities
├── selects/observes Licence Postures
├── contains Compatibility Decisions
├── links milestone 1 and issues #47/#48/#107
└── is verified by CLI, repository, metadata, and test evidence
```
