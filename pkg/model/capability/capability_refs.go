package capability

import (
	"github.com/google/uuid"
)

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

// CapabilityVersionRef references a [CapabilityVersion] by resolved object and/or id.
type CapabilityVersionRef struct {
	CapabilityVersion   CapabilityVersion
	CapabilityVersionId uuid.UUID
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

// VariantRef references a [Variant] by resolved object and/or id.
type VariantRef struct {
	Variant   Variant
	VariantId uuid.UUID
}

// ResolvedVariant returns the embedded [Variant] when present, or nil.
func (r *VariantRef) ResolvedVariant() Variant {
	if r == nil {
		return nil
	}
	return r.Variant
}

// EffectiveVariantID returns the variant id from the embedded object or from
// [VariantRef.VariantId].
func (r *VariantRef) EffectiveVariantID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.Variant != nil {
		return r.Variant.GetVariantId()
	}
	return r.VariantId
}
