package parameter

import "github.com/google/uuid"

// ValidValueRef references a [ValidValue] by resolved object and/or id.
type ValidValueRef struct {
	ValidValue   ValidValue
	ValidValueId uuid.UUID
}

// ResolvedValidValue returns the embedded [ValidValue] when present, or nil.
func (r *ValidValueRef) ResolvedValidValue() ValidValue {
	if r == nil {
		return nil
	}
	return r.ValidValue
}

// EffectiveValidValueID returns the valid value id from the embedded object or from
// [ValidValueRef.ValidValueId].
func (r *ValidValueRef) EffectiveValidValueID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.ValidValue != nil {
		return r.ValidValue.GetValidValueId()
	}
	return r.ValidValueId
}
