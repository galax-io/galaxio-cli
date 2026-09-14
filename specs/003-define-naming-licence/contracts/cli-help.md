# Contract: `galaxio report` Namespace

## Scope

This contract reserves and exposes the public reporting namespace. It does not define a
report operation, input source, calculation, flag, or result schema.

## Root discovery

Given `galaxio --help`:

- exit code is `0`;
- stdout contains a root command entry named `report`;
- the short description is `Report on finished load-test runs.`;
- stderr is empty;
- every previously published root command remains present.

## Direct invocation

Both invocations are valid:

```text
galaxio report
galaxio report --help
```

For each invocation:

- exit code is `0`;
- stdout contains `Report on finished load-test runs.`, a long-description statement that
  operational subcommands are introduced separately, `Usage:`, and
  `galaxio report [flags]` (or the Cobra-equivalent usage line);
- stderr is empty;
- no input is opened, no report is calculated, and no file is written.

The group inherits the existing root flags `--verbose`, `--quiet`, and `--no-color` through
Cobra. Because the group emits help rather than a report result, it does not introduce
`-o`, text output, JSON output, or a `runX` domain function. Those become mandatory on the
first operational report subcommand and must be specified before implementation.

## Invalid arguments

Given an unrecognized child or positional argument, for example:

```text
galaxio report unexpected
```

- exit code is `2`;
- stdout is empty;
- stderr contains an actionable Cobra argument/unknown-command diagnostic;
- the error is normalized as `UsageError`, not `RuntimeError`.

## Compatibility

- The change is additive: no command, alias, flag, default, exit code, JSON structure,
  schema, or generated output is removed or changed.
- `report` is the only accepted public name for this namespace. Rejected repository names
  are not CLI aliases.
- The command is not feature-gated because root-help visibility is acceptance evidence.
- Adding any operational subcommand is a separate compatibility-sensitive feature.

## Verification seam

Command tests MUST call `runCLI`, not the constructor directly, so root registration,
global handling, exit codes, stdout, and stderr are exercised together.
