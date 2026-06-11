package authz

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/annotations"
)

// Ownable is implemented by model resources that carry ownership annotations.
type Ownable interface {
	GetResourceId() uuid.UUID
	GetAnnotations() annotations.Annotations
}

// VisibilityRule grants read access for a resource under specific conditions.
// Scope rules (e.g. K8s cluster owner inheritance) implement this interface.
type VisibilityRule interface {
	Name() string
	Grants(p Principal, rt events.ResourceType, r Ownable) bool
}

// OwnershipRule grants access when the principal matches owner annotations.
type OwnershipRule struct{}

func (OwnershipRule) Name() string { return "ownership" }

// Grants reports whether the principal matches this resource's owner annotations.
// rt is unused: ownership is read only from annotations and does not vary by resource
// type. The parameter is required by VisibilityRule so future scope rules (e.g. type-
// or context-specific inheritance) can use rt without changing the evaluator loop.
func (OwnershipRule) Grants(p Principal, _ events.ResourceType, r Ownable) bool {
	ann := r.GetAnnotations()
	for _, id := range OwnerIdentities(ann) {
		if id == p.Subject {
			return true
		}
	}
	for _, g := range OwnerGroups(ann) {
		if p.InGroup(g) {
			return true
		}
	}
	return false
}
