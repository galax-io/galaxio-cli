# Implementation Plan: Align Releases with Milestones

**Branch**: `galaxio/align-sdd-release-process`

## Approach

Keep `.github/workflows/ci.yml` as the merge verification workflow. Replace its
main-only release and publish jobs with the ordinary Docker validation job.

Add `.github/workflows/release.yml`, triggered only by `vX.Y.Z` tag pushes. Its
jobs are ordered as follows:

1. Guard the tag's branch, release range, and milestone linkage.
2. Re-run all Go, shell, and container verification for the tagged source.
3. Run GoReleaser to create the GitHub Release.
4. Build and publish the Docker image after the release succeeds.

The workflow takes the version only from `github.ref_name`, so conventional
commit parsing cannot create an unrelated version. `AGENTS.md` and the
constitution will be amended in the implementation commit because their release
rules are part of the executable contract.

## Validation

- Add `scripts/release_workflow_test.sh` before implementation and demonstrate
  that it fails against the automatic-release workflow.
- Run every shell suite, `go vet ./...`, `go build ./...`, `go test -race ./...`,
  and `go test -tags=integration -race ./...`.
- Parse both workflows as YAML when PyYAML is installed and run `actionlint` when
  available.
- Run `scripts/check-linkage.sh --tag 2` after the issue closes; do not push a
  version tag as part of this change.
