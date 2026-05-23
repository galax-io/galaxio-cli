# Galaxio Templates

Galaxio registries describe where the CLI can discover template packs.

The CLI now supports discovery, validation, and rendering for local sources and
GitHub-backed template packs.

## Default Registry

The CLI should use this registry by default:

```text
github:galax-io/galaxio-template-registry
```

Users only need `galaxio template configure --registry ...` when they want to
override the default registry with another source.

## Concepts

- **Registry**: an index of template packs available to the CLI.
- **Pack**: a repository or local directory that groups related templates and
  declares the pack version used for GitHub release resolution.
- **Template**: one renderable project skeleton with declared inputs and file
  mappings.

## Registry

A registry points the CLI to template packs:

```yaml
apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: gatling
    source: github:galax-io/templates-gatling
    description: Gatling performance testing templates
```

For GitHub packs, `template init` renders from the release tag that matches the
pack version. A listed pack version `0.3.0` resolves to repository tag `v0.3.0`.
The registry still points at the repository, not at a pinned tag:

```yaml
packs:
  - name: gatling
    source: github:galax-io/templates-gatling
```

`template list` reads the latest registry and pack manifests. `template init`
uses the listed pack version as the immutable GitHub release tag.

## Local Examples

The repository includes a complete local example under
`examples/templates`:

```text
examples/templates/
  registry/galaxio-registry.yaml
  packs/basic/galaxio-pack.yaml
  packs/basic/service/galaxio-template.yaml
  packs/basic/service/files/...
```

Use it as a starting point for your own local repositories:

```sh
galaxio template list --registry local:examples/templates/registry
galaxio template validate local:examples/templates/packs/basic
galaxio template init examples/service --registry local:examples/templates/registry --destination ./tmp/example-service
```

The minimal required manifests are:

Registry:

```yaml
apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: examples
    source: local:examples/templates/packs/basic
```

Pack:

```yaml
apiVersion: galaxio.io/v1
kind: TemplatePack
name: examples
version: 0.1.0
templates:
  - name: service
    version: 0.1.0
    path: service
```

Template:

```yaml
apiVersion: galaxio.io/v1
kind: Template
name: service
engine: go-template
inputs:
  Name:
    type: string
    default: myservice
files:
  - from: files
    to: .
  - from: files/plugins/kafka
    to: plugins
    if: '{{ .KafkaEnabled }}'
```

`files[].if` is optional. When present, the mapping is rendered only if the
expression evaluates to a truthy value such as `true`, `1`, `yes`, or `on`.
The condition is evaluated against the same data used by the rest of the
template, including manifest defaults and `--values` / `--set` overrides.

`local:` sources are resolved from the current working directory.

## CLI Reference

```sh
galaxio template list
galaxio template list --registry local:../galaxio-template-registry
galaxio template init gatling/scala-sbt
galaxio template validate local:../templates-gatling
galaxio template validate github:galax-io/templates-gatling
galaxio doctor
```

Persistent configuration:

```sh
galaxio template configure --show
galaxio template configure --registry github:my-org/my-template-registry
```
