package capacity

import "github.com/google/uuid"

// CapacityResourceTypeRef references a [CapacityResourceType] by resolved object and/or id.
type CapacityResourceTypeRef struct {
	CapacityResourceType   CapacityResourceType
	CapacityResourceTypeId uuid.UUID
}

// ResolvedCapacityResourceType returns the embedded [CapacityResourceType] when present, or nil.
func (r *CapacityResourceTypeRef) ResolvedCapacityResourceType() CapacityResourceType {
	if r == nil {
		return nil
	}
	return r.CapacityResourceType
}

// EffectiveCapacityResourceTypeID returns the type id from the embedded object or from
// [CapacityResourceTypeRef.CapacityResourceTypeId].
func (r *CapacityResourceTypeRef) EffectiveCapacityResourceTypeID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.CapacityResourceType != nil {
		return r.CapacityResourceType.GetCapacityResourceTypeId()
	}
	return r.CapacityResourceTypeId
}
