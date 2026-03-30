package context

import "github.com/google/uuid"

// ContextRef references a [Context] by resolved object and/or id.
type ContextRef struct {
	Context   Context
	ContextId uuid.UUID
}

// ResolvedContext returns the embedded [Context] when present, or nil.
func (r *ContextRef) ResolvedContext() Context {
	if r == nil {
		return nil
	}
	return r.Context
}

// EffectiveParentContextID returns the parent id from the embedded object or from [ContextRef.ContextId].
func (r *ContextRef) EffectiveParentContextID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.Context != nil {
		return r.Context.GetContextId()
	}
	return r.ContextId
}
