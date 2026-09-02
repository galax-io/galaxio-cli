# galaxio-cli — Agent Guide

Command-line interface for Galaxio: scaffolds load-test projects from templates, generates Gatling scripts from API specifications, and reports on finished runs; released automatically from main, so a feat commit cuts a minor release

> The sections above the `---` are **project-specific** — fill them in for each new
> project. Everything below the `---` is the **stack-agnostic development process**
> and is meant to be reused verbatim across all projects.

## Role

Principal Engineer: Go, cobra command-line design, code generation from API specifications, load-test result reporting

## Stack

Go 1.25, spf13/cobra, pb33f/libopenapi, gopkg.in/yaml.v3; standard-library testing only, table-driven with golden files; goreleaser and a distroless image

## Commands

```bash
# format      gofmt -w .
# verify      go vet ./... && go test ./...
# build/test  go build ./... && go test ./...
# integration go test -tags=integration ./...
```

## Structure

<!-- A LIGHT search index, not a full tree. List only the entry points an agent needs
     to FIND code fast — one terse line per area (`dir/ -> what lives there`). Omit
     anything discoverable by looking; an exhaustive tree is noise and rots fast. -->
cmd/galaxio/ -> one file per command, registered in root.go; internal/codegen/ -> parsers and Scala renderers for swagger, HAR and Postman; internal/templatecatalog/ -> template registry client; internal/featureflags/ -> environment gate for experimental commands; internal/buildinfo/ -> version embedding

## Architecture

Each command is a thin cobra wrapper over a runX(ctx, opts) (XOutput, error) function; human output goes to stdout and diagnostics to stderr; every command offers -o text|json; usage errors exit 2 and runtime errors 1 via UsageError and RuntimeError. Release is automatic from main: the version bump is computed from conventional commits, so a feat commit publishes.

## Test Model

Standard-library testing, no assertion library; table-driven subtests; golden files under testdata; command-level tests drive execute() through runCLI and assert exit codes; integration tests behind the integration build tag; coverage gate 80 percent enforced in CI; race detector always on.

---

<!-- ===================================================================== -->
<!-- STACK-AGNOSTIC DEVELOPMENT PROCESS — reuse verbatim across projects.   -->
<!-- ===================================================================== -->

## Boundaries

**Always:** format before commit, branch from `main`, keep commits semantic and green, preserve backward compat for published public APIs and any downstream consumers. `go.mod` = dependency truth, `.github/workflows/` = CI/release truth.

**Ask first:** new deps or upgrades, changing public API signatures / observable behavior / serialized formats, editing another repo, release/publish workflow changes.

**Never:** force-push or commit to `main`, merge commits in PR branches (rebase only), commit broken code, opportunistic refactors outside scope, mock external systems where a real integration path exists.

## Milestones (ALWAYS)

Every piece of work is tied to a milestone. No exceptions unless explicitly told otherwise.

- **Every PR** must be assigned to the active milestone before merging. No milestone = do not merge.
- **Every issue** fixed by a PR must be closed when that PR lands on `main`. Do not leave completed issues open.
- **Spec work** (`specs/NNN-*/`) belongs to the milestone that owns the spec. Link the spec PR to the milestone immediately when creating it.
- **Active milestone** = the lowest-numbered open milestone that matches the current spec/plan. Check `gh api repos/galax-io/galaxio-cli/milestones` if unsure.

## Commits & PRs

- **Spec-first.** `specs/NNN-*/` artifacts → `docs(speckit): add NNN-<feature> spec/plan/tasks` commit BEFORE any `feat`/`fix`. Never folded into implementation.
- **1 issue = 1 commit.** Each tracked GitHub issue maps to one semantic commit (`feat(scope): … (#NNN)`), green on its own (`go build ./... && go test ./...`). Docs, tweaks, and out-of-scope improvements go in separate PRs — never mixed with issue commits.
- **Intent, not path.** No add-then-remove within a PR. Squash churn before review.
- **1 concern per PR.** Feature ≠ docs/README. Stack dependent PRs; update with `--force-with-lease`.
- **Idiomatic code.** Follow the language's idioms and the conventions already in the codebase; no control-flow-by-exception, no dead/duplicated code.

## Release Process (MANDATORY)

Trunk-based with release branches. Trunk is `main`; `release/*` branches are cut from `main` for stabilization. Pushing a `vX.Y.Z` tag on `main` or a `release/*` branch triggers the release workflow (Go module proxy (tag-based)) and creates a GitHub Release (git-cliff).

### Minor/Major release (e.g. 1.2.0, 2.0.0)

1. `git checkout -b release/X.Y.0 main` — cut release branch from `main`
2. `git push -u origin release/X.Y.0`
3. `git tag vX.Y.0` on the release branch
4. `git push origin vX.Y.0` — triggers release workflow

### Patch release (e.g. 1.2.1)

1. Fix lands on `main` first (via PR as usual)
2. `git cherry-pick <fix-sha>` onto `release/X.Y.0`
3. `git tag vX.Y.1` on the release branch
4. `git push origin vX.Y.1` — triggers release workflow

### Rules

- **Every minor version gets its own `release/X.Y.0` branch** — no exceptions
- **Tags ONLY on `release/*` branches or `main`** — `release.yml` validates this
- **Branch name must match tag version**: `release/1.2.0` → `v1.2.0`, `v1.2.1`, etc.
- **Never delete a release tag** after the registry deployment starts — creates stuck deployments
- **Never reuse a version number** — most package registries reject duplicates permanently
- **Before tagging**: every PR merged since the previous tag must be assigned to the milestone; every issue in the milestone whose fix is on `main` must be closed

<!-- The issue↔PR↔milestone contract above is enforced mechanically by         -->
<!-- scripts/check-linkage.sh + the .claude/hooks/linkage-guard.sh PreToolUse   -->
<!-- hook (gates release tagging only; normal push/PR/merge untouched).         -->
