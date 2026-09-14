<!--
Sync Impact Report
==================
Version change: 1.1.1 → 2.0.0
Bump rationale: MAJOR. galaxio-cli#109 replaces split issue/PR delivery with one reviewed
PR per milestone and one commit per task. Agents may prepare and update a PR, but may not
merge or close it without an explicit post-review instruction for that exact PR. Principle I
now distinguishes operational commands from help-only namespace parents and names the actual
root flags.

Principles modified: I (operational-command contract and namespace-parent exception),
VI (out-of-scope work stays out of the milestone rather than automatically creating another
PR).

Added sections: none. Removed sections: none.

Modified:
- Development Workflow: specification and implementation stay in one milestone PR; every
  task maps to exactly one green commit; maintainer review owns merge and closure.
- Governance: amendment PRs remain open until maintainer review and explicit merge action.
- AGENTS.md: records the same single-PR, task-commit, and review ownership rules.
- Principle I: applies `runX` and `-o text|json` to operational commands, permits help-only
  namespace parents, and removes `-o` from the root-flag list.

Templates:
- ✅ .specify/templates/plan-template.md — existing Constitution Check can assess the
  clarified command contract.
- ✅ .specify/templates/tasks-template.md — requires one commit per task and one PR per
  milestone; removes optional-test wording.
- ✅ .specify/templates/spec-template.md — unchanged.
- ✅ AGENTS.md — runtime guidance matches this amendment.

Follow-up TODOs (carried from 1.0.0 unless noted):
- `scripts/linkage_test.sh` covers the release-range helpers used by
  `scripts/check-linkage.sh`. A direct suite for its `gh` calls still needs a `gh` stub
  and remains a follow-up; the `shell suites` job discovers the existing helper suite.
- The release workflow has no end-to-end test against a disposable GitHub repository; its
  local regression suite checks trigger, guard, verification, and publication ordering.
- `.claude/skills/speckit-tasks/SKILL.md` (spec-kit managed) may still generate "OPTIONAL"
  test headings; this repository's template and constitution require tests.
- Skills classification pinned to `samber/cc-skills-golang` 2.0.1 and
  `galaxio/galaxio-gatling` 2.4.0 as installed on 2026-09-13; re-read on every plugin update.
-->
# galaxio-cli Constitution

## Core Principles

### I. Command Contract

- Every operational command is a thin cobra wrapper over a
  `runX(ctx, opts) (XOutput, error)` function in `cmd/galaxio/`, one file per command,
  registered in `root.go`. The cobra layer parses and dispatches; it MUST NOT hold logic a
  test cannot reach through `runX`.
- A help-only namespace parent MAY route directly to `cmd.Help()` and define neither a
  result type, `runX`, nor `-o` until it gains operational behavior. It MUST reject
  positional arguments, expose its child commands, and inherit the root flags.
- Human output goes to stdout, diagnostics to stderr. Every operational command offers
  `-o text|json`. The JSON form MUST be a documented structure, never the text table wrapped
  in a string, and MUST end with a single newline.
- Exit codes are a contract: `0` success, `1` runtime failure (`RuntimeError`), `2` usage
  failure (`UsageError`). A usage error is a bad flag, argument or unknown command; anything
  that fails while executing a valid command is runtime. An error MUST name what was
  searched, read or refused, so the user can act on it.
- Experimental commands are gated through `internal/featureflags` (an environment variable
  read with `strconv.ParseBool`). A gated command that is disabled MUST exit `2` naming the
  variable that enables it, and the gate MUST be removed — not left in place — when the
  command is promoted.
- Root flags (`--verbose`, `--quiet`, `--no-color`) behave identically across commands. The
  `-o` flag is local to operational commands and MUST follow the output contract above.

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

Rationale: this CLI computes reports over finished logs while `parsec` supplies shared
definitions rather than aggregate implementations. Keeping report arithmetic here prevents
library consumers from inheriting CLI policy, and the honesty rules keep results comparable.

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
  needs. Three similar lines beat a premature helper. A refactor outside the current
  milestone stays out; an in-scope refactor is an explicit task and its own commit.
- Follow the conventions already in the codebase before adding a new one; a new convention
  is named in the plan and applied consistently in the PR that introduces it.
- Comments say why, not what. A comment that restates the identifier is deleted.

Rationale: a CLI is read by every contributor who adds a command and by every user who
debugs one; predictable Go keeps both fast.

## Quality Gates & Tooling

Toolchain: Go 1.27 (`go.mod`), used by CI through `go-version-file`. The `toolchain`
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

- CI has no `paths-ignore`: a docs-only merge still runs every verification job, but it does
  not publish. Publication is the tag-triggered `release.yml` workflow and remains an
  ask-first release-workflow change.
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
| existing code is restructured | `golang-refactoring` | Principle VI requires an explicit in-scope task and its own commit; this is how it stays behaviour-preserving. |
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

- **Spec-first, same PR.** Every feature starts as `specs/NNN-<feature>/` (spec, plan,
  tasks), committed as `docs(speckit): add NNN-<feature> spec/plan/tasks` BEFORE any `feat`
  or `fix` commit. Specification and implementation stay in the same milestone PR.
- **Milestones.** One active milestone is delivered by one PR containing its specification,
  implementation, validation, and documentation commits. Split or stacked PRs require an
  explicit maintainer request. The PR MUST carry the active milestone when created, and
  every completed issue is linked for closure only when that reviewed PR lands on `main`.
  `scripts/check-linkage.sh --pr N` is the merge gate. A completed milestone is released by
  one `vX.Y.Z` tag; before pushing it, `scripts/check-linkage.sh --for-tag vX.Y.Z` must pass.
  `.claude/hooks/linkage-guard.sh` and `.githooks/pre-push` protect that irreversible tag
  push.
- **Commits.** Semantic messages (`feat(scope): … (#NNN)`); exactly one task per commit and
  one commit per task; every commit green on its own; format before commit. A validation-only
  task commits its checkbox and evidence. Intent, not path: squash churn before review. The
  type records intent for the changelog: `feat` for a user-visible
  capability, `fix` for a defect, `docs`/`ci`/`chore`/`test`/`refactor` for everything else.
- **Branches and PRs.** Branch from `main`, push every milestone task to the same PR, and
  rebase only: no merge commits in PR branches. Never force-push to `main` or commit to it
  directly. Agents may prepare and update the milestone PR but leave it open for maintainer
  review.
- **Ask first.** New dependencies or upgrades; changes to public API signatures, observable
  behaviour or serialized formats (Principle V); edits to another repository; release or
  publish workflow changes; a licence change.
- **Never.** Commit broken code; refactor outside the milestone's scope; mock an external
  system where a real integration path exists; phrase a command to evade a guard; merge or
  close a PR without an explicit post-review instruction for that exact PR from the
  maintainer. Agents leave prepared PRs open for maintainer review.
- **Releases are tag-triggered.** Merges run `ci.yml` verification and never create a tag,
  GitHub Release, or published image. One completed `vX.Y.0` milestone is released by one
  deliberate `vX.Y.Z` tag on `main` or `release/X.Y.0`. `release.yml` first validates the
  tag's branch and the complete release range through `check-linkage.sh --for-tag`, reruns
  all Go, shell, and image gates, then runs GoReleaser and publishes the Docker image. Never
  delete a release tag once the job has started; never reuse a version number.
- **Hotfix.** A fix lands on `main` through a PR. Cherry-pick it to the matching release
  branch when required, then create the next patch tag after its milestone audit passes.

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
  through PR review, remains open until the maintainer explicitly merges it, and is
  committed as
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

**Version**: 2.0.0 | **Ratified**: 2026-09-02 | **Last Amended**: 2026-09-14
