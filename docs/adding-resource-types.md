# Adding resource types

Resource types are defined as annotated Go structs and the rest is generated. The
authoritative guide for the generator, including the marker/tag reference and the exact
generation order, is [code-generation.md](code-generation.md). This page is a short
orientation.

## Add or change a field

Edit the type's struct in `tools/gen/modeldefs/model_defs.go` (add the field with a doc
comment and the appropriate `emeland:` tag), then run `make gen && make generate`. For a
relational field or a type with custom methods, also edit its entry in
`tools/gen/wiring.go`. See [code-generation.md](code-generation.md#how-to-add-a-scalar-field-to-an-existing-type).

## Add a new resource type

Summary of the touchpoints (details in
[code-generation.md](code-generation.md#how-to-add-a-new-resource-type)):

1. Add an annotated struct to `tools/gen/modeldefs/model_defs.go` (and a `wiring.go` entry
   if it has references, custom methods, or non-trivial test setup).
1. Hand-author its endpoints and any new shared schemas in
   `api/openapi/EmergingEnterpriseLandscape-0.1.0-oapi-3.0.3.yaml`, and add it to the
   `ResourceRef` / `ResourceView` enums there.
1. Register it in the non-generated touchpoints: `pkg/events/events.go`,
   `pkg/ingress/document.go` (+ the relevant `pkg/ingress/apply_*.go` and CSV id column),
   `pkg/model/structure.go` (Model sub-interface, `modelData` maps, FK validation),
   `pkg/model/common/errors.go`, `pkg/model/resource_types.go`, and the enrichment tables
   in `tools/gen` (`wire_meta.go`; `domain_meta.go` if a new sub-package; `alltypes.go`
   order).
1. Define any missing Ref types in the resource's `pkg/model` sub-package.
1. Run `make gen && make generate`, then add hand-written tests for domain rules (FK
   validation, uniqueness) and YAML ingress (see `pkg/filesensor/filesensor_test.go`).

Do not hand-edit generated files (`*_gen.go`, `*.gen.go`, `mock_*.go`, or the per-resource
schema blocks in the OpenAPI file); see the generated-outputs list in
[code-generation.md](code-generation.md#what-is-hand-authored-vs-generated).

## Ownership visibility

When `--trust-auth-headers` is enabled, list and get-by-id handlers generated from
`tools/gen/server_handler.tmpl` automatically enforce ownership visibility for new resource
types (unless listed in `skipAuthzByName`). Owners are set via annotations
(`emeland.io/owner-identities`, `emeland.io/owner-groups`). See
[adr/ownership-visibility.md](adr/ownership-visibility.md).
