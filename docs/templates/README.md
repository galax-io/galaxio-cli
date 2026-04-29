# Galaxio Templates

Galaxio registries describe where the CLI can discover template packs.

This first step only defines the default registry contract. Template pack and
template rendering formats will be added separately. Commands currently operate
at discovery and validation level, not rendering level.

## Default Registry

The CLI should use this registry by default:

```text
github:galax-io/galaxio-template-registry
```

Users only need `galaxio template configure --registry ...` when they want to
override the default registry with another source.

## Concepts

- **Registry**: an index of template packs available to the CLI.
- **Pack**: a repository that groups related templates. Pack structure is not
  defined in this step.

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

## Planned CLI Flow

```sh
galaxio template list
galaxio template list --registry local:../galaxio-template-registry
galaxio template init gatling/scala-sbt
galaxio template validate local:../templates-gatling
galaxio template validate github:galax-io/templates-gatling
galaxio doctor
```

Later steps will add persistent configuration and rendering:

```sh
galaxio template configure --show
galaxio template configure --registry github:my-org/my-template-registry
```
