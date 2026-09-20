# Quality validation

**Feature**: Restore legacy Gatling statistics files
**Validated**: 2026-09-20
**Environment**: native macOS checkout

## Results

| Gate | Command | Result |
|---|---|---|
| Formatting | `test -z "$(gofmt -l .)"` | PASS |
| Module hygiene | `go mod tidy -diff` | PASS, no module changes |
| Vet | `go vet ./...` | PASS |
| Race and coverage | `go test -race -coverprofile=<temp>/coverage.out ./...` | PASS |
| Coverage floor | `go tool cover -func=<temp>/coverage.out` | PASS, 87.3% total |
| Build | `go build -trimpath -o <temp>/galaxio ./cmd/galaxio` | PASS |
| Integration | `go test -tags=integration -race -count=1 ./...` | PASS |
| Shell suites | every `*_test.sh` under `scripts/`, `.claude/hooks/` and `.githooks/` | PASS, 8 suites |

## Feature checks

- Gatling 3.11.5, 3.12.0, 3.13.1, 3.14.9 and 3.15.1 corpus exports pass.
- `stats.json` and `global_stats.json` are independently selectable and silent on success.
- Repeated exports with equal options produce equal bytes.
- Existing selected files require `--overwrite`; unselected files remain unchanged.
- Existing directories and symlinks at publication paths are refused.
- Damaged or truncated logs publish neither legacy file.
- Empty OK or KO columns retain zero-valued legacy placeholders with a zero count.

No Jenkins fixture, JVM oracle, Docker build or operating-system matrix was introduced or
required for this validation.
