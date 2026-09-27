# Adding Resource Types

As may be obvious in the multiple empty pages in the EmELand book, that there are still a number of resource types missing from the model.

This document should help you when adding an additional resource type.

**Ordering constraint:** regenerate OpenAPI (`go generate ./internal/oapi/...` or `make gen`) **before** running `tools/gen` (`make generate` / `go generate ./pkg/model/...`). The `tools/gen` templates reference oapi-codegen DTO type names that must already exist.

## Adding a scalar field to an existing type

Adding a plain scalar field (`string`, `bool`, or a numeric type like `int`/`int64`/`float64`) to a resource type that is **not** in `skipConvertByName` needs only two edits:

1. Add the property to that type's schema in `api/openapi/EmergingEnterpriseLandscape-0.1.0-oapi-3.0.3.yaml`. Whether you list it under `required` decides the DTO shape:
   - listed in `required` → oapi-codegen emits a value field (e.g. `Owner string`).
   - not required → oapi-codegen emits a pointer field (e.g. `Owner *string`).
1. Add one line to the type's `Fields` in `tools/gen/specs.go`, for example `{Name: "Owner", Type: "string"}` for a required property, or `{Name: "Owner", Type: "string", Optional: true}` for an optional one. Set `Optional: true` **iff** the YAML property is not required, so the generated converters match the pointer/value shape of the DTO.

Then regenerate in the usual order (`make gen` then `make generate`). The domain interface/impl accessors, mocks, and the `FromDto`/`ToDto` converters are all generated. Optional fields get nil-safe conversion automatically (dereferenced on the way in; emitted only when non-zero on the way out).

Caveats:
- Types in `skipConvertByName` (e.g. `Node`, `System`, `Finding`) keep hand-written converters in `internal/oapi/convert_special.go`; a new scalar field on those must be mapped there by hand as well.
- If the field should be settable from YAML/CSV/JSON ingress, wire it into the relevant `pkg/ingress/apply_*.go` path (and the CSV column mapping in `pkg/ingress/csv.go`).
- Reference fields (pointing at another resource) are not scalars; they need `TypeRefLink`/`RefByRef` wiring as described in the resource-type checklist below.

## Checklist

1. Update the resource type enum in **both** `ResourceRef` schemas of the OpenAPI spec in `api/openapi/EmergingEnterpriseLandscape-0.1.0-oapi-3.0.3.yaml`.
1. Add schemas and list/get-by-id endpoints for the new type to the same OAPI file (tag with the matching phase tag, e.g. `p6_observability`).
1. Run `go generate ./internal/oapi/...` so `server.gen.go` / `client.gen.go` pick up the new DTOs and routes.
1. Add the type to the list of resource types in `pkg/events/events.go` (constant + `resourceTypeValues` map).
1. Add the type to the `documentKinds` map in `pkg/ingress/document.go`.
1. Add the id-column mapping in `pkg/ingress/csv.go`.
1. Implement declarative apply in `pkg/ingress/apply_*.go` and wire the switch in `pkg/ingress/apply.go`.
1. Add the package dir to `tools/gen/domain_meta.go` (if new) and wire-meta maps in `tools/gen/wire_meta.go` (`wireKindToEventsResource`, `restListPathByName`, `serverResourceLabelByName`; `skipConvertByName` when refs need hand-written DTO converters; `skipAuthzByName` for public vocabulary types).
1. Define the new resource type in `tools/gen/specs.go`. Include fields, any required Ref structures (`TypeRefLink` / `RefByRefs` / `CustomMethods`), client/OAPI method names, and `TestSetup` (primary resource variable **first** — used by generated replication tests).
1. Define any missing Ref types in the sub-package of `pkg/model` in which you have just added the new resource type (e.g. `metric_ref.go`).
1. Create a Model sub-interface for the new resource type and add it to the full `Model` in `pkg/model/structure.go`.
1. Add the Id-to-resource maps to the `modelData` structure and required initialization in `NewModel` in `pkg/model/structure.go`.
1. Implement the missing methods for the compound `Model` interface (often `pkg/model/structure_<domain>.go`), including FK validation when the type references another resource.
1. Add error codes for missing resources to `pkg/model/common/errors.go`.
1. Register a count entry in `pkg/model/resource_types.go`.
1. If the type is in `skipConvertByName`, add `FromDto` / `ToDto` in `internal/oapi/convert_special.go`.
1. Run `make generate` (runs `tools/gen` via `pkg/model/structure.go` and mockgen).
1. Add hand-written tests for domain rules (FK validation, uniqueness) and YAML ingress (see `pkg/filesensor/filesensor_test.go`).

Generated outputs (do not hand-edit): `pkg/model/<pkg>/*_gen.go`, `pkg/model/handlers_*_gen.go`, `pkg/client/client_gen.go` (+ tests), `internal/oapi/{server_handlers,replication_*,convert_*}_gen.go`, `pkg/mocks/mock_model.go`.

## Ownership visibility

When `--trust-auth-headers` is enabled, list and get-by-id handlers generated from `tools/gen/server_handler.tmpl` automatically enforce ownership visibility for new resource types (unless listed in `skipAuthzByName`). Owners are set via annotations (`emeland.io/owner-identities`, `emeland.io/owner-groups`). See [adr/ownership-visibility.md](adr/ownership-visibility.md).
