# Implementation plan

## Approach

1. Extend the existing report scan with an optional per-position collector so the log is
   still read once.
2. Convert the root and position summaries into the historical JSON field structure using
   the standard library JSON encoder.
3. Publish only the selected fixed filenames under the run's `js` directory, refusing
   existing files unless `--overwrite` is set.
4. Exercise the feature through the real CLI against every committed Gatling corpus
   version, plus focused selection, overwrite and name-preservation cases.

## Scope

- Production code stays in `internal/report/`, `internal/report/legacy/` and the existing
  `report` cobra command.
- No new dependency, service, container, CI job or platform runner is required.
- Memory grows with distinct request/group positions and rendered JSON, not sample count.

## Verification

```sh
gofmt -w .
go vet ./...
go test -race ./...
go build ./...
```
