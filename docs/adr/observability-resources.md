# ADR: Phase 6 observability resources

## Status

Accepted

## Context

EmELand Phase 6 introduces observability modelling: abstract measurements (metrics), conditions of
arbitrary complexity (thresholds), and current-state readings (metric values). A lean core schema
keeps the landscape model small; domain-specific metadata belongs in documented `emeland.io/*`
annotations.

Phase 6 scope is **current-state only** — no snapshot timestamps, history, or time-series fields on
the schema or in the observability annotation registry. Metrics are abstract measurements with
business value (including compound metrics), not time-series definitions. A MetricValue holds only
the current value, not a timeline.

This ADR locks the object model before implementation (codegen, Sensor, query API, replication).

## Decision

### Naming: MetricValue

The book concept **Value** is named **MetricValue** everywhere in modelsrv to avoid collision with
generic “value” language in APIs, annotations (`annotation.value`), and generated identifiers
(`GetValues` already exists on Parameter). The inner field that stores the reading remains `Value`
(accessors `GetValue` / `SetValue`).

### Target shape (lean core)

**Metric** — shared vocabulary (like ContextType / FindingType / CapacityResourceType):

| Field | Purpose |
|-------|---------|
| Identifier | Primary key (`MetricId`) |
| Display name | Human-readable name |
| Description | Optional detail |
| Annotations | Extended metadata — see [observability-annotations.md](../observability-annotations.md) |

Metrics do **not** represent time series. They name abstract measurements, including compound
metrics whose composition formula lives in annotations.

**Threshold** — a condition of arbitrary complexity attached to a Metric:

| Field | Purpose |
|-------|---------|
| Identifier | Primary key (`ThresholdId`) |
| Display name | Human-readable name |
| Description | Optional detail |
| Metric reference | → Metric (validated) |
| Annotations | Condition expression and language — see registry |

Thresholds do **not** have a fixed operator or scalar match value. The condition is unvalidated
metadata in annotations (`emeland.io/threshold.expression`, `emeland.io/threshold.language`).

**MetricValue** — current reading of a Metric:

| Field | Purpose |
|-------|---------|
| Identifier | Primary key (`MetricValueId`) |
| Display name | Human-readable name |
| Description | Optional detail |
| Metric reference | → Metric (validated) |
| Value | Current value as an unvalidated string |
| Annotations | Extended metadata |

`Value` is not parsed (unlike Capacity `Amount`). Compound or non-numeric readings remain strings.

### Reference model

- **Metric ← Threshold / MetricValue**: first-class typed `MetricRef`. Validated on `Add` —
  missing Metric yields `ErrMetricNotFound`.
- **Base resource → Threshold / MetricValue**: annotation lists on the base resource
  (`emeland.io/thresholds`, `emeland.io/metric-values`). Opaque strings with **no** referential
  integrity in the model.

### Book Phase 6 relationship

modelsrv deliberately trims the core to identifiers, names, Metric refs, and the MetricValue
reading. Units, composition formulas, and threshold conditions live in the
[annotation registry](../observability-annotations.md) instead of first-class columns.

### Ingestion and query access

- **Sensor-first**: create, update, and delete via declarative YAML through the file Sensor — not
  via landscape write endpoints on the query API.
- **Read-only query API**: list and get-by-id only.
- **Replication**: all three types participate in cross-node event apply (create/update/delete).

### Uniqueness

No tuple uniqueness. Resources are keyed by id alone; `Add` is a plain upsert by id. A Metric may
have many MetricValues; the subject of a reading is established via annotations on base resources,
not via a field on MetricValue.

### Read visibility

| Resource | Visibility |
|----------|------------|
| **Metric** | **Public vocabulary** — listed in `--public-resource-types` (like ContextType, FindingType, CapacityResourceType). |
| **Threshold** | **Owner/auditor restricted** when `--trust-auth-headers` is enabled. |
| **MetricValue** | **Owner/auditor restricted** when `--trust-auth-headers` is enabled. |

Owners via `emeland.io/owner-*` annotations; semantics in [ownership-visibility.md](ownership-visibility.md).
Non-owners: omitted from list; get-by-id returns 404.

### Extended metadata (annotations)

All well-known observability annotation keys are documented in
[observability-annotations.md](../observability-annotations.md). The ADR does not duplicate that
registry.

No snapshot or time-series annotation keys are registered. Annotation values are plain strings.

### Annotation-reference limitations

- Deleting a Threshold or MetricValue leaves stale UUIDs in base-resource annotations.
- Reverse lookup (“which resources reference this Threshold?”) requires a full scan of annotated
  resources.
- A follow-up `pkg/eventfilter/observability` filter may raise `MissingResourceReference` findings
  for dangling annotation UUIDs; clearing them reuses `pkg/eventfilter/resolvefindings`.

## Consequences

### Positive

- Lean schema stays stable as integrators add domain metadata via annotations.
- MetricValue naming avoids collision with generic value language.
- Public vocabulary for Metric; protected instance data for Threshold and MetricValue.
- Validated Metric refs keep Threshold and MetricValue attachable to a known measurement.

### Negative / trade-offs

- Base-resource linkage to Thresholds and MetricValues requires convention discipline (annotation
  lists) and has no model-enforced integrity.
- Threshold conditions are unvalidated strings — modelsrv does not interpret PromQL, CEL, or similar.
- No built-in history; consumers needing trends must integrate outside Phase 6 storage.

### Follow-up work

- Vertical slices: Metric, then Threshold, then MetricValue (model, Sensor, query, replication).
- Optional: dangling annotation UUID findings via `pkg/eventfilter/observability`.
