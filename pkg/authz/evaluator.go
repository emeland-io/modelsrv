package authz

import (
	"go.emeland.io/modelsrv/pkg/events"
)

// Config holds authorization settings for the visibility evaluator.
type Config struct {
	AuditorIdentity string
	AuditorGroup    string
	PublicTypes     map[events.ResourceType]bool
}

// Evaluator decides whether a principal may read a resource.
type Evaluator struct {
	cfg   Config
	rules []VisibilityRule
}

// NewEvaluator constructs an Evaluator with the base OwnershipRule.
func NewEvaluator(cfg Config) *Evaluator {
	return &Evaluator{
		cfg:   cfg,
		rules: []VisibilityRule{OwnershipRule{}},
	}
}

// CanSee reports whether principal p may read resource r of type rt.
func (e *Evaluator) CanSee(p Principal, rt events.ResourceType, r Ownable) bool {
	if e.cfg.PublicTypes[rt] {
		return true
	}
	if e.isAuditor(p) {
		return true
	}
	for _, rule := range e.rules {
		if rule.Grants(p, rt, r) {
			return true
		}
	}
	return false
}

// FilterVisible returns only items visible to principal p.
func FilterVisible[T Ownable](e *Evaluator, p Principal, rt events.ResourceType, items []T) []T {
	if e == nil {
		return items
	}
	out := make([]T, 0, len(items))
	for _, item := range items {
		if e.CanSee(p, rt, item) {
			out = append(out, item)
		}
	}
	return out
}

func (e *Evaluator) isAuditor(p Principal) bool {
	if p.AuditorHeader {
		return true
	}
	if e.cfg.AuditorIdentity != "" && p.Subject == e.cfg.AuditorIdentity {
		return true
	}
	if e.cfg.AuditorGroup != "" && p.InGroup(e.cfg.AuditorGroup) {
		return true
	}
	return false
}

// ParsePublicResourceTypes parses a comma-separated list of resource type names.
func ParsePublicResourceTypes(s string) map[events.ResourceType]bool {
	out := make(map[events.ResourceType]bool)
	for _, part := range parseList(s) {
		rt := events.ParseResourceType(part)
		if rt != events.UnknownResourceType {
			out[rt] = true
		}
	}
	return out
}
