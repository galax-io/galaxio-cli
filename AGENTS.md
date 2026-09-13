# galaxio-cli — Agent Guide

Command-line interface for Galaxio: scaffolds load-test projects from templates, generates Gatling scripts from API specifications, and reports on finished runs; one completed milestone is released by one deliberate version tag

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

Each command is a thin cobra wrapper over a runX(ctx, opts) (XOutput, error) function; human output goes to stdout and diagnostics to stderr; every command offers -o text|json; usage errors exit 2 and runtime errors 1 via UsageError and RuntimeError. CI verifies merges; a version tag for a completed milestone starts the release.

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
- **1 issue = 1 commit.** Each tracked GitHub issue maps to one semantic commit (`feat(scope): … (#NNN)`), green on its own (`go build ./... && go test ./...`). Verified Dependabot PRs may omit an issue, but must carry the active milestone and remain green on their own. Docs, tweaks, and out-of-scope improvements go in separate PRs — never mixed with issue commits.
- **Intent, not path.** No add-then-remove within a PR. Squash churn before review.
- **1 concern per PR.** Feature ≠ docs/README. Stack dependent PRs; update with `--force-with-lease`.
- **Idiomatic code.** Follow the language's idioms and the conventions already in the codebase; no control-flow-by-exception, no dead/duplicated code.

## Release Process (MANDATORY)

Merges to `main` run verification only. Pushing a `vX.Y.Z` tag on `main` or the matching `release/X.Y.0` branch starts `.github/workflows/release.yml`: it checks the milestone, reruns the release gates, creates the GitHub Release with GoReleaser, then publishes the Docker image.

### Preparing a release

1. Assign each PR to its active milestone before merging; link any fixed issues so GitHub closes them when the PR lands on `main`.
2. When the milestone is complete, fetch complete history and tags, then run `scripts/check-linkage.sh --for-tag vX.Y.Z` from its release commit.
3. Create and push the annotated tag only after the audit passes: `git tag -a vX.Y.Z -m "Release vX.Y.Z"` followed by `git push origin vX.Y.Z`.
4. Check the `release` and `publish image` jobs in `release.yml`.

### Rules

- **One completed milestone produces one release.** A version tag maps to the `vX.Y.0` milestone and is only pushed after its PRs are merged and issues closed.
- **Tags only on `main` or the matching `release/X.Y.0` branch.** The release workflow rejects any other location.
- **Never delete a release tag** after deployment starts.
- **Never reuse a version number.**
- **Before tagging**: every PR merged since the previous tag must be assigned to the release milestone; every issue in the milestone whose fix is on `main` must be closed.
- **Release audit:** fetch complete history and tags, then run `scripts/check-linkage.sh --for-tag vX.Y.Z`. It checks the existing target tag, or `HEAD` if the tag does not exist yet, against the preceding reachable release tag.

The `linkage` CI job validates each PR's milestone and closing link before it reaches
`main`; the `shell suites` job runs every `*_test.sh` suite under `scripts/`,
`.claude/hooks/`, and `.githooks/`. The tag-triggered `release` workflow repeats the
linkage audit and all release gates. The Claude `PreToolUse` hook and the optional
`.githooks/pre-push` hook protect the irreversible tag push; `LINKAGE_OFF=1` is their
sanctioned local bypass. Changes to the publishing workflow require separate approval.
