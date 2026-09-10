# ADR: Mapping EmELand resources onto C4 abstractions

## Status

Accepted

## Context

`pkg/c4injector` renders landscape resources as C4-PlantUML. The original mapping treated an
EmELand **System** as a C4 Container (level 2) and an EmELand **Component** as a C4 Component
(level 3). Re-reading the [C4 container definition](https://c4model.com/abstractions/container)
against the domain model showed that mapping to be wrong in both directions, and the renderer
did not even implement what its own doc comment claimed: `RenderContainer` emitted `System()`
and `System_Ext()`, so `container.puml` contained no C4 Containers at all and was really a second
System Context diagram.

### An EmELand System is not a container

A C4 container is "something that needs to be running in order for the overall software system to
work" — a runtime boundary around executing code or stored data. An EmELand `System`:

- is documented as a high-level entity that "encapsulates various APIs and software components",
  i.e. a composition, not a runnable unit;
- uses `abstract` specifically "to represent external systems", which is `System_Ext` semantics
  (C4 spells the container equivalent `Container_Ext`);
- has an optional `Parent` System, and C4 containers never nest;
- carries no technology, while technology is a mandatory positional argument of `Container()`.

### An EmELand Component is a container

The definition's sharpest test is that two things communicating "via an inter-process/remote
communication mechanism (e.g. JSON/HTTPS)" are two separate containers. In EmELand:

- `Component.Provides` / `Consumes` are `[]api.ApiRef`, and `ApiType` is only ever `OpenAPI`,
  `GraphQL`, `GRPC`, or `Other`. All component-to-component communication is remote by
  construction, so every Component is its own process space.
- Each `ComponentInstance` is deployed separately, and its `ApiInstance` carries an
  `emeland.io/endpoint.host` annotation that `pkg/endpointprobe` opens a TLS connection to.
  Its own host and its own handshake make it a runtime construct with a network identity.
- The definition's counter-example — JARs, assemblies, DLLs and modules, which "organise the code
  within those applications" — matches nothing in the model. `Component` has no `Parent` and no
  sub-parts, so it is not a code-organisation unit.

### There is no level-3 source

Because `Component` has no sub-parts, no EmELand resource can populate a genuine C4 Component
diagram. The former level 3 was a container diagram with the wrong labels.

## Decision

| EmELand resource | C4 abstraction | PlantUML |
|---|---|---|
| `Landscape` (hardcoded, not a resource) | landscape boundary | `Enterprise_Boundary` |
| `Context` | landscape / deployment boundary | nested `Boundary` |
| `System` (concrete) | Software System | `System_Boundary` on level 2 |
| `System` (abstract) | external Software System | `System_Ext` |
| `Component` | **Container** | `Container`, technology = provided API types |
| `SystemInstance` | deployed Software System | `System_Boundary` on the deployment diagram |
| `ComponentInstance` | deployed Container | `Container` |
| `ApiInstance` | deployed endpoint | `Container` tagged `endpoint` |

Level 2 (`/documents/c4/container.puml`) is the only structural type-level diagram: one
`System_Boundary` per concrete System, one `Container` per Component inside it, `System_Ext` for
abstract Systems reached through `consumes`. Container-to-container edges within a single System
are kept — they are still remote API calls, and there is no lower level to defer them to.

Level 3 (`/documents/c4/component.puml`) returns HTTP 404 with `ReasonComponentLevel`, alongside
level 4 (`code.puml`) with `ReasonCodeLevel`.

### Every instance is drawn somewhere

An instance that is not drawn is worse than one drawn imprecisely: the model looks complete when
it is not. So the deployment diagram is the total inventory of instance resources, and no instance
is filtered out for having incomplete references.

- `ApiInstance` is rendered, having previously appeared on no diagram at all. It uses `Container`
  with an `endpoint` element tag, which gives it an `«endpoint»` stereotype distinct from a
  `ComponentInstance`. Technology is `protocol:port` and the description is `host` + `path`, both
  read from the `emeland.io/endpoint.*` annotations that `pkg/endpointprobe` also consumes — that
  annotation set is the model's only record of where a deployed API answers.
- A `ComponentInstance` or `ApiInstance` whose `SystemInstance` ref is unset *or* points at a UUID
  with no matching SystemInstance is grouped under the synthetic `(no Context)` boundary. Only the
  nil-ref case was previously bucketed, so a dangling UUID meant silent disappearance.
- An `ApiInstance` with no resolvable API ref is still drawn; it just carries no type name in its
  label and can source no edge. Edge projection continues to require both refs.

Type-level diagrams stay free of instances, since mixing them defeats the abstraction. Level 2
instead annotates each `Container` description with its `ComponentInstance` count, stating
`[no instances]` explicitly so an undeployed Component is visible as such.

Placement gaps are recorded as Findings rather than by omission — `SystemInstanceContextMissing`
and `ApiInstanceSystemInstanceMissing` are the inventory of instances whose position is unknown.

## Consequences

- `container.puml` changes shape: PlantUML aliases for Components move from `comp_<uuid>` to
  `container_<uuid>`, and `RelEdge.FromKind` / `ToKind` use `"container"` instead of `"component"`.
- `component.puml` stops returning a diagram. Anything consuming it should read `container.puml`.
- The deployment diagram draws `ComponentInstance` as `Container` inside a `System_Boundary` and
  includes `C4_Container` rather than `C4_Component`.
- Level 2 descriptions gain an instance-count line, so `container.puml` output now depends on
  ComponentInstances as well as types.
- Level 1 still shows only `Context` boundaries and no Systems, so it is a System Context diagram
  without systems. Placing concrete Systems on level 1 is unresolved: a `System` is a type and only
  `SystemInstance` is scoped to a `Context`, so there is no type-level containment to draw.

## Known gap: data stores

The container definition explicitly counts databases, blob stores, file systems, shell scripts and
serverless functions as containers, and says to treat services like S3 and RDS as containers
"because they are an integral part of your software architecture, although they are hosted
elsewhere". The model cannot express these:

- An owned Postgres schema is not an *external* system, so `System.Abstract` is the wrong tool.
- Modelling it as a `Component` forces an `API` on it, and `ApiType` has no data-store member —
  only the `Other` escape hatch.
- Neither `System` nor `Component` has a technology field, so there is nowhere to record
  "PostgreSQL 16 schema"; level 2 borrows the provided API type instead, which yields a protocol
  (`OpenAPI`) rather than a technology.

Closing this gap needs a `ContainerDb`-style shape and a first-class technology field. Deferred;
not designed here.
