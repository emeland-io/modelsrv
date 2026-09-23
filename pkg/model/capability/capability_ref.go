package capability

import "github.com/google/uuid"

// CapabilityRef references a [Capability] by resolved object and/or id.
type CapabilityRef struct {
	Capability   Capability
	CapabilityId uuid.UUID
}

// ResolvedCapability returns the embedded [Capability] when present, or nil.
func (r *CapabilityRef) ResolvedCapability() Capability {
	if r == nil {
		return nil
	}
	return r.Capability
}

// EffectiveCapabilityID returns the capability id from the embedded object or from
// [CapabilityRef.CapabilityId].
func (r *CapabilityRef) EffectiveCapabilityID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.Capability != nil {
		return r.Capability.GetCapabilityId()
	}
	return r.CapabilityId
}
