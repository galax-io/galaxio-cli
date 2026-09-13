# Tasks: Align Releases with Milestones

**Tracking**: galax-io/galaxio-cli#71

- [X] T001 Add the release-workflow regression suite and show it fails on the
  current automatic-release implementation.
- [X] T002 Add spec, plan, and task artifacts before the implementation commit.
- [ ] T003 Convert `.github/workflows/ci.yml` to verification-only CI.
- [ ] T004 Add tag-triggered `.github/workflows/release.yml` with linkage,
  verification, GitHub Release, and Docker publication stages.
- [ ] T005 Amend `AGENTS.md` and the constitution to the shared release process.
- [ ] T006 Run the regression suite, shell suites, Go verification, workflow
  syntax checks, and review the diff.
- [ ] T007 Open a milestone-linked PR with `Closes #71`; after merge, verify the
  milestone is tag-ready without creating a tag automatically.
