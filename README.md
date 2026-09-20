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
galaxio report gatling  # read the latest Gatling run, describe it and summarise it
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

### Read a finished run

`galaxio report <tool> [PATH]` reads a finished run once, describes it, and summarises it as
a whole. Without `-o` it writes no file and prints no row for a single request or group.

```sh
galaxio report gatling                               # the latest run under target/gatling
galaxio report gatling target/gatling/mysim-20260906044741110
galaxio report gatling path/to/simulation.log
galaxio report gatling build/reports/gatling         # a results root: lastRun.txt, else the newest run
galaxio report gatling --percentiles 90,99.9         # other percentile ranks
galaxio report gatling --bounds 5,1000               # other response-time bands
galaxio report                                       # help
```

```text
run         io.galaxio.parsec.corpus.CorpusSimulation
id          io.galaxio.parsec.corpus.CorpusSimulation
started     2026-09-06T04:47:41.110Z
tool        gatling 3.13.1
log         binary, internal/report/testdata/corpus/gatling/3.13.1/simulation.log
found by    path
span        2026-09-06T04:47:41.603Z .. 2026-09-06T04:47:44.829Z (3.226s)
requests    102 (84 ok, 18 failed)
groups      12 traversals
users       12 events
errors      6
assertions  10 payloads the run declared

102 requests · 25.5 req/s      ✓ 84 ok · 82.35 % · 21/s      ✗ 18 failed · 17.65 % · 4.5/s

response time, ms       min   mean    std    p50    p75    p95    p99    max
all                       0     89    353      1      1   1427   1502   1503
✓ ok                      0    108    387      1      1   1502   1502   1503
✗ failed                  0      1      1      1      2      4      4      4

███████████████░░░░░  76.47 %    78  ok under 800 ms
░░░░░░░░░░░░░░░░░░░░      0 %     0  ok 800 to 1200 ms
█░░░░░░░░░░░░░░░░░░░   5.88 %     6  ok 1200 ms and over
████░░░░░░░░░░░░░░░░  17.65 %    18  failed

times in ms · percentiles are galaxio's t-digest estimates, interpolated
```

**Tool.** The first argument names the tool that produced the run. `gatling` is the only
tool this release reads: text logs from Gatling 3.11.5 through 3.12.0 and binary logs from
3.13.1 through 3.15.1, identified from the log's content, never its name. A version below
that range, and 3.13.0, are refused with a message naming the version and the range; a newer
version is read and a warning goes to standard error.

**PATH.** A run directory, its `simulation.log`, or a results root holding run directories.
With no path the Maven and sbt results root `target/gatling` is searched (Gradle writes to
`build/reports/gatling`; pass it). In a results root the run named by `lastRun.txt` wins if
it still exists, otherwise the most recently modified run. The report says which run was
read and by which rule.

**The description.** `started` is the run's own recorded start and `span` is the interval
the run actually covers, which is a different quantity; the span line is left out when the
log holds an item the source could not place in time. `requests` splits by the outcome the
source recorded, never inferred, and a further `unknown` line appears only when the source
lost a sample's outcome, counting those requests apart rather than guessing either way.
`groups` counts traversals and `users` counts events, each saying so, because a run enters a
group once per iteration and emits a start and an end per virtual user. `assertions` counts
the opaque payloads the run declared, wherever the log put them, and is left out when it
declared none. Anything the log recorded is quoted if it is not printable, so a run from
somebody else's CI cannot colour your terminal or erase the line it is on. `--quiet`
suppresses the description, the summary and the progress block, and leaves errors and the
warnings, which qualify every number under them; `--verbose` adds what this source can never
record, such as a response code, which Gatling does not put in its log.

**The summary.** A headline with the number of requests and their rate, then ok and failed
with their share of all requests and their rate; a table of response times with a row for
all, ok and failed requests; the response-time bands as bars, each with its share of all
requests and its count; and a closing line with the unit of every time and what the
percentiles are. The layout is this tool's own, the same for every tool the command reads.
Every figure but the percentiles is exact:

- **count** — the requests of that outcome; **share** — the count over all requests, at most
  two decimals.
- **rate** — the count over the run's span in whole seconds, rounded up, which is the divisor
  Gatling uses; `-` when the run cannot be placed in time.
- **min** and **max** — over the requests with a recorded end.
- **mean** — rounded half up to a whole millisecond; **std** — the population standard
  deviation, rounded the same way.
- **bands** — successful responses under the lower boundary, from it to under the upper one,
  and at the upper one or more (`t < LOW`, `LOW <= t < HIGH`, `t >= HIGH`); the `failed` band
  holds every failed request whatever its time. A bar fills its share rounded to a
  twentieth, at least one cell for a band that holds a request and never all twenty for one
  that does not hold them all.
- `-` marks a figure that does not exist — the times of an outcome no request reached, or
  every share of a run with no request — never `0`.

**Percentiles.** Estimates from a t-digest, `github.com/caio/go-tdigest` at compression 100,
read as its default quantile reads it: the numbers Gatling 3.11 and 3.12 print for the same
log, which compute them as `Math.round(quantile(rank / 100))` over `AVLTreeDigest(100)` of
t-digest 3.1. The read is this command's own, written in t-digest 3.1's order of operations
and rounded as `Math.round` rounds, so that a percentile that lies at a half rounds the same
way on every platform. The tests assert the equality on every recorded run against t-digest
3.1 itself. Gatling 3.13.0 and later print other numbers because t-digest 3.3's `AVLTreeDigest`
miscounts ([tdunning/t-digest#230](https://github.com/tdunning/t-digest/issues/230)): for the
recorded 3.13.1 run Gatling printed 1072 ms as the 95th percentile of all requests, where
Gatling 3.11's digest and this command give 1427. One difference from Gatling 3.11 is known
and rare: when a percentile falls on the edge of a run of equal response times, the two
digests can round it to the two sides — on one live run, 5 ms where Gatling 3.11 printed 6,
with no request recorded between 5 and 6 — because they order centroids of an equal mean
differently. For the same reason of last bits, an arm64 and an amd64 build of this command
can differ from each other: inside the digest library arm64 computes a centroid's mean in
one fused operation and amd64 in two, so on rare runs the two hold slightly different
centroids. At the default ranks 2 percentiles in 7 500 differed, each by 1 ms; one build's
output is always the same for the same log.

**Known limitation.** The default quantile interpolates between neighbouring response
times, as Gatling 3.11's does, so where a run's response times have a gap a percentile can
be a value no request had. For the recorded 3.13.1 run the 95th percentile of all requests
is printed as 1427 ms, while the request at that rank took 1502 ms and no request took
between 8 and 1501 ms. [caio/go-tdigest#42](https://github.com/caio/go-tdigest/pull/42)
proposes a read by rank that would print 1502, and so part from Gatling 3.11's numbers.
How far a percentile may differ from the response times a run recorded is a rule the tests
hold every percentile to: while an outcome holds at most 200 requests, a percentile lies
between the two recorded response times around its position, and at any size it misplaces
its rank by at most 4·q·(1−q)/100 of the requests plus one, q being the rank over 100.

**`--percentiles` and `--bounds`.** `--percentiles 90,99.9` reports those ranks instead of
the default `50,75,95,99`, each once and in increasing order; a rank is a number above 0
and at most 100. `--bounds 5,1000` sets the two boundaries of the bands instead of the
default `800,1200`, in whole, non-negative milliseconds with the second greater than the
first. A value that is neither exits 2 quoting it, while the flag is parsed and so before
any help.

**Colour.** On a terminal that understands escape sequences, `ok` is green, `failed` is red
when anything failed, and the headings, the empty part of a bar and the closing line are
faint. Written to a pipe, a file or a device that is no terminal, and with `--no-color` or
`NO_COLOR`, the same text carries no escape sequence. Whether the stream is a terminal is
asked of the device itself; on Linux and macOS `TERM` must also name a terminal that is not
`dumb`, and on Windows, which sets no `TERM`, the console must have virtual-terminal
processing on, which this command reads and never changes.

**Progress.** A long read shows a block of six lines on standard error: how much of the log
has been read, the time left, and the count, share and times so far of all, ok and failed
requests. It appears 500 ms into the read, is redrawn in place a few times a second, and is
erased before anything else is written. It is shown only when standard error is a terminal
by the same test as colour, and never with `--quiet`; standard output and the exit code are
the same with and without it. No terminal mode is changed, so a terminal narrower than 80
columns wraps the block and may keep remains of it.

**`-o stats,global_stats`.** Writes `js/stats.json` and `js/global_stats.json` under the
selected run. Either product can also be selected alone. Successful export is silent.
Export requires four distinct percentile ranks after sorting and
deduplication; their values are the deterministic t-digest estimates described above.

```sh
galaxio report gatling target/gatling/mysim-20260906044741110 -o global_stats
galaxio report gatling target/gatling/mysim-20260906044741110 -o stats
galaxio report gatling target/gatling/mysim-20260906044741110 -o stats,global_stats
```

`yml` remains reserved. Unknown or unavailable products and empty list entries
exit 2 before reading a run. There is no `-o json` or `-o text`.

**Exit codes.** `0` when the run was read to the end and summarised; `1` for a runtime
failure — no run under the directory, a path that cannot be read, a log that is not a
Gatling `simulation.log`, an unsupported version, a log cut short (the description and the
summary of what it did hold are still printed, so a script cannot mistake a partial run for a
complete one), a damaged log (nothing is printed, because the records before the damage are
not a result), a run that holds requests and spans no time (the summary is printed with every
rate `-`, because no rate can be computed; a run that holds no request at all reports
`0 requests` and exits 0), or a run holding requests whose outcome the source lost (the
summary is printed, and its outcomes do not add up to its requests); `2` for a usage error —
an unsupported tool, an empty or third argument, an unknown flag, a bad `--percentiles` or
`--bounds`, or an invalid `-o` product list, each rejected while the flag is parsed and so before any help. An
interrupt cancels the read and exits 1 naming the log it stopped in. The two exits for a run
that spans no time and for lost outcomes were `0` before this release; no Gatling log is
known to produce either. Requests with no recorded end are counted, take part in no timing
figure, and are named in a warning on standard error.

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
