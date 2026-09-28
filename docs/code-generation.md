# Code generation

The resource types of the model are defined once, as annotated Go structs, and everything
else (the OpenAPI schema, the Go domain layer, the HTTP server/client, converters, mocks,
and their tests) is generated from them.

## Source of truth

- `tools/gen/modeldefs/model_defs.go` — one annotated struct per resource type. Field
  types, `emeland:` struct tags, and doc comments describe both the OpenAPI wire schema and
  the Go domain model. This file is parsed as source (it carries `//go:build ignore` and is
  never compiled into a binary).
- `tools/gen/wiring.go` — the freeform, code-like parts that do not belong in struct tags:
  custom method signatures, relational ref links (`TypeRefLink` / `ParentLink` /
  `RefByRefs`), per-type test setup, and domain-field overrides. Keyed by type name; only
  types that need it have an entry. Simple vocabulary types (e.g. `NodeType`, `OrgUnit`)
  have none.

To add or change a resource field you edit the struct in `model_defs.go` (and, for
relational types, its `wiring.go` entry), then run generation. You do not edit the OpenAPI
schema blocks or any `*_gen.go` file by hand.

## What is hand-authored vs generated

Hand-authored:
- `tools/gen/modeldefs/model_defs.go` and `tools/gen/wiring.go` (above).
- The non-resource parts of `api/openapi/EmergingEnterpriseLandscape-0.1.0-oapi-3.0.3.yaml`:
  `info`, `tags`, `servers`, all `paths` (endpoints), and the shared/support schemas
  (`InstanceList`, `ResourceRef`, `*Ref`, `*View`, `Event`, `Version`, `Annotation`,
  `ProductionVersion`, `ErrorString`, enums). The generator only rewrites the per-resource
  schema blocks in place; everything else passes through untouched.
- Small internal tables in `tools/gen`, changed only when adding a whole new resource type
  or model sub-package: enrichment maps in `wire_meta.go` (`skipConvertByName`,
  `wireKindToEventsResource`, etc.), the dir→import-alias map in `domain_meta.go`, and
  `canonicalTypeOrder` in `alltypes.go`.

Generated — never edit by hand (all marked `DO NOT EDIT`):
- `pkg/model/<pkg>/*_gen.go`, `pkg/model/handlers_*_gen.go`
- `pkg/client/client_gen.go` (+ `client_gen_test.go`)
- `internal/oapi/{server,client}.gen.go`
- `internal/oapi/{server_handlers,replication_*,convert_*}_gen.go`
- `pkg/mocks/mock_*.go`
- `pkg/model/model_store_gen_test.go`
- the per-resource schema blocks inside the OpenAPI YAML

## Generation order

Run `make gen` then `make generate`. The order matters: each stage consumes the previous
stage's output.

1. **gen-spec** (`go run ./tools/gen -mode=spec`, run first by `make gen`): parse the
   structs in `modeldefs/`, emit each resource's OpenAPI schema block, and merge those
   blocks in place into the OpenAPI file. Everything non-resource passes through unchanged.
2. **oapi-codegen** (`go generate ./internal/oapi/... ./pkg/client/...`): reads the merged
   OpenAPI file and produces the wire DTO structs and HTTP server/client (`server.gen.go`,
   `client.gen.go`).
3. **tools/gen** (`go run ./tools/gen`, via `go generate ./...` in `make generate`): builds
   the resource metadata from the structs (`buildAllTypes`: parse `modeldefs` → merge
   `wiring.go` → order by `canonicalTypeOrder` → enrich) and renders the Go code that
   depends on the oapi DTO names: domain interfaces/impls, handlers, client wrappers,
   replication, and the `FromDto`/`ToDto` converters.
4. **mockgen** (`go generate ./...`): regenerates mocks from the updated interfaces.

Dependency chain: `modeldefs` structs → OpenAPI resource schemas → oapi-codegen DTOs →
tools/gen Go code (references those DTOs) → mocks.

## How to add a scalar field to an existing type

1. Add the field to the type's struct in `tools/gen/modeldefs/model_defs.go`, with a doc
   comment (becomes the OpenAPI description) and the appropriate tag. For an optional field
   use `emeland:"optional"` (the DTO field becomes a pointer and conversion is nil-safe);
   omit it for a required field.
2. Run `make gen && make generate`.

That regenerates the OpenAPI schema, the domain accessors, the converters, the client, and
the mocks. Nothing else to touch for a plain scalar on a non-`skipConvert` type.

Caveats:
- Types in `skipConvertByName` keep hand-written converters in
  `internal/oapi/convert_special.go`; a new field there must be mapped by hand as well.
- If the field should be settable from YAML/CSV/JSON ingress, wire it into the relevant
  `pkg/ingress/apply_*.go` path (CSV columns are generic and need no code).
- A reference to another resource is not a scalar; it needs a `ref=`/`arrayref=` tag and,
  usually, a `wiring.go` entry (see below).

## How to add a new resource type

1. Add an annotated struct to `tools/gen/modeldefs/model_defs.go` (see the marker
   reference below).
2. If the type has relational references, custom methods, or non-trivial test setup, add a
   `wiring.go` entry for it.
3. Add its endpoints and any new shared/support schemas to the OpenAPI file by hand
   (`paths`, and its entry in the `ResourceRef`/`ResourceView` enums).
4. Register the type in the non-generated touchpoints: `pkg/events/events.go`,
   `pkg/ingress/document.go` (+ `apply_*.go`), `pkg/model/structure.go` (Model
   sub-interface, `modelData` maps, FK validation), `pkg/model/common/errors.go`,
   `pkg/model/resource_types.go`, and the enrichment tables in `tools/gen`
   (`wire_meta.go`, `domain_meta.go` if a new sub-package, `alltypes.go` order).
5. Run `make gen && make generate` and add hand-written tests for domain rules and ingress.

## Marker and tag reference

Type-level markers (in the struct's doc comment, one per line):

| Marker | Meaning |
|--------|---------|
| `+emeland:resource` | Marks the struct as a resource type (required). |
| `+emeland:dir=<pkg>` | Model sub-package directory (default: lower-cased type name). |
| `+emeland:handler` | Emit a `handlers_<type>_gen.go` handler. |
| `+emeland:client` | Generate client wrapper + client test + server handlers. |
| `+emeland:list=<path>` | REST list path, e.g. `/landscape/nodeTypes` (drives oapi method names). |
| `+emeland:event=<Name>` | Override the events resource name (e.g. `ApiInstance` → `APIInstance`). |
| `+emeland:handleralias=<alias>` | Import alias for the handler's package. |
| `+emeland:handlerdelete=<Method>` | Override the Model delete method name. |

Field tags (`emeland:"..."`, comma-separated):

| Tag | Meaning |
|-----|---------|
| `id` | Primary identifier field (not emitted as a wire property description). |
| `name` | The display-name field. |
| `required` | Property is in the schema's `required` list. |
| `optional` | Optional scalar: DTO field is a pointer; conversion is nil-safe. |
| `annotations` | The annotations bag (array of `Annotation` refs). |
| `ref=<Schema>` | Bare `$ref` to another schema (e.g. `ref=Version`). |
| `arrayref=<Schema>` | Array whose items are `$ref` to `<Schema>`. |
| `enum=A\|B\|C` | Enum values (pipe-separated). |
| `wire=<name>` | Explicit JSON property name (e.g. acronym ids: `wire=apiId`). |
| `pattern=<regex>` | OpenAPI `pattern`. |
| `example=<value>` | OpenAPI `example`. |

A field's doc comment becomes its OpenAPI `description`; a field with no doc comment emits
no description.

## Fidelity and tests

- `tools/gen` unit/semantic tests: `TestSchemaEmitter_Semantic` and
  `TestMergeSchemas_Semantic` assert the schema generated from the structs parses to the
  same OpenAPI model as the committed spec (semantic equivalence, not byte-for-byte);
  `TestConvertGen_*` cover the scalar converter logic.
- `make gen` is idempotent: re-running produces no changes. The committed generated files
  are kept in git; a clean regenerate should leave the tree unchanged.
- The generated `*_gen_test.go` files (store CRUD, client round-trips) are integration
  coverage of the produced domain/client/server code and regenerate for free.

See [adr/single-source-resource-fields.md](adr/single-source-resource-fields.md) for why
the model is defined in Go rather than YAML.
