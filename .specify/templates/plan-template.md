# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]

**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

[Extract from feature spec: primary requirement + technical approach from research]

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: [e.g., Python 3.11, Swift 5.9, Rust 1.75 or NEEDS CLARIFICATION]

**Primary Dependencies**: [e.g., FastAPI, UIKit, LLVM or NEEDS CLARIFICATION]

**Storage**: [if applicable, e.g., PostgreSQL, CoreData, files or N/A]

**Testing**: [e.g., pytest, XCTest, cargo test or NEEDS CLARIFICATION]

**Target Platform**: [e.g., Linux server, iOS 15+, WASM or NEEDS CLARIFICATION]

**Project Type**: [e.g., library/cli/web-service/mobile-app/compiler/desktop-app or NEEDS CLARIFICATION]

**Performance Goals**: [domain-specific, e.g., 1000 req/s, 10k lines/sec, 60 fps or NEEDS CLARIFICATION]

**Constraints**: [domain-specific, e.g., <200ms p95, <100MB memory, offline-capable or NEEDS CLARIFICATION]

**Scale/Scope**: [domain-specific, e.g., 10k users, 1M LOC, 50 screens or NEEDS CLARIFICATION]

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Answer each gate PASS or FAIL with one line of evidence. A FAIL needs a Complexity Tracking row.

| # | Gate (constitution principle) | Status |
|---|---|---|
| I | Command Contract — every new command is a thin cobra wrapper over `runX(ctx, opts)`; offers `-o`, valued `text\|json` where it encodes one output and named by product where it selects among several, with a documented structure for any machine-readable form and no interim one invented; exits 0/1/2 via `UsageError`/`RuntimeError`; honours `--verbose`/`--quiet`/`--no-color`; experimental commands sit behind `internal/featureflags`. | |
| II | Report Arithmetic Lives Here — (report features only) statistics are computed in `internal/report/` over `parsec` primitives, not requested from the library; success and failure accumulated separately; every figure exact, percentiles included, with a refusal rather than an estimate where a bounded accumulator cannot keep one exact; no parity with another tool's percentiles claimed or tested for; one pass, bounded memory, with the peak-memory goal stated in Technical Context; absence reported as absent; source detected by content. Mark N/A for non-report features. | |
| III | Tests Land With The Change — stdlib `testing`, table-driven, golden files under `testdata/`; command-level tests through `runCLI` asserting exit code and output; integration tests behind the `integration` tag on real packs/registries/specs; race on; coverage stays ≥ 80%; every fix carries a regression test; test tasks are never optional. | |
| IV | Minimal, Explicit Dependencies — no new module unless named here with the reason the standard library or an existing dependency is insufficient, recorded in `research.md`, licence-compatible with GPL-2.0-only, and asked for first. | |
| V | Published Surfaces — any change to a command, flag, default, exit code, `-o json` structure, manifest/registry schema or generated output is listed; breaking ones are approved before implementation and will be committed with `!`; README updated in the same PR; deprecations keep working one minor release. | |
| VI | Idiomatic, Simple Go — gofmt/vet clean; errors as values wrapped into `UsageError`/`RuntimeError` at the boundary; no panic control flow; no dead or duplicated code; no refactor outside this issue's scope. | |

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
# [REMOVE IF UNUSED] Option 1: Single project (DEFAULT)
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/

# [REMOVE IF UNUSED] Option 2: Web application (when "frontend" + "backend" detected)
backend/
├── src/
│   ├── models/
│   ├── services/
│   └── api/
└── tests/

frontend/
├── src/
│   ├── components/
│   ├── pages/
│   └── services/
└── tests/

# [REMOVE IF UNUSED] Option 3: Mobile + API (when "iOS/Android" detected)
api/
└── [same as backend above]

ios/ or android/
└── [platform-specific structure: feature modules, UI flows, platform tests]
```

**Structure Decision**: [Document the selected structure and reference the real
directories captured above]

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
