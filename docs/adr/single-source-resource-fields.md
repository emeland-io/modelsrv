# ADR: Single source of truth for resource fields

## Status

Proposed

## Context

A resource field feeds two consumers: the Go domain layer (interface, impl, FK
validation, converters, mocks, client, replication) and the OpenAPI wire layer (DTO
structs, HTTP server/client via oapi-codegen). Today both are authoritative and edited
by hand:

- `tools/gen/specs.go` holds a large `[]TypeSpec` data literal (fields plus wiring), with
  additional per-type metadata spread across lookup maps in `tools/gen/wire_meta.go` and
  `tools/gen/domain_meta.go`.
- `api/openapi/EmergingEnterpriseLandscape-0.1.0-oapi-3.0.3.yaml` (~2565 lines) holds the
  schemas and endpoints.

Adding one scalar field means editing `specs.go` and the YAML; forgetting the YAML breaks
the generated converters. A field's name, type, description, wire facts, and behavior are
scattered across a data literal, several maps, and a YAML block.

The code is young and not in production, so a foundational refactor is acceptable.

## Decision

**Go is the single source of truth, expressed as real annotated structs.** Each resource
type is declared once as a Go struct whose fields carry the domain type; doc comments carry
the OpenAPI `description`; struct tags and `+emeland:` markers carry wire facts and wiring.
Everything else is generated from these structs.

```go
// NodeType classifies nodes in the landscape.
//
// +emeland:resource=NodeType
// +emeland:list=/landscape/nodeTypes
type NodeType struct {
    // Stable identifier, assigned at creation.
    NodeTypeID  uuid.UUID `emeland:"id"`
    // Human-readable name.
    DisplayName string
    // A brief description of the node type's purpose.
    Description string `emeland:"optional"`
    // Extended metadata.
    Annotations annotations.Annotations `emeland:"annotations"`
}
```

A loader parses these structs with `go/ast` + `go/doc` and populates the existing
`TypeSpec` shape, so the current templates keep working. From `TypeSpec` the generator
produces, as it does now: domain code, mocks, client, replication, converters, and, newly,
the OpenAPI **schema fragment** for each resource.

This follows the proven controller-gen / kubebuilder model (Go struct + markers generate
the OpenAPI/CRD schema).

### Ownership split

- **Generated from Go structs:** the `components/schemas/<Type>` fragment for each resource
  type (its own scalar and simple-ref properties, formats, enums, required set,
  descriptions).
- **Hand-authored in `api/openapi/base.yaml`:** `info`, `tags`, `servers`, all `paths`
  (endpoints), and all shared/non-resource schemas (`InstanceList`, `ResourceRef`, `*Ref`,
  `*View`, `Event`, `Version`, `Annotation`, `ProductionVersion`, `ErrorString`, enums).

A merge step combines generated fragments with `base.yaml` into the final spec that
oapi-codegen consumes.

### Marker/tag vocabulary (initial)

- `emeland:"id"` — primary identifier field.
- `emeland:"optional"` — property is not required; DTO field is a pointer.
- `emeland:"annotations"` — annotations bag (not a wire scalar).
- `emeland:"ref=<Schema>"` — property is a `$ref` to another schema.
- `emeland:"format=uuid"` — OpenAPI `format`.
- `emeland:"enum=A|B|C"` — enum values.
- `+emeland:resource=<Name>` / `+emeland:list=<path>` — type-level resource name and REST
  list path.
- `+emeland:convert=manual` — escape hatch: converter stays hand-written in
  `internal/oapi/convert_special.go` (replaces today's `skipConvertByName`).

### Generation order

1. Loader parses resource structs → `TypeSpec` values.
2. Generator emits schema fragments and merges with `base.yaml` → final OpenAPI file.
3. oapi-codegen runs on the final file (DTOs, server/client).
4. Generator runs Go/DTO-dependent passes (domain, converters, replication, handlers).

Schema generation reads only struct data, so no bootstrapping cycle with oapi-codegen.

## Consequences

Positive:
- One readable, compiler-checked, refactorable place per resource: field, type, docs, and
  wiring together. Adding a scalar field is one line (plus a doc comment) in one Go file.
- The scattered `wire_meta.go` / `domain_meta.go` maps collapse into struct tags/markers.
- Descriptions are ordinary Go docs and also document the generated code.

Risks and mitigations:
- **Fidelity.** Generated OpenAPI YAML and generated Go must be byte-identical to today's
  for unchanged types. Mitigation: golden tests assert equality before the old `TypeSpec`
  literal is removed; migrate type by type under that guard.
- **Comment/marker fragility.** Parsing is less rigid than data literals. Mitigation:
  strict marker parsing that fails loudly with the offending type/field named.
- **Expressing complex wiring.** Ref links, FK validation, and hand-written converters must
  be expressible as markers or kept behind `+emeland:convert=manual`. Keep the escape hatch
  rather than forcing everything into markers.

## Rollout plan (each step independently reviewable)

1. **Fidelity harness (done).** `tools/gen/openapi_fidelity_test.go` asserts specs.go scalar
   fields match the committed OpenAPI schemas (name/type/optionality). Guards drift during
   migration.
2. **Struct loader.** Add the `go/ast`-based loader and marker vocabulary; parse a pilot
   type into a `TypeSpec` and assert it equals the current hand-written `TypeSpec` for that
   type. No deletion yet.
3. **Schema emitter + base split.** Extract `base.yaml`; generate resource schema fragments;
   merge to a file byte-identical to today's committed YAML (golden test). No field moves.
4. **Flip the pipeline.** Point oapi-codegen at the generated file; rework Makefile /
   `go:generate` ordering; update `docs/adding-resource-types.md`.
5. **Migrate all types** to structs, remove the `[]TypeSpec` literal and the metadata maps,
   under the golden tests.

## Alternatives considered

- **YAML authoritative (generate Go from YAML).** Rejected: the hard, type-sensitive logic
  lives in Go (ref resolution, FK validation, converters); YAML has no types or compiler and
  would become a weak DSL describing Go.
- **`TypeSpec` data literal enriched with description strings.** Reaches single-sourcing but
  keeps the scattered-metadata problem and duplicated string literals; less readable than
  structs with doc comments.
- **Per-type YAML files.** Smaller files, but still two sources (Go + YAML) unless Go is
  generated from YAML, which the first alternative rejects.
