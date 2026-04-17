package finding

import "github.com/google/uuid"

// FindingTypeRef references a [FindingType] by resolved object and/or id.
type FindingTypeRef struct {
	FindingType   FindingType
	FindingTypeId uuid.UUID
}

// ResolvedFindingType returns the embedded [FindingType] when present, or nil.
func (r *FindingTypeRef) ResolvedFindingType() FindingType {
	if r == nil {
		return nil
	}
	return r.FindingType
}

// EffectiveFindingTypeID returns the type id from the embedded object or from
// [FindingTypeRef.FindingTypeId].
func (r *FindingTypeRef) EffectiveFindingTypeID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.FindingType != nil {
		return r.FindingType.GetFindingTypeId()
	}
	return r.FindingTypeId
}
