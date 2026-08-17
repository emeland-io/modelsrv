# Model Server

The EmELand model server provides the example mapping of the Emerging Enterprise Landscape as an OpenAPI service. It gives access to the objects of the model. It is implemented as a golang library to allow users to quickly build adapters to their existing IT management solutions.

**Warning: This project is in early alpha and all of the APIs will change!**

## Usage

Endpoint annotations (`emeland.io/endpoint.*`) declare where an ApiInstance is reachable.
**Writing OTel collector config** and **probing those endpoints** are separate concerns:

| Mechanism | Role |
|-----------|------|
| `modelsrv server --otel-config-out` | **Writes** collector YAML on ApiInstance events (does not probe) |
| `modelsrv certprobe` | **Writes** the same collector YAML on demand (does not probe) |
| `modelsrv-otel-exporter --config …` | **Probes** via the OTel `http_check` receiver using that YAML |

Despite the name, `modelsrv certprobe` only generates config — it never contacts the annotated hosts. modelsrv itself does not probe endpoints.

### Server startup subscribers

Pre-register downstream model servers (replicas, sensors, consumers) so they receive landscape events without a manual `POST /api/events/register` after every restart:

```bash
modelsrv server \
  --subscribers http://replica:8080/api,http://sensor:8081/api
# or: SUBSCRIBERS=http://replica:8080/api,http://sensor:8081/api
```

Initial YAML/JSON in `--data-dir` is applied synchronously first; each URL is then registered via the same in-process path as the register API (including synchronous replay of live-state Creates). Later changes are pushed asynchronously to `POST …/events/push`. Invalid URLs are logged and skipped; duplicate URLs are idempotent.

### File sensor Sources and formats

The reference `modelsrv server` acts as a **file-sensor**: it obtains landscape documents from a
**Source** and applies them via [`pkg/ingress`](pkg/ingress) (YAML, JSON, CSV). Format parsing lives
in the modelsrv library; Sensors own persistence. See
[docs/adr/multi-format-ingress-sources.md](docs/adr/multi-format-ingress-sources.md).

**`--data-dir` (default)** — one local directory. Watches `.yaml` / `.yml` / `.json` / `.jsonl` /
`.csv` with `fsnotify`. CSV without `--sensor-config` uses the default column layout
(`resourcetype`, `uuid`, `displayname`, `description`, `annotations`).

**`--sensor-config` / `SENSOR_CONFIG`** — YAML listing one or more Sources. When set, it overrides
`--data-dir`:

```yaml
sources:
  - uri: file:///var/lib/emeland/data
    watch: true
  - uri: https://example.com/landscape.json
    poll: 1m
  - uri: s3://landscapes/org/data/
    poll: 1m
    files:
      "*.csv":
        format: csv
        delimiter: ","
        columns:
          resourcetype: kind
          uuid: id
          displayname: displayName
          description: description
          annotations: annotations
      "*.yaml":
        format: yaml
```

CSV rows carry `resourcetype` (e.g. `Context`, `System`); `uuid` is written to that kind's primary
ID field (`contextId`, `systemId`, …). `annotations` is a JSON object cell (e.g. `{"owner":"ops"}`).
Config does not need a fixed `kind` / `version` (version defaults to `emeland.io/v1`). See
[`config/sensor.yaml`](config/sensor.yaml).

Under `files`, the most specific glob wins: a literal name (`landscape.csv`) is matched before a
wildcard (`*.csv`), so overlapping patterns resolve the same way on every start. A `files` entry
that cannot be understood (unknown `format` or `kind`) fails startup rather than being ignored.

The format of a document is taken from the explicit `format` if given, otherwise from the file
extension, otherwise from the HTTP `Content-Type` — so an extension-less endpoint such as
`https://example.com/api/landscape` serving `application/json` needs no configuration.

Supported URI schemes: `file://`, `http://` / `https://`, `s3://`. Local Sources use `watch: true`
(fsnotify). HTTP and S3 Sources poll (`poll`) and re-apply when ETag / Last-Modified changes.
S3 auth uses the default AWS credential chain (env, shared config, IAM role). File/object deletes
do **not** remove landscape resources in v1.

### OpenTelemetry Collector integration

Keep an OpenTelemetry Collector
[`http_check`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/httpcheckreceiver)
config in sync with ApiInstance endpoint annotations. modelsrv only maintains the YAML;
the collector (or `modelsrv-otel-exporter`) performs the actual HTTP/TLS checks.
The receiver component type is `http_check` (snake_case); the older `httpcheck` type is a
deprecated alias per the [httpcheckreceiver README](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/receiver/httpcheckreceiver/README.md#configuration).

#### Event-driven (normal path)

Start modelsrv with `--otel-config-out` so ApiInstance create/update/delete events rewrite the
collector config automatically:

```bash
go run ./cmd/modelsrv server \
  --data-dir ./data \
  --otel-config-out ./collector.yaml \
  --otel-subscriber http://modelsrv:8080
```

Relevant flags (also available as env vars where noted):

| Flag | Env | Default | Purpose |
|------|-----|---------|---------|
| `--otel-config-out` | `OTEL_CONFIG_OUT` | *(empty / off)* | Path to keep in sync |
| `--otel-config-debounce` | `OTEL_CONFIG_DEBOUNCE` | `2s` | Debounce before rewrite |
| `--otel-collection-interval` | — | `5m` | `collection_interval` for http_check |
| `--otel-listen-addr` | — | `0.0.0.0:24200` | emeland exporter listen address |
| `--otel-expiry-threshold` | — | `720h` | Certificate expiry threshold |
| `--otel-subscriber` | — | *(none)* | Downstream modelsrv URL (repeatable) |

The written file is a complete collector config:

```yaml
receivers:
  http_check:
    collection_interval: 5m
    metrics:
      httpcheck.tls.cert_remaining:
        enabled: true
    targets:
      - method: GET
        endpoint: https://payments.prod.eu.example.com:443/api/v1/health
exporters:
  emeland:
    listen_addr: 0.0.0.0:24200
    expiry_threshold: 720h
    subscribers:
      - http://modelsrv:8080
    endpoint_mapping:
      https://payments.prod.eu.example.com:443/api/v1/health: 88888888-0000-4000-8000-000000000001
service:
  pipelines:
    metrics:
      receivers:
        - http_check
      exporters:
        - emeland
```

Point `modelsrv-otel-exporter --config <path>` at the same file. When ApiInstances change,
modelsrv rewrites the file within the debounce window; unchanged content is not rewritten.

#### On-demand CLI (`modelsrv certprobe`)

`modelsrv certprobe` is a **config generator**, not a probe runner. It queries ApiInstances from a
running modelsrv and writes collector YAML (http_check receiver + emeland exporter + pipeline).
It does **not** dial annotated hosts; use `modelsrv-otel-exporter` for that.

```bash
go run ./cmd/modelsrv certprobe \
  --server http://localhost:8080/api/ \
  --otel-config-out ./collector.yaml \
  --subscriber http://modelsrv:8080 \
  --collection-interval 5m
```

The command queries the landscape over HTTP (`--server`, default `http://localhost:8080/api/` or
`MODELSRV_URL`), so resources replicated from upstream subscribers are included — not only local
YAML. It uses the same annotation rules and host:port dedupe as the background writer.

#### Reload strategy

modelsrv owns rewriting the config file. Reloading the collector remains operator-owned:

1. **File write + SIGHUP** — after modelsrv (or `certprobe`) updates the file, send `SIGHUP` to
   the collector so it reloads from disk.
2. **OpAMP supervisor** — when an [OpAMP](https://opentelemetry.io/docs/collector/management/)
   supervisor manages the collector, push the updated config through OpAMP instead of signaling
   the process directly.

Collector-side `httpcheck.tls.cert_remaining` is the source of truth for certificate lifetime. EmELand certificate Findings are produced separately from those metrics (not by an in-process prober in modelsrv).

## Contributing

Merge requests to expand and improve the service are greatly appreciated.

Please make sure that all tests are running before you create a merge request.

## License

Copyright 2025 Lutz Behnke <lutz.behnke@gmx.de>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Project status

This project is under active development.
