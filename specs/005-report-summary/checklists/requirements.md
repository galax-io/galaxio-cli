# Specification Quality Checklist: Summarise a Finished Gatling Run

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-09-17
**Re-validated**: 2026-09-17, after `/speckit-clarify` and after the maintainer's decisions
on files, layout, wording and progress of the same day
**Feature**: [spec.md](../spec.md)

## Content Quality

- [ ] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [ ] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [ ] No implementation details leak into specification

## Notes

- **13 of 16 complete; three items knowingly accepted, and all three are the same
  deviation.** The specification names the digest library, the quantile it gives by default
  and the upstream pull request that changes it, and it names the command's flags, exit
  codes and the two words its output uses for an outcome. The first is not a leak: recording
  that choice is what the maintainer asked this specification to do ("только пометь что есть
  pr … сейчас мы используем по умолчанию тот что дает caio"), and FR-020 and SC-006 exist so
  that the choice and its known consequence — 1427 ms printed where the request at the rank
  took 1502 ms — are visible to users and to the change that later removes it. The second
  follows `specs/004-report-records`: for a CLI the command surface is the deliverable.
- **Every figure quoted in an acceptance scenario was read from what Gatling recorded** for
  the 3.13.1 run — its console summary and its `js/global_stats.json` in the shared
  library's corpus — and the 1427 ms percentile was measured with the named library at its
  default settings. None is an estimate of what the command will print.
- **At `/speckit-specify` no clarification was asked.** Three points had more than one
  reasonable reading, and each was settled by an existing decision rather than a question:
  `-o json` from the issue is superseded by constitution v2.1.0 Principle I; the two
  exit-code policies (zero span, lost outcome) are taken from issue #51 as written and called
  out under Principle V; the per-message error table is left out because issue #51 settles
  no figure for it. Each is recorded in Assumptions; `/speckit-clarify` has since confirmed
  the third.
- **Clarified 2026-09-17, seven answers, all integrated.** (1) The digest decision does not
  meet the exactness clause of constitution Principle II, and the clause is amended first —
  its own issue and pull request inside milestone `v0.14.0`, as #112 was — before any
  implementation. (2) The console carries the whole-run summary only; figures for every
  request and group go to the products `-o` names, which are the only files this command is
  to write and arrive with their own milestones, so this milestone writes no file (FR-002,
  FR-016). (3) The per-message error table is out of this milestone. (4) Group rows by
  wall-clock time are deferred, with every other group figure. (5) Progress while reading is
  in, as a P3 story and as a live block (FR-037 to FR-042, SC-009, SC-010). (6) The layout is
  this tool's own and tool-independent, chosen by the maintainer from three mock-ups
  (FR-006, FR-007, SC-011). (7) The outcomes are `ok` and `failed` everywhere, which changes
  one word of the released v0.13.0 description, with the maintainer's approval (FR-005).
- **Answer (2) was first misread.** A draft of this specification added a text file of its
  own for the rows per request and group; the maintainer rejected it — no file other than
  the `-o` products — and then confirmed that the console carries the summary only. Nothing
  of that file remains here or in the plan; it is recorded in research §10 as a rejected
  alternative.
- **Re-validated after those answers:** six user stories (P1, four P2, P3), FR-001 to FR-042
  contiguous and in page order, SC-001 to SC-011 contiguous, every cross-reference resolved,
  no statement left that contradicts an answer. Plan, research, data model, contract and
  quickstart were updated in the same change.
- **Left for the plan on purpose:** the names of the two new options (percentile ranks, band
  boundaries), the exact layout of the summary and of the progress block, the number of
  decimals a rate is printed with where the corpus gives no evidence, and the peak-memory
  figure. Each is either published surface the plan must name under Principle V or research
  against Gatling's source, not a question for the maintainer.
- FR-022 constrains a change that is outside this milestone, so nothing in this milestone
  tests it; it is kept because it is the other half of the note the maintainer asked for.
