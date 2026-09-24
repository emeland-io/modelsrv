package parameter

import "github.com/google/uuid"

// ParameterRef references a [Parameter] by resolved object and/or id.
type ParameterRef struct {
	Parameter   Parameter
	ParameterId uuid.UUID
}

// ResolvedParameter returns the embedded [Parameter] when present, or nil.
func (r *ParameterRef) ResolvedParameter() Parameter {
	if r == nil {
		return nil
	}
	return r.Parameter
}

// EffectiveParameterID returns the parameter id from the embedded object or from
// [ParameterRef.ParameterId].
func (r *ParameterRef) EffectiveParameterID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.Parameter != nil {
		return r.Parameter.GetParameterId()
	}
	return r.ParameterId
}
