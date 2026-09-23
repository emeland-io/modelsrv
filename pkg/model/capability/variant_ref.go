package capability

import "github.com/google/uuid"

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
