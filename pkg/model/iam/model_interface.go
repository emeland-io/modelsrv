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

type PermissionSpecModel interface {
	AddPermissionSpec(PermissionSpec) error
	DeletePermissionSpec(uuid.UUID) error
	GetPermissionSpecs() ([]PermissionSpec, error)
	GetPermissionSpecById(uuid.UUID) PermissionSpec
}

type RoleSpecModel interface {
	AddRoleSpec(RoleSpec) error
	DeleteRoleSpec(uuid.UUID) error
	GetRoleSpecs() ([]RoleSpec, error)
	GetRoleSpecById(uuid.UUID) RoleSpec
}

type PermissionModel interface {
	AddPermission(Permission) error
	DeletePermission(uuid.UUID) error
	GetPermissions() ([]Permission, error)
	GetPermissionById(uuid.UUID) Permission
}

type RoleModel interface {
	AddRole(Role) error
	DeleteRole(uuid.UUID) error
	GetRoles() ([]Role, error)
	GetRoleById(uuid.UUID) Role
}

type BindingModel interface {
	AddBinding(Binding) error
	DeleteBinding(uuid.UUID) error
	GetBindings() ([]Binding, error)
	GetBindingById(uuid.UUID) Binding
}
