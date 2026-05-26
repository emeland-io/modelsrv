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

// ResolvedIdentity returns the embedded [Identity] when present, or nil.
func (r *IdentityRef) ResolvedIdentity() Identity {
	if r == nil {
		return nil
	}
	return r.Identity
}

// EffectiveIdentityID returns the identity id from the embedded object or from [IdentityRef.IdentityId].
func (r *IdentityRef) EffectiveIdentityID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.Identity != nil {
		return r.Identity.GetIdentityId()
	}
	return r.IdentityId
}

// GroupRef references a [Group] by resolved object and/or id.
type GroupRef struct {
	Group   Group
	GroupId uuid.UUID
}

// ResolvedGroup returns the embedded [Group] when present, or nil.
func (r *GroupRef) ResolvedGroup() Group {
	if r == nil {
		return nil
	}
	return r.Group
}

// EffectiveGroupID returns the group id from the embedded object or from [GroupRef.GroupId].
func (r *GroupRef) EffectiveGroupID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.Group != nil {
		return r.Group.GetGroupId()
	}
	return r.GroupId
}

// PermissionSpecRef references a [PermissionSpec] by resolved object and/or id.
type PermissionSpecRef struct {
	PermissionSpec   PermissionSpec
	PermissionSpecId uuid.UUID
}

// ResolvedPermissionSpec returns the embedded [PermissionSpec] when present, or nil.
func (r *PermissionSpecRef) ResolvedPermissionSpec() PermissionSpec {
	if r == nil {
		return nil
	}
	return r.PermissionSpec
}

// EffectivePermissionSpecID returns the spec id from the embedded object or from [PermissionSpecRef.PermissionSpecId].
func (r *PermissionSpecRef) EffectivePermissionSpecID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.PermissionSpec != nil {
		return r.PermissionSpec.GetPermissionSpecId()
	}
	return r.PermissionSpecId
}

// RoleSpecRef references a [RoleSpec] by resolved object and/or id.
type RoleSpecRef struct {
	RoleSpec   RoleSpec
	RoleSpecId uuid.UUID
}

// ResolvedRoleSpec returns the embedded [RoleSpec] when present, or nil.
func (r *RoleSpecRef) ResolvedRoleSpec() RoleSpec {
	if r == nil {
		return nil
	}
	return r.RoleSpec
}

// EffectiveRoleSpecID returns the spec id from the embedded object or from [RoleSpecRef.RoleSpecId].
func (r *RoleSpecRef) EffectiveRoleSpecID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.RoleSpec != nil {
		return r.RoleSpec.GetRoleSpecId()
	}
	return r.RoleSpecId
}

// PermissionRef references a [Permission] by resolved object and/or id.
type PermissionRef struct {
	Permission   Permission
	PermissionId uuid.UUID
}

// ResolvedPermission returns the embedded [Permission] when present, or nil.
func (r *PermissionRef) ResolvedPermission() Permission {
	if r == nil {
		return nil
	}
	return r.Permission
}

// EffectivePermissionID returns the permission id from the embedded object or from [PermissionRef.PermissionId].
func (r *PermissionRef) EffectivePermissionID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.Permission != nil {
		return r.Permission.GetPermissionId()
	}
	return r.PermissionId
}

// RoleRef references a [Role] by resolved object and/or id.
type RoleRef struct {
	Role   Role
	RoleId uuid.UUID
}

// ResolvedRole returns the embedded [Role] when present, or nil.
func (r *RoleRef) ResolvedRole() Role {
	if r == nil {
		return nil
	}
	return r.Role
}

// EffectiveRoleID returns the role id from the embedded object or from [RoleRef.RoleId].
func (r *RoleRef) EffectiveRoleID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.Role != nil {
		return r.Role.GetRoleId()
	}
	return r.RoleId
}

// SubjectKind distinguishes which principal a [SubjectRef] references.
type SubjectKind int

const (
	SubjectNone SubjectKind = iota
	SubjectKindGroup
	SubjectKindIdentity
)

// SubjectRef references either a group or an identity as the subject of a [Binding].
type SubjectRef struct {
	Group    *GroupRef
	Identity *IdentityRef
}

// EffectiveKind returns whether the subject refers to a group, an identity, or is unset / ambiguous.
func (r *SubjectRef) EffectiveKind() SubjectKind {
	if r == nil {
		return SubjectNone
	}
	g := r.Group != nil && r.Group.EffectiveGroupID() != uuid.Nil
	i := r.Identity != nil && r.Identity.EffectiveIdentityID() != uuid.Nil
	if g && !i {
		return SubjectKindGroup
	}
	if i && !g {
		return SubjectKindIdentity
	}
	return SubjectNone
}

// EffectiveGroupID returns the referenced group id when [SubjectKind] is [SubjectKindGroup].
func (r *SubjectRef) EffectiveGroupID() uuid.UUID {
	if r == nil || r.Group == nil {
		return uuid.Nil
	}
	return r.Group.EffectiveGroupID()
}

// EffectiveIdentityID returns the referenced identity id when [SubjectKind] is [SubjectKindIdentity].
func (r *SubjectRef) EffectiveIdentityID() uuid.UUID {
	if r == nil || r.Identity == nil {
		return uuid.Nil
	}
	return r.Identity.EffectiveIdentityID()
}
