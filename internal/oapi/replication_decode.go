package oapi

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/artifact"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.emeland.io/modelsrv/pkg/model/iam"
	"go.emeland.io/modelsrv/pkg/model/node"
	mdlprod "go.emeland.io/modelsrv/pkg/model/product"
)

func decodeReplicationResourceFromMap(m model.Model, rt events.ResourceType, res *map[string]interface{}) (uuid.UUID, any, error) {
	if res == nil {
		return uuid.Nil, nil, fmt.Errorf("nil resource map")
	}
	normalizeReplicationWireMap(rt, *res)
	raw, err := json.Marshal(res)
	if err != nil {
		return uuid.Nil, nil, err
	}
	switch rt {
	case events.SystemResource:
		var os System
		if err := json.Unmarshal(raw, &os); err != nil {
			return uuid.Nil, nil, err
		}
		s, err := systemFromWire(m, os)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return s.GetSystemId(), s, nil

	case events.SystemInstanceResource:
		var os SystemInstance
		if err := json.Unmarshal(raw, &os); err != nil {
			return uuid.Nil, nil, err
		}
		s, err := systemInstanceFromWire(m, os)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return s.GetInstanceId(), s, nil

	case events.APIResource:
		var oa API
		if err := json.Unmarshal(raw, &oa); err != nil {
			return uuid.Nil, nil, err
		}
		a, err := apiFromWire(m, oa)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return a.GetApiId(), a, nil

	case events.APIInstanceResource:
		var oa ApiInstance
		if err := json.Unmarshal(raw, &oa); err != nil {
			return uuid.Nil, nil, err
		}
		a, err := apiInstanceFromWire(m, oa)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return a.GetInstanceId(), a, nil

	case events.ComponentResource:
		var oc Component
		if err := json.Unmarshal(raw, &oc); err != nil {
			return uuid.Nil, nil, err
		}
		c, err := componentFromWire(m, oc)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return c.GetComponentId(), c, nil

	case events.ComponentInstanceResource:
		var oc ComponentInstance
		if err := json.Unmarshal(raw, &oc); err != nil {
			return uuid.Nil, nil, err
		}
		c, err := componentInstanceFromWire(m, oc)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return c.GetInstanceId(), c, nil

	case events.ContextResource:
		var oc Context
		if err := json.Unmarshal(raw, &oc); err != nil {
			return uuid.Nil, nil, err
		}
		c, err := contextFromWire(m, oc)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return c.GetContextId(), c, nil

	case events.ContextTypeResource:
		var oct ContextType
		if err := json.Unmarshal(raw, &oct); err != nil {
			return uuid.Nil, nil, err
		}
		ct, err := contextTypeFromWire(m, oct)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return ct.GetContextTypeId(), ct, nil

	case events.NodeResource:
		var on Node
		if err := json.Unmarshal(raw, &on); err != nil {
			return uuid.Nil, nil, err
		}
		n, err := nodeFromWire(m, on)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return n.GetNodeId(), n, nil

	case events.NodeTypeResource:
		var ont NodeType
		if err := json.Unmarshal(raw, &ont); err != nil {
			return uuid.Nil, nil, err
		}
		nt, err := nodeTypeFromWire(m, ont)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return nt.GetNodeTypeId(), nt, nil

	case events.OrgUnitResource:
		var oo OrgUnit
		if err := json.Unmarshal(raw, &oo); err != nil {
			return uuid.Nil, nil, err
		}
		o, err := orgUnitFromWire(m, oo)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return o.GetOrgUnitId(), o, nil

	case events.GroupResource:
		var og Group
		if err := json.Unmarshal(raw, &og); err != nil {
			return uuid.Nil, nil, err
		}
		g, err := groupFromWire(m, og)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return g.GetGroupId(), g, nil

	case events.IdentityResource:
		var oi Identity
		if err := json.Unmarshal(raw, &oi); err != nil {
			return uuid.Nil, nil, err
		}
		i, err := identityFromWire(m, oi)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return i.GetIdentityId(), i, nil

	case events.PermissionSpecResource:
		var o PermissionSpec
		if err := json.Unmarshal(raw, &o); err != nil {
			return uuid.Nil, nil, err
		}
		ps, err := permissionSpecFromWire(m, o)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return ps.GetPermissionSpecId(), ps, nil

	case events.RoleSpecResource:
		var o RoleSpec
		if err := json.Unmarshal(raw, &o); err != nil {
			return uuid.Nil, nil, err
		}
		rs, err := roleSpecFromWire(m, o)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return rs.GetRoleSpecId(), rs, nil

	case events.PermissionResource:
		var o Permission
		if err := json.Unmarshal(raw, &o); err != nil {
			return uuid.Nil, nil, err
		}
		p, err := permissionFromWire(m, o)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return p.GetPermissionId(), p, nil

	case events.RoleResource:
		var o Role
		if err := json.Unmarshal(raw, &o); err != nil {
			return uuid.Nil, nil, err
		}
		r, err := roleFromWire(m, o)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return r.GetRoleId(), r, nil

	case events.BindingResource:
		var o Binding
		if err := json.Unmarshal(raw, &o); err != nil {
			return uuid.Nil, nil, err
		}
		b, err := bindingFromWire(m, o)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return b.GetBindingId(), b, nil

	case events.ProductResource:
		var op Product
		if err := json.Unmarshal(raw, &op); err != nil {
			return uuid.Nil, nil, err
		}
		p, err := productFromWire(m, op)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return p.GetProductId(), p, nil

	case events.FindingResource:
		var of Finding
		if err := json.Unmarshal(raw, &of); err != nil {
			return uuid.Nil, nil, err
		}
		f, err := findingFromWire(m, of)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return f.GetFindingId(), f, nil

	case events.FindingTypeResource:
		var oft FindingType
		if err := json.Unmarshal(raw, &oft); err != nil {
			return uuid.Nil, nil, err
		}
		ft, err := findingTypeFromWire(m, oft)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return ft.GetFindingTypeId(), ft, nil

	case events.ArtifactResource:
		var oa Artifact
		if err := json.Unmarshal(raw, &oa); err != nil {
			return uuid.Nil, nil, err
		}
		a, err := artifactFromWire(m, oa)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return a.GetArtifactId(), a, nil

	case events.ArtifactInstanceResource:
		var oai ArtifactInstance
		if err := json.Unmarshal(raw, &oai); err != nil {
			return uuid.Nil, nil, err
		}
		ai, err := artifactInstanceFromWire(m, oai)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return ai.GetArtifactInstanceId(), ai, nil

	default:
		return uuid.Nil, nil, fmt.Errorf("unsupported resource type for upsert: %s", rt)
	}
}

func contextFromWire(m model.Model, oc Context) (mdlctx.Context, error) {
	id := uuid.UUID(oc.ContextId)
	c := mdlctx.NewContext(m.GetSink(), id)
	c.SetDisplayName(oc.DisplayName)
	if oc.Description != nil {
		c.SetDescription(*oc.Description)
	}
	c.SetContextTypeById(uuid.UUID(oc.Type))
	if oc.Parent != nil {
		c.SetParentById(uuid.UUID(*oc.Parent))
	}
	mergeOapiAnnotations(c.GetAnnotations(), oc.Annotations)
	return c, nil
}

func contextTypeFromWire(m model.Model, oct ContextType) (mdlctx.ContextType, error) {
	id := uuid.UUID(oct.ContextTypeId)
	ct := mdlctx.NewContextType(m.GetSink(), id)
	ct.SetDisplayName(oct.DisplayName)
	if oct.Description != nil {
		ct.SetDescription(*oct.Description)
	}
	mergeOapiAnnotations(ct.GetAnnotations(), oct.Annotations)
	return ct, nil
}

func nodeFromWire(m model.Model, on Node) (node.Node, error) {
	id := uuid.UUID(on.NodeId)
	n := node.NewNode(m.GetSink(), id)
	n.SetDisplayName(on.DisplayName)
	ntid := uuid.UUID(on.NodeType)
	n.SetTypeRef(&node.NodeTypeRef{
		NodeTypeId: ntid,
		NodeType:   m.GetNodeTypeById(ntid),
	})
	mergeOapiAnnotations(n.GetAnnotations(), on.Annotations)
	return n, nil
}

func nodeTypeFromWire(m model.Model, ont NodeType) (node.NodeType, error) {
	id := uuid.UUID(ont.NodeTypeId)
	nt := node.NewNodeType(m.GetSink(), id)
	nt.SetDisplayName(ont.DisplayName)
	mergeOapiAnnotations(nt.GetAnnotations(), ont.Annotations)
	return nt, nil
}

func orgUnitFromWire(m model.Model, oo OrgUnit) (iam.OrgUnit, error) {
	id := uuid.UUID(oo.OrgUnitId)
	o := iam.NewOrgUnit(m.GetSink(), id)
	o.SetDisplayName(oo.DisplayName)
	if oo.Description != nil {
		o.SetDescription(*oo.Description)
	}
	mergeOapiAnnotations(o.GetAnnotations(), oo.Annotations)
	return o, nil
}

func groupFromWire(m model.Model, og Group) (iam.Group, error) {
	id := uuid.UUID(og.GroupId)
	g := iam.NewGroup(m.GetSink(), id)
	g.SetDisplayName(og.DisplayName)
	if og.Description != nil {
		g.SetDescription(*og.Description)
	}
	mergeOapiAnnotations(g.GetAnnotations(), og.Annotations)
	return g, nil
}

func identityFromWire(m model.Model, oi Identity) (iam.Identity, error) {
	id := uuid.UUID(oi.IdentityId)
	i := iam.NewIdentity(m.GetSink(), id)
	i.SetDisplayName(oi.DisplayName)
	if oi.Description != nil {
		i.SetDescription(*oi.Description)
	}
	mergeOapiAnnotations(i.GetAnnotations(), oi.Annotations)
	return i, nil
}

func permissionSpecFromWire(m model.Model, op PermissionSpec) (iam.PermissionSpec, error) {
	id := uuid.UUID(op.PermissionSpecId)
	ps := iam.NewPermissionSpec(m.GetSink(), id)
	ps.SetDisplayName(op.DisplayName)
	if op.Description != nil {
		ps.SetDescription(*op.Description)
	}
	mergeOapiAnnotations(ps.GetAnnotations(), op.Annotations)
	return ps, nil
}

func roleSpecFromWire(m model.Model, ors RoleSpec) (iam.RoleSpec, error) {
	id := uuid.UUID(ors.RoleSpecId)
	rs := iam.NewRoleSpec(m.GetSink(), id)
	rs.SetDisplayName(ors.DisplayName)
	if ors.Description != nil {
		rs.SetDescription(*ors.Description)
	}
	if ors.Permissions != nil {
		refs := make([]*iam.PermissionSpecRef, 0, len(*ors.Permissions))
		for _, pid := range *ors.Permissions {
			u := uuid.UUID(pid)
			refs = append(refs, &iam.PermissionSpecRef{
				PermissionSpecId: u,
				PermissionSpec:   m.GetPermissionSpecById(u),
			})
		}
		rs.SetPermissions(refs)
	}
	mergeOapiAnnotations(rs.GetAnnotations(), ors.Annotations)
	return rs, nil
}

func permissionFromWire(m model.Model, op Permission) (iam.Permission, error) {
	id := uuid.UUID(op.PermissionId)
	p := iam.NewPermission(m.GetSink(), id)
	p.SetDisplayName(op.DisplayName)
	if op.Description != nil {
		p.SetDescription(*op.Description)
	}
	p.SetPermissionSpecById(uuid.UUID(op.Spec))
	mergeOapiAnnotations(p.GetAnnotations(), op.Annotations)
	return p, nil
}

func roleFromWire(m model.Model, orole Role) (iam.Role, error) {
	id := uuid.UUID(orole.RoleId)
	r := iam.NewRole(m.GetSink(), id)
	r.SetDisplayName(orole.DisplayName)
	if orole.Description != nil {
		r.SetDescription(*orole.Description)
	}
	r.SetRoleSpecById(uuid.UUID(orole.Spec))
	if orole.Permissions != nil {
		refs := make([]*iam.PermissionRef, 0, len(*orole.Permissions))
		for _, pid := range *orole.Permissions {
			u := uuid.UUID(pid)
			refs = append(refs, &iam.PermissionRef{
				PermissionId: u,
				Permission:   m.GetPermissionById(u),
			})
		}
		r.SetPermissions(refs)
	}
	if orole.Resources != nil {
		res := make([]*common.ResourceRef, 0, len(*orole.Resources))
		for i := range *orole.Resources {
			rr := (*orole.Resources)[i]
			res = append(res, &common.ResourceRef{
				ResourceId:   uuid.UUID(rr.ResourceId),
				ResourceType: resourceTypeFromWireField(rr.ResourceType),
			})
		}
		r.SetResources(res)
	}
	r.SetContextRef(&mdlctx.ContextRef{
		ContextId: uuid.UUID(orole.Context),
		Context:   m.GetContextById(uuid.UUID(orole.Context)),
	})
	mergeOapiAnnotations(r.GetAnnotations(), orole.Annotations)
	return r, nil
}

func bindingFromWire(m model.Model, ob Binding) (iam.Binding, error) {
	id := uuid.UUID(ob.BindingId)
	b := iam.NewBinding(m.GetSink(), id)
	b.SetDisplayName(ob.DisplayName)
	if ob.Description != nil {
		b.SetDescription(*ob.Description)
	}
	rid := uuid.UUID(ob.Role)
	b.SetRole(&iam.RoleRef{RoleId: rid, Role: m.GetRoleById(rid)})
	sub := &iam.SubjectRef{}
	if ob.Subject.GroupId != nil {
		gid := uuid.UUID(*ob.Subject.GroupId)
		sub.Group = &iam.GroupRef{GroupId: gid, Group: m.GetGroupById(gid)}
	}
	if ob.Subject.IdentityId != nil {
		iid := uuid.UUID(*ob.Subject.IdentityId)
		sub.Identity = &iam.IdentityRef{IdentityId: iid, Identity: m.GetIdentityById(iid)}
	}
	if sub.EffectiveKind() == iam.SubjectNone {
		return nil, fmt.Errorf("binding %s: subject must set exactly one of groupId or identityId", id)
	}
	b.SetSubject(sub)
	mergeOapiAnnotations(b.GetAnnotations(), ob.Annotations)
	return b, nil
}

func productFromWire(m model.Model, op Product) (mdlprod.Product, error) {
	id := uuid.UUID(op.ProductId)
	p := mdlprod.NewProduct(m.GetSink(), id)
	p.SetDisplayName(op.DisplayName)
	if op.Description != nil {
		p.SetDescription(*op.Description)
	}
	if op.Vendor != nil {
		p.SetVendor(refOrgUnit(m, uuid.UUID(*op.Vendor)))
	}
	if op.Versions != nil {
		list := make([]mdlprod.ProductionVersion, 0, len(*op.Versions))
		for i := range *op.Versions {
			list = append(list, productionVersionFromWire((*op.Versions)[i]))
		}
		p.SetVersions(list)
	}
	mergeOapiAnnotations(p.GetAnnotations(), op.Annotations)
	return p, nil
}

func productionVersionFromWire(in ProductionVersion) mdlprod.ProductionVersion {
	var out mdlprod.ProductionVersion
	out.AvailableFrom = in.AvailableFrom
	out.DeprecatedFrom = in.DeprecatedFrom
	out.TerminatedFrom = in.TerminatedFrom
	if in.Artefacts != nil {
		for _, aid := range *in.Artefacts {
			out.Artefacts = append(out.Artefacts, uuid.UUID(aid))
		}
	}
	return out
}

func findingFromWire(m model.Model, of Finding) (finding.Finding, error) {
	id := uuid.UUID(of.FindingId)
	f := finding.NewFinding(m.GetSink(), id)
	f.SetSummary(of.Summary)
	if of.Description != nil {
		f.SetDescription(*of.Description)
	}
	refs := make([]*common.ResourceRef, 0, len(of.Resources))
	for i := range of.Resources {
		r := of.Resources[i]
		rt := resourceTypeFromWireField(r.ResourceType)
		refs = append(refs, &common.ResourceRef{
			ResourceId:   uuid.UUID(r.ResourceId),
			ResourceType: rt,
		})
	}
	f.SetResources(refs)
	if of.Type != nil {
		f.SetFindingTypeById(uuid.UUID(*of.Type))
	}
	mergeOapiAnnotations(f.GetAnnotations(), of.Annotations)
	return f, nil
}

// resourceTypeFromWireField decodes Finding.ResourceRef.resourceType after JSON decoding
// (string label from OpenAPI-shaped payloads, or numeric enum from encoding/json on domain types).
func resourceTypeFromWireField(v interface{}) events.ResourceType {
	if v == nil {
		return events.UnknownResourceType
	}
	if s, ok := v.(string); ok {
		if rt := events.ParseResourceType(s); rt != events.UnknownResourceType {
			return rt
		}
	}
	switch n := v.(type) {
	case float64:
		return events.ResourceType(int(n))
	case int:
		return events.ResourceType(n)
	}
	return events.ParseResourceType(fmt.Sprint(v))
}

func findingTypeFromWire(m model.Model, oft FindingType) (finding.FindingType, error) {
	if oft.FindingTypeId == nil {
		return nil, fmt.Errorf("finding type event missing findingTypeId")
	}
	id := uuid.UUID(*oft.FindingTypeId)
	ft := finding.NewFindingType(m.GetSink(), id)
	if oft.DisplayName != nil {
		ft.SetDisplayName(*oft.DisplayName)
	} else {
		ft.SetDisplayName("")
	}
	if oft.Description != nil {
		ft.SetDescription(*oft.Description)
	} else {
		ft.SetDescription("")
	}
	mergeOapiAnnotations(ft.GetAnnotations(), oft.Annotations)
	return ft, nil
}

func artifactFromWire(m model.Model, oa Artifact) (artifact.Artifact, error) {
	id := uuid.UUID(oa.ArtifactId)
	a := artifact.NewArtifact(m.GetSink(), id)
	a.SetDisplayName(oa.DisplayName)
	if oa.Description != nil {
		a.SetDescription(*oa.Description)
	}
	if oa.Hash != nil {
		a.SetHash(*oa.Hash)
	}
	mergeOapiAnnotations(a.GetAnnotations(), oa.Annotations)
	return a, nil
}

func artifactInstanceFromWire(m model.Model, oai ArtifactInstance) (artifact.ArtifactInstance, error) {
	id := uuid.UUID(oai.ArtifactInstanceId)
	ai := artifact.NewArtifactInstance(m.GetSink(), id)
	ai.SetDisplayName(oai.DisplayName)
	if oai.Description != nil {
		ai.SetDescription(*oai.Description)
	}
	if oai.Artifact != nil {
		artId := uuid.UUID(*oai.Artifact)
		ai.SetArtifactRef(&artifact.ArtifactRef{
			ArtifactId: artId,
			Artifact:   m.GetArtifactById(artId),
		})
	}
	mergeOapiAnnotations(ai.GetAnnotations(), oai.Annotations)
	return ai, nil
}
