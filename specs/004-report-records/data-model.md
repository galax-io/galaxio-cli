# Data Model: Read a Finished Gatling Run

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Contract**:
[contracts/cli.md](contracts/cli.md)

Nothing here is persisted, and almost nothing here is a new vocabulary. Three of the five
entities are types the shared library already defines and this feature uses unchanged. Two
are this repository's: the tally, because counting is the consumer's job and the library
counts nothing, and the open source, because pairing a reader with the format detected for
the report is this command's concern and not the library's.

## Run location (library type)

Where the run's artefacts sit and how they were chosen. Produced by the library's run
finder, consumed when opening the log and when reporting which run was read, then discarded.

| Attribute | Meaning |
|---|---|
| directory | the run directory, cleaned; absolute only if the argument was |
| log | the `simulation.log` inside it |
| found by | the rule that chose it: named path, last-run marker, or newest |

Resolution: an empty argument means the Maven and sbt results root; a path holding a
`simulation.log`, or that file itself, is the run; any other directory is a results root
searched by the marker, then by modification time. A root with no run is an absence of
runs; a path that cannot be read is a read failure and never an absence.

## Run description (library type)

Everything about the run that does not grow with its length, exactly as the library returns
it: identifier, simulation name, description, recorded start, tool, tool version, the
warnings its version gate raised, what the source can never record, and the opaque
assertion payloads. This feature reports a subset of it and stores none of it.

The identifier is not unique across runs of one simulation; identity is the pair of
identifier and start.

## Run span (library type)

Where the run begins and ends, as the library's bounds define it: the earliest of every
sample start, group start and virtual-user start, and the latest of every end. The library
owns this definition because every rate a later milestone prints divides by it. This feature
extends one bounds value over the pass and reports both instants and the time between them.
It derives no rate and no statistic from them.

An item the source could not place in time makes the bounds unusable rather than being
skipped, and the span line is then left out of the report.

## Read tally (this repository)

What one pass counted. One of the two types this feature declares, because the library
exports no count; the other is the open source above.

| Field | Meaning |
|---|---|
| requests | samples walked |
| successes / failures | those samples split by the outcome the source recorded, never inferred from any other field |
| unknown outcomes | samples whose outcome the source lost; counted in requests and in neither split |
| groups | group traversals walked |
| users | virtual-user events walked |
| errors | run-level errors walked |
| assertions | declared-assertion payloads the source yielded among its events rather than ahead of them; zero for a Gatling log, which writes its own before the events |
| other | items of a kind this release does not know, which a later library release could introduce; counted so that they are not walked past in silence |
| bounds | the run's span, extended by every item the pass walked; the library's type, carried on the tally because one fold produces both |

Invariant: successes + failures + unknown outcomes = requests, and each is exact. The
report names what each count counts, so that a traversal is not read as a group nor an
event as a virtual user: a run emits one traversal per entry and a start and an end per
user, so both numbers are larger than the thing a reader might take them for.

These are tallies of records walked. They are not statistics: no mean, percentile, range,
deviation or rate is derived from them in this milestone, and the failure counts exist so
that the statistics milestone inherits a split that was never conflated.

## Read source (this repository)

The open run: the log format detected for the report, and the library's reader that yields
the stream. It is closed when the read is over and outlives nothing.

It exists because the format is a fact the report prints and the library's reader does not
expose, and because the file, the reader and the format have one lifetime. It restates no
library definition: the format is the library's own type, and the reader is its interface.

## What is deliberately not declared here

The library already defines each of the following, and this feature uses it rather than
restating it: the position of a sample, the bounds of a run, the outcome of a sample, an
optional value the source may not have recorded, the statement of what a source can never
record, and the discriminated item a run's stream yields. Declaring any of them again would
fork the definitions the ecosystem shares.
