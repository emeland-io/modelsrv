package iam

import (
	"github.com/google/uuid"
)

type OrgUnitModel interface {
	AddOrgUnit(OrgUnit) error
	DeleteOrgUnit(uuid.UUID) error
	GetOrgUnits() ([]OrgUnit, error)
	GetOrgUnitById(uuid.UUID) OrgUnit
}

type GroupModel interface {
	AddGroup(Group) error
	DeleteGroup(uuid.UUID) error
	GetGroups() ([]Group, error)
	GetGroupById(uuid.UUID) Group
}

type IdentityModel interface {
	AddIdentity(Identity) error
	DeleteIdentity(uuid.UUID) error
	GetIdentities() ([]Identity, error)
	GetIdentityById(uuid.UUID) Identity
}
