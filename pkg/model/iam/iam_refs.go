package iam

import (
	"github.com/google/uuid"
)

// OrgUnitRef references an [OrgUnit] by resolved object and/or id.
type OrgUnitRef struct {
	OrgUnit   OrgUnit
	OrgUnitId uuid.UUID
}

// ResolvedOrgUnit returns the embedded [OrgUnit] when present, or nil.
func (r *OrgUnitRef) ResolvedOrgUnit() OrgUnit {
	if r == nil {
		return nil
	}
	return r.OrgUnit
}

// EffectiveParentOrgUnitID returns the parent id from the embedded object or from [OrgUnitRef.OrgUnitId].
func (r *OrgUnitRef) EffectiveParentOrgUnitID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.OrgUnit != nil {
		return r.OrgUnit.GetOrgUnitId()
	}
	return r.OrgUnitId
}

// IdentityRef references an [Identity] by resolved object and/or id.
type IdentityRef struct {
	Identity   Identity
	IdentityId uuid.UUID
}

// GroupRef references a [Group] by resolved object and/or id.
type GroupRef struct {
	Group   Group
	GroupId uuid.UUID
}
