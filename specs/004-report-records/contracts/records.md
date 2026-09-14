# Contract: the record stream

**Feature**: [spec.md](../spec.md) | **Model**: [data-model.md](../data-model.md)

This schema is a published surface from the release that ships it (constitution
Principle V). Removing or renaming a key, or changing a key's encoding, is a breaking
change. Adding a key is not. Consumers must ignore keys they do not know.

## JSON Lines (the command's output; no `-o` needed)

One JSON object per line, UTF-8, `\n` terminated, no array, no preamble, no trailer. Every
object has `kind`. Keys appear in the order listed. Instants are integers in milliseconds
since the Unix epoch (UTC). Durations are integers in milliseconds. A key whose value the
source did not record is absent from the object, never `null` or `0`.

### `run` — first line, exactly once

```json
{"kind":"run","id":"io.galaxio.parsec.corpus.CorpusSimulation","name":"io.galaxio.parsec.corpus.CorpusSimulation","start":1788670094356,"tool":"gatling","toolVersion":"3.15.1","absent":["sample.scenario","sample.responseCode","sample.bytesSent","sample.bytesReceived","sample.failureType","sample.userIdentity","timing.connect","timing.dns","timing.tls","requirements","intervalSeries"],"assertions":["AAEBAAEFAAAAAAAAgFlA","…"]}
```

| Key | Type | Present |
|---|---|---|
| `id` | string | always |
| `name` | string | always |
| `description` | string | when the run carried one |
| `start` | int (epoch ms) | when resolved |
| `tool` | string | always (`"gatling"`) |
| `toolVersion` | string | always |
| `absent` | array of string | always; identifiers of what this source can never record |
| `warnings` | array of `{"version": string, "reason": string}` | when the version gate raised one |
| `assertions` | array of string (base64) | when the run declared any; opaque, uninterpreted |

### `request`

```json
{"kind":"request","groups":["outer","inner, with comma"],"name":"GET /fail","start":1788670095021,"duration":3,"outcome":"failure","failure":{"message":"status.find.is(200), found 500"}}
```

| Key | Type | Present |
|---|---|---|
| `groups` | array of string | always; `[]` outside any group; outermost first |
| `name` | string | always |
| `start` | int (epoch ms) | when resolved |
| `duration` | int (ms) | when the source recorded an end; `0` is a real zero |
| `outcome` | `"success"` \| `"failure"` (\| `"unknown"`) | always, as recorded |
| `failure` | `{"type"?: string, "message": string}` | iff `outcome` is `"failure"` |
| `scenario` | string | when recorded (never for Gatling) |
| `responseCode` | string | when recorded (never for Gatling) |
| `bytesSent`, `bytesReceived` | int | when recorded (never for Gatling) |

### `group`

```json
{"kind":"group","groups":["outer","inner, with comma"],"start":1788670095016,"duration":1504,"cumulatedDuration":1503,"outcome":"failure"}
```

| Key | Type | Present |
|---|---|---|
| `groups` | array of string | always; the group's own path, own name last |
| `start` | int (epoch ms) | when resolved |
| `duration` | int (ms) | when set; wall clock, pauses included |
| `cumulatedDuration` | int (ms) | when set; sum of enclosed request durations |
| `outcome` | string | always; the group's own |

### `user`

```json
{"kind":"user","scenario":"Corpus recording","event":"start","at":1788670094885}
```

| Key | Type | Present |
|---|---|---|
| `scenario` | string | always |
| `event` | `"start"` \| `"end"` | always |
| `at` | int (epoch ms) | when resolved |

### `error`

```json
{"kind":"error","message":"unresolvable url: No attribute named 'undefinedAttribute' is defined ","at":1788670096632}
```

| Key | Type | Present |
|---|---|---|
| `message` | string | always, verbatim |
| `at` | int (epoch ms) | when resolved |

### `assertion` — only for a source that interleaves payloads with events

```json
{"kind":"assertion","payload":"AAEBAAEFAAAAAAAAgFlA"}
```

Never produced for a Gatling log: both formats put every payload on the `run` header.

## Report formats (`-o`)

`-o` does not select an encoding of this stream; it requests a report format instead, and
every known format is reserved for a later milestone: `stats` and `global_stats` (Gatling's
legacy files, v0.14.0/v0.15.0) and `yml` (OpenNFR-style YAML, postponed). Until then any
`-o` is a usage error. There is no `-o json` and no `-o text`.

## Compatibility with the live sidecar

The sidecar (`comet`) will emit the JSON Lines form of this schema. The keys, kinds and
encodings above are the shared contract; `absent` and `warnings` are what let a consumer
tell a live source from an archived one without a second schema.
