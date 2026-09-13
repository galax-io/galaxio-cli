# Implementation Plan: Harden the Linkage Guard

**Branch**: `001-harden-linkage-guard` | **Date**: 2026-09-13 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-harden-linkage-guard/spec.md` (galax-io/galaxio-cli#62, milestone v0.11.0)

## Summary

Replace the raw-text release guard with the shared version from the bootstrap template, which judges only the git segments a command would actually run and reads the version from the tag alone; add the git `pre-push` hook that gates a tag from any client; rename the installer suite to the `*_test.sh` convention; and add two CI jobs, one that discovers and runs every shell suite and one that runs the pull-request merge gate. No Go code changes. The guard and hook are adopted byte-for-byte from `galax-io/spec-kit-galaxio-bootstrap` (see [research.md](research.md) R1) so the next template update replaces them cleanly.

## Technical Context

**Language/Version**: Bash 5 (guard, hook, suites; `#!/usr/bin/env bash`, `set -uo pipefail`), GitHub Actions YAML. No Go.

**Primary Dependencies**: `git`, `jq` (the guard reads the hook's JSON payload; already required by `check-linkage.sh`), `gh` (merge gate only, in CI via `GITHUB_TOKEN`). All present on `ubuntu-latest`; no new Go module.

**Storage**: N/A. State lives in git refs and GitHub milestones, read through the existing `scripts/check-linkage.sh`.

**Testing**: Shell suites beside each script: `.claude/hooks/linkage-guard_test.sh` (32 cases), `.githooks/pre-push_test.sh` (12 cases), `scripts/install_test.sh` (renamed from `test_install.sh`). Both hook suites stub the checker, so they are hermetic and offline. A CI job discovers `scripts/*_test.sh .claude/hooks/*_test.sh .githooks/*_test.sh` with a floor of 3.

**Target Platform**: Developer machines (macOS/Linux, bash ≥ 4 for `${BASH_REMATCH}` and `[[ =~ ]]`; macOS `/bin/bash` 3.2 is not used because the shebang resolves to Homebrew bash 5 via `env`), `ubuntu-latest` runners. Windows contributors get the pre-push hook through Git for Windows' bash.

**Project Type**: CLI repository tooling (hooks and CI), not a product change.

**Performance Goals**: The guard runs before every agent shell command, so its non-release path must be invisible: measured at ~30 ms per invocation on the reference guard for a non-release command (no network, no checker call). The release path calls the checker and is bounded by GitHub API latency, as today.

**Constraints**: Guard and hook files MUST stay byte-identical to upstream (FR-017); local fixes go upstream first. The CI addition touches `ci.yml`, whose `main` path is the release job, so the new jobs are added without altering `release`, `docker-image-main` or `publish-image` steps; only `needs` edges are added so the suites gate the image and release. `core.hooksPath` is per-clone config and cannot be shipped; documentation is the delivery.

**Scale/Scope**: 4 adopted files, 1 rename, 2 CI jobs, README and AGENTS.md notes, a PATCH constitution amendment for the gate table. Roughly 300 lines of shell, 60 lines of YAML.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Gate (constitution principle) | Status |
|---|---|---|
| I | Command Contract — every new command is a thin cobra wrapper over `runX(ctx, opts)`; offers `-o text\|json` with a documented JSON structure; exits 0/1/2 via `UsageError`/`RuntimeError`; honours `--verbose`/`--quiet`/`--no-color`; experimental commands sit behind `internal/featureflags`. | N/A — no command, flag or output of the `galaxio` binary changes. |
| II | Report Arithmetic Lives Here — (report features only). | N/A — not a report feature. |
| III | Tests Land With The Change — stdlib `testing`, table-driven, golden files under `testdata/`; command-level tests through `runCLI`; integration tests behind the `integration` tag; race on; coverage stays ≥ 80%; every fix carries a regression test; test tasks are never optional. | PASS — no Go changes, so Go coverage is untouched. The change is a bug fix in shell and ships with the regression suites the constitution's shell-suite rule demands; the current guard fails 9 of the 32 cases ([research.md](research.md) R2), which is the "fails without the fix" evidence. A CI job runs every suite (constitution Quality Gates, third additional constraint). |
| IV | Minimal, Explicit Dependencies — no new module unless named here, recorded in `research.md`, licence-compatible, asked for first. | PASS — no Go module. Runtime tools (`bash`, `git`, `jq`, `gh`) are already required by `check-linkage.sh` and the existing guard; `gh` is only needed in the merge-gate CI job, where it is preinstalled. No GitHub Action is added; the new jobs use `actions/checkout@v4` already pinned in `ci.yml`. |
| V | Published Surfaces — any change to a command, flag, default, exit code, `-o json` structure, manifest/registry schema or generated output is listed; breaking ones approved and committed with `!`; README updated in the same PR. | PASS — no public surface of the binary changes. New surfaces are contributor-facing: two CI job names (`shell suites`, `linkage`) that branch protection may reference, the `LINKAGE_OFF=1` bypass (already existed upstream, now documented), and `git config core.hooksPath .githooks`. README gains a Contributing note in the same PR. Commit type is `fix(hooks)` for the guard and `ci` for the jobs; neither is a `feat`, so the automatic release cuts a patch. |
| VI | Idiomatic, Simple Go — gofmt/vet clean; errors as values; no panic control flow; no dead or duplicated code; no refactor outside scope. | PASS (applied to shell) — the four adopted files are `shellcheck -S warning` clean ([research.md](research.md) R2). No opportunistic edits to `check-linkage.sh` or `install.sh`; the only rename is the suite whose name breaks the discovery glob. |

**Post-design re-check (after Phase 1)**: unchanged; the contracts introduce no command, dependency or binary surface. The one item that needed a decision, whether the shell-suite job should gate the release chain, is resolved in R5 by adding `needs` edges only.

## Project Structure

### Documentation (this feature)

```text
specs/001-harden-linkage-guard/
├── plan.md              # This file
├── research.md          # Phase 0: upstream provenance, regression evidence, CI wiring decisions
├── data-model.md        # Phase 1: guarded command, pushed ref, suite, decision states
├── quickstart.md        # Phase 1: run the suites, prove the regression, enable the hook, verify CI
├── contracts/
│   ├── guard.md         # PreToolUse guard: input, decision table, exit codes, messages, bypass
│   ├── pre-push.md      # git hook: ref-line contract and decisions
│   └── ci.md            # the two CI jobs: names, triggers, permissions, what fails them
└── tasks.md             # Phase 2 (/speckit-tasks), not created here
```

### Source Code (repository root)

```text
.claude/
├── hooks/
│   ├── linkage-guard.sh          # REPLACED byte-for-byte from upstream (102 lines)
│   └── linkage-guard_test.sh     # NEW, from upstream (32 cases)
└── settings.json                 # unchanged: already wires the guard as PreToolUse(Bash)

.githooks/
├── pre-push                      # NEW, from upstream, executable
└── pre-push_test.sh              # NEW, from upstream (12 cases)

scripts/
├── check-linkage.sh              # unchanged: the authority both guards call
├── install.sh                    # unchanged
└── install_test.sh               # RENAMED from test_install.sh so the *_test.sh glob finds it

.github/workflows/
└── ci.yml                        # + job `shell-suites` (name: shell suites), + job `linkage` (PR only);
                                  #   docker-image-pr / docker-image-main gain `needs: shell-suites`

README.md                         # + "Contributing" section: enable the hook, the bypass
AGENTS.md                         # HTML comment near the milestone rules now names .githooks/pre-push
.specify/memory/constitution.md   # PATCH 1.0.1: Linkage gate row → "linkage" job; #62 follow-up closed
```

**Structure Decision**: Files land where their consumers look for them: Claude Code reads `.claude/hooks/` through `.claude/settings.json`, git reads `.githooks/` once `core.hooksPath` points there, and CI discovers suites by the three-directory glob the constitution names. No new directory is introduced.

## Complexity Tracking

No constitution violations; table intentionally empty.

## Implementation Notes for /speckit-tasks

Ordering that keeps every commit green on its own:

1. `docs(speckit): add 001-harden-linkage-guard spec/plan` — the artifacts in this directory.
2. `fix(hooks): adopt the shared linkage guard and add pre-push (#62)` — the four upstream files plus the `test_install.sh` rename. Green because the suites pass locally and nothing runs them in CI yet.
3. `ci: run every shell suite and the PR merge gate (#62)` — `ci.yml` jobs and `needs` edges. Green because commit 2 already made the suites pass. The `linkage` job will be red on the PR until it carries the milestone and a `Closes #62` line, which is the gate doing its job; assign both before requesting review.
4. `docs: document the release guards for contributors (#62)` — README Contributing section and the AGENTS.md comment.
5. `docs(speckit): amend constitution to v1.0.1 (linkage gate runs in CI)` — gate table row and the #62 follow-up in the Sync Impact Report.

Commits 2 to 5 are one PR (one issue, one concern: the guard and what documents and runs it). AGENTS.md's rule of one issue per commit is met because every commit references #62 and no other issue is touched.
