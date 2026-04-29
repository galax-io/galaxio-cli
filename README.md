# galaxio-cli

[![CI](https://github.com/galax-io/galaxio-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/galax-io/galaxio-cli/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/galax-io/galaxio-cli/branch/main/graph/badge.svg)](https://codecov.io/gh/galax-io/galaxio-cli)
[![Latest Release](https://img.shields.io/github/v/release/galax-io/galaxio-cli?sort=semver)](https://github.com/galax-io/galaxio-cli/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/galax-io/galaxio-cli)](https://goreportcard.com/report/github.com/galax-io/galaxio-cli)
[![License](https://img.shields.io/github/license/galax-io/galaxio-cli)](https://github.com/galax-io/galaxio-cli/blob/main/LICENSE)

`galaxio` is the command-line interface for Galaxio workflows.

This is the initial version of the CLI. It includes help, version output, shell
completion, and stable exit codes for scripts and CI.

## Install

Install the latest version with Go:

```sh
go install github.com/galax-io/galaxio-cli/cmd/galaxio@latest
```

Or download a prebuilt binary from the
[GitHub Releases](https://github.com/galax-io/galaxio-cli/releases) page.

## Usage

Show available commands and flags:

```sh
galaxio --help
```

Print build information:

```sh
galaxio version
galaxio --version
```

Control diagnostic output:

```sh
galaxio --verbose version
galaxio --quiet version
galaxio --no-color version
```

Generate shell completion:

```sh
galaxio completion bash
galaxio completion zsh
galaxio completion fish
galaxio completion powershell
```

## Exit Codes

`galaxio` keeps exit codes stable for scripts and CI:

| Code | Meaning |
| --- | --- |
| `0` | Command completed successfully. |
| `1` | Runtime failure while executing a valid command. |
| `2` | Usage error such as invalid flags, arguments, or commands. |

## Build From Source

Clone the repository and build the binary:

```sh
git clone https://github.com/galax-io/galaxio-cli.git
cd galaxio-cli
go build -o bin/galaxio ./cmd/galaxio
```

Run the test suite:

```sh
go test ./...
```

## License

This project is distributed under the terms described in [LICENSE](LICENSE).
