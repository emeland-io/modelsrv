package authz

import (
	"context"
	"strings"
)

type ctxKey struct{}

// Principal holds the authenticated caller identity forwarded by the BFF.
type Principal struct {
	Subject       string
	Groups        []string
	AuditorHeader bool
}

// InGroup reports whether the principal belongs to the given group id.
func (p Principal) InGroup(groupID string) bool {
	if groupID == "" {
		return false
	}
	for _, g := range p.Groups {
		if g == groupID {
			return true
		}
	}
	return false
}

// WithPrincipal stores a Principal in the context.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// PrincipalFromCtx retrieves the Principal from the context, or a zero Principal if absent.
func PrincipalFromCtx(ctx context.Context) Principal {
	p, _ := ctx.Value(ctxKey{}).(Principal)
	return p
}

// Header names injected by the BFF (modelsrv trusts these on the closed mgmt network).
const (
	HeaderAuthSubject = "X-Auth-Subject"
	HeaderAuthGroups  = "X-Auth-Groups"
	HeaderAuthAuditor = "X-Auth-Auditor"
)

// ParseGroups splits a comma-separated group list header value.
func ParseGroups(header string) []string {
	if header == "" {
		return nil
	}
	var groups []string
	for _, part := range strings.Split(header, ",") {
		if s := strings.TrimSpace(part); s != "" {
			groups = append(groups, s)
		}
	}
	return groups
}
