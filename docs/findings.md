# Findings

## What is a Finding?

A **Finding** is a model resource that represents a detected condition in the
landscape that warrants attention. The most common findings are
referential-integrity violations — a resource that references another resource
by UUID, but that referenced resource does not (yet) exist in the model.

Findings are first-class resources: they have a stable UUID, participate in
the event stream (`FindingResource`), and are queryable via the model's
`GetFindings` / `GetFindingById` methods alongside nodes, contexts, and all
other resource types.

A finding carries:

| Field | Purpose |
|-------|---------|
| `DisplayName` | Human-readable name of the finding (also the resource name). |
| `Description` | Detailed explanation of the violation (formerly carried in `Summary`). |
| `TypeRef` | Reference to the `FindingType` that classifies this finding (see below). |
| `Resources` | Ordered list of `ResourceRef` values — subject resource first, then any referenced-but-missing resources. |
| `Annotations` | Arbitrary key/value metadata. |

### Read API shape

`GET /api/landscape/findings` and `GET /api/landscape/findings/{findingId}`
return an enriched **FindingView** that includes `findingId`, resolved `type` and
`resources` objects (with human-readable `displayName` values), a `reference`
URI, and `annotations`.
Phase-0 findings use `displayName: "Phase 0 Integrity check"` and put the
detailed violation text in `description`.

## What is a FindingType?

A **FindingType** classifies findings, in the same way that a `NodeType`
classifies `Node` resources or a `ContextType` classifies `Context` resources.
It is itself a first-class resource with a UUID, `DisplayName`, and
`Description`.

A `Finding` holds a `FindingTypeRef` which stores the UUID of its
`FindingType`. The resolved object is accessible via `GetFindingType()` once
the `FindingType` is registered in the model; before that, only the UUID is
available via `GetFindingTypeId()`.

### Well-known FindingType UUIDs

The built-in finding categories each have a **deterministic UUID** derived from
the category name using UUID v5 (namespace
`c3d4e5f6-a7b8-9012-cdef-012345678901`). The helper `finding.TypeIDForKind`
computes these:

```go
finding.TypeIDForKind(finding.ContextTypeMissing)          // stable UUID for ContextTypeMissing
finding.TypeIDForKind(finding.ContextParentNotFound)       // stable UUID for ContextParentNotFound
finding.TypeIDForKind(finding.NodeTypeMissing)             // stable UUID for NodeTypeMissing
finding.TypeIDForKind(finding.ReferencedResourceNotFound)  // stable UUID for ReferencedResourceNotFound
finding.TypeIDForKind(finding.MissingResourceReference)    // stable UUID for MissingResourceReference
```

This means filter code can call `f.SetFindingTypeById(finding.TypeIDForKind(kind))`
without requiring the `FindingType` resource to already exist in the model. If
you want `GetFindingType()` to resolve to a full object (with a human-readable
`DisplayName`), register a `FindingType` in the model under the same UUID — for
example via a YAML file ingested by the file sensor.

## How findings are generated

Findings are produced as **side-effects of filter functions** registered in the
`eventfilter.Chain`. A filter function receives every model event and the
current model state; it can inspect the event, check referential integrity, and
call `model.AddFinding` or `model.DeleteFindingById` to upsert or remove
findings. The original event is always forwarded downstream unchanged.

### Finding UUID stability

Built-in filters derive finding UUIDs deterministically from the subject
resource's UUID and the `FindingKind` string (using UUID v5, namespace
`7a3f2c1e-4b8d-5e9f-a0b1-c2d3e4f56789`). This guarantees:

- **No duplicates** — applying the same event multiple times produces exactly
  one finding, not many.
- **Upsert semantics** — if the condition changes (e.g. the description is updated),
  the finding is replaced in-place.
- **Coexistence** — a single subject resource can have multiple findings
  simultaneously (e.g. a `Context` that is both missing its type and referencing
  a non-existent parent), because different `FindingKind` strings yield different
  UUIDs.

### Finding lifecycle

A finding is **created** when the violating condition is first detected on a
`Create` or `Update` event. It is **deleted** automatically when the same filter
sees that the condition has been resolved — for example, when a missing
`ContextType` is added to the model and the `Context` is subsequently updated.

## Phase 0 findings

The `pkg/eventfilter/phase0` filter covers **phase-0 resources** (Node,
NodeType, Context, ContextType) and is registered automatically in
`pkg/backend/backend.go`.

### ContextTypeMissing

**Trigger:** A `Context` Create or Update event arrives and either:

- The context has no `ContextType` set at all (`GetContextTypeId()` returns
  `uuid.Nil`), **or**
- The context references a `ContextType` by UUID but that UUID is not
  registered in the model.

**Resolved by:** A subsequent Update event on the context in which
`GetContextTypeById(typeId)` returns a non-nil result.

**Resources in the finding:**
1. The `Context` (subject).
2. The referenced `ContextType` UUID, if one was set (so consumers can identify
   exactly which type is missing).

---

### ContextParentNotFound

**Trigger:** A `Context` Create or Update event arrives and the context has a
non-nil parent UUID (`GetParentId()`) but that UUID is not registered in the
model.

**Not a violation:** Having no parent at all (`GetParentId()` returns
`uuid.Nil`) is valid — parent is an optional relationship.

**Resolved by:** A subsequent Update event on the context in which the parent
UUID is either cleared (`uuid.Nil`) or a `Context` with that UUID now exists in
the model.

**Resources in the finding:**
1. The `Context` (subject).
2. The referenced parent `Context` UUID.

---

### NodeTypeMissing

**Trigger:** A `Node` Create or Update event arrives and the node has no
`NodeType` assigned (`GetNodeType()` returns `nil`).

**Resolved by:** A subsequent Update event on the node in which
`GetNodeType()` returns a non-nil result.

**Resources in the finding:**
1. The `Node` (subject).

## Resolve-findings filter

The `pkg/eventfilter/resolvefindings` filter cleans up findings when later
ingress supplies the missing information. It is registered automatically in
`pkg/backend/backend.go` after phase 0. It does **not** create findings —
Sensors (for example
[modelsrv-k8s-sensor#34](https://github.com/emeland-io/modelsrv-k8s-sensor/issues/34))
emit them; this filter only deletes them when resolved
([modelsrv#152](https://github.com/emeland-io/modelsrv/issues/152)).

**Unknown kinds are never discarded.** Findings whose `FindingType` / kind is
not known to modelsrv remain in the model until a positive resolution rule
matches (or another owner deletes them). The filter always forwards the
incoming event unchanged.

### Structural dangling-ref resolution (kind-agnostic)

When a landscape resource is created or updated, the filter scans pending
findings. If a finding’s `Resources` list has length ≥ 2 (subject first, then
one or more referenced-but-missing resources) and any missing ref matches the
event’s resource id (and type when set), and that resource now exists in the
model, the finding is deleted.

This path covers documented `ReferencedResourceNotFound` **and** custom /
future Sensor kinds that follow the same `Resources` layout. Phase-0 kinds
(`ContextTypeMissing`, `ContextParentNotFound`, `NodeTypeMissing`) are skipped
here so phase 0 retains ownership of their negation.

K8s Context parent annotations (`emeland.io/k8s-sensor/context-parent`) should
map into `Context.Parent` and rely on phase 0’s `ContextParentNotFound`, not
this filter.

### ReferencedResourceNotFound

**Meaning:** A subject cites a resource UUID that is not registered in the
local model.

**Resources in the finding:**
1. The subject resource (e.g. `ApiInstance`, `ComponentInstance`).
2. The referenced-but-missing resource (e.g. `API`, `Component`).

**Resolved by:** Create/Update of the missing resource when it becomes present
in the model (structural path above).

### MissingResourceReference

**Meaning:** A subject lacks a required EmELand reference (workflow
“missing-resource” — e.g. an `ApiInstance` without an API ref annotation).

**Resources in the finding:**
1. The subject resource only.

**Resolved by:** Create/Update of the subject when its first-class reference is
populated:

- `ApiInstance` → `ApiRef` with non-nil `ApiID`
- `ComponentInstance` → `ComponentRef` with non-nil `ComponentId`

Other subject types are left untouched until a rule is documented.

### Sensor contract

External Sensors that emit findings intended for this filter SHOULD:

1. Prefer `finding.TypeIDForKind` when the kind is one of the documented kinds
   above.
2. Always use **subject-first** `Resources`, with missing targets after the
   subject, so unknown kinds can still be resolved structurally.
3. Prefer the same deterministic finding UUID scheme as phase 0
   (UUID v5 from subject id + kind) so re-emits upsert rather than duplicate.

Findings that do not match any resolution rule are retained — including
unknown kinds with only a subject ref and no missing target.

## Registering FindingTypes for well-known kinds

To give the built-in findings a human-readable `DisplayName` and `Description`,
register `FindingType` resources in the model under the well-known UUIDs.  The
easiest way is via a YAML file processed by the file sensor.

The stable UUIDs for the built-in kinds are:

| Kind | UUID |
|------|------|
| `ContextTypeMissing` | `fa538332-fb6d-51ef-99f3-87831ac140fb` |
| `ContextParentNotFound` | `daf948a3-f77d-582e-9bbe-72251e22373f` |
| `NodeTypeMissing` | `808c222c-3e02-5d38-9a82-4b16c792b075` |
| `ReferencedResourceNotFound` | `26a693f2-996d-5310-9e5b-a357722dcda5` |
| `MissingResourceReference` | `904c4012-fa93-5bbf-a8fe-7907eccce5d5` |

Example YAML:

```yaml
---
version: emeland.io/v1
kind: FindingType
spec:
  findingTypeId: "fa538332-fb6d-51ef-99f3-87831ac140fb"
  displayName: "ContextTypeMissing"
  description: >
    A Context resource references a ContextType by UUID that is not registered
    in the model, or has no ContextType assigned at all.
---
version: emeland.io/v1
kind: FindingType
spec:
  findingTypeId: "daf948a3-f77d-582e-9bbe-72251e22373f"
  displayName: "ContextParentNotFound"
  description: >
    A Context resource references a parent Context by UUID that is not
    registered in the model.
---
version: emeland.io/v1
kind: FindingType
spec:
  findingTypeId: "808c222c-3e02-5d38-9a82-4b16c792b075"
  displayName: "NodeTypeMissing"
  description: >
    A Node resource has no NodeType assigned.
---
version: emeland.io/v1
kind: FindingType
spec:
  findingTypeId: "26a693f2-996d-5310-9e5b-a357722dcda5"
  displayName: "ReferencedResourceNotFound"
  description: >
    A resource references another resource by UUID that is not registered in
    the local model.
---
version: emeland.io/v1
kind: FindingType
spec:
  findingTypeId: "904c4012-fa93-5bbf-a8fe-7907eccce5d5"
  displayName: "MissingResourceReference"
  description: >
    A resource lacks a required EmELand reference to another resource.
```

## Test manifest repository

[`emeland-io/test-gitsensor-target`](https://github.com/emeland-io/test-gitsensor-target)
is a companion repository that contains YAML manifests whose purpose is to
deliberately trigger specific findings. The repository is structured as
follows:

```
watchedDir/    ← manifests in this directory are processed by the file sensor
unwatchedDir/  ← files here are intentionally ignored
```

Each manifest file under `watchedDir/` is named after the finding it is
designed to produce, for example:

```
watchedDir/ContextTypeMissing.yaml      ← creates a Context with no ContextType
watchedDir/ContextParentNotFound.yaml   ← creates a Context whose parent UUID doesn't exist
watchedDir/NodeTypeMissing.yaml         ← creates a Node with no NodeType
```

This naming convention makes it easy to see at a glance which conditions the
repository exercises and to verify that a deployed `modelsrv` instance responds
with the expected findings when the file sensor watches `watchedDir/`.

### Adding a manifest for a new finding

When a new `FindingKind` is integrated into `modelsrv`, add a corresponding
manifest to the `watchedDir/` directory of
[`test-gitsensor-target`](https://github.com/emeland-io/test-gitsensor-target):

1. Create a YAML file named `<FindingKind>.yaml` (e.g. `NodeParentNotFound.yaml`).
2. The manifest should contain the minimum resource definition that triggers the
   new finding — typically a resource with a deliberate referential-integrity
   violation.
3. Open a pull request against `emeland-io/test-gitsensor-target` alongside (or
   shortly after) the `modelsrv` PR that introduces the new kind.

This ensures there is always a concrete, version-controlled example that
documents the conditions under which each finding is generated.

## Adding new findings

To add a new finding category:

1. Add a new `FindingKind` constant to `pkg/model/finding/finding_kind.go`.
2. Implement creation checks in the appropriate filter (e.g.
   `pkg/eventfilter/phase0` for phase-0 resources) or in an external Sensor.
3. For dangling-reference findings that Sensors emit, prefer the
   subject-first + missing-ref `Resources` layout so
   `pkg/eventfilter/resolvefindings` can clear them structurally — even before
   modelsrv knows the kind by name.
4. For kinds that need subject-field checks (like `MissingResourceReference`),
   extend the documented rules in `resolvefindings`.
5. Add tests next to the creating filter / Sensor and, when resolution is
   modelsrv-owned, in `pkg/eventfilter/resolvefindings`.
6. Optionally register a `FindingType` YAML definition so the kind has a
   human-readable description in the model.
7. Add a manifest to `watchedDir/` in
   [`emeland-io/test-gitsensor-target`](https://github.com/emeland-io/test-gitsensor-target)
   named `<FindingKind>.yaml` that triggers the new finding (see
   [Test manifest repository](#test-manifest-repository) above).
