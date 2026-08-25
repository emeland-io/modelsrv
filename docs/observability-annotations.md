# Observability annotation registry

Well-known `emeland.io/*` annotation keys for **Metric**, **Threshold**, **MetricValue**, and for
base resources that reference Thresholds or MetricValues. modelsrv stores these as ordinary
annotations (`map[string]string` in the model; `{ key, value }` objects on the query API). No
runtime validation is applied — recommendation levels guide integrators and downstream tooling.

**Related docs**:

- [ADR: Phase 6 observability resources](adr/observability-resources.md) — lean core schema
- [ADR: Annotation-based ownership visibility](adr/ownership-visibility.md) — owner key semantics

## Value format

All annotation values are **plain UTF-8 strings**. UUIDs MUST use standard UUID string form
(e.g. `550e8400-e29b-41d4-a716-446655440000`). modelsrv does not parse structured data from
annotation values for registry keys.

Declarative YAML uses a map under `spec.annotations`:

```yaml
spec:
  annotations:
    emeland.io/unit: ms
```

Comma- or space-separated UUID lists (on base resources) follow the same parsing convention as
`emeland.io/owner-identities` in `pkg/authz`.

## Recommendation levels

| Level | Meaning |
|-------|---------|
| **recommended** | Interoperability expectation; omit only when the metadata does not apply |
| **optional** | Useful hint or override; safe to omit |

## Keys on Metric

| Key | Recommendation | Purpose | Example |
|-----|----------------|---------|---------|
| `emeland.io/unit` | optional | Measurement unit | `ms`, `requests/s` |
| `emeland.io/dimension` | optional | UI / grouping family | `latency`, `availability` |
| `emeland.io/metric.expression` | optional | Composition formula for a compound metric | `a + b` |
| `emeland.io/metric.language` | optional | Language of the composition formula | `promql`, `cel`, `text` |

Owner keys (`emeland.io/owner-identities`, `emeland.io/owner-groups`) may appear when read
visibility should be restricted, but **Metric** is public vocabulary by default — see
[ownership-visibility.md](adr/ownership-visibility.md).

## Keys on Threshold

| Key | Recommendation | Purpose | Example |
|-----|----------------|---------|---------|
| `emeland.io/threshold.expression` | recommended | Condition of arbitrary complexity (unvalidated) | `histogram_quantile(0.99, ...) > 0.5` |
| `emeland.io/threshold.language` | recommended | Expression language | `promql`, `cel`, `sql`, `text` |

Owner keys are recommended when `--trust-auth-headers` is enabled (Threshold is owner-restricted).

### Suggested `emeland.io/threshold.language` values

| Value | Typical use |
|-------|-------------|
| `promql` | Prometheus query language |
| `cel` | Common Expression Language |
| `sql` | SQL-like predicate |
| `text` | Free-form human-readable condition |

## Keys on MetricValue

No Phase 6-specific keys beyond ownership. The current reading is the first-class `value` field.
Owner keys are recommended when `--trust-auth-headers` is enabled.

## Keys on base resources

These keys may appear on any landscape resource that should link to Thresholds or MetricValues
(e.g. System, SystemInstance, ApiInstance, Component, Capacity).

| Key | Recommendation | Purpose | Example |
|-----|----------------|---------|---------|
| `emeland.io/thresholds` | optional | Comma/space-separated Threshold UUIDs | `550e8400-...,7c9e6679-...` |
| `emeland.io/metric-values` | optional | Comma/space-separated MetricValue UUIDs | `a1b2c3d4-...` |

Linkage is metadata only. modelsrv does not validate that listed UUIDs exist. Deleting a Threshold
or MetricValue does not clear these annotations. Reverse lookup requires scanning annotated
resources.

## Worked examples

### Metric with unit and compound expression

```yaml
version: emeland.io/v1
kind: Metric
spec:
  metricId: 11111111-1111-1111-1111-111111111111
  displayName: p99 API latency
  description: End-to-end latency for the payments API
  annotations:
    emeland.io/unit: ms
    emeland.io/dimension: latency
    emeland.io/metric.expression: histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))
    emeland.io/metric.language: promql
```

### Threshold with condition annotations

```yaml
version: emeland.io/v1
kind: Threshold
spec:
  thresholdId: 22222222-2222-2222-2222-222222222222
  displayName: Latency SLO breach
  description: p99 latency must stay under 500ms
  metricRef:
    metricId: 11111111-1111-1111-1111-111111111111
  annotations:
    emeland.io/threshold.expression: histogram_quantile(0.99, ...) > 0.5
    emeland.io/threshold.language: promql
    emeland.io/owner-identities: sre-oncall
```

### MetricValue (current reading)

```yaml
version: emeland.io/v1
kind: MetricValue
spec:
  metricValueId: 33333333-3333-3333-3333-333333333333
  displayName: Current p99 latency
  metricRef:
    metricId: 11111111-1111-1111-1111-111111111111
  value: "412"
  annotations:
    emeland.io/owner-identities: sre-oncall
```

### Base resource linking to Threshold and MetricValue

```yaml
version: emeland.io/v1
kind: SystemInstance
spec:
  systemInstanceId: 550e8400-e29b-41d4-a716-446655440000
  displayName: Payments API (prod EU)
  annotations:
    emeland.io/thresholds: 22222222-2222-2222-2222-222222222222
    emeland.io/metric-values: 33333333-3333-3333-3333-333333333333
```

## Out of scope (explicit exclusions)

The following MUST NOT be used as observability annotation keys in Phase 6:

- Snapshot or observation timestamps (`emeland.io/observed-at`, `emeland.io/valid-from`, …)
- Time-series or history markers
- Any key whose purpose is storing temporal snapshots (snapshotting is handled outside this model)

Phase 6 stores **current-state** MetricValues only.

Metadata covered by this registry MUST NOT be promoted to first-class schema fields on Metric,
Threshold, or MetricValue (see [observability ADR](adr/observability-resources.md)).
