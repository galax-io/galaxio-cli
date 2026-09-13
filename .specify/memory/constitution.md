<!--
Sync Impact Report
==================
Version change: 1.0.0 → 1.0.1
Bump rationale: PATCH. No principle changes. The Quality Gates table and its additional
constraints now describe what CI runs after galaxio-cli#62 (spec 001-harden-linkage-guard):
the `linkage` job runs `scripts/check-linkage.sh --pr` on every pull request, and the
`shell suites` job runs every `*_test.sh` under `scripts/`, `.claude/hooks/` and
`.githooks/`. The linkage row no longer says "manual today", and the Development Workflow
bullet no longer points at #62 for the guards, which now exist.

Principles: unchanged (I–VI).

Added sections: none. Removed sections: none.

Modified:
- Quality Gates table: Linkage row's CI job `(manual today — see #62)` → `linkage`; new row
  "Shell suites" → `shell suites`.
- Additional constraints: the paragraph saying the `--pr` gate is not run by CI is replaced
  by the sentence that names the two jobs.
- Development Workflow, Milestones bullet: `(#62)` dropped after `.githooks/pre-push`.

Templates: no template text depends on the changed lines; none touched.
- ✅ .specify/templates/plan-template.md — Constitution Check gates unchanged.
- ✅ .specify/templates/tasks-template.md — unchanged.
- ✅ .specify/templates/spec-template.md — unchanged.
- ⚠ AGENTS.md — the shared "Release Process (MANDATORY)" section still describes release
  branches this repository does not have (carried from 1.0.0; belongs to
  `spec-kit-galaxio-bootstrap`). The enforcement comment under it now names all three
  callers of `check-linkage.sh` (#62).

Follow-up TODOs (carried from 1.0.0 unless noted):
- NEW: `scripts/check-linkage.sh` has no `*_test.sh` suite, which the shell-suite rule
  requires of every script under `scripts/`; it needs a `gh` stub and is its own issue. The
  `shell suites` job will pick it up with no change once it exists.
- `scripts/check-linkage.sh --for-tag` and the milestone-per-version convention assume a
  human decides the version; here the version is computed from commits, so a milestone
  title's version is a plan. Decision still deferred.
- `.claude/skills/speckit-tasks/SKILL.md` (spec-kit managed) still generates "OPTIONAL" test
  headings; the template is corrected, the generator is not ours.
- Skills classification pinned to `samber/cc-skills-golang` 2.0.1 and
  `galaxio/galaxio-gatling` 2.4.0 as installed on 2026-09-13; re-read on every plugin update.
-->
# galaxio-cli Constitution

## Core Principles

### I. Command Contract

- Every command is a thin cobra wrapper over a `runX(ctx, opts) (XOutput, error)` function
  in `cmd/galaxio/`, one file per command, registered in `root.go`. The cobra layer parses
  and dispatches; it MUST NOT hold logic a test cannot reach through `runX`.
- Human output goes to stdout, diagnostics to stderr. Every command offers `-o text|json`.
  The JSON form MUST be a documented structure, never the text table wrapped in a string,
  and MUST end with a single newline.
- Exit codes are a contract: `0` success, `1` runtime failure (`RuntimeError`), `2` usage
  failure (`UsageError`). A usage error is a bad flag, argument or unknown command; anything
  that fails while executing a valid command is runtime. An error MUST name what was
  searched, read or refused, so the user can act on it.
- Experimental commands are gated through `internal/featureflags` (an environment variable
  read with `strconv.ParseBool`). A gated command that is disabled MUST exit `2` naming the
  variable that enables it, and the gate MUST be removed — not left in place — when the
  command is promoted.
- Global flags (`--verbose`, `--quiet`, `--no-color`, `-o`) behave identically across
  commands. A new command MUST honour them.

Rationale: CI jobs and scripts are the primary consumers of this binary. A stable exit-code
and output contract is what lets them be written once; the `runX` seam is what lets that
contract be tested without a terminal.

### II. Report Arithmetic Lives Here

- `galaxio report` computes its statistics in this repository, in `internal/report/`, over
  the primitives `github.com/galax-io/parsec` yields: an addressable request position, the
  bounds of a run, the outcome of every sample. `parsec` exports no count, mean, percentile,
  range or series, and this repository MUST NOT ask it to (parsec constitution v2.0.0,
  parsec#8).
- What the library owns is the definitions — what counts as a failure, what a request
  position is, where a run begins and ends. This repository MUST NOT re-derive them.
- Successful and failed samples are accumulated separately; a failure MUST NOT contribute to
  a success statistic. Counts, minimum, maximum, mean and standard deviation MUST be exact.
  Percentiles are estimates, and every place that prints one MUST say so and MUST NOT claim
  digit-for-digit parity with Gatling.
- Accumulation MUST work in one pass without retaining every sample: memory MUST NOT grow
  with the sample count, and a plan for a report feature states its peak-memory goal.
- What a source cannot provide is reported as absent — never as a zero, an average or a
  guess. Rendering absence is this repository's job; inventing a value is nobody's.
- Source detection is by file content, never by file name or extension; an explicit override
  flag takes precedence over detection.

Rationale: two consumers compute — this command over a finished log, the `comet` sidecar
over a log still being written — and the library refuses to pick one implementation so that
the part that must not diverge stays in one place. The arithmetic here is deliberately a
second implementation, and the honesty rules are what keep it comparable.

### III. Tests Land With The Change (NON-NEGOTIABLE)

- Standard-library `testing`, table-driven subtests, no assertion library. Golden files live
  under `testdata/` next to the package that reads them and are updated deliberately, never
  regenerated to make a test pass.
- Command-level tests drive `execute()` through `runCLI` and assert the exit code, stdout
  and stderr. A new command or flag is not done until such a test exists.
- Integration tests sit behind the `integration` build tag, use real template packs, real
  registries and real API specifications, and `t.Skip` with a reason when a resource is
  unavailable rather than fake it. Mocking is reserved for what no real path can reach.
- The race detector is always on: `go test -race ./...` is the CI command and the local
  verify step. Coverage is gated at 80% for the module and enforced in CI; a change that
  takes the module below the floor MUST NOT merge.
- A bug fix MUST include a regression test that fails without the fix. Test tasks are never
  optional in a spec, plan or task list.
- Generated code (Scala from swagger, HAR and Postman) is tested against golden output, and
  a template or generator change that alters what a user receives MUST update the golden
  file in the same commit.

Rationale: the CLI's value is that a rendered project compiles and a generated script runs.
#45 and #46 — a rendered project that did not compile, a wrapper without its executable bit —
were exactly the class of defect a golden test catches and a unit test does not.

### IV. Minimal, Explicit Dependencies

- `go.mod` is dependency truth: `go mod tidy` MUST leave the tree unchanged, and `main`
  carries no `replace` directive.
- Adding or upgrading a module requires asking first. A plan that introduces one names it,
  explains why the standard library or an existing dependency is insufficient, and records
  the decision in `research.md`. Direct dependencies at ratification: `spf13/cobra`,
  `pb33f/libopenapi`, `gopkg.in/yaml.v3`.
- Licence compatibility is part of the decision. This repository is GPL-2.0-only; a module
  under a licence the FSF treats as incompatible with it (Apache-2.0 among them) MUST NOT be
  imported until the licence question is settled. `parsec` (MIT) is compatible. The
  relicensing decision is galaxio-cli#48 and is recorded there, not here.
- The binary is static (`CGO_ENABLED=0`) and ships in a distroless image; a dependency that
  needs cgo or a runtime library is out unless the plan shows how the image still works.

Rationale: every module here becomes part of a binary users install from a shell script and
a container they run in CI; its licence, its CVEs and its size are theirs the moment it lands.

### V. Published Surfaces Are Compatibility-Sensitive

- The public surface is what users and scripts depend on: command names, flags and their
  defaults, exit codes, the `-o json` structures, the template manifest (`galaxio-template.yaml`)
  and registry schema (`schemas/galaxio-registry.schema.json`), and the shape of generated
  Scala and rendered projects.
- Changing any of them in a way that breaks an existing invocation, a parseable output or a
  published pack is a breaking change: it MUST be called out in the spec, approved before
  implementation, and committed with a `!` (or `BREAKING CHANGE:` footer) so the release job
  cuts a major version. It MUST NOT be slipped into a `feat` or `fix`.
- Deprecate before removing: a superseded command or flag keeps working for at least one
  minor release and prints a deprecation notice naming its replacement on stderr.
- Every command and flag is documented in `README.md` in the same PR that adds or changes it.
  `CHANGELOG.md` is generated from conventional commits; the commit message is therefore the
  changelog entry and MUST be written as one.

Rationale: `galaxio` is invoked from Jenkins, GitLab and shell scripts that nobody in this
repository can see. A renamed flag breaks them long after the release that caused it.

### VI. Idiomatic, Simple Go

- Code MUST pass `gofmt` and `go vet` with zero findings. Format before every commit.
- Errors are values: functions return `error`, callers wrap with `%w` and inspect with
  `errors.Is` and `errors.As`. `UsageError` and `RuntimeError` are the only two the command
  layer distinguishes; everything else is wrapped into one of them at the boundary. `panic`
  and `recover` MUST NOT be used for control flow.
- No dead code, no duplicated code, no speculative abstraction: build what the current spec
  needs. Three similar lines beat a premature helper. A refactor outside the scope of the
  current issue goes in its own PR.
- Follow the conventions already in the codebase before adding a new one; a new convention
  is named in the plan and applied consistently in the PR that introduces it.
- Comments say why, not what. A comment that restates the identifier is deleted.

Rationale: a CLI is read by every contributor who adds a command and by every user who
debugs one; predictable Go keeps both fast.

## Quality Gates & Tooling

Toolchain: Go 1.25 (`go.mod`), used by CI through `go-version-file`. The `toolchain`
directive, when set, is bumped in a dedicated PR. Release binaries are built by GoReleaser
(`.goreleaser.yaml`) with `CGO_ENABLED=0`; the image is built from `Dockerfile` and
validated by running `galaxio version` inside it.

Every PR MUST be green on all CI jobs before merge:

| Gate | Command | CI job |
|------|---------|--------|
| Format | `test -z "$(gofmt -l .)"` | test |
| Module hygiene | `go mod tidy && git diff --exit-code -- go.mod go.sum` | test |
| Vet | `go vet ./...` | test |
| Tests + race | `go test -race -coverprofile=coverage.out ./...` | test |
| Coverage floor | total ≥ 80.0%, enforced | test |
| Build | `go build -trimpath -o dist/galaxio ./cmd/galaxio` | test |
| Integration | `go test -tags=integration -race -count=1 ./...` | integration tests |
| Image | build from `Dockerfile`, run `galaxio version` | docker image |
| Linkage | `scripts/check-linkage.sh --pr N` | linkage |
| Shell suites | every `*_test.sh` under `scripts/`, `.claude/hooks/`, `.githooks/` | shell suites |

Local equivalents: `gofmt -w .` before every commit; `go vet ./... && go test ./...` to
verify; `go build ./... && go test ./...` is the definition of a green commit;
`go test -tags=integration ./...` runs the integration suite.

Additional constraints:

- CI has no `paths-ignore`: a docs-only merge still runs every job and still cuts a patch
  release. That is the current design, not an accident; changing it is a release-workflow
  change and is asked for first.
- The `linkage` job runs `check-linkage.sh --pr` on every pull request, and the `shell suites`
  job runs every `*_test.sh` suite under `scripts/`, `.claude/hooks/` and `.githooks/` on every
  pull request and push to `main`. Both stay red until fixed; a reviewer relies on them
  instead of checking the milestone by hand.
- Shell scripts under `scripts/`, `.claude/hooks/` and `.githooks/` ship with a `*_test.sh`
  suite, and every such suite MUST be run by a CI job. Two repositories regressed the guard
  twice because a suite existed and nothing ran it.

### Engineering Guidance (Skills)

Coding agents here have `samber/cc-skills-golang` (classified against **2.0.1**) and
`galaxio/galaxio-gatling` (**2.4.0**). This is a rule about **reading**, not a build gate: a
skill is versioned outside this repository, can drift, and may be absent for a contributor.
The gate table enforces the outcomes; the skills are how a change is got right the first time.

Three bounds apply to all of them:

- **This document wins.** Where a skill and this constitution disagree, the constitution is
  followed and the disagreement is recorded in the feature's `research.md`.
- **A skill never justifies a dependency, a layout change, a weakened gate or a lowered
  coverage floor.** Those are Principles III and IV, against which a skill has no standing.
  An orchestrator (`golang-how-to`) surfacing a skill does not change this.
- **Unavailability blocks nothing.** Without the skill set a contributor follows Principles
  I–VI, which say the same things in less detail.

**Required reading.** Before writing code in the area named, the skill MUST be read.

| When a change… | Skill | Why it is required |
|---|---|---|
| adds or changes a command, flag, exit path or output format | `golang-cli`, `golang-spf13-cobra` | Principle I is this subject; Principle V makes a flag name effectively permanent. |
| adds or changes an error, or a path that returns one | `golang-error-handling` | Principle I's `UsageError`/`RuntimeError` boundary and Principle VI's errors-as-values rule. |
| adds or changes a test — which is every change | `golang-testing` | Principle III is NON-NEGOTIABLE. Where it reaches for a third-party library, that is a Principle IV proposal — argued, not ignored. |
| adds, renames or changes an exported identifier | `golang-naming` | Principle V for anything a user sees; Principle VI for the rest. |
| adds an exported identifier or a command | `golang-documentation` | Principle V requires the README entry in the same PR. |
| touches a template pack, generated Scala, or a Gatling version question | `galaxio-gatling-pro`, `gatling-versions`, `gatling-build`, `gatling-migration` | Rendered projects and generated scripts are Principle V surfaces and Principle III's golden output; the Gatling line decides what compiles. |

**Consult when the occasion arises.** SHOULD, with the occasion stated.

| Occasion | Skill | Assessment |
|---|---|---|
| a report feature states a peak-memory or throughput figure | `golang-benchmark`, then `golang-performance` | Principle II's one-pass rule is measured with a `testing.B` over the largest corpus run; optimisation follows the profile, never precedes it. |
| reading untrusted input — every log, spec, pack and archive this binary opens | `golang-safety`, `golang-security` | Archive extraction (`safeJoin`) and spec parsing are the exposed edges. |
| designing a constructor, an options struct or a streaming reader | `golang-design-patterns` | Functional options and `io.Reader` pipelines recur in `internal/report/`. |
| a dependency is proposed under the ask-first rule | `golang-pkg-go-dev`, `golang-dependency-management` | Principle IV: licence, CVEs and the real API are what the case has to survive. |
| existing code is restructured | `golang-refactoring` | Principle VI sends it to its own PR; this is how it stays behaviour-preserving. |
| a new `internal/` package is added or a helper needs a home | `golang-project-layout` | `cmd/galaxio/` and `internal/<area>/` are settled; where a shared helper goes is the open question. |
| a lint is added or the toolchain is bumped | `golang-lint`, `golang-modernize` | No `golangci-lint` config exists at ratification; adding one is a gate change, asked for first. |
| a bug resists the obvious explanation | `golang-troubleshooting` | Cheap to reach for, and only then. |

**No occasion has arisen** for `golang-grpc`, `golang-graphql`, `golang-database`,
`golang-observability`, `golang-samber-slog`, `golang-swagger` (it annotates Go servers; this
repository parses OpenAPI with `libopenapi`), the dependency-injection family
(`golang-dependency-injection`, `golang-google-wire`, `golang-uber-dig`, `golang-uber-fx`,
`golang-samber-do`), the `golang-samber-*` helpers, `golang-stretchr-testify`,
`golang-spf13-viper` or `golang-continuous-integration` (the pipeline exists and the gate table
is authoritative). **These are not prohibitions.** A skill that reaches for a dependency is
making a Principle IV proposal, decided in the feature's `research.md`. Two are worth
watching: `golang-concurrency` and `golang-context` become relevant the moment `galaxio report`
streams a multi-gigabyte log (#50), and `golang-spf13-viper` the moment configuration grows
past `~/.galaxio/config.yaml`.

**Review.** This classification is re-read at every minor release and whenever a skill plugin
is updated: diff the inventory, re-check the rules the tiers rest on, update the versions
pinned above, and fix every `research.md` left pointing at a file that moved.

## Development Workflow & Release Process

`AGENTS.md` holds the step-by-step procedure; the rules below are the ones it MUST NOT
contradict.

- **Spec-first.** Every feature starts as `specs/NNN-<feature>/` (spec, plan, tasks),
  committed as `docs(speckit): add NNN-<feature> spec/plan/tasks` BEFORE any `feat` or `fix`
  commit, and never folded into implementation. Spec work belongs to the milestone that owns
  the spec.
- **Milestones.** Every PR MUST carry the active milestone (the lowest-numbered open
  milestone matching the current spec) before merge; no milestone, no merge. Every issue a
  PR fixes MUST be closed when the PR lands on `main`. `scripts/check-linkage.sh --pr N` is
  the merge gate. Because tags here are cut by CI, the merge gate is the one that protects a
  release; `.claude/hooks/linkage-guard.sh` and `.githooks/pre-push` guard the rare
  manual tag.
- **Commits.** Semantic messages (`feat(scope): … (#NNN)`); one tracked issue per commit;
  every commit green on its own; format before commit; intent, not path: squash churn
  before review. The commit type is a release decision (below), so it MUST be chosen
  deliberately: `feat` for a user-visible capability, `fix` for a defect, `docs`/`ci`/
  `chore`/`test`/`refactor` for everything else.
- **Branches and PRs.** Branch from `main`. One concern per PR (feature ≠ docs). Rebase
  only: no merge commits in PR branches; stacked PRs are updated with `--force-with-lease`.
  Never force-push to `main`, never commit to `main` directly.
- **Ask first.** New dependencies or upgrades; changes to public API signatures, observable
  behaviour or serialized formats (Principle V); edits to another repository; release or
  publish workflow changes; a licence change.
- **Never.** Commit broken code; refactor outside the issue's scope; mock an external system
  where a real integration path exists; phrase a command to evade a guard.
- **Releases are automatic from `main`.** On every push to `main`, `ci.yml` computes the next
  version from conventional commits since the latest `v*` tag — `patch` by default, `minor`
  when any commit is `feat`, `major` when any carries `!` or `BREAKING CHANGE` — tags it,
  runs GoReleaser and publishes the image. There is no release branch and no manual tag in
  the normal path. Consequences that MUST be understood before merging: a merge is a
  release; a `feat` in a PR is a minor bump for everyone; a milestone title's version is a
  plan, and the version actually cut is whatever the commits say. Never delete a release tag
  once the job has started; never reuse a version number.
- **Hotfix.** A fix lands on `main` through a PR like any other change and is released by the
  same job. There is no cherry-pick path because there is no release branch.

## Governance

- This constitution supersedes every other practice document in the repository. `AGENTS.md`
  (loaded by coding agents through `CLAUDE.md`) is the runtime guidance file: it MUST agree
  with this document, and where the two conflict this document wins and `AGENTS.md` is
  corrected in the same PR — or, where the conflicting text is shared org boilerplate, an
  issue is filed against `spec-kit-galaxio-bootstrap` and the conflict is recorded in the
  Sync Impact Report until it is resolved.
- **Amendment procedure.** An amendment is a PR that edits
  `.specify/memory/constitution.md`, bumps the version below, rewrites the Sync Impact
  Report comment at the top of the file, and updates `.specify/templates/*` and any affected
  guidance doc in the same PR. It is approved by a maintainer of `galax-io/galaxio-cli`
  through PR review and committed as
  `docs(speckit): amend constitution to vX.Y.Z (<summary>)`.
- **Versioning policy.** MAJOR: a principle or governance rule is removed or redefined in a
  backward-incompatible way. MINOR: a principle or section is added, or guidance is
  materially expanded. PATCH: clarification, wording or typo fixes with no semantic change.
- **Compliance review.** Every `plan.md` MUST complete the Constitution Check in the plan
  template before Phase 0 and again after Phase 1; each FAIL needs a Complexity Tracking row
  naming the simpler alternative that was rejected. `/speckit-analyze` treats a violated MUST
  as CRITICAL. A reviewer MUST NOT approve a PR that violates a MUST without such a recorded
  justification. At every minor release the maintainer re-reads Principles I–VI against the
  milestone's merged PRs and files an issue for each gap in the next milestone.

**Version**: 1.0.1 | **Ratified**: 2026-09-02 | **Last Amended**: 2026-09-13
