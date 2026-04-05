package iam

import (
	"github.com/google/uuid"
)

// OrgUnitRef references an [OrgUnit] by resolved object and/or id.
type OrgUnitRef struct {
	OrgUnit   OrgUnit
	OrgUnitId uuid.UUID
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
