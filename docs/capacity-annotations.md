# Capacity annotation registry

Well-known `emeland.io/*` annotation keys for **CapacityResourceType** and **Capacity**. modelsrv
stores these as ordinary annotations (`map[string]string` in the model; `{ key, value }` objects on
the query API). No runtime validation is applied in Phase 1 — recommendation levels guide integrators
and downstream tooling.

**Related docs**:

- [ADR: Phase 7 capacity resources](adr/capacity-resources.md) — lean core schema
- [ADR: Annotation-based ownership visibility](adr/ownership-visibility.md) — owner key semantics

## Value format

All annotation values are **plain UTF-8 strings**. UUIDs (for example subject linkage) MUST use
standard UUID string form (e.g. `550e8400-e29b-41d4-a716-446655440000`). modelsrv does not parse
structured data from annotation values for registry keys.

Declarative YAML uses a map under `spec.annotations`:

```yaml
spec:
  annotations:
    emeland.io/source: planned
```

## Recommendation levels

| Level | Meaning |
|-------|---------|
| **recommended** | Interoperability expectation; omit only when the metadata does not apply |
| **optional** | Useful hint or override; safe to omit |

## Keys on both resource types

These keys may appear on **CapacityResourceType** and **Capacity**.

| Key | Recommendation | Purpose | Example |
|-----|----------------|---------|---------|
| `emeland.io/owners` | recommended* | Owner OIDC subjects or Owner group ids | `alice,bob,platform-team` |

\*Recommended when read visibility should be restricted. **Visibility rules, parsing, and enforcement
are defined only in [ownership-visibility.md](adr/ownership-visibility.md)** — not redefined here.
Key strings match `pkg/authz` constants (`OwnerIdentitiesKey`, `OwnerGroupsKey`).

## Keys on CapacityResourceType

| Key | Recommendation | Purpose | Example |
|-----|----------------|---------|---------|
| `emeland.io/dimension` | recommended | UI / grouping family | `compute`, `storage`, `license`, `budget` |
| `emeland.io/value-kind` | optional | Validation hint for amounts | `integer`, `decimal` |
| `emeland.io/granularity` | optional | Smallest meaningful step | `0.001` |

## Keys on Capacity

| Key | Recommendation | Purpose | Example |
|-----|----------------|---------|---------|
| `emeland.io/source` | recommended | Provenance of the amount | `measured`, `reported`, `planned`, `estimated` |
| `emeland.io/subject-kind` | recommended | Landscape type of related subject | `OrgUnit`, `SystemInstance` |
| `emeland.io/subject-id` | recommended | Subject resource UUID (string) | `550e8400-e29b-41d4-a716-446655440000` |
| `emeland.io/unit` | optional | Unit override when entry differs from type default | `mcores` |
| `emeland.io/reserved-amount` | optional | Allocated but not consumed | `8` |
| `emeland.io/soft-limit` | optional | Warning threshold | `0.8` |
| `emeland.io/hard-limit` | optional | Hard cap | `1.0` |

### Subject linkage

When a Capacity entry relates to another landscape resource, set **both**:

- `emeland.io/subject-kind` — resource type name (PascalCase, as in declarative `kind`)
- `emeland.io/subject-id` — UUID string of that resource

If either key is missing, treat the entry as **unattached**. Subject linkage is metadata only;
uniqueness remains `(context, capacity resource type, category)`.

### Provenance (`emeland.io/source`)

Suggested values (convention, not validated by modelsrv):

| Value | Typical use |
|-------|-------------|
| `measured` | Observed from monitoring or agents |
| `reported` | Declared by an owner or upstream system |
| `planned` | Target or forecast capacity |
| `estimated` | Approximation when exact measurement is unavailable |

## Worked examples

### Source on a provided entry

```yaml
version: emeland.io/v1
kind: Capacity
spec:
  capacityId: a1b2c3d4-e5f6-7890-abcd-ef1234567890
  displayName: Measured memory provided
  resourceTypeRef: 11111111-1111-1111-1111-111111111111
  contextRef: 22222222-2222-2222-2222-222222222222
  category: provided
  amount: "64"
  annotations:
    emeland.io/source: measured
```

### Subject linkage on a requested entry

```yaml
version: emeland.io/v1
kind: Capacity
spec:
  capacityId: 7c9e6679-7425-40de-944b-e07fc1f90ae7
  displayName: Requested vCPU for prod instance
  resourceTypeRef: 33333333-3333-3333-3333-333333333333
  contextRef: 22222222-2222-2222-2222-222222222222
  category: requested
  amount: "16"
  annotations:
    emeland.io/source: planned
    emeland.io/subject-kind: SystemInstance
    emeland.io/subject-id: 550e8400-e29b-41d4-a716-446655440000
    emeland.io/owner-identities: platform-lead
```

### Dimension on a capacity resource type

```yaml
version: emeland.io/v1
kind: CapacityResourceType
spec:
  capacityResourceTypeId: 44444444-4444-4444-4444-444444444444
  displayName: CPU cores
  unit: cores
  annotations:
    emeland.io/dimension: compute
    emeland.io/value-kind: decimal
    emeland.io/granularity: "0.001"
```

## Out of scope (explicit exclusions)

The following MUST NOT be used as capacity annotation keys in Phase 7:

- Snapshot or observation timestamps (`emeland.io/observed-at`, `emeland.io/valid-from`, …)
- Time-series or history markers
- Any key whose purpose is storing temporal snapshots (snapshotting is handled outside this model)

Phase 7 stores **current-state** capacity only. Time aggregation belongs to Phase 4 Billing.

Metadata covered by this registry MUST NOT be promoted to first-class schema fields on
CapacityResourceType or Capacity (see [capacity ADR](adr/capacity-resources.md)).
