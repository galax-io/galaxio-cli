# galaxio-cli

`galaxio` is a minimal, enterprise-style command-line application scaffolded for
predictable automation, stable releases, and idiomatic Go development.

## Features

- Minimal help and version commands.
- Stable exit codes: `0` for success, `1` for runtime failures, and `2` for
  command-line usage errors.
- Global `--no-color`, `--verbose`, and `--quiet` flags.
- Version metadata embedded at build time with Go linker flags.
- Unit-tested Cobra command execution.
- CI for formatting, module tidiness, vet, tests, and binary build.
- Tag-driven releases with GoReleaser.

## Requirements

- Go 1.24 or newer.

## Usage

Print help:

```sh
galaxio --help
```

Print version information:

```sh
galaxio version
galaxio --version
```

Use global output-control flags:

```sh
galaxio --verbose version
galaxio --quiet version
galaxio --no-color version
```

## Build

Build a local binary:

```sh
go build -o bin/galaxio ./cmd/galaxio
```

Build with version metadata:

```sh
go build \
  -ldflags "-X github.com/galax-io/galaxio-cli/internal/buildinfo.Version=v0.1.0 \
    -X github.com/galax-io/galaxio-cli/internal/buildinfo.Commit=$(git rev-parse --short HEAD) \
    -X github.com/galax-io/galaxio-cli/internal/buildinfo.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o bin/galaxio ./cmd/galaxio
```

## Test

Run the test suite:

```sh
go test ./...
```

Run the same core checks as CI:

```sh
test -z "$(gofmt -l .)"
go mod tidy
go vet ./...
go test -race -coverprofile=coverage.out ./...
go build -trimpath -o dist/galaxio ./cmd/galaxio
```

## Install

Install from the current checkout:

```sh
go install ./cmd/galaxio
```

Install from the module path:

```sh
go install github.com/galax-io/galaxio-cli/cmd/galaxio@latest
```

## Release

Releases are driven by Git tags. Create and push a semver tag:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The release workflow runs GoReleaser, builds cross-platform archives, publishes
checksums, and generates release notes from Conventional Commits.

## Conventional Commits

Use Conventional Commits so generated changelogs stay readable:

```text
feat: add profile configuration
fix: return usage exit code for invalid arguments
chore: update ci workflow
```
