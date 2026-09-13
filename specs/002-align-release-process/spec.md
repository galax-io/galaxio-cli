# Feature Specification: Align Releases with Milestones

**Feature Branch**: `galaxio/align-sdd-release-process`

**Created**: 2026-09-13

**Tracking**: galax-io/galaxio-cli#71, milestone `v0.11.0 SDD bootstrap`

## Background

The SDD bootstrap requires exactly one release for a completed milestone. The
current `ci.yml` creates a version tag, GitHub Release, and Docker image after
every push to `main`. Merging several PRs from one milestone therefore creates
several releases before the milestone is complete.

## User Scenarios & Testing

### User Story 1 - Merging work does not publish it (Priority: P1)

A maintainer merges a green PR into `main`. CI verifies the source and image,
but creates no version tag, GitHub Release, or published Docker image.

**Independent Test**: the release workflow regression suite rejects a `ci.yml`
that contains tag creation or GoReleaser, and accepts the completed workflow.

### User Story 2 - A completed milestone has one deliberate release (Priority: P1)

After every issue is closed and every PR for milestone `vX.Y.0` is merged, a
maintainer pushes `vX.Y.Z` from `main` or the matching `release/X.Y.0` branch.
The release workflow checks the tag's location and linkage before publishing a
GitHub Release and Docker image.

**Independent Test**: `scripts/check-linkage.sh --for-tag vX.Y.Z` runs before
GoReleaser and Docker Hub publication in `.github/workflows/release.yml`.

### User Story 3 - The documented process and automation agree (Priority: P2)

A contributor follows `AGENTS.md` and the constitution to prepare a release.
Both documents describe the same tag-triggered procedure that CI implements.

**Independent Test**: the documents name the tag workflow and require the
linkage audit before the irreversible tag push.

## Requirements

- **FR-001**: `ci.yml` MUST run validation on pull requests and `main` pushes,
  but MUST NOT create a release tag, GitHub Release, or publish an image.
- **FR-002**: `release.yml` MUST run only when a semantic `vX.Y.Z` tag is pushed.
- **FR-003**: Before publication, the release workflow MUST confirm that the tag
  is on `main` or its matching `release/X.Y.0` branch and call
  `scripts/check-linkage.sh --for-tag`.
- **FR-004**: The release workflow MUST rerun formatting, module hygiene, vet,
  unit tests with the coverage floor, integration tests, shell suites, build,
  and container validation before publishing.
- **FR-005**: GoReleaser MUST create the GitHub Release only after those gates;
  Docker Hub publication MUST run only after the GitHub Release succeeds.
- **FR-006**: `AGENTS.md` and the constitution MUST describe the tag-triggered
  process and the one-milestone/one-release invariant.

## Success Criteria

- A main-branch merge cannot create a new `v*` tag or external release.
- A tag fails before publication when its milestone is incomplete or mismatched.
- A successful tag produces one GitHub Release and the version, minor, and
  `latest` Docker tags.
