# galaxio-cli Constitution

galaxio-cli is the command-line face of Galaxio: it scaffolds load-test projects, generates test code from API descriptions, and reports on finished runs. It is installed by people who then type its commands every day, and it releases itself from `main`, so every merged change is a change users get.

**Version**: 1.0.0 · **Ratified**: 2026-09-02 · **Last amended**: 2026-09-02

## Core Principles

### I. A command is a thin shell over a function

Each command is a cobra wrapper around a `runX(ctx, opts) (XOutput, error)` function that holds the logic and knows nothing about cobra. Commands live one per file under `cmd/galaxio/` and are registered in `root.go`. Logic that could be tested without a terminal is tested without one.

### II. Every command speaks both to a person and to a script

Human output goes to standard output, diagnostics and warnings to standard error, and every command offers `-o text|json` with a documented structure behind the JSON. A usage mistake exits 2, a runtime failure exits 1, through `UsageError` and `RuntimeError`; nothing exits zero after failing.

### III. Merging publishes

The release is computed from conventional commits on `main`: a `feat:` commit cuts a minor release and reaches users within minutes. A change that is not ready to ship is not merged, and a chore that must not release is labelled as one.

### IV. Parsing belongs to the library

Reading a load-testing tool's results is `parsec`'s job, pinned to an exact version. This repository discovers files, chooses formats, renders output and sets exit codes. A decoder is never reimplemented here, and neither is the OpenNFR document model, which comes from its own SDK.

### V. Tests use the standard library

Standard-library testing only: table-driven subtests, golden files under `testdata`, and command-level tests that drive `execute()` and assert exit codes. No assertion library. Integration tests sit behind the `integration` build tag. Coverage is gated at 80 percent and the race detector is always on.

### VI. Say what happened

An error names what went wrong and what to do about it: which directory was searched, which versions are supported, which flag would change the outcome. A silent failure, or output that looks like a result when nothing was read, is a defect.

## Code Conventions

Go 1.25. `gofmt` clean, `go vet` green, `go mod tidy` leaves no diff. `context.Context` first where it applies; no `init()`; errors wrap with `%w`. Experimental commands are gated through `internal/featureflags` until their behaviour is settled. Conventional Commits; branches named `feat/N-slug`; one issue, one commit, one pull request that closes it.

## Quality Gates

CI runs formatting, tidiness, vet, build, race-enabled tests with the coverage gate, and integration tests. A red build is never merged. The linkage check must pass: every pull request belongs to a milestone and closes an issue.

## Governance

This constitution governs. A change to it is a pull request stating what changed and why, with the version above bumped: major for a removed or reversed principle, minor for a new one, patch for wording.

Work follows the organisation's spec-driven flow: a milestone states the problem, `/speckit.specify` turns it into a spec, and the spec's tasks become the commits.
