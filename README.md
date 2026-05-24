# galaxio-cli

[![CI](https://github.com/galax-io/galaxio-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/galax-io/galaxio-cli/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/galax-io/galaxio-cli/branch/main/graph/badge.svg)](https://codecov.io/gh/galax-io/galaxio-cli)
[![Latest Release](https://img.shields.io/github/v/release/galax-io/galaxio-cli?sort=semver)](https://github.com/galax-io/galaxio-cli/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/galax-io/galaxio-cli)](https://goreportcard.com/report/github.com/galax-io/galaxio-cli)
[![License](https://img.shields.io/github/license/galax-io/galaxio-cli)](https://github.com/galax-io/galaxio-cli/blob/main/LICENSE)

`galaxio` is a CLI for Gatling performance testing workflows. It scaffolds new
load-test projects from templates and generates Gatling scripts from API
specifications.

**Key features:**

- `template init` — scaffold a ready-to-compile Gatling project from a template
- `template list` — discover available templates from any registry
- `generate swagger / har / postman` — generate Gatling scripts from an existing API spec
- `doctor` — validate CLI configuration and registry access
- `update` — self-update from GitHub Releases

## Install

### Supported platforms

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

### Option 4 — Docker image

The image is published to Docker Hub as `galax-io/galaxio:<release-version>`,
plus the alias `galax-io/galaxio:<major.minor>` and `galax-io/galaxio:latest`.

```sh
docker pull galax-io/galaxio:0.1.1
docker pull galax-io/galaxio:0.1
docker pull galax-io/galaxio:latest
docker run --rm galax-io/galaxio --help
```

For commands that create files, mount a working directory and set it as the
container workdir:

```sh
docker run --rm -v "$PWD":/work -w /work galax-io/galaxio template list
```

### Windows

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

### Troubleshooting installation

#### `command not found` after install

The binary is not on your `PATH`. Check where it was installed:

```sh
ls ~/.local/bin/galaxio        # shell installer default
ls "$(go env GOPATH)/bin/galaxio"  # go install default
```

Open a new terminal after changing `PATH`.

#### `unsupported arch` from the shell installer

The installer only recognises `x86_64`/`amd64` and `arm64`/`aarch64`. For
other architectures (e.g. `armv7l`), use `go install` instead.

#### GitHub API / network / proxy errors

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| `curl: (7) Failed to connect` | Corporate proxy or firewall | Set `https_proxy` before running the installer: `https_proxy=http://proxy:port sh scripts/install.sh` |
| `curl: (28) Connection timed out` | Slow or blocked network | Retry, or download the binary manually |
| HTTP 403 / rate-limit error | GitHub API rate limit (60 req/h unauthenticated) | Wait an hour, or download manually from the [Releases](https://github.com/galax-io/galaxio-cli/releases) page |
| `release asset not found` | OS/arch not in the release | Verify your platform is in the support matrix; fall back to `go install` |

#### Checksum verification fails

The download may be corrupted. Delete the partial install and retry:

```sh
rm -f ~/.local/bin/galaxio
curl -fsSL https://raw.githubusercontent.com/galax-io/galaxio-cli/main/scripts/install.sh | sh
```

## Usage

```sh
galaxio --help          # show commands and flags
galaxio version         # print build info
galaxio --verbose ...   # verbose diagnostic output
galaxio --quiet ...     # suppress non-error output
galaxio --no-color ...  # disable colour
```

Update `galaxio` from GitHub Releases:

```sh
galaxio update
galaxio update --dry-run
galaxio update --version 0.1.1
```

Generate shell completions:

```sh
galaxio completion bash
galaxio completion zsh
galaxio completion fish

# Install for zsh:
mkdir -p ~/.zsh/completion
galaxio completion zsh > ~/.zsh/completion/_galaxio
```

## Template Workflow

The CLI ships with a default registry (`github:galax-io/galaxio-template-registry`).
Template discovery works out of the box — no configuration needed.

### List available templates

```sh
galaxio template list
galaxio template list -o json
```

### Scaffold a project

Files are written to `--destination` (defaults to the current directory):

```sh
galaxio template init gatling/scala-sbt -d ./perf-tests

# Override template inputs
galaxio template init gatling/scala-sbt \
  --set Name=orders-api \
  --set Package=org.example.perf \
  --set PackagePath=org/example/perf \
  -d ./perf-tests

# Supply inputs from a YAML file
galaxio template init gatling/scala-sbt --values ./values.yaml -d ./perf-tests
```

### Enable optional plugins

Gatling templates support optional Kafka, JDBC, and AMQP plugin modules:

```sh
galaxio template init gatling/scala-sbt \
  --set KafkaPluginEnabled=true \
  --set Name=orders-api \
  -d ./perf-tests
```

See [templates-gatling](https://github.com/galax-io/templates-gatling) for the
full list of available templates and inputs.

### Scaffold and overlay generated files together

Use `--init` on a `generate` command to scaffold a full project first, then
overlay the generated API files on top in one step:

```sh
galaxio generate swagger --from ./petstore.yaml --init -d ./perf-tests
```

### Custom registries

The default registry is used automatically. Override it only when you need a
different source:

```sh
galaxio template configure --registry github:my-org/my-registry
galaxio template configure --show   # inspect current setting

# Override for a single command without changing config
galaxio template init mypack/mytemplate --registry local:/path/to/registry
```

Registry source formats:

| Format | Example | Notes |
| --- | --- | --- |
| `github:owner/repo` | `github:galax-io/templates-gatling` | Resolves pack `version` to GitHub release tag `v{version}` |
| `local:path` | `local:../my-templates` | Resolved from current working directory |

### Validate a template pack

```sh
galaxio template validate local:./my-pack
galaxio template validate github:my-org/my-templates
```

### Clear the template cache

Downloaded GitHub template archives are cached locally. Clear the cache when
you need to force a fresh download:

```sh
galaxio template clear-cache
```

## Code Generation

Generate Gatling load-test scripts from existing API specifications.
The `--template` flag selects the Gatling DSL flavour (default: `scala-sbt`).

### From Swagger / OpenAPI

```sh
galaxio generate swagger --from ./petstore.yaml
galaxio generate swagger --from ./petstore.yaml --dest ./out
galaxio generate swagger --from ./petstore.yaml --package org.example.perf
galaxio generate swagger --from ./petstore.yaml -o json   # machine-readable summary
```

### From HAR recording

```sh
galaxio generate har --from ./recording.har
galaxio generate har --from ./recording.har --dest ./out --include-static
```

### From Postman collection

```sh
galaxio generate postman --from ./collection.json
galaxio generate postman --from ./collection.json --dest ./out
```

### Conflict strategy (`--if-exists`)

When generating into a directory that already contains files, control how
conflicts are resolved:

| Strategy | Behaviour |
| --- | --- |
| `suffix` (default) | Writes `file.generated.scala` alongside the existing `file.scala` |
| `merge` | Writes conflict markers (`<<<< generated / ==== / >>>> existing`) into the existing file |
| `skip` | Leaves the existing file untouched and emits a warning to stderr |
| `overwrite` | Replaces the existing file; skips the write if content is identical |

```sh
galaxio generate swagger --from ./petstore.yaml --if-exists overwrite
galaxio generate swagger --from ./petstore.yaml --if-exists skip
```

## Diagnostics

Check CLI configuration and registry connectivity:

```sh
galaxio doctor
galaxio doctor --registry github:my-org/my-registry
galaxio doctor -o json
```

`doctor` verifies that the configured registry is reachable, the pack manifest
is valid, and all declared templates can be resolved.

## Environment Variables

| Variable | Default | Purpose |
| --- | --- | --- |
| `GALAXIO_CONFIG` | `~/.galaxio/config.yaml` | Override the config file path (useful in CI or Docker) |
| `GALAXIO_CACHE_DIR` | OS user cache dir | Override the template cache directory |

## Exit Codes

| Code | Meaning |
| --- | --- |
| `0` | Command completed successfully. |
| `1` | Runtime failure while executing a valid command. |
| `2` | Usage error — invalid flags, arguments, or commands. |

## Build From Source

```sh
git clone https://github.com/galax-io/galaxio-cli.git
cd galaxio-cli
go build -o bin/galaxio ./cmd/galaxio
go test ./...
```

## License

This project is distributed under the terms described in [LICENSE](LICENSE).
