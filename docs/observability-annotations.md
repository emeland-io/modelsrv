# Observability annotation registry

Well-known `emeland.io/*` annotation keys for **Metric**, **MetricInstance**, **Threshold**, and
**MetricValue**. modelsrv stores these as ordinary annotations (`map[string]string` in the model;
`{ key, value }` objects on the query API). No runtime validation is applied — recommendation levels
guide integrators and downstream tooling.

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

## Recommendation levels

| Level | Meaning |
|-------|---------|
| **recommended** | Interoperability expectation; omit only when the metadata does not apply |
| **optional** | Useful hint or override; safe to omit |

## Keys on Metric

A Metric is the abstract measurement (an organizational grouping). It carries **no query** — the
concrete query lives on the MetricInstance.

| Key | Recommendation | Purpose | Example |
|-----|----------------|---------|---------|
| `emeland.io/unit` | optional | Measurement unit | `ms`, `requests/s` |
| `emeland.io/dimension` | optional | UI / grouping family | `latency`, `availability` |

Owner keys (`emeland.io/owner-identities`, `emeland.io/owner-groups`) may appear when read
visibility should be restricted, but **Metric** is public vocabulary by default — see
[ownership-visibility.md](adr/ownership-visibility.md).

## Keys on MetricInstance

A MetricInstance is the concrete measured thing. It carries the query that produces its value; the
query is instance-specific (its labels/aggregation differ per subject).

| Key | Recommendation | Purpose | Example |
|-----|----------------|---------|---------|
| `emeland.io/metric.expression` | recommended | The query that produces the instance's value | `sum(up{cluster="prod"})` |
| `emeland.io/metric.language` | recommended | Language of the expression | `promql`, `cel`, `text` |
| `emeland.io/unit` | optional | Measurement unit (may restate the Metric's) | `ms` |

The prometheus sensor evaluates MetricInstances carrying `emeland.io/metric.expression` with
language `promql` (or unset) and emits a MetricValue. Owner keys are recommended when
`--trust-auth-headers` is enabled (MetricInstance is owner-restricted).

## Keys on Threshold

A Threshold references a MetricInstance and applies a condition to its value.

| Key | Recommendation | Purpose | Example |
|-----|----------------|---------|---------|
| `emeland.io/threshold.expression` | optional | Condition of arbitrary complexity (unvalidated, free-form) | `histogram_quantile(0.99, ...) > 0.5` |
| `emeland.io/threshold.language` | optional | Expression language for `threshold.expression` | `promql`, `cel`, `sql`, `text` |
| `emeland.io/threshold.operator` | optional | Structured comparison operator applied to the MetricInstance's value | `gt`, `ge`, `lt`, `le`, `eq`, `ne` |
| `emeland.io/threshold.limit` | optional | Numeric bound the operator compares against | `500`, `0.99` |

The prometheus sensor uses the structured `emeland.io/threshold.operator` + `emeland.io/threshold.limit`
pair: it compares the referenced MetricInstance's current value (`value <operator> limit`) and raises
a `ThresholdBreached` finding on breach. The free-form `threshold.expression` is documentation for
conditions that do not reduce to a simple operator+limit. Owner keys are recommended when
`--trust-auth-headers` is enabled (Threshold is owner-restricted).

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

## Referencing between observability resources

References are **first-class typed fields**, not annotations:

- `MetricValue.metricInstanceRef` → the MetricInstance it is a reading of.
- `Threshold.metricInstanceRef` → the MetricInstance it bounds.
- `MetricInstance.metricRef` → an optional abstract Metric grouping.
- `MetricInstance.subject` → an optional `ResourceRef` to the landscape resource being measured
  (any resource type, like `Finding.resources`).

None of these are validated on `Add`, so references may be delivered out of order or dangle. There
are no `emeland.io/thresholds` / `emeland.io/metric-values` base-resource annotation lists in this
model; a measurement's subject is the first-class `MetricInstance.subject`, not an annotation on the
measured resource.

## Worked examples

### Metric (abstract grouping)

```yaml
version: emeland.io/v1
kind: Metric
spec:
  metricId: 11111111-1111-1111-1111-111111111111
  displayName: p99 API latency
  description: End-to-end latency of the payments API (abstract)
  annotations:
    emeland.io/unit: ms
    emeland.io/dimension: latency
```

### MetricInstance (concrete, carries the query)

```yaml
version: emeland.io/v1
kind: MetricInstance
spec:
  metricInstanceId: 22222222-2222-2222-2222-222222222222
  displayName: p99 latency — payments-api (prod EU)
  metricRef:
    metricId: 11111111-1111-1111-1111-111111111111
  subject:
    resourceId: 550e8400-e29b-41d4-a716-446655440000
    resourceType: ApiInstance
  annotations:
    emeland.io/metric.expression: histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{service="payments"}[5m])) by (le))
    emeland.io/metric.language: promql
    emeland.io/unit: ms
```

### Threshold with a structured condition

```yaml
version: emeland.io/v1
kind: Threshold
spec:
  thresholdId: 33333333-3333-3333-3333-333333333333
  displayName: Latency SLO breach
  description: p99 latency must stay under 500ms
  metricInstanceRef:
    metricInstanceId: 22222222-2222-2222-2222-222222222222
  annotations:
    emeland.io/threshold.operator: gt
    emeland.io/threshold.limit: "500"
    emeland.io/owner-identities: sre-oncall
```

### MetricValue (current reading)

```yaml
version: emeland.io/v1
kind: MetricValue
spec:
  metricValueId: 44444444-4444-4444-4444-444444444444
  displayName: Current p99 latency
  metricInstanceRef:
    metricInstanceId: 22222222-2222-2222-2222-222222222222
  value: "412"
  annotations:
    emeland.io/owner-identities: sre-oncall
```

## Out of scope (explicit exclusions)

The following MUST NOT be used as observability annotation keys in Phase 6:

- Snapshot or observation timestamps (`emeland.io/observed-at`, `emeland.io/valid-from`, …)
- Time-series or history markers
- Any key whose purpose is storing temporal snapshots (snapshotting is handled outside this model)

Phase 6 stores **current-state** MetricValues only.

Metadata covered by this registry MUST NOT be promoted to first-class schema fields on Metric,
MetricInstance, Threshold, or MetricValue (see [observability ADR](adr/observability-resources.md)).
