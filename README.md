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

Install a tagged release binary:

```sh
curl -fsSL https://raw.githubusercontent.com/galax-io/galaxio-cli/main/scripts/install.sh | sh
GALAXIO_VERSION=0.1.1 sh scripts/install.sh
```

The installer writes to `$HOME/.local/bin` by default. Add it to `PATH` if
needed:

```sh
export PATH="$HOME/.local/bin:$PATH"
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

Example zsh install:

```sh
mkdir -p ~/.zsh/completion
galaxio completion zsh > ~/.zsh/completion/_galaxio
```

Update `galaxio` from GitHub Releases:

```sh
galaxio update
galaxio update --dry-run
galaxio update --version 0.1.1
```

## Template Workflow

The CLI ships with a default registry (`github:galax-io/galaxio-template-registry`),
so template discovery works out of the box with no configuration.

Browse available templates:

```sh
galaxio template list
```

Scaffold a new project from a template (files are written to `--destination`,
which defaults to the current directory):

```sh
galaxio template init gatling/scala-sbt -d ./my-project
```

Override template inputs with `--set`:

```sh
galaxio template init gatling/scala-sbt --set Name=orders -d ./my-project
```

Validate a template pack (useful for template authors):

```sh
galaxio template validate local:../my-templates
```

### End-to-end example

```sh
# 1. See what templates are available
galaxio template list

# 2. Create a project from a template
galaxio template init gatling/scala-sbt -d ./perf-tests
```

### For template authors

Validate a template pack before publishing:

```sh
galaxio template validate local:./my-pack
```

### Custom registries

The default registry is used automatically. Run `template configure` only when
you need a different registry:

```sh
galaxio template configure --registry github:my-org/my-registry
galaxio template configure --show
```

For manifest format details and local examples, see
[docs/templates/README.md](docs/templates/README.md) and `examples/templates/`.

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
