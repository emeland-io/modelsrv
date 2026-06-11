# Ticket: BFF — forward trusted identity headers to modelsrv

**Repo:** `modelsrv-web-ui-server`  
**Depends on:** modelsrv ownership visibility (`pkg/authz`, `--trust-auth-headers`)  
**Blocks:** End-to-end ownership filtering in the web UI demo

## Summary

modelsrv now enforces read visibility based on owner annotations when `--trust-auth-headers` is enabled. It does **not** verify OIDC tokens; it expects the BFF to inject trusted identity headers after authentication. This ticket implements that forwarding in `modelsrv-web-ui-server`.

## Background

Deployment flow: OIDC IdP (Dex) → `modelsrv-web-ui-server` (BFF) → modelsrv.

The BFF already:

- Verifies OIDC JWTs (`internal/auth/auth.go`) or uses dev `StubMiddleware`
- Extracts `Claims{Subject, Groups}` from the token
- Gates API access to authenticated users (`internal/authz/authz.go`)
- Determines auditors via `--auditor-group` / `AUDITOR_GROUP_ID`

modelsrv list endpoints return only id/name/reference (no owner annotations), so the BFF cannot filter responses itself. Filtering must happen in modelsrv, which reads owner data from per-resource annotations.

## Required changes (modelsrv-web-ui-server)

### 1. Inject trusted headers on proxied API requests

After auth middleware runs, set these headers on the outgoing request to modelsrv (strip any client-supplied values first):

| Header | Value |
|--------|--------|
| `X-Auth-Subject` | `claims.Subject` (OIDC `sub`) |
| `X-Auth-Groups` | Comma-separated `claims.Groups` |
| `X-Auth-Auditor` | `true` when caller is in the configured auditor group; omit otherwise |

**Security:** Remove all inbound `X-Auth-*` headers from the client request before setting the above (anti-spoofing). modelsrv trusts these only on the closed mgmt network.

Natural implementation: custom `httputil.ReverseProxy.Director` wrapping `NewSingleHostReverseProxy`, or a small `internal/proxy` package.

Header names must match modelsrv `pkg/authz` constants:

- `X-Auth-Subject`
- `X-Auth-Groups`
- `X-Auth-Auditor`

### 2. Simplify `internal/authz` middleware

Remove the obsolete TODO “Implement owner-context filtering on proxy responses”. The middleware should only ensure the caller is authenticated; visibility is enforced downstream in modelsrv.

Keep `--auditor-group` / `AUDITOR_GROUP_ID` in the BFF so it can set `X-Auth-Auditor` correctly.

### 3. Tests

- Proxy forwards `X-Auth-Subject`, `X-Auth-Groups`, `X-Auth-Auditor` for an authenticated auditor.
- Client-supplied `X-Auth-Subject` / `X-Auth-Auditor` are stripped and replaced with values from claims.
- Existing auth rejection tests still pass.

## modelsrv configuration (for integration testing)

Run modelsrv with:

```bash
modelsrv server --trust-auth-headers \
  --auditor-group=<audit-group-uuid> \
  --public-resource-types=ContextType,FindingType
```

Set resource owners via annotations:

- `emeland.io/owner-identities` — OIDC subject(s)
- `emeland.io/owner-groups` — group id(s)

See [adr/ownership-visibility.md](../adr/ownership-visibility.md).

## Acceptance criteria

- [ ] Authenticated API proxy requests to modelsrv include correct `X-Auth-*` headers.
- [ ] Spoofed client `X-Auth-*` headers never reach modelsrv.
- [ ] Auditor users get `X-Auth-Auditor: true` when their groups include `--auditor-group`.
- [ ] With modelsrv `--trust-auth-headers`, owners see their resources; non-auditors do not see unowned resources; auditors see all.
- [ ] Tests cover header injection and spoof stripping.

## Out of scope

- modelsrv changes (already implemented in `modelsrv` repo).
- Write authorization on modelsrv.
- Missing-owner Finding generation.

## References

- modelsrv ADR: [docs/adr/ownership-visibility.md](../adr/ownership-visibility.md)
- BFF auth: `modelsrv-web-ui-server/internal/auth/auth.go`
- BFF authz: `modelsrv-web-ui-server/internal/authz/authz.go`
- Demo deploy: `modelsrv-web-ui-server/deploy/demo.yaml`
