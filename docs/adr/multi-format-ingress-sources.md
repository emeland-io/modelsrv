# ADR: Multi-format Ingress and file-sensor Sources

## Status

Accepted

## Context

Landscape authorship today is YAML-only and local-directory-only (`pkg/filesensor` +
`modelsrv server --data-dir`). Operators also need JSON and CSV imports, and file-like
origins beyond the local disk (HTTP, S3). The team decided (project-discussion#5):

1. **modelsrv** owns format parsers (YAML, JSON, CSV, …).
2. **Sensors** own persistence (how bytes are obtained).
3. File-shaped Sensors must pass **parser configuration** into those parsers (CSV cannot
   carry `kind` / field layout in-band).

Alternatives considered:

- **New NodeType per origin** (s3-sensor, http-sensor) — proliferates binaries for the same
  “list files + parse + apply” loop.
- **Parsers inside each Sensor** — duplicates apply logic and diverges document contracts.
- **Native S3 event watches in v1** — higher ops cost; polling covers the portable case.

## Decision

### Parsers live in `pkg/ingress`

`Document`, YAML/JSON/CSV decode, and `ApplyDocument` / `ApplyAll` live in
[`pkg/ingress`](../../pkg/ingress). Sensors pass `[]byte` + `ParseOptions` and receive the
same documents YAML always produced.

- YAML / JSON are self-describing (`version`, `kind`, `spec`).
- CSV uses a header→field column map. Kind is taken per row from a column mapped to `kind`
  (e.g. `resourcetype`); a config-level `Kind` is only an optional default. A column mapped to
  `id` expands to that kind's primary UUID field (`contextId`, `systemId`, …). Version defaults
  to `emeland.io/v1` when omitted.
- Format selection is explicit `ParseOptions.Format`, else the file extension, else the
  transport MIME hint (`ParseOptions.ContentType`, filled from HTTP `Content-Type`). The hint is
  a fallback for extension-less names, never an override.

### File-like origins are Source backends of the file-sensor

[`pkg/filesensor`](../../pkg/filesensor) introduces a `Source` interface (`List` / `Read`)
with optional `Watcher`. Backends:

| URI | Backend | Change detection |
|-----|---------|------------------|
| `file://…` / `--data-dir` | local directory | `fsnotify` |
| `http(s)://…` | HTTP GET | poll + ETag / Last-Modified |
| `s3://bucket/prefix/` | S3 list + get (AWS SDK, default credential chain) | poll listing |

S3 and HTTP are **not** new NodeTypes. Git and Kubernetes remain separate NodeTypes
(protocol-specific objects/lifecycle) and may still call `pkg/ingress` for file-shaped
payloads.

### Sensor config carries parser options

`--sensor-config` points at YAML listing sources and per-glob `ParseOptions`. `--data-dir`
remains a shorthand for one local Source with extension-based format detect (CSV uses
`DefaultCSVColumns` when no mapping is configured).

Globs come from a YAML mapping, which is unordered, so rules are sorted most-specific-first
(literal names before wildcards) to keep resolution deterministic across restarts. A parser
entry that cannot be built is a startup error, not a silently-empty `ParseOptions`.

Sources are opened once (`OpenSources`) and the resulting handles feed both the initial
`ApplySources` and the background `StartSources`. Poll loops therefore start from the state that
the initial apply already observed, instead of re-emitting a full round of events at startup.

### Out of scope for v1

- Propagating file/object deletes into landscape deletes (matches today’s watcher).
- Native S3 notifications / Git NodeType / K8s ConfigMap file ingest.

## Consequences

- External Sensors (K8s, future Git) import `pkg/ingress`, not `pkg/filesensor`.
- Adding a format is a modelsrv change; adding a file-like origin is a filesensor Source.
- Mixed-kind CSV works without a fixed config `kind`; operators supply a `resourcetype` column
  (or equivalent mapped to `kind`).
- AWS SDK becomes a dependency of modelsrv for the S3 Source.
