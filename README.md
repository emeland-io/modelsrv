# Model Server

The EmELand model server provides the example mapping of the Emerging Enterprise Landscape as an OpenAPI service. It gives access to the objects of the model. It is implemented as a golang library to allow users to quickly build adapters to their existing IT management solutions.

**Warning: This project is in early alpha and all of the APIs will change!**

## Usage

### In-process certificate probing

Run the model server with certprobe enabled (default) to probe ApiInstance endpoints declared via [`emeland.io/endpoint.*` annotations](docs/endpoint-annotations.md):

```bash
go run ./cmd/modelsrv server --data-dir ./data --metrics-addr :9090
```

### OpenTelemetry Collector integration

As an alternative (or complement) to in-process certprobe, export an OpenTelemetry Collector [`http_check`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/httpcheckreceiver) receivers fragment from ApiInstances on a **running** modelsrv:

```bash
go run ./cmd/modelsrv certprobe \
  --server http://localhost:8080/api/ \
  --otel-config-out ./http_check.yaml \
  --collection-interval 5m
```

The command queries the landscape over HTTP (`--server`, default `http://localhost:8080/api/` or `MODELSRV_URL`), so resources replicated from upstream subscribers are included — not only local YAML. It discovers probe URLs with the same annotation rules and host:port dedupe as the background daemon, and writes a fragment like:

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
```

Merge `receivers.http_check` into your full collector config (exporters and pipelines remain operator-owned). Enable TLS certificate metrics as shown; see the httpcheckreceiver docs for optional timing and validation metrics.

#### Reload strategy

When ApiInstances change, regenerate the fragment and reload the collector:

1. **File write + SIGHUP** — rewrite `--otel-config-out` (cron or on model change), then send `SIGHUP` to the collector process so it reloads configuration from disk.
2. **OpAMP supervisor** — when an [OpAMP](https://opentelemetry.io/docs/collector/management/) supervisor manages the collector, push the updated config through OpAMP instead of signaling the process directly.

In-process certprobe (`modelsrv server --enable-certprobe`) and collector-side `httpcheck.tls.cert_remaining` are complementary: the former also writes EmELand Findings; the latter fits existing OTel pipelines.

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
