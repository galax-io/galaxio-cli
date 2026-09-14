# Contract: Naming and Licence Posture

## Canonical identities

The following mapping is exact and exhaustive for this milestone:

| Role | Name | Canonical path/invocation |
|---|---|---|
| Public result-primitives library | `parsec` | `github.com/galax-io/parsec` |
| Private live-metrics sidecar | `comet` | `galax-io/comet` |
| Finished-run CLI namespace | `report` | `galaxio report` |

Active documentation MUST NOT use `galaxio-results`, `galaxio-tail`, or another descriptive
placeholder as a second active name.

## CLI licence surfaces

| Surface | Contract | Verification |
|---|---|---|
| `LICENSE` | Contains the unchanged GNU GPL Version 2 terms and an approved project selection that is unambiguously version 2 only | Inspect the notice and confirm the licence body remains verbatim |
| `README.md` | Names the exact SPDX expression `GPL-2.0-only` and links `LICENSE` and the decision record | Text search and link review |
| `Dockerfile` | OCI label equals `GPL-2.0-only` | Inspect `org.opencontainers.image.licenses` |
| GitHub repository | Detects the GPL v2 licence file; the API may expose the legacy classifier `gpl-2.0` | `gh repo view galax-io/galaxio-cli --json licenseInfo` |

GitHub's classifier is accepted only when it resolves from the repository's v2-only licence
file and displays GNU GPLv2. It is documented as a platform alias, not copied into
repository-owned SPDX fields.

## Shared-library boundary

- `galax-io/parsec` is public and reports SPDX `MIT`.
- Its MIT notice is retained by that repository and by any distribution obligation that
  later consumption creates.
- The FSF Expat/MIT entry is the authoritative compatibility source for combining the MIT
  library with this GPL-2.0-only program.
- No dependency on `parsec` is added in this milestone.

## Incompatible alternative boundary

Apache-2.0 is classified by the FSF as incompatible with GPL version 2 by itself. A future
direct Apache-2.0 dependency therefore remains blocked until a separate, explicit review
and approval records how the conflict is resolved.

This contract is scoped to the prospective MIT `parsec` relationship. It does not certify
the whole existing module graph: Cobra is a pre-existing Apache-2.0 dependency and requires
its own audit rather than being silently declared resolved here.

## Private sidecar boundary

- `galax-io/comet` must remain visible to authorized maintainers as a private repository.
- Public documentation may name its role and organization-qualified path.
- This milestone assigns no licence to `comet`, promises no public access, and makes no
  source, behavior, or commercial commitment about it.

## Failure conditions

The milestone evidence is not complete if any of these is true:

- an audited CLI surface contradicts `GPL-2.0-only`;
- README leaves the version selection implicit;
- the GPL terms are altered rather than accompanied by a project notice;
- `parsec` is not public, not organization-controlled, or no longer identifies as MIT;
- `comet` visibility cannot be checked by an authorized maintainer;
- root help lacks `report`;
- the decision record omits the authoritative compatibility sources or rejected choices.

No missing or inaccessible value is inferred from a different surface.
