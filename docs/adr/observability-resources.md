# ADR: Phase 6 observability resources

## Status

Accepted

## Context

EmELand Phase 6 introduces observability modelling: abstract measurements (metrics), concrete
instances of those measurements (metric instances, which carry the query and bind to a subject),
conditions of arbitrary complexity (thresholds), and current-state readings (metric values). A lean
core schema keeps the landscape model small; domain-specific metadata belongs in documented
`emeland.io/*` annotations.

Phase 6 scope is **current-state only** — no snapshot timestamps, history, or time-series fields on
the schema or in the observability annotation registry. Metrics are abstract measurements with
business value (including compound metrics), not time-series definitions. A MetricValue holds only
the current value, not a timeline.

This document describes the implemented observability object model (codegen, Sensor, query API,
replication).

## Decision

### Naming: MetricValue

The book concept **Value** is named **MetricValue** everywhere in modelsrv to avoid collision with
generic “value” language in APIs, annotations (`annotation.value`), and generated identifiers
(`GetValues` already exists on Parameter). The inner field that stores the reading remains `Value`
(accessors `GetValue` / `SetValue`).

### Target shape (lean core)

**Metric** — shared, abstract vocabulary (like ContextType / FindingType / CapacityResourceType):

| Field | Purpose |
|-------|---------|
| Identifier | Primary key (`MetricId`) |
| Display name | Human-readable name |
| Description | Optional detail |
| Annotations | Extended metadata — see [observability-annotations.md](../observability-annotations.md) |

A Metric is an abstract measurement ("p99 latency"), like a ContextType. It does **not** carry a
query and does **not** represent a time series — the concrete query lives on the MetricInstance.
Metric is an optional organizational grouping; a MetricInstance may exist without one.

**MetricInstance** — a concrete instantiation of a Metric, bound to a subject and carrying the query
(mirrors System/SystemInstance, API/ApiInstance):

| Field | Purpose |
|-------|---------|
| Identifier | Primary key (`MetricInstanceId`) |
| Display name | Human-readable name |
| Description | Optional detail |
| Metric reference | → Metric (**optional**, not validated on `Add`) |
| Subject | Optional first-class `ResourceRef` to the landscape resource being measured (any resource type, like `Finding.Resources`) |
| Annotations | Concrete query and language — `emeland.io/metric.expression`, `emeland.io/metric.language` — see registry |

The MetricInstance is the concrete measured thing: it carries the PromQL query (with its specific
labels/aggregation), because the query differs per subject. A scraped instance (e.g. from
AlertManager) may have no reliable parent Metric, so the Metric reference is optional.

**Threshold** — a condition of arbitrary complexity attached to a MetricInstance:

| Field | Purpose |
|-------|---------|
| Identifier | Primary key (`ThresholdId`) |
| Display name | Human-readable name |
| Description | Optional detail |
| MetricInstance reference | → MetricInstance (**not validated on `Add`**) |
| Annotations | Condition expression and language — see registry |

Thresholds do **not** have a fixed operator or scalar match value at the schema level. The condition
is unvalidated metadata in annotations (`emeland.io/threshold.expression`, `emeland.io/threshold.language`;
sensors may also use structured `emeland.io/threshold.operator` + `emeland.io/threshold.limit`).

**MetricValue** — current reading of a MetricInstance:

| Field | Purpose |
|-------|---------|
| Identifier | Primary key (`MetricValueId`) |
| Display name | Human-readable name |
| Description | Optional detail |
| MetricInstance reference | → MetricInstance (**not validated on `Add`**) |
| Value | Current value as an unvalidated string |
| Annotations | Extended metadata |

`Value` is not parsed (unlike Capacity `Amount`). Compound or non-numeric readings remain strings.

### Reference model

- **MetricInstance → Metric**: first-class typed `MetricRef`, **optional** and **not validated on
  `Add`**. An instance may reference an absent Metric (a dangling ref) or none at all.
- **Threshold / MetricValue → MetricInstance**: first-class typed `MetricInstanceRef`, **not
  validated on `Add`**. The referenced MetricInstance need not already exist.
- **MetricInstance → subject**: first-class generic `ResourceRef` (`Subject`), optional and
  unvalidated, pointing at the landscape resource being measured.

No observability reference is checked for existence on `Add` — this matches the other instance
types (ApiInstance/SystemInstance/ComponentInstance have no Add-time reference validation) and keeps
event apply **order-tolerant**: a replication or snapshot stream may deliver a MetricValue or
Threshold before the MetricInstance it references, or a MetricInstance before its Metric, without
the event being rejected. Dangling references can be surfaced later via findings rather than by
failing the write.

### Book Phase 6 relationship

modelsrv deliberately trims the core to identifiers, names, the MetricInstance query/subject, the
typed references (MetricInstance→Metric, Threshold/MetricValue→MetricInstance), and the MetricValue
reading. Units, composition formulas, and threshold conditions live in the
[annotation registry](../observability-annotations.md) instead of first-class columns.

### Ingestion and query access

- **Sensor-first**: create, update, and delete via declarative YAML through the file Sensor — not
  via landscape write endpoints on the query API.
- **Read-only query API**: list and get-by-id only.
- **Replication**: all four types (Metric, MetricInstance, Threshold, MetricValue) participate in
  cross-node event apply (create/update/delete).

### Uniqueness

No tuple uniqueness. Resources are keyed by id alone; `Add` is a plain upsert by id. A Metric may
have many MetricInstances (one per subject); a MetricInstance may have many MetricValues over
successive readings. The subject of a measurement is established via the first-class
`MetricInstance.Subject` reference, not via annotations on base resources.

### Read visibility

| Resource | Visibility |
|----------|------------|
| **Metric** | **Public vocabulary** — listed in `--public-resource-types` (like ContextType, FindingType, CapacityResourceType). |
| **MetricInstance** | **Owner/auditor restricted** when `--trust-auth-headers` is enabled. |
| **Threshold** | **Owner/auditor restricted** when `--trust-auth-headers` is enabled. |
| **MetricValue** | **Owner/auditor restricted** when `--trust-auth-headers` is enabled. |

Owners via `emeland.io/owner-*` annotations; semantics in [ownership-visibility.md](ownership-visibility.md).
Non-owners: omitted from list; get-by-id returns 404.

### Extended metadata (annotations)

All well-known observability annotation keys are documented in
[observability-annotations.md](../observability-annotations.md). The ADR does not duplicate that
registry.

No snapshot or time-series annotation keys are registered. Annotation values are plain strings.

### Reference limitations

- Because references are not validated on `Add`, a Threshold or MetricValue may reference a
  MetricInstance that does not exist (yet), and a MetricInstance may reference an absent Metric.
- Deleting a MetricInstance leaves any Threshold/MetricValue that referenced it pointing at a
  now-missing id (a dangling ref).
- A follow-up `pkg/eventfilter/observability` filter may raise `MissingResourceReference` findings
  for dangling observability references; clearing them reuses `pkg/eventfilter/resolvefindings`.

## Consequences

### Positive

- Lean schema stays stable as integrators add domain metadata via annotations.
- MetricValue naming avoids collision with generic value language.
- Public vocabulary for Metric; protected instance data for MetricInstance, Threshold and MetricValue.
- The MetricInstance layer lets one abstract Metric have many concrete, per-subject measurements,
  each with its own query and readings.
- Unvalidated references keep event apply order-tolerant (no snapshot/replication ordering constraint).

### Negative / trade-offs

- References are not existence-checked on `Add`, so the model does not guarantee referential
  integrity; dangling references are possible and must be surfaced via findings if desired.
- Threshold conditions are unvalidated strings — modelsrv does not interpret PromQL, CEL, or similar.
- No built-in history; consumers needing trends must integrate outside Phase 6 storage.

### Follow-up work

- Optional: dangling observability reference findings via `pkg/eventfilter/observability`.
