# ADR: Phase 3 ordering resources

## Status

Accepted

## Context

EmELand Phase 3 models how organizational units order capabilities: discrete parameters and
valid values, versioned capability offerings, selectable variants with dependencies, and
orders that bind concrete values. The book describes a richer ordering graph; modelsrv keeps
a lean, Sensor-first landscape model with first-class resources for query/replication and
nested value objects only where the book objects are not independently addressable.

Slice 1 already landed **Parameter** (with `values` as ValidValue UUIDs) and **ValidValue**.
This ADR locks the remaining Phase 3 object model (CapabilityVersion, Variant, Order,
OrderItem, BoundValue) and the domain rules that gate Sensor apply.

## Decision

### First-class landscape resources

These are addressable resources (YAML `kind`, events, list/get query paths, replication):

| Resource | Path | Package |
|----------|------|---------|
| ValidValue | `/landscape/validValues` | `pkg/model/parameter` |
| CapabilityVersion | `/landscape/capabilityVersions` | `pkg/model/capability` |
| Variant | `/landscape/variants` | `pkg/model/capability` |
| Order | `/landscape/orders` | `pkg/model/order` |
| OrderItem | `/landscape/orderItems` | `pkg/model/order` |
| BoundValue | `/landscape/boundValues` | `pkg/model/order` |

**Capability** remains first-class; its `versions` field is a list of
**CapabilityVersionRef** (id only). **CapabilityVersion** holds the version metadata and a
required FK to Capability.

### Nested value objects (not landscape resources)

On **Variant** only:

- **ParameterValueSet** — `{parameterId, validValueIds[]}`
- **VariantDependency** — `{capabilityId, required[]ParameterValueSet}`

These are not `kind`s, have no query paths, and are converted hand-written with the Variant DTO.

### Parameter.values break

`Parameter.values` is `[]uuid.UUID` referencing **ValidValue** resources (not embedded value
objects). AddParameter validates each id exists and belongs to that Parameter. ValidValue
uniqueness remains `(parameterId, displayName)`.

### CapabilityVersion

| Field | Purpose |
|-------|---------|
| Identifier | Primary key |
| Display name | Human-readable name |
| Version | `common.Version` (semver string + lifecycle timestamps) |
| Capability reference | Required FK → Capability |
| Annotations | Extended metadata |

AddCapability rejects unknown CapabilityVersion ids in `versions[]`.

### Variant domain rules

1. CapabilityVersion must exist.
2. Each Parameter/ValidValue in `provides` must exist; each ValidValue must belong to its Parameter.
3. A dependency on Capability C is satisfied only if **some** Variant of C provides a
   **superset** of the required ValidValue ids for **each** ParameterValueSet in `required`.
4. The Variant→Capability dependency graph (edge from a Variant’s owning Capability to each
   dependency CapabilityId) must be **acyclic**.

Sentinels: `ErrVariantDependencyUnsatisfied`, `ErrVariantDependencyCycle`.

### Order / OrderItem / BoundValue

- **Order** — required OrgUnit FK; optional `items[]` OrderItemRefs.
- **OrderItem** — required Order + Capability FKs; optional CapabilityVersion and Variant refs;
  optional BoundValueRefs.
- **BoundValue** — required OrderItem + Parameter + ValidValue FKs.
  - Uniqueness: at most one BoundValue per `(orderItemId, parameterId)` → `ErrBoundValueConflict`.
  - ValidValue must belong to Parameter.
  - If the OrderItem has a Variant, the ValidValue must appear in that Variant’s `provides`
    for the Parameter.

### Explicit non-goals

- **No graph walking APIs** — dependency/cycle checks run at write (Add*) time only; no
  path-finding or “resolve order” query endpoints in Phase 3.
- **No SystemInstance link** — Orders are not tied to deployment instances; that remains a
  later concern (or annotations), not a core FK.

### Ingestion and query access

Same automation-first pattern as other landscape resources:

- **Sensor-first** create/update/delete via declarative YAML.
- **Read-only query API** — list and get-by-id.
- **Replication** — create/update/delete event apply for all Phase 3 resources above.

## Consequences

### Positive

- Clear split between addressable resources and nested Variant value objects.
- Parameter/ValidValue FK model avoids duplicating discrete values on Parameter.
- Dependency satisfaction and acyclicity are enforced at ingest, keeping the stored graph
  consistent without a separate analyzer service.

### Negative / trade-offs

- Apply order matters (Capability before CapabilityVersion before Variant with deps;
  Order before OrderItem before BoundValue).
- Superset dependency semantics are strict: one provider Variant must cover all required
  ParameterValueSets for a dependency.

### Follow-up work

- Optional ownership/visibility annotations on Order/BoundValue if Product needs them.
- Optional analytics over Variant dependency graphs outside the core model.
