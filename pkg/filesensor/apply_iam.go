package filesensor

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/iam"
)

func parseUUIDStringList(spec map[string]any, key string) ([]uuid.UUID, error) {
	raw, ok := spec[key]
	if !ok || raw == nil {
		return nil, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%q must be an array", key)
	}
	out := make([]uuid.UUID, 0, len(arr))
	for i, item := range arr {
		s, ok := item.(string)
		if !ok {
			s = strings.TrimSpace(fmt.Sprint(item))
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, fmt.Errorf("%q[%d]: empty UUID", key, i)
		}
		id, err := uuid.Parse(s)
		if err != nil || id == uuid.Nil {
			return nil, fmt.Errorf("%q[%d]: invalid UUID", key, i)
		}
		out = append(out, id)
	}
	return out, nil
}

func applyPermissionSpec(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "permissionSpecId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	ps := iam.NewPermissionSpec(m.GetSink(), id)
	ps.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		ps.SetDescription(desc)
	}
	if err := applyAnnotations(ps.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddPermissionSpec(ps)
}

func applyRoleSpec(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "roleSpecId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	rs := iam.NewRoleSpec(m.GetSink(), id)
	rs.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		rs.SetDescription(desc)
	}
	plist, err := parseUUIDStringList(spec, "permissions")
	if err != nil {
		return err
	}
	if len(plist) > 0 {
		refs := make([]*iam.PermissionSpecRef, 0, len(plist))
		for _, pid := range plist {
			refs = append(refs, &iam.PermissionSpecRef{
				PermissionSpecId: pid,
				PermissionSpec:   m.GetPermissionSpecById(pid),
			})
		}
		rs.SetPermissions(refs)
	}
	if err := applyAnnotations(rs.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddRoleSpec(rs)
}

func applyPermission(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "permissionId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	sid, err := parseUUIDField(spec, "spec")
	if err != nil {
		return fmt.Errorf(`spec must reference a permission specification: %w`, err)
	}
	p := iam.NewPermission(m.GetSink(), id)
	p.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		p.SetDescription(desc)
	}
	p.SetPermissionSpecById(sid)
	if err := applyAnnotations(p.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddPermission(p)
}

func applyRole(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "roleId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	rsid, err := firstUUIDField(spec, "spec", "roleSpecId")
	if err != nil {
		return fmt.Errorf("role requires spec referencing a RoleSpec UUID: %w", err)
	}
	cid, err := firstUUIDField(spec, "context", "contextId")
	if err != nil {
		return fmt.Errorf(`role requires "context": %w`, err)
	}
	r := iam.NewRole(m.GetSink(), id)
	r.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		r.SetDescription(desc)
	}
	r.SetRoleSpecById(rsid)
	r.SetContextRef(&mdlctx.ContextRef{ContextId: cid, Context: m.GetContextById(cid)})
	plist, err := parseUUIDStringList(spec, "permissions")
	if err != nil {
		return err
	}
	if len(plist) > 0 {
		refs := make([]*iam.PermissionRef, 0, len(plist))
		for _, pid := range plist {
			refs = append(refs, &iam.PermissionRef{
				PermissionId: pid,
				Permission:   m.GetPermissionById(pid),
			})
		}
		r.SetPermissions(refs)
	}
	refs, err := parseResourceRefs(spec)
	if err != nil {
		return err
	}
	if len(refs) > 0 {
		r.SetResources(refs)
	}
	if err := applyAnnotations(r.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddRole(r)
}

func parseBindingSubject(spec map[string]any) (*iam.SubjectRef, error) {
	raw, ok := spec["subject"]
	if !ok || raw == nil {
		return nil, fmt.Errorf("subject is required")
	}
	subm, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("subject must be an object with groupId or identityId")
	}
	gid, gh, ge := optionalUUIDRefFromMap(subm, "groupId")
	iid, ih, ie := optionalUUIDRefFromMap(subm, "identityId")
	if ge != nil {
		return nil, fmt.Errorf(`subject.groupId: %w`, ge)
	}
	if ie != nil {
		return nil, fmt.Errorf(`subject.identityId: %w`, ie)
	}
	if gh && ih {
		return nil, fmt.Errorf("subject must not set both groupId and identityId")
	}
	if !gh && !ih {
		return nil, fmt.Errorf("subject must set exactly one of groupId or identityId")
	}
	sub := &iam.SubjectRef{}
	switch {
	case gh:
		sub.Group = &iam.GroupRef{GroupId: gid}
	case ih:
		sub.Identity = &iam.IdentityRef{IdentityId: iid}
	}
	return sub, nil
}

func optionalUUIDRefFromMap(subm map[string]any, key string) (uuid.UUID, bool, error) {
	s, ok := stringField(subm, key)
	if !ok || strings.TrimSpace(s) == "" {
		return uuid.Nil, false, nil
	}
	id, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, true, fmt.Errorf("%q invalid UUID", key)
	}
	return id, true, nil
}

func applyBinding(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "bindingId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	rid, err := firstUUIDField(spec, "role", "roleId")
	if err != nil {
		return fmt.Errorf(`binding requires "role" UUID: %w`, err)
	}
	subject, err := parseBindingSubject(spec)
	if err != nil {
		return err
	}
	if subject.Group != nil {
		subject.Group.Group = m.GetGroupById(subject.Group.GroupId)
	}
	if subject.Identity != nil {
		subject.Identity.Identity = m.GetIdentityById(subject.Identity.IdentityId)
	}
	b := iam.NewBinding(m.GetSink(), id)
	b.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		b.SetDescription(desc)
	}
	b.SetRole(&iam.RoleRef{RoleId: rid, Role: m.GetRoleById(rid)})
	b.SetSubject(subject)
	if err := applyAnnotations(b.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddBinding(b)
}
