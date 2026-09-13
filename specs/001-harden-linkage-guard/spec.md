# Feature Specification: Harden the Linkage Guard

**Feature Branch**: `001-harden-linkage-guard`

**Created**: 2026-09-13

**Status**: Draft

**Input**: User description: "https://github.com/galax-io/galaxio-cli/issues/62 — The linkage guard matches raw command text, and treats a release-branch push as a release"

**Tracking**: galax-io/galaxio-cli#62 (milestone v0.11.0 SDD bootstrap). Reference fix: galax-io/spec-kit-galaxio-bootstrap#6 (merged 2026-09-06), adopted downstream in galax-io/parsec#51.

## Background

This repository ships a guard that runs before every shell command a coding agent executes. Its only job is to stop a release tag from being created or pushed while the milestone that owns that version is not ready: an issue still open, a pull request not merged, a pull request without a milestone. The check itself already exists (`scripts/check-linkage.sh --for-tag`); the guard decides *when* to invoke it and *which version* to check.

Today the guard decides by looking at the raw text of the whole command. That has two consequences, both observed in more than one repository:

- **False blocks.** Any command whose text mentions a tag-like version is treated as a release. A pull-request description that documents the guard was blocked by the guard, which reported a version taken from prose. Contributors learn to rephrase commands until the guard stops recognising them, which is a habit of evading a safety check.
- **Wrong version.** A push to a release branch is treated as a release and the version is read from the branch name. A release branch serves a whole minor line, so for every patch release the name is the wrong version and the guard demands a milestone that never existed.

There is also a gap no command-text parsing can close: the guard only sees commands an agent runs. A maintainer tagging from a terminal or an IDE never reaches it. A git-level pre-push hook does reach every client, because git hands it the exact references about to be pushed.

Finally, the guard reached its current shape by being wrong repeatedly, and it regressed twice because the test suites that exist in some repositories are run by nothing.

In this repository, releases are cut automatically by CI on every push to `main` (constitution v1.0.0), so a human-pushed tag is the exception. The guard and the pre-push hook protect that exception; the merge gate (`check-linkage.sh --pr N`) is what protects the normal path, and the constitution assigns wiring it into CI to this issue as well.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - The guard judges only what would actually run (Priority: P1)

A contributor, or a coding agent acting for them, runs an ordinary command that happens to contain a version string: opening a pull request whose description documents the release procedure, committing a changelog line, writing a file that mentions `v1.2.3`. The command completes. The guard only steps in when the command would actually create or push a release tag, and then it checks exactly the version the tag names.

**Why this priority**: This is the defect that blocked real work in two repositories and trained contributors to evade the guard. Everything else in this feature is secondary to a guard that is right about what it is looking at.

**Independent Test**: Feed the guard a corpus of commands (prose mentioning a version, heredoc bodies, compound commands, wrapped commands, read-only tag queries, real tag creations) and confirm it blocks exactly the commands that create or push a release tag, naming the version from the tag itself.

**Acceptance Scenarios**:

1. **Given** a command that opens a pull request whose body text mentions `v1.2.3` and the guard, **When** the command runs, **Then** it is not blocked.
2. **Given** a command whose only mention of a version is inside a heredoc body, **When** the command runs, **Then** it is not blocked.
3. **Given** a single command line that first commits and then creates a tag `v1.2.3`, **When** it runs, **Then** the tag creation is judged and blocked if the milestone is not ready, even though the same line also contains a commit.
4. **Given** a tag creation invoked through a wrapper or with leading environment assignments (for example the token-saving proxy this project's agents use for every git command), **When** it runs, **Then** it is judged exactly as the unwrapped command would be.
5. **Given** an annotated tag creation with the version and message flags in any order, **When** it runs, **Then** it is judged.
6. **Given** a read-only or destructive tag subcommand (list, delete, verify, or the numbered-list form), **When** it runs, **Then** it is not gated.
7. **Given** a command that would create or push a release tag whose milestone is ready, **When** it runs, **Then** it proceeds without user-visible friction.
8. **Given** a blocked command, **When** the block is reported, **Then** the message names the version that was checked and the rule that failed, so the contributor can fix the milestone rather than rephrase the command.

---

### User Story 2 - A branch push is not a release (Priority: P2)

A maintainer pushes a branch, including a branch whose name looks like a release line. Nothing is published by that push, so the guard lets it through. Only pushing a tag is a release, and the version checked is the one the tag carries.

**Why this priority**: This repository has no release branches today, so the wrong-version failure is latent here rather than active. It still must be fixed because the guard is shared text across all Galaxio repositories and the corrected version is the one to adopt.

**Independent Test**: Push a branch named like a release line with the guard active and confirm it is not blocked; push an explicit tag, a tag by full reference, and all tags, and confirm each is judged by the version of the tag being pushed.

**Acceptance Scenarios**:

1. **Given** a push of a branch named `release/1.2.0`, **When** it runs, **Then** it is not treated as a release and is not blocked.
2. **Given** a push of tag `v1.2.1`, **When** it runs, **Then** the guard checks milestone `v1.2.1`, not a version derived from any branch name.
3. **Given** a push of all tags, **When** it runs, **Then** it is treated as a release and judged.
4. **Given** a push of a tag without any explicit version in the command text, **When** it runs, **Then** the guard refuses and tells the contributor how to verify the release manually.

---

### User Story 3 - Manual tags from any client are guarded (Priority: P2)

A maintainer tags a release from a terminal, an IDE, or any client other than a coding agent. When they push, git itself asks a pre-push hook whether the tag may go out. The hook sees the exact references being pushed, so there is nothing to parse: a tag reference is checked against its milestone, a branch push passes, a deletion passes. The hook is enabled once per clone with a documented one-line configuration, and the repository documentation tells every contributor to do so.

**Why this priority**: Closes the gap that command parsing structurally cannot. Lower than P1 only because in this repository tags are normally pushed by CI, not people.

**Independent Test**: With the hook enabled in a clone, attempt to push a tag whose milestone is not ready and confirm the push is refused with the checker's explanation; push a branch and a tag deletion and confirm both pass; remove the checker and confirm the tag push is refused rather than waved through.

**Acceptance Scenarios**:

1. **Given** the hook is enabled and a tag `v1.2.3` whose milestone is not ready, **When** the maintainer pushes it, **Then** the push is refused and the checker's findings are shown.
2. **Given** the hook is enabled, **When** the maintainer pushes a branch, **Then** the push proceeds with no check.
3. **Given** the hook is enabled, **When** the maintainer pushes a tag deletion, **Then** the push proceeds with no check.
4. **Given** the hook is enabled, **When** the maintainer pushes all tags at once, **Then** every tag reference is judged individually and the push is refused if any one fails.
5. **Given** the hook is enabled but the linkage checker is missing from the checkout, **When** a tag is pushed, **Then** the push is refused with a message saying the release cannot be verified.
6. **Given** a fresh clone, **When** a contributor reads the repository's contributor guidance, **Then** they find the one-time command that enables the hook.

---

### User Story 4 - Every shell suite runs in CI (Priority: P2)

Both the guard and the pre-push hook ship with their test suites, and CI runs every shell suite in the repository on every pull request and every push to `main`. A regression in the guard fails the build before it can reach a contributor.

**Why this priority**: The guard regressed twice in other repositories because suites existed and nothing ran them. Without this story, the fix in Story 1 has the same life expectancy.

**Independent Test**: Introduce a deliberate regression in the guard on a branch and confirm the CI run for that branch fails on the shell-suite job; revert it and confirm the job passes.

**Acceptance Scenarios**:

1. **Given** the guard suite (32 cases) and the pre-push suite (12 cases) are present, **When** CI runs for a pull request, **Then** a job executes both suites and passes.
2. **Given** the existing installer suite that no job runs today, **When** CI runs, **Then** it is executed by the same job.
3. **Given** a change that breaks one guard case, **When** CI runs, **Then** the shell-suite job fails and names the failing case.
4. **Given** a new shell script with a suite under the governed directories, **When** it is added, **Then** it is picked up by CI without editing the job.

---

### User Story 5 - The merge gate runs in CI (Priority: P3)

A pull request is checked automatically for its milestone, its closing link to an issue, and that issue sharing the milestone. A reviewer no longer checks these by hand, and a pull request that fails the gate is visibly red before merge.

**Why this priority**: In this repository the merge is the release, so this gate is the one that actually protects a version. The constitution assigns wiring it into CI to this issue. It is P3 because it is independent of the guard fix and can land separately.

**Independent Test**: Open a pull request without a milestone and confirm the gate job fails naming the missing milestone; assign the milestone and add the closing link and confirm the job passes.

**Acceptance Scenarios**:

1. **Given** a pull request with no milestone, **When** CI runs, **Then** the gate job fails and says the milestone is missing.
2. **Given** a pull request with a milestone, a closing link to an issue, and the issue in the same milestone, **When** CI runs, **Then** the gate job passes.
3. **Given** a push to `main` (not a pull request), **When** CI runs, **Then** the gate job does not run or is skipped without failing the release.

---

### Edge Cases

- A command mentions a version only in prose, a file path, a commit message, or a pull-request body: not gated.
- A heredoc body contains a line that looks like a tag command: ignored; only text outside the heredoc is judged.
- One command line chains a commit and a tag creation: each part is judged on its own, so the commit exemption does not cover the tag.
- A command creates a tag through a wrapper, with leading environment assignments, or via a sub-shell runner: judged as the underlying command.
- Tag list, delete, verify, and numbered-list forms: not gated, in any flag order.
- Annotated tag creation with the version before or after the message flag: gated.
- Push of a release-named branch: not gated. Push of all tags, of a tag by full reference, or of a tag by name: gated.
- The linkage checker is missing or not executable: the guard and the hook both refuse and say the release cannot be verified, rather than passing.
- The deliberate bypass variable is set: the guard stands down, and the bypass is documented so its use is visible in a review rather than hidden in phrasing.
- The pre-push hook receives a tag deletion (all-zero local id): passes.
- The pre-push hook receives several tags in one push: each judged; one failure refuses the whole push.
- CI shell-suite job on a runner where a suite depends on tooling that is absent: the suite fails loudly rather than skipping silently.
- The merge-gate job on a pull request from a fork or without repository access to milestones: the job reports why it cannot verify rather than passing.

## Requirements *(mandatory)*

### Functional Requirements

**Guard scope (what is judged)**

- **FR-001**: The guard MUST judge only the parts of a command that would actually execute a git operation. Text inside heredoc bodies, and any segment that does not invoke git, MUST NOT influence the decision.
- **FR-002**: A command line containing several commands joined by shell separators MUST be split and each segment judged independently. Exemptions for commit, log, and show MUST apply only to the segment that invokes them, never to the whole line.
- **FR-003**: Leading environment assignments and known wrappers (the token-saving proxy, sudo, env, xargs, time, exec) MUST be peeled before a segment is judged, so a wrapped tag creation is judged as the bare one.
- **FR-004**: Tag creation MUST be recognised in any flag order, including the annotated form with a message. Tag list, delete, verify, and numbered-list forms MUST NOT be gated.
- **FR-005**: Pushing a branch MUST NOT be treated as a release, regardless of the branch name. Pushing a tag by name, by full tag reference, or with the all-tags option MUST be treated as a release.
- **FR-006**: The version checked MUST be taken from the tag being created or pushed, never from surrounding prose, a file path, or a branch name. If a release push carries no explicit version, the guard MUST refuse and tell the contributor how to verify the release manually.

**Guard outcome**

- **FR-007**: A blocked command MUST produce a message that names the version checked and the reason the milestone is not ready, and MUST NOT suggest rephrasing the command.
- **FR-008**: If the linkage checker is missing or not executable, the guard MUST refuse a release rather than allow it.
- **FR-009**: The deliberate bypass (an environment variable) MUST remain available and MUST be documented in the contributor guidance as the sanctioned way to skip the guard, so that evasion by phrasing has no reason to exist.
- **FR-010**: Commands that do not create or push a release tag MUST experience no observable change in behaviour or latency from the guard.

**Pre-push hook**

- **FR-011**: The repository MUST ship a git pre-push hook that judges the references git is about to push: a tag reference is checked against its milestone via the linkage checker; a branch reference passes; a deletion passes; when several references are pushed, each is judged and any failure refuses the push.
- **FR-012**: The hook MUST refuse a tag push when the linkage checker is missing, stating that the release cannot be verified.
- **FR-013**: The hook MUST be opt-in per clone via git configuration, and the contributor guidance MUST document the one-time enabling command.

**Test suites and CI**

- **FR-014**: The guard MUST ship with its test suite (32 cases in the reference) and the pre-push hook with its suite (12 cases in the reference), each case being a command or reference shape the guard was at some point wrong about.
- **FR-015**: CI MUST run every shell test suite under the governed script directories on every pull request and every push to `main`, discovering suites by convention rather than by an explicit list, and a failing case MUST fail the job. The existing installer suite MUST be included.
- **FR-016**: CI MUST run the pull-request merge gate (milestone present, closing link present, linked issue in the same milestone) on every pull request, and a gate failure MUST fail the check visibly on the pull request.

**Provenance**

- **FR-017**: The guard and hook adopted here MUST be the shared versions from the bootstrap template, not a local fork, so the next template update replaces them cleanly. Any local divergence MUST be recorded in the plan with the reason.
- **FR-018**: Contributor guidance MUST describe the guard, the bypass, the pre-push hook, and how to enable it, and MUST NOT contradict the constitution's statement that releases are cut automatically from `main`.

### Key Entities

- **Guarded command**: a shell command an agent is about to run, decomposed into segments; only git segments that create or push a release tag are subject to the milestone check.
- **Release tag**: a version-named tag (`vX.Y.Z`); creating or pushing one is the only action the guard and the hook gate. Its name is the sole source of the version to check.
- **Milestone readiness**: the state the linkage checker evaluates for a version: every issue closed, every pull request merged and carrying the milestone. Not defined by this feature; consumed by it.
- **Pushed reference**: a branch or tag reference git presents to the pre-push hook, with its local and remote ids; a tag reference with a non-zero local id is a release.
- **Shell test suite**: a self-contained script next to the script it tests, named by convention, exiting non-zero on any failing case; CI discovers and runs all of them.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Zero false blocks: every command in the guard suite that does not create or push a release tag passes, including the pull-request-creation and heredoc shapes that were blocked before.
- **SC-002**: Zero misses: every command in the guard suite that does create or push a release tag is judged, including the chained commit-then-tag and wrapped forms that were let through before.
- **SC-003**: All 32 guard cases and all 12 pre-push cases pass in CI on the pull request that lands this feature, and the shell-suite job is required for merge from then on.
- **SC-004**: A deliberately broken guard case turns the CI run red within one push; the regression class that went unnoticed twice is now detected before merge.
- **SC-005**: A push of a release-named branch is never blocked and never demands a milestone that does not exist.
- **SC-006**: With the hook enabled, a tag pushed from a plain terminal for an unready milestone is refused, with the checker's findings shown, in the same run.
- **SC-007**: A pull request without a milestone shows a failing check within its first CI run, and a reviewer can rely on that check instead of inspecting the milestone by hand.
- **SC-008**: Contributors have no documented or observed reason to rephrase a command to get past the guard; the only sanctioned skip is the documented bypass.

## Assumptions

- The behaviour to adopt is the one merged in galax-io/spec-kit-galaxio-bootstrap#6 and verified downstream in galax-io/parsec#51; this spec restates that behaviour as requirements rather than redesigning it. The reference suites (32 and 12 cases) are taken as the acceptance corpus.
- The linkage checker (`scripts/check-linkage.sh`) already exists here and its rules are unchanged by this feature; the guard and the hook only decide when to call it and with which version.
- Because this repository cuts releases automatically from `main`, a human tag push is rare; the guard and hook protect that rare path, and the merge gate in CI (Story 5) protects the normal path. The constitution v1.0.0 assigns the merge gate to this issue, which is why it is in scope despite not appearing in the issue's suggested list.
- Enabling the pre-push hook is per clone and cannot be automated by files in the repository; documentation is the delivery mechanism.
- The token-saving proxy that rewrites every git command for this project's agents is one of the wrappers the guard peels; without that, Story 1 would fail here on every agent-run command.
- CI job scope is a release-workflow change under the constitution's ask-first rule; adding a job that runs suites and the merge gate is proposed here and approved through this spec's review, without altering the existing release job.
- The existing contributor guidance section describing a manual release-branch procedure is known to be out of date (constitution Sync Impact Report) and is corrected only insofar as it describes the guard and hook; the broader correction is tracked separately.
- Out of scope: changing the linkage checker's rules, changing the automatic release job, changing the milestone naming convention, and any behaviour of the guard beyond release tagging.
