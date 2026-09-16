# ADR: Level-indexed structure and a third structural tier

## Status

Accepted (not implemented)

The decisions below are locked. None of them are in the tree yet: level-indexed paths, configurable labels, `Part` / `PartInstance`, and `Technology` remain unbuilt. The C4 injector currently implements only [adr/c4-abstraction-mapping.md](c4-abstraction-mapping.md) (context / container / deployment; component and code return 404). Treat this document as a design lock, not as shipped work.

## Context

[adr/c4-abstraction-mapping.md](c4-abstraction-mapping.md) established that an EmELand `System` is a
C4 Software System and an EmELand `Component` is a C4 Container, and that no resource can populate a
C4 Component diagram: `Component` is a leaf. Its whole surface is id, display name, description,
version, `System`, `Consumes`, `Provides`, `Annotations` — no parent, no children — and the only
resource in `pkg/model` that references a Component is `ComponentInstance`, which is the deployed
copy of it rather than a piece of it. Containment therefore bottoms out at
`Context → System → Component → nothing`.

We want a third structural tier with finer granularity than the deployable, and it must carry the
same type/instance duality as the rest of the model.

### The naming problem

The obvious names are taken or overloaded:

- **`Container`** is inseparable from Docker for most readers. The C4 site concedes the point:
  "it's unfortunate that containerisation has become popular, because many software developers now
  associate the term container with Docker."
- **`Component`** is already used in EmELand for the deployable tier, and outside EmELand it means a
  UI element to anyone who has touched React, Vue or Angular.
- **`Module`** collides with Go modules and Terraform modules; **`building block`** is already spoken
  for in this project, where it denotes Sensor / Filter / Injector.

Renaming `Component` → `Container` to free up the noun is not a cheap diff either. The strings in
`resourceTypeValues` (`pkg/events/events.go`) are externally visible:

```go
ComponentResource:         "Component",
ComponentInstanceResource: "ComponentInstance",
```

They appear in replication payloads, the OpenAPI `ResourceRef` enum, the `/api/landscape/components`
REST path, and `kind: Component` in ingested YAML — so `modelsrv-k8s-sensor`,
`test-gitsensor-target` and any replication peer break on a rename.

C4 anticipates all of this: "While many teams successfully use the C4 model as is, feel free to
change the terminology if needed." The lesson we take from that is to stop borrowing C4 nouns for
identity, and treat them as labels instead.

### Levels, not nouns

C4's own structure is already level-indexed, and each structural element appears twice: as the
*content* of its own level and as the *boundary* on the level below. EmELand's spine works the same
way — `System` is content on level 1 and a boundary on level 2; `Component` is content on level 2 and
would be a boundary on level 3. Identifying tiers by level rather than by noun makes that regularity
explicit and stops each new tier from requiring a new word.

## Decision

### 1. The diagram layer becomes level-indexed

Diagram identity moves from C4 nouns to level numbers. Paths become
`/documents/c4/level1.puml` … `/documents/c4/level4.puml`, with the existing
`context` / `container` / `component` / `deployment` paths retained as permanent aliases so nothing
that reads them today breaks. `Level` is already an enum in `pkg/c4injector/http.go`, so this is a
routing and naming change rather than a rendering one.

### 2. Level labels are configuration, not schema

The noun rendered for each level — in boundary types and element labels — comes from a per-landscape
label map with C4's own words as defaults. A shop that runs Kubernetes can label level 2 "Deployable"
or "Service" without the model or the renderer changing. This is the configurable naming the model
owes its users, and it is the layer where the overloaded-noun problem actually bites, since that is
the text a human reads on a diagram.

The map lives in `emeland.io/*` annotations rather than server flags, so labels travel with the
model through replication instead of being per-process configuration that can disagree between
peers. Flags stay limited to landscape identity (`--c4-landscape-name`).

### 3. The third tier is `Part` / `PartInstance`

`System` and `Component` keep their current names and semantics; a new `Part` type is added below
`Component`, with a `PartInstance` counterpart. `Part` is short, reads as "part of", and avoids the
Docker, React and Go-module collisions catalogued above. This is Option B in
[Model layer options](#model-layer-options).

### 4. Relationships on `Part` are untyped in-process dependencies

`Part` gets `DependsOn []PartRef` and does **not** get `Provides` / `Consumes`. Parts inside one
deployable communicate in-process; attaching an `API` to a Part would assert a remote call on a code
element and contradict the reasoning that made `Component` a container in the first place. Keeping
APIs off `Part` is the invariant that stops the tier drifting back into being a container.

Dependencies are unlabelled. Edge kinds (calls / reads / extends) would be the fourth vocabulary in
a model that already has `ApiType`, `ContextType` and `FindingType`, and level-3 diagrams read fine
with plain arrows. If typed edges are wanted later they can arrive as an annotation before they
become schema.

### 5. Type/instance duality is preserved

`PartInstance` references both its `Part` type and the enclosing `ComponentInstance`, matching
`System`/`SystemInstance` and `Component`/`ComponentInstance`.

### 6. Technology becomes a first-class field

Level 2 currently synthesises its C4 technology string from provided API types, which yields a
protocol (`OpenAPI`) where C4 wants a technology (`Go / net/http`), and yields nothing at all for a
deployable that only consumes. A real `Technology` field on `Component` and `Part` closes the gap
recorded in the mapping ADR.

## Model layer options

The diagram-layer decisions above are independent of how the third tier is modelled. Two candidates
were considered; decision 3 selects Option B.

### Option A — one recursive, level-indexed structural type

A single type carrying `Level`, `Parent`, `DependsOn`, replacing `System` and `Component` as
distinct types. Depth becomes data, so a fourth or fifth tier never requires a new noun or a new
resource type — which is precisely the problem this ADR is trying to stop recurring.

The cost is the largest migration in the model's history: it removes two resource types that are
externally visible, and it gives up per-type foreign-key validation and the type-level authz
surface. It also weakens the schema's ability to state that only level 2 may own APIs.

### Option B — a third distinct type with a neutral noun (chosen)

`System` and `Component` stay exactly as they are; one new type and its instance are added below
`Component` following the 18-step checklist in [adding-resource-types.md](../adding-resource-types.md).
Nothing breaks, validation stays per type, and the "only level 2 owns APIs" rule
stays expressible in the schema. The cost is that `Component` permanently denotes the deployable
tier, so the C4 reader still has to hold a translation in their head — mitigated, but not removed, by
the configurable labels in decision 2.

Option A remains the better end state if the model is ever opened for a breaking major version, and
every decision above is compatible with migrating to it later: level-indexed paths, annotation-borne
labels and an untyped `DependsOn` all survive a move to a recursive type.

## Resulting mapping

| EmELand | C4 abstraction | Level |
|---|---|---|
| `Context` | landscape / deployment boundary | 1 |
| `System` / `SystemInstance` | Software System | 1 content, 2 boundary |
| `Component` / `ComponentInstance` | Container | 2 content, 3 boundary |
| `Part` / `PartInstance` | Component | 3 content |
| — | Code | 4 (out of scope) |

## Consequences

- Level 3 gains a real source and stops returning 404; level 4 (code) stays out of scope.
- `container.puml` and `component.puml` keep working as aliases, so the path change in decision 1 is
  not breaking for existing readers.
- Adding `Part` and `PartInstance` touches OpenAPI, `tools/gen/specs.go`, `pkg/events`,
  `pkg/ingress`, `pkg/model/structure.go`, authz wiring and replication — two types, so the
  18-step checklist runs twice. Codegen is idempotent, so the regen step is safe to repeat.
- `Part` is a leaf by construction, so the model keeps three structural tiers and cannot express a
  fourth without another ADR. That is deliberate: level 4 is code, and code structure changes on
  every commit.
- The C4 injector only depicts landscape state and does not raise Findings. Any future validation
  for `Part` (missing description, `Part` with no `Component`, `DependsOn` cycle) would be a
  separate concern, not part of diagram rendering.
- `Technology` on `Component` supersedes the synthesised API-type string that level 2 renders today,
  so existing diagrams change text once the field is populated.

## Follow-on work not covered here

Implementation of this ADR (`Part` / `PartInstance`, `Technology`, level-indexed paths, annotation-borne labels) is a separate issue from the #173 spike that shipped the mapping injector.

The data-store gap from [adr/c4-abstraction-mapping.md](c4-abstraction-mapping.md) is untouched: an
owned database still cannot be modelled, because `System.Abstract` means external and `ApiType` has
no data-store member. Decision 6 supplies the `Technology` field such a shape would need, but the
`ContainerDb` equivalent remains unspecified.

Level 1 still renders only `Context` boundaries and no Systems. Placing concrete Systems on level 1
is unresolved for the reason given in the mapping ADR: `System` is a type and only `SystemInstance`
is scoped to a `Context`, so there is no type-level containment to draw.
