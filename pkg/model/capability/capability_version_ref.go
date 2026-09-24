package capability

import (
	"github.com/google/uuid"
)

// CapabilityVersionRef is a reference to a CapabilityVersion resource.
type CapabilityVersionRef struct {
	CapabilityVersion   CapabilityVersion
	CapabilityVersionId uuid.UUID `json:"capabilityVersionId"`
}

// ResolvedCapabilityVersion returns the embedded [CapabilityVersion] when present, or nil.
func (r *CapabilityVersionRef) ResolvedCapabilityVersion() CapabilityVersion {
	if r == nil {
		return nil
	}
	return r.CapabilityVersion
}

// EffectiveCapabilityVersionID returns the version id from the embedded object or from
// [CapabilityVersionRef.CapabilityVersionId].
func (r *CapabilityVersionRef) EffectiveCapabilityVersionID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.CapabilityVersion != nil {
		return r.CapabilityVersion.GetCapabilityVersionId()
	}
	return r.CapabilityVersionId
}
