# ADR: Annotation-based ownership visibility

## Status

Accepted

## Context

The EmELand web UI reaches modelsrv through a BFF (`modelsrv-web-ui-server`) that verifies OIDC tokens. Resources need per-owner visibility: callers see only resources they own (by identity or group membership), while unowned resources are visible only to auditors. Ownership is environment-specific to the OIDC demo setup and should not be a first-class model field.

## Decision

### Ownership storage

Owners are stored in existing per-resource annotations:

- `emeland.io/owner-identities` — comma/space-separated OIDC subject values (identities).
- `emeland.io/owner-groups` — comma/space-separated group ids.

Only Identity and Group principals may be owners (not OrgUnit). Detection lives in `pkg/authz` (`HasOwner`, `OwnerIdentities`, `OwnerGroups`) for reuse by future finding filters.

### Trust boundary (BFF → modelsrv)

OIDC verification stays in the BFF. modelsrv does **not** verify JWTs. On the closed management network, the BFF injects trusted headers after stripping any client-supplied `X-Auth-*`:

| Header | Meaning |
|--------|---------|
| `X-Auth-Subject` | OIDC subject |
| `X-Auth-Groups` | Comma-separated group ids from the token |
| `X-Auth-Auditor` | `true` when the BFF determined the caller is an auditor |

modelsrv enables this with `--trust-auth-headers`. When disabled, no filtering is applied (dev/test default).

BFF implementation is tracked separately: [bff-forward-trusted-identity-headers.md](../tickets/bff-forward-trusted-identity-headers.md) (`modelsrv-web-ui-server` repo).

### Visibility rules

`pkg/authz.Evaluator` applies rules in order:

1. Resource type in `--public-resource-types` → visible to all.
2. Caller is auditor → visible. Auditor if `X-Auth-Auditor=true`, or subject matches `--auditor-identity`, or groups contain `--auditor-group`.
3. Any `VisibilityRule` grants access. Base rule: `OwnershipRule` matches owner annotations against the principal.
4. Otherwise hidden (list omits; get-by-id returns **404**, not 403, to avoid leaking existence).

Scope rules (e.g. K8s cluster owner inheritance) implement `authz.VisibilityRule` in follow-up work.

### Enforcement

Generated read handlers in `internal/oapi/server_handlers_gen.go` filter via `tools/gen/server_handler.tmpl`. New resource types are enforced automatically when added to the generator.

## Future: missing-owner findings

A future `pkg/eventfilter/ownership` filter should call `authz.HasOwner()` on resource upserts and upsert/delete Findings using the same pattern as `pkg/eventfilter/phase0`. Visibility and findings share only the predicate, not evaluator logic.

## Configuration (modelsrv)

```
--trust-auth-headers          Enable header trust and visibility filtering
--auditor-identity            OIDC subject treated as auditor
--auditor-group               Group id treated as auditor
--public-resource-types       Comma-separated types always visible (e.g. ContextType,FindingType)
```

Environment variables: `TRUST_AUTH_HEADERS`, `AUDITOR_IDENTITY`, `AUDITOR_GROUP`, `PUBLIC_RESOURCE_TYPES`.
