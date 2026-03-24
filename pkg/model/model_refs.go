package model

import "github.com/google/uuid"

// ApiRef references an [API] by resolved object and/or id.
// ApiRef may embed an [EntityVersion] when the reference is versioned (e.g. contract / semantic binding).
type ApiRef struct {
	API    API
	ApiID  uuid.UUID
	ApiRef *EntityVersion
}

// ContextRef references a [Context] by resolved object and/or id.
type ContextRef struct {
	Context   Context
	ContextId uuid.UUID
}

// NodeTypeRef references a [NodeType] by resolved object and/or id.
type NodeTypeRef struct {
	NodeType   NodeType
	NodeTypeId uuid.UUID
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

// ResolvedNodeType returns the embedded [NodeType] when present, or nil.
func (r *NodeTypeRef) ResolvedNodeType() NodeType {
	if r == nil {
		return nil
	}
	return r.NodeType
}
