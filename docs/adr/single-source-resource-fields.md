# ADR: Single source of truth for resource fields

## Status

Accepted.

## Context

A resource field feeds two consumers: the Go domain layer (interfaces, FK validation,
converters, mocks, client, replication) and the OpenAPI wire layer (DTOs and HTTP
server/client via oapi-codegen). Originally both were authored by hand, a large
`[]TypeSpec` literal plus lookup maps in `tools/gen`, and the OpenAPI YAML, so adding one
field meant editing two places and keeping them in sync. The project is young and not in
production, so a foundational refactor was acceptable.

## Decision

Define each resource type once as an annotated Go struct; generate everything else from it.

- The structs in `tools/gen/modeldefs/model_defs.go` are the single source of truth: field
  types and `emeland:` tags describe the wire schema, doc comments become descriptions.
- Freeform, code-like wiring that does not fit struct tags (custom method signatures,
  relational ref links, per-type test setup, domain-field overrides) lives in a small,
  type-checked Go supplement, `tools/gen/wiring.go`, keyed by type name (**Path B**).
- The generator emits each resource's OpenAPI schema block and merges it in place into the
  hand-authored spec (endpoints and shared schemas stay hand-authored), then oapi-codegen
  and the Go-side generators run.

This follows the controller-gen / kubebuilder model (Go struct + markers generate the
OpenAPI/CRD schema). Fidelity is verified by **semantic equivalence** (the generated schema
parses to the same OpenAPI model as the committed spec) rather than byte-for-byte, and the
committed generated files plus `make gen` idempotency guard against drift.

How to use it: see [../code-generation.md](../code-generation.md).

## Rationale

- **Go, not YAML, as the source.** The hard, type-sensitive logic (ref resolution, FK
  validation, converters) lives in Go; a Go struct can describe the schema far more easily
  than YAML could describe the Go. YAML has no types or compiler and would become a weak
  DSL.
- **A `wiring.go` supplement, not everything-in-tags (Path B).** Much of the old literal is
  freeform Go (test-setup code, method signatures, relational wiring). Cramming that into
  string tags would be less readable and uncheckable. Keeping it as type-checked Go, while
  the structs own the plain data and schema, reaches "a field lives in one place" without
  hiding code in strings.

## Alternatives considered and rejected

- **YAML authoritative (generate Go from YAML).** Rejected: pushes type-sensitive Go
  concerns into an untyped DSL.
- **Everything in struct tags/markers (Path A).** Rejected: forces freeform Go
  (TestSetup, CustomMethods, a multi-field ref DSL) into uncheckable string tags.
- **`TypeSpec` literal enriched with description strings.** Reaches single-sourcing but
  keeps scattered metadata and duplicated string literals; less readable than structs with
  doc comments.
- **Per-type YAML files.** Smaller files, but still two sources unless Go is generated from
  YAML (see the first alternative).
