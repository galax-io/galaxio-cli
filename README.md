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

### Supported platforms

Release binaries are built for every OS/architecture combination below.

| OS | Arch | `go install` | Shell installer | Manual download |
| --- | --- | :---: | :---: | :---: |
| macOS (darwin) | amd64 | yes | yes | yes |
| macOS (darwin) | arm64 (Apple Silicon) | yes | yes | yes |
| Linux | amd64 | yes | yes | yes |
| Linux | arm64 | yes | yes | yes |
| Windows | amd64 | yes | no* | yes |
| Windows | arm64 | yes | no* | yes |

> \* The shell installer targets POSIX shells. Windows users should use
> `go install` or download a `.zip` from GitHub Releases (see below).

### Option 1 — `go install` (all platforms)

Requires Go 1.24 or later on your `PATH`.

```sh
go install github.com/galax-io/galaxio-cli/cmd/galaxio@latest
```

The binary is placed in `$(go env GOPATH)/bin`. Make sure that directory is on
your `PATH` (see [PATH setup](#path-setup)).

### Option 2 — Shell installer (macOS and Linux)

Prerequisites: `curl`, `shasum`, `tar`, `unzip`.

```sh
curl -fsSL https://raw.githubusercontent.com/galax-io/galaxio-cli/main/scripts/install.sh | sh
```

Pin a specific version:

```sh
curl -fsSL https://raw.githubusercontent.com/galax-io/galaxio-cli/main/scripts/install.sh | GALAXIO_VERSION=0.1.1 sh
```

The installer writes to `$HOME/.local/bin` by default. Override with
`GALAXIO_BIN_DIR`:

```sh
curl -fsSL https://raw.githubusercontent.com/galax-io/galaxio-cli/main/scripts/install.sh | GALAXIO_BIN_DIR=/usr/local/bin sh
```

### Option 3 — Manual download (all platforms)

Download a prebuilt binary from the
[GitHub Releases](https://github.com/galax-io/galaxio-cli/releases) page.
Archives are named `galaxio_<version>_<os>_<arch>.tar.gz` (`.zip` on Windows).

1. Download the archive for your platform.
2. Verify the checksum against `checksums.txt` from the same release.
3. Extract the `galaxio` binary (or `galaxio.exe` on Windows).
4. Move it to a directory on your `PATH`.

### Windows recommended path

Windows does not ship with a POSIX shell, so use one of:

1. **`go install`** — simplest if Go is already installed.
2. **Manual download** — grab the `.zip` from GitHub Releases, extract
   `galaxio.exe`, and place it in a directory listed in your `Path` environment
   variable (for example `C:\Users\<you>\bin`).

To add a directory to your `Path` on Windows:

```
setx Path "%Path%;C:\Users\<you>\bin"
```

Then open a **new** terminal for the change to take effect.

### PATH setup

After installing, verify the binary is reachable:

```sh
galaxio --version
```

If the command is not found, add the install location to your `PATH`:

**macOS / Linux (shell installer default)**

```sh
# Add to ~/.bashrc, ~/.zshrc, or equivalent:
export PATH="$HOME/.local/bin:$PATH"
```

**macOS / Linux (`go install` default)**

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

**Windows**

```
setx Path "%Path%;%USERPROFILE%\go\bin"
```

### Troubleshooting

#### `command not found` after install

The binary is not on your `PATH`. Check where it was installed and add that
directory (see [PATH setup](#path-setup)). Open a new terminal session after
changing `PATH`.

```sh
# Find the binary:
ls ~/.local/bin/galaxio        # shell installer default
ls "$(go env GOPATH)/bin/galaxio"  # go install default
```

#### `unsupported arch` from the shell installer

The installer only recognises `x86_64`/`amd64` and `arm64`/`aarch64`. If
`uname -m` returns something else (for example `armv7l` on a 32-bit ARM
device), use `go install` to cross-compile for your architecture instead.

#### GitHub API / network / proxy errors

The shell installer downloads from the GitHub API and Releases CDN. Common
failures:

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| `curl: (7) Failed to connect` | Corporate proxy or firewall | Set `https_proxy` before running the installer: `https_proxy=http://proxy:port sh scripts/install.sh` |
| `curl: (28) Connection timed out` | Slow or blocked network | Retry, or download the binary manually from a browser |
| HTTP 403 / rate-limit error | GitHub API rate limit (60 req/h unauthenticated) | Wait an hour, or download the binary manually from the [Releases](https://github.com/galax-io/galaxio-cli/releases) page |
| `release asset not found` | OS/arch not in the release | Verify your platform is in the support matrix above; fall back to `go install` |

#### Checksum verification fails

If `shasum` reports a mismatch, the download may be corrupted or tampered with.
Delete the partial install and retry:

```sh
rm -f ~/.local/bin/galaxio
curl -fsSL https://raw.githubusercontent.com/galax-io/galaxio-cli/main/scripts/install.sh | sh
```

If the problem persists, download the archive and `checksums.txt` manually and
compare.

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
