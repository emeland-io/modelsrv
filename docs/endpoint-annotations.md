# ApiInstance endpoint annotation registry

Well-known `emeland.io/endpoint.*` annotation keys for **ApiInstance** resources.
These declare where an API instance is reachable on the network so external tooling
(for example an OpenTelemetry Collector [`httpcheck`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/httpcheckreceiver) receiver)
can perform synthetic HTTP/TLS checks.

modelsrv stores annotations as `map[string]string` in the model and `{ key, value }`
objects on the query API. Values MUST be flat UTF-8 strings — nested YAML maps under
`spec.annotations` are stringified by the file sensor and MUST NOT be used.

**Related docs**

- [Findings](findings.md) — certificate findings (future milestone)
- [README: OpenTelemetry Collector integration](../README.md#opentelemetry-collector-integration) — export `http_check` config via `modelsrv certprobe`

## Scope

| Resource | Endpoint annotations |
|----------|---------------------|
| **ApiInstance** | Yes — probe URL declared here |
| SystemInstance | No — deployment metadata only |
| API | No — type/spec definition, not a deployed endpoint |

## Value format

Declarative YAML uses a flat map under `spec.annotations`:

```yaml
spec:
  annotations:
    emeland.io/endpoint.protocol: https
    emeland.io/endpoint.host: payments.prod.eu.example.com
    emeland.io/endpoint.port: "443"
    emeland.io/endpoint.path: /api/v1/health
```

Port values MUST be quoted in YAML when numeric (`"443"`) so they remain strings.

## Keys on ApiInstance

| Key | Required | Purpose | Example |
|-----|----------|---------|---------|
| `emeland.io/endpoint.protocol` | yes | URL scheme | `https` |
| `emeland.io/endpoint.host` | yes | Hostname or IP | `payments.prod.eu.example.com` |
| `emeland.io/endpoint.port` | no | TCP port (string) | `443` |
| `emeland.io/endpoint.path` | no | HTTP path | `/api/v1/health` |

### Defaults

When optional keys are omitted:

| Key | Default |
|-----|---------|
| `emeland.io/endpoint.port` | `443` if protocol is `https`; `80` if `http` |
| `emeland.io/endpoint.path` | `/` |

### URL construction

Probe URL is built as:

```
{protocol}://{host}:{port}{path}
```

- `protocol` MUST be `http` or `https`.
- `path` MUST start with `/` (a leading slash is added if missing).
- An ApiInstance without `emeland.io/endpoint.host` is not a probe target.

## Certificate metadata

**Do not** declare certificate expiry or issuer in ApiInstance annotations for v1.
Certificate state is discovered live by the OpenTelemetry Collector httpcheck receiver
and exposed as `httpcheck.tls.cert_remaining`. EmELand Findings derived from those
metrics are handled outside the in-process modelsrv probing path.

## `reference` vs probe URL

List and detail API responses include a `reference` URI pointing at the resource in
the EmELand catalog (for example `https://emeland.local/v1/landscape/api-instances/{uuid}`).
That URI is **not** the live service endpoint and MUST NOT be used as a probe target.

## Worked example

```yaml
version: emeland.io/v1
kind: ApiInstance
spec:
  apiInstanceId: "88888888-0000-4000-8000-000000000001"
  displayName: "Payments API (prod EU)"
  api: "aaaaaaaa-0000-4000-8000-000000000001"
  systemInstance: "77777777-0000-4000-8000-000000000102"
  annotations:
    env: prod
    emeland.io/endpoint.protocol: https
    emeland.io/endpoint.host: payments.prod.eu.example.com
    emeland.io/endpoint.port: "443"
    emeland.io/endpoint.path: /api/v1/health
```

## Out of scope (explicit exclusions)

The following MUST NOT be used as endpoint annotation keys in v1:

- `cert.notAfter`, `cert.expires`, `cert.issuer`, or similar cert inventory keys
- Nested YAML structures under `annotations` (use flat keys or a single JSON string value)
- Probe URLs on resource types other than **ApiInstance**
