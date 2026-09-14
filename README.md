# galaxio-cli

[![CI](https://github.com/galax-io/galaxio-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/galax-io/galaxio-cli/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/galax-io/galaxio-cli/branch/main/graph/badge.svg)](https://codecov.io/gh/galax-io/galaxio-cli)
[![Latest Release](https://img.shields.io/github/v/release/galax-io/galaxio-cli?sort=semver)](https://github.com/galax-io/galaxio-cli/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/galax-io/galaxio-cli)](https://goreportcard.com/report/github.com/galax-io/galaxio-cli)
[![License](https://img.shields.io/github/license/galax-io/galaxio-cli)](https://github.com/galax-io/galaxio-cli/blob/main/LICENSE)<br>
[![Docker Hub](https://shieldcn.dev/badge/dynamic/json.png?url=https%3A%2F%2Fapi.github.com%2Frepos%2Fgalax-io%2Fgalaxio-cli%2Freleases%2Flatest&query=%24.tag_name&label=docker&logo=docker)](https://github.com/galax-io/galaxio-cli/releases)
[![Utility Size](https://shieldcn.dev/badge/utility%20size-5.8%20MB.png)](https://github.com/galax-io/galaxio-cli/releases/latest)
[![Image Size](https://shieldcn.dev/docker/size/galaxioteam/galaxio.png)](https://hub.docker.com/r/galaxioteam/galaxio)

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

The image is published to Docker Hub as `galaxioteam/galaxio:<release-version>`,
plus the alias `galaxioteam/galaxio:<major.minor>` and `galaxioteam/galaxio:latest`.
The badges at the top show the current Docker Hub tag, the latest release
utility size, and the compressed Docker image size.

```sh
docker pull galaxioteam/galaxio:0.1.1
docker pull galaxioteam/galaxio:0.1
docker pull galaxioteam/galaxio:latest
docker run --rm galaxioteam/galaxio --help
```

For commands that create files, mount a working directory and set it as the
container workdir:

```sh
docker run --rm -v "$PWD":/work -w /work galaxioteam/galaxio template list
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
galaxio report gatling  # write the latest Gatling run as JSON Lines
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

## Reporting Ecosystem

Galaxio uses one canonical name for each reporting component:

- `parsec` is the public result-primitives library at
  [`github.com/galax-io/parsec`](https://github.com/galax-io/parsec).
- `galaxio report` is the CLI namespace for finished-run reporting.

See the [naming and licence decision record](specs/003-define-naming-licence/research.md)
for the selected identities and rejected alternatives.

### Write a run as records

`galaxio report <tool> [PATH]` turns a finished run into a stream of records on standard
output: one JSON object per line, the run header first, then every request, group
traversal, virtual-user event and run-level error in the order the log recorded them. It
computes nothing; statistics and report formats follow in later releases.

```sh
galaxio report gatling                               # the latest run under target/gatling
galaxio report gatling target/gatling/mysim-20260906044741110
galaxio report gatling path/to/simulation.log
galaxio report gatling build/reports/gatling         # a results root: lastRun.txt, else the newest run
galaxio report gatling --quiet | jq -c 'select(.kind=="request")'
galaxio report                                       # help
```

**Tool.** The first argument names the tool that produced the run. `gatling` is the only
tool this release reads: text logs from Gatling 3.11.5 through 3.12.0 and binary logs from
3.13.1 through 3.15.1, identified from the log's content, never its name. A version below
that range, and 3.13.0, are refused with a message naming the version and the range; a
newer version is read and a warning goes to standard error and into the header.

**PATH.** A run directory, its `simulation.log`, or a results root holding run directories.
With no path the Maven and sbt results root `target/gatling` is searched (Gradle writes to
`build/reports/gatling`; pass it). In a results root the run named by `lastRun.txt` wins
if it still exists, otherwise the most recently modified run. Standard error says which run
was read and by which rule; `--quiet` silences that line, `--verbose` adds the log format,
version and a count of records per kind after the stream.

**`-o`.** Reserved for report formats that later releases produce: `stats` and
`global_stats` (Gatling's legacy `js/stats.json` and `js/global_stats.json`) and `yml`
(the OpenNFR YAML report). Passing any of them exits 2 naming the release that delivers
it; without `-o` the record stream is written.

**Exit codes.** `0` when the whole log was written; `1` for a runtime failure — no run under
the directory, a path that cannot be read, a log that is not a Gatling `simulation.log`, an
unsupported version, a log cut short (every record it held is still written, so a script
cannot mistake a partial run for a complete one), a damaged log, or a failed write; `2` for
a usage error — an unsupported tool, a third argument, an unknown flag, or any `-o`. When
the reader closes the pipe the process ends quietly, as any Unix filter does.

**Records.** Every line carries `kind`. Instants are milliseconds since the Unix epoch;
durations are milliseconds. A value the source did not record is omitted, never written as
zero; what the source can never record is listed in the header's `absent`.

| `kind` | Keys |
|---|---|
| `run` | `id`, `name`, `description`, `start`, `tool`, `toolVersion`, `absent`, `warnings`, `assertions` (base64, opaque) |
| `request` | `groups`, `name`, `start`, `duration`, `outcome`, `failure` (`type`, `message`), `scenario`, `responseCode`, `bytesSent`, `bytesReceived` |
| `group` | `groups`, `start`, `duration`, `cumulatedDuration`, `outcome` |
| `user` | `scenario`, `event` (`start`/`end`), `at` |
| `error` | `message`, `at` |
| `assertion` | `payload` (base64, opaque; never produced for a Gatling log) |

```json
{"kind":"run","id":"corpussimulation","name":"io.galaxio.parsec.corpus.CorpusSimulation","start":1788379676999,"tool":"gatling","toolVersion":"3.12.0","absent":["sample.scenario","sample.responseCode","sample.bytesSent","sample.bytesReceived","sample.failureType","sample.userIdentity","timing.connect","timing.dns","timing.tls","requirements","intervalSeries"],"assertions":["…"]}
{"kind":"user","scenario":"Corpus recording","event":"start","at":1788379677532}
{"kind":"request","groups":["outer","inner  with comma"],"name":"GET /slow","start":1788379677646,"duration":1501,"outcome":"success"}
{"kind":"request","groups":["outer","inner  with comma"],"name":"GET /fail","start":1788379679147,"duration":1,"outcome":"failure","failure":{"message":"status.find.is(200), but actually found 500"}}
{"kind":"group","groups":["outer","inner  with comma"],"start":1788379677645,"duration":1515,"cumulatedDuration":1502,"outcome":"failure"}
{"kind":"error","message":"unresolvable url: No attribute named 'undefinedAttribute' is defined ","at":1788379679269}
```

The schema is a published surface: a key is never removed, renamed or re-encoded without a
major version. The full contract, with the presence rule of every key, is
[specs/004-report-records/contracts/records.md](specs/004-report-records/contracts/records.md).

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

## Contributing

Two guards protect release tags. Both call `scripts/check-linkage.sh`, which holds the
issue ↔ PR ↔ milestone rules; the same script runs on every pull request as the `linkage`
CI job.

- **Enable the release-tag hook once per clone.** Without it `.githooks/pre-push` never
  runs. The `linkage` CI job validates a PR before it reaches `main`; this is the local
  fast failure for an exceptional manual tag.

  ```sh
  git config core.hooksPath .githooks
  ```

- **Skipping the guard deliberately.** A release step that must proceed while its
  milestone is still being closed is run as `LINKAGE_OFF=1 <command>`. That is the only
  sanctioned bypass: a command is never rephrased to get past the guard. A block names the
  version and the rule that failed, so fix the milestone rather than the wording.

## License

`galaxio-cli` is licensed under the GNU General Public License, version 2 only
(`GPL-2.0-only`); see [LICENSE](LICENSE).

The public [`parsec`](https://github.com/galax-io/parsec) result-primitives
library is separately licensed under `MIT`. The reviewed MIT-to-GPL relationship
is compatible for this planned dependency, but it is not a whole-module licence
audit. See the [naming and licence decision record](specs/003-define-naming-licence/research.md)
for the compatibility rationale and its scope.
