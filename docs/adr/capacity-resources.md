# ADR: Phase 7 capacity resources

## Status

Accepted

## Context

EmELand Phase 7 introduces capacity modelling: how much of a measurable resource exists in a
deployment **Context**, classified as requested, provided, or consumed. The book uses the term
**ResourceType** for capacity vocabulary; modelsrv already uses “resource type” for event and HTTP
naming (e.g. `FindingResource`, replication payloads). A lean core schema keeps the landscape model
small; domain-specific metadata belongs in documented `emeland.io/*` annotations.

Phase 7 scope is **current-state only** — no snapshot timestamps, history, or time-series fields on
the schema or in the capacity annotation registry. Phase 4 Billing owns time aggregation; runway and
bottleneck forecasts remain derived analytics, not stored fields.

This ADR locks the object model before Phase 2–3 implementation (codegen, Sensor, query API,
replication).

## Decision

### Naming: CapacityResourceType

Rename the book **ResourceType** to **CapacityResourceType** everywhere in modelsrv to avoid
collision with event resource-type naming and generic “resource type” language in APIs and
replication.

### Target shape (lean core)

**CapacityResourceType** — shared vocabulary (like ContextType / FindingType):

| Field | Purpose |
|-------|---------|
| Identifier | Primary key |
| Display name | Human-readable name |
| Description | Optional detail |
| Unit | Measurement unit for amounts of this type (e.g. `cores`, `GiB`) |
| Annotations | Extended metadata — see [capacity-annotations.md](../capacity-annotations.md) |

**Capacity** — current-state row: how much of a type exists in a Context:

| Field | Purpose |
|-------|---------|
| Identifier | Primary key |
| Display name | Human-readable name |
| Description | Optional detail |
| Resource type reference | → CapacityResourceType |
| Context reference | → existing Phase 0 **Context** (no new Context type) |
| Category | `requested` \| `provided` \| `consumed` |
| Amount | Non-negative decimal: how much of the type in this context/category |
| Annotations | Extended metadata — see [capacity-annotations.md](../capacity-annotations.md) |

Unit for a Capacity entry comes from the linked CapacityResourceType. An optional
`emeland.io/unit` annotation may override for display or integration; it is not a core field.

### Book Phase 7 relationship

The book describes capacity types and amounts with richer metadata. modelsrv deliberately trims the
core to identifier, names, refs, category, amount, and unit-on-type. Fields such as provenance,
subject linkage, limits, reserved amount, and UI grouping live in the
[annotation registry](../capacity-annotations.md) instead of first-class columns.

### Ingestion and query access

- **Sensor-first**: create, update, and delete via declarative YAML through the file Sensor — not
  via landscape write endpoints on the query API.
- **Read-only query API**: list and get-by-id only (same automation-first pattern as other landscape
  resources).
- **Replication**: both types participate in cross-node event apply (create/update/delete).

### Capacity uniqueness and tuple-keyed upsert

At most one Capacity per `(context reference, capacity resource type reference, category)` tuple.

Declarative apply and replication apply use **tuple-keyed upsert**:

| Case | Behaviour |
|------|-----------|
| No row for tuple | Create with the document's CapacityId |
| Row exists, CapacityId matches | Update fields in place (CapacityId unchanged) |
| Row exists, CapacityId differs | Reject with diagnosable error; existing row unchanged |

### Read visibility

| Resource | Visibility |
|----------|------------|
| **CapacityResourceType** | **Public vocabulary** — listed in `--public-resource-types` (like ContextType, FindingType). No per-entry ownership required for vocabulary reads. |
| **Capacity** | **Owner/auditor restricted** when `--trust-auth-headers` is enabled. Owners via `emeland.io/owner-*` annotations; semantics in [ownership-visibility.md](ownership-visibility.md). Non-owners: omitted from list; get-by-id returns 404. |

### Extended metadata (annotations)

All well-known capacity annotation keys are documented in
[capacity-annotations.md](../capacity-annotations.md). The ADR does not duplicate that registry.

Owner keys (`emeland.io/owner-identities`, `emeland.io/owner-groups`) are listed there for
discoverability but defined only in [ownership-visibility.md](ownership-visibility.md).

No snapshot or time-series annotation keys are registered. Annotation values are plain strings.

## Consequences

### Positive

- Lean schema stays stable as integrators add domain metadata via annotations.
- CapacityResourceType naming avoids collision with event resource types.
- Tuple-keyed upsert gives idempotent Sensor apply for the natural capacity key.
- Public vocabulary for types; protected entries for instance-level capacity data.

### Negative / trade-offs

- Subject linkage and provenance require convention discipline (annotation pairs, documented keys).
- Unit override via annotation duplicates type-level unit in edge cases — acceptable for display
  variants (e.g. `mcores` vs `cores`).
- No built-in history; consumers needing trends must integrate outside Phase 7 storage.

### Follow-up work

- Phase 2: CapacityResourceType vertical slice (model, Sensor, query, replication).
- Phase 3: Capacity vertical slice with ownership visibility on reads.
- Optional: `pkg/model/capacity/doc.go` well-known-keys section linking to this registry.
