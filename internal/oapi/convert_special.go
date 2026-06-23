package oapi

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.emeland.io/modelsrv/pkg/model"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/artifact"
	mdlcapability "go.emeland.io/modelsrv/pkg/model/capability"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/component"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	mdlfilterrule "go.emeland.io/modelsrv/pkg/model/filterrule"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.emeland.io/modelsrv/pkg/model/iam"
	mdlmergerule "go.emeland.io/modelsrv/pkg/model/mergerule"
	"go.emeland.io/modelsrv/pkg/model/node"
	mdlparameter "go.emeland.io/modelsrv/pkg/model/parameter"
	mdlprod "go.emeland.io/modelsrv/pkg/model/product"
	"go.emeland.io/modelsrv/pkg/model/system"
)

// ContextFromDto builds a domain Context from a wire DTO.
func ContextFromDto(m model.Model, oc *Context) (mdlctx.Context, error) {
	if oc == nil {
		return nil, fmt.Errorf("nil context")
	}
	id := uuid.UUID(oc.ContextId)
	c := mdlctx.NewContext(id)
	c.SetDisplayName(oc.DisplayName)
	if oc.Description != nil {
		c.SetDescription(*oc.Description)
	}
	c.SetContextTypeById(uuid.UUID(oc.Type))
	if oc.Parent != nil {
		c.SetParentById(uuid.UUID(*oc.Parent))
	}
	MergeAnnotationsFromDto(c.GetAnnotations(), oc.Annotations)
	return c, nil
}

func ContextToDto(c mdlctx.Context) Context {
	if c == nil {
		return Context{}
	}
	out := Context{
		ContextId:   uuidToOpenAPI(c.GetContextId()),
		DisplayName: c.GetDisplayName(),
		Annotations: AnnotationsToDto(c.GetAnnotations()),
	}
	if desc := c.GetDescription(); desc != "" {
		out.Description = &desc
	}
	if typeID := c.GetContextTypeId(); typeID != uuid.Nil {
		out.Type = uuidToOpenAPI(typeID)
	}
	if parentID := c.GetParentId(); parentID != uuid.Nil {
		out.Parent = uuidPtr(parentID)
	}
	return out
}

// NodeFromDto builds a domain Node from a wire DTO.
func NodeFromDto(m model.Model, on *Node) (node.Node, error) {
	if on == nil {
		return nil, fmt.Errorf("nil node")
	}
	id := uuid.UUID(on.NodeId)
	n := node.NewNode(id)
	n.SetDisplayName(on.DisplayName)
	if on.Description != nil {
		n.SetDescription(*on.Description)
	}
	ntid := uuid.UUID(on.NodeType)
	n.SetTypeRef(&node.NodeTypeRef{
		NodeTypeId: ntid,
		NodeType:   nilsafeGetNodeType(m, ntid),
	})
	MergeAnnotationsFromDto(n.GetAnnotations(), on.Annotations)
	return n, nil
}

func nilsafeGetNodeType(m model.Model, ntid uuid.UUID) node.NodeType {
	if m == nil {
		return nil
	}
	return m.GetNodeTypeById(ntid)
}

func NodeToDto(n node.Node) Node {
	if n == nil {
		return Node{}
	}
	out := Node{
		NodeId:      uuidToOpenAPI(n.GetNodeId()),
		DisplayName: n.GetDisplayName(),
		Annotations: AnnotationsToDto(n.GetAnnotations()),
	}
	if desc := n.GetDescription(); desc != "" {
		out.Description = &desc
	}
	if typeID := n.GetNodeTypeId(); typeID != uuid.Nil {
		out.NodeType = uuidToOpenAPI(typeID)
	}
	return out
}

// ProductionVersionFromDto maps a wire production version payload.
func ProductionVersionFromDto(in ProductionVersion) mdlprod.ProductionVersion {
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

// ProductFromDto builds a Product from a wire DTO.
func ProductFromDto(m model.Model, op *Product) (mdlprod.Product, error) {
	if op == nil {
		return nil, fmt.Errorf("nil product")
	}
	id := uuid.UUID(op.ProductId)
	p := mdlprod.NewProduct(id)
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
			list = append(list, ProductionVersionFromDto((*op.Versions)[i]))
		}
		p.SetVersions(list)
	}
	MergeAnnotationsFromDto(p.GetAnnotations(), op.Annotations)
	return p, nil
}

func ProductToDto(p mdlprod.Product) Product {
	if p == nil {
		return Product{}
	}
	desc := p.GetDescription()
	out := Product{
		ProductId:   uuidToOpenAPI(p.GetProductId()),
		DisplayName: p.GetDisplayName(),
		Description: &desc,
		Annotations: AnnotationsToDto(p.GetAnnotations()),
	}
	if v := p.GetVendor(); v != nil {
		vid := v.OrgUnitId
		if v.OrgUnit != nil {
			vid = v.OrgUnit.GetOrgUnitId()
		}
		out.Vendor = uuidPtr(vid)
	}
	if vers := productionVersionsToDto(p.GetVersions()); vers != nil {
		out.Versions = vers
	}
	return out
}

// FindingFromDto builds a Finding from a wire DTO.
func FindingFromDto(m model.Model, of *Finding) (finding.Finding, error) {
	if of == nil {
		return nil, fmt.Errorf("nil finding")
	}
	id := uuid.UUID(of.FindingId)
	f := finding.NewFinding(id)
	f.SetDisplayName(of.DisplayName)
	if of.Description != nil {
		f.SetDescription(*of.Description)
	}
	refs := make([]*common.ResourceRef, 0, len(of.Resources))
	for i := range of.Resources {
		r := of.Resources[i]
		rt := ResourceTypeFromWireField(r.ResourceType)
		refs = append(refs, &common.ResourceRef{
			ResourceId:   uuid.UUID(r.ResourceId),
			ResourceType: rt,
		})
	}
	f.SetResources(refs)
	if of.Type != nil {
		f.SetFindingTypeById(uuid.UUID(*of.Type))
	}
	MergeAnnotationsFromDto(f.GetAnnotations(), of.Annotations)
	return f, nil
}

func FindingToDto(f finding.Finding) Finding {
	if f == nil {
		return Finding{}
	}
	desc := f.GetDescription()
	out := Finding{
		FindingId:   uuidToOpenAPI(f.GetFindingId()),
		DisplayName: f.GetDisplayName(),
		Description: &desc,
		Resources:   resourceRefsToDto(f.GetResources()),
		Annotations: AnnotationsToDto(f.GetAnnotations()),
	}
	if typeID := f.GetFindingTypeId(); typeID != uuid.Nil {
		out.Type = uuidPtr(typeID)
	}
	return out
}

// FindingTypeFromDto builds a FindingType from a wire DTO.
func FindingTypeFromDto(m model.Model, oft *FindingType) (finding.FindingType, error) {
	if oft == nil {
		return nil, fmt.Errorf("nil finding type")
	}
	if oft.FindingTypeId == nil {
		return nil, fmt.Errorf("finding type event missing findingTypeId")
	}
	id := uuid.UUID(*oft.FindingTypeId)
	ft := finding.NewFindingType(id)
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
	MergeAnnotationsFromDto(ft.GetAnnotations(), oft.Annotations)
	return ft, nil
}

func FindingTypeToDto(ft finding.FindingType) FindingType {
	if ft == nil {
		return FindingType{}
	}
	id := ft.GetFindingTypeId()
	name := ft.GetDisplayName()
	desc := ft.GetDescription()
	return FindingType{
		FindingTypeId: uuidPtr(id),
		DisplayName:   &name,
		Description:   &desc,
		Annotations:   AnnotationsToDto(ft.GetAnnotations()),
	}
}

// ArtifactInstanceFromDto builds an ArtifactInstance from a wire DTO.
func ArtifactInstanceFromDto(m model.Model, oai *ArtifactInstance) (artifact.ArtifactInstance, error) {
	if oai == nil {
		return nil, fmt.Errorf("nil artifact instance")
	}
	id := uuid.UUID(oai.ArtifactInstanceId)
	ai := artifact.NewArtifactInstance(id)
	ai.SetDisplayName(oai.DisplayName)
	if oai.Description != nil {
		ai.SetDescription(*oai.Description)
	}
	if oai.Artifact != nil {
		artID := uuid.UUID(*oai.Artifact)
		ai.SetArtifactRef(&artifact.ArtifactRef{
			ArtifactId: artID,
			Artifact:   nilsafeGetArtifact(m, artID),
		})
	}
	MergeAnnotationsFromDto(ai.GetAnnotations(), oai.Annotations)
	return ai, nil
}

func nilsafeGetArtifact(m model.Model, artID uuid.UUID) artifact.Artifact {
	if m == nil {
		return nil
	}
	return m.GetArtifactById(artID)
}

func ArtifactInstanceToDto(ai artifact.ArtifactInstance) ArtifactInstance {
	if ai == nil {
		return ArtifactInstance{}
	}
	desc := ai.GetDescription()
	out := ArtifactInstance{
		ArtifactInstanceId: uuidToOpenAPI(ai.GetArtifactInstanceId()),
		DisplayName:        ai.GetDisplayName(),
		Description:        &desc,
		Annotations:        AnnotationsToDto(ai.GetAnnotations()),
	}
	if ref := ai.GetArtifactRef(); ref != nil {
		out.Artifact = uuidPtr(ref.ArtifactId)
	}
	return out
}

// SystemFromDto builds a System from a wire DTO.
func SystemFromDto(m model.Model, os *System) (system.System, error) {
	if os == nil {
		return nil, fmt.Errorf("nil system")
	}
	if os.SystemId == nil {
		return nil, fmt.Errorf("system event missing systemId")
	}
	id := uuid.UUID(*os.SystemId)
	sys := system.NewSystem(id)
	sys.SetDisplayName(os.DisplayName)
	if os.Description != nil {
		sys.SetDescription(*os.Description)
	}
	sys.SetAbstract(os.Abstract)
	if os.Version != nil {
		sys.SetVersion(versionFromDto(os.Version))
	}
	if os.Parent != nil {
		sys.SetParent(refSystem(m, uuid.UUID(*os.Parent)))
	}
	MergeAnnotationsFromDto(sys.GetAnnotations(), os.Annotations)
	return sys, nil
}

func SystemToDto(sys system.System) System {
	if sys == nil {
		return System{}
	}
	id := sys.GetSystemId()
	desc := sys.GetDescription()
	out := System{
		SystemId:    uuidPtr(id),
		DisplayName: sys.GetDisplayName(),
		Description: &desc,
		Abstract:    sys.GetAbstract(),
		Annotations: AnnotationsToDto(sys.GetAnnotations()),
	}
	if v := versionToDto(sys.GetVersion()); v != nil {
		out.Version = v
	}
	if parent, _ := sys.GetParent(); parent != nil {
		out.Parent = uuidPtr(parent.GetSystemId())
	}
	return out
}

// SystemInstanceFromDto builds a SystemInstance from a wire DTO.
func SystemInstanceFromDto(m model.Model, os *SystemInstance) (system.SystemInstance, error) {
	if os == nil {
		return nil, fmt.Errorf("nil system instance")
	}
	id := uuid.UUID(os.SystemInstanceId)
	si := system.NewSystemInstance(id)
	si.SetDisplayName(os.DisplayName)
	si.SetSystemRef(refSystem(m, uuid.UUID(os.System)))
	if os.Context != nil {
		si.SetContextRef(&mdlctx.ContextRef{ContextId: uuid.UUID(*os.Context)})
	}
	MergeAnnotationsFromDto(si.GetAnnotations(), os.Annotations)
	return si, nil
}

func SystemInstanceToDto(si system.SystemInstance) SystemInstance {
	if si == nil {
		return SystemInstance{}
	}
	out := SystemInstance{
		SystemInstanceId: uuidToOpenAPI(si.GetInstanceId()),
		DisplayName:      si.GetDisplayName(),
		Annotations:      AnnotationsToDto(si.GetAnnotations()),
	}
	if ref := si.GetSystemRef(); ref != nil {
		out.System = uuidToOpenAPI(ref.SystemId)
	}
	if ref := si.GetContextRef(); ref != nil {
		out.Context = uuidPtr(ref.ContextId)
	}
	return out
}

// APIFromDto builds an API from a wire DTO.
func APIFromDto(m model.Model, oa *API) (mdlapi.API, error) {
	if oa == nil {
		return nil, fmt.Errorf("nil API")
	}
	if oa.ApiId == nil {
		return nil, fmt.Errorf("API event missing apiId")
	}
	id := uuid.UUID(*oa.ApiId)
	dom := mdlapi.NewAPI(id)
	dom.SetDisplayName(oa.DisplayName)
	if oa.Description != nil {
		dom.SetDescription(*oa.Description)
	}
	dom.SetType(parseAPIType(oa.Type))
	if oa.Version != nil {
		dom.SetVersion(versionFromDto(oa.Version))
	}
	if oa.System != nil {
		dom.SetSystem(refSystem(m, uuid.UUID(*oa.System)))
	}
	MergeAnnotationsFromDto(dom.GetAnnotations(), oa.Annotations)
	return dom, nil
}

func APIToDto(api mdlapi.API) API {
	if api == nil {
		return API{}
	}
	id := api.GetApiId()
	desc := api.GetDescription()
	out := API{
		ApiId:       uuidPtr(id),
		DisplayName: api.GetDisplayName(),
		Description: &desc,
		Type:        api.GetType().String(),
		Annotations: AnnotationsToDto(api.GetAnnotations()),
	}
	if v := versionToDto(api.GetVersion()); v != nil {
		out.Version = v
	}
	if s := api.GetSystem(); s != nil {
		out.System = uuidPtr(s.SystemId)
	}
	return out
}

// ApiInstanceFromDto builds an ApiInstance from a wire DTO.
func ApiInstanceFromDto(m model.Model, oa *ApiInstance) (mdlapi.ApiInstance, error) {
	if oa == nil {
		return nil, fmt.Errorf("nil API instance")
	}
	id := uuid.UUID(oa.ApiInstanceId)
	ai := mdlapi.NewApiInstance(id)
	ai.SetDisplayName(oa.DisplayName)
	if oa.Api != nil {
		if ref := refAPI(m, uuid.UUID(*oa.Api)); ref != nil {
			ai.SetApiRef(ref)
		} else {
			log.Printf("WARNING: skipping unresolvable API ref %s", *oa.Api)
		}
	}
	if oa.SystemInstance != nil {
		if ref := refSystemInstance(m, uuid.UUID(*oa.SystemInstance)); ref != nil {
			ai.SetSystemInstance(ref)
		} else {
			log.Printf("WARNING: skipping unresolvable SystemInstance ref %s", *oa.SystemInstance)
		}
	}
	MergeAnnotationsFromDto(ai.GetAnnotations(), oa.Annotations)
	return ai, nil
}

func ApiInstanceToDto(ai mdlapi.ApiInstance) ApiInstance {
	if ai == nil {
		return ApiInstance{}
	}
	out := ApiInstance{
		ApiInstanceId: uuidToOpenAPI(ai.GetInstanceId()),
		DisplayName:   ai.GetDisplayName(),
		Annotations:   AnnotationsToDto(ai.GetAnnotations()),
	}
	if ref := ai.GetApiRef(); ref != nil {
		out.Api = uuidPtr(ref.ApiID)
	}
	if ref := ai.GetSystemInstance(); ref != nil {
		out.SystemInstance = uuidPtr(ref.InstanceId)
	}
	return out
}

// ComponentFromDto builds a Component from a wire DTO.
func ComponentFromDto(m model.Model, oc *Component) (component.Component, error) {
	if oc == nil {
		return nil, fmt.Errorf("nil component")
	}
	if oc.ComponentId == nil {
		return nil, fmt.Errorf("component event missing componentId")
	}
	id := uuid.UUID(*oc.ComponentId)
	c := component.NewComponent(id)
	c.SetDisplayName(oc.DisplayName)
	if oc.Description != nil {
		c.SetDescription(*oc.Description)
	}
	if oc.Version != nil {
		c.SetVersion(versionFromDto(oc.Version))
	}
	c.SetSystem(refSystem(m, uuid.UUID(oc.System)))
	if oc.Consumes != nil {
		c.SetConsumes(apiRefsFromUUIDs(m, *oc.Consumes))
	}
	if oc.Provides != nil {
		c.SetProvides(apiRefsFromUUIDs(m, *oc.Provides))
	}
	MergeAnnotationsFromDto(c.GetAnnotations(), oc.Annotations)
	return c, nil
}

func ComponentToDto(c component.Component) Component {
	if c == nil {
		return Component{}
	}
	id := c.GetComponentId()
	desc := c.GetDescription()
	out := Component{
		ComponentId: uuidPtr(id),
		DisplayName: c.GetDisplayName(),
		Description: &desc,
		Annotations: AnnotationsToDto(c.GetAnnotations()),
	}
	if v := versionToDto(c.GetVersion()); v != nil {
		out.Version = v
	}
	if s := c.GetSystem(); s != nil {
		out.System = uuidToOpenAPI(s.SystemId)
	}
	if consumes := c.GetConsumes(); len(consumes) > 0 {
		ids := make([]openapi_types.UUID, len(consumes))
		for i, ref := range consumes {
			ids[i] = uuidToOpenAPI(ref.ApiID)
		}
		out.Consumes = &ids
	}
	if provides := c.GetProvides(); len(provides) > 0 {
		ids := make([]openapi_types.UUID, len(provides))
		for i, ref := range provides {
			ids[i] = uuidToOpenAPI(ref.ApiID)
		}
		out.Provides = &ids
	}
	return out
}

// ComponentInstanceFromDto builds a ComponentInstance from a wire DTO.
func ComponentInstanceFromDto(m model.Model, oc *ComponentInstance) (component.ComponentInstance, error) {
	if oc == nil {
		return nil, fmt.Errorf("nil component instance")
	}
	id := uuid.UUID(oc.ComponentInstanceId)
	ci := component.NewComponentInstance(id)
	ci.SetDisplayName(oc.DisplayName)
	if ref := refComponent(m, uuid.UUID(oc.Component)); ref != nil {
		ci.SetComponentRef(ref)
	} else {
		log.Printf("WARNING: skipping unresolvable Component ref %s", oc.Component)
	}
	if ref := refSystemInstance(m, uuid.UUID(oc.SystemInstance)); ref != nil {
		ci.SetSystemInstance(ref)
	} else {
		log.Printf("WARNING: skipping unresolvable SystemInstance ref %s", oc.SystemInstance)
	}
	MergeAnnotationsFromDto(ci.GetAnnotations(), oc.Annotations)
	return ci, nil
}

func ComponentInstanceToDto(ci component.ComponentInstance) ComponentInstance {
	if ci == nil {
		return ComponentInstance{}
	}
	out := ComponentInstance{
		ComponentInstanceId: uuidToOpenAPI(ci.GetInstanceId()),
		DisplayName:         ci.GetDisplayName(),
		Annotations:         AnnotationsToDto(ci.GetAnnotations()),
	}
	if ref := ci.GetComponentRef(); ref != nil {
		out.Component = uuidToOpenAPI(ref.ComponentId)
	}
	if ref := ci.GetSystemInstance(); ref != nil {
		out.SystemInstance = uuidToOpenAPI(ref.InstanceId)
	}
	return out
}

// PermissionSpecFromDto builds a domain PermissionSpec from a wire DTO.
func PermissionSpecFromDto(m model.Model, o *PermissionSpec) (iam.PermissionSpec, error) {
	if o == nil {
		return nil, fmt.Errorf("nil permission specification")
	}
	id := uuid.UUID(o.PermissionSpecId)
	v := iam.NewPermissionSpec(id)
	v.SetDisplayName(o.DisplayName)
	if o.Description != nil {
		v.SetDescription(*o.Description)
	}
	MergeAnnotationsFromDto(v.GetAnnotations(), o.Annotations)
	return v, nil
}

func PermissionSpecToDto(v iam.PermissionSpec) PermissionSpec {
	if v == nil {
		return PermissionSpec{}
	}
	desc := v.GetDescription()
	return PermissionSpec{
		PermissionSpecId: uuidToOpenAPI(v.GetPermissionSpecId()),
		DisplayName:      v.GetDisplayName(),
		Description:      &desc,
		Annotations:      AnnotationsToDto(v.GetAnnotations()),
	}
}

// RoleSpecFromDto builds a domain RoleSpec from a wire DTO.
func RoleSpecFromDto(m model.Model, o *RoleSpec) (iam.RoleSpec, error) {
	if o == nil {
		return nil, fmt.Errorf("nil role specification")
	}
	id := uuid.UUID(o.RoleSpecId)
	v := iam.NewRoleSpec(id)
	v.SetDisplayName(o.DisplayName)
	if o.Description != nil {
		v.SetDescription(*o.Description)
	}
	if o.Permissions != nil {
		refs := make([]*iam.PermissionSpecRef, 0, len(*o.Permissions))
		for _, pid := range *o.Permissions {
			u := uuid.UUID(pid)
			refs = append(refs, &iam.PermissionSpecRef{
				PermissionSpecId: u,
				PermissionSpec:   nilsafeGetPermissionSpec(m, u),
			})
		}
		v.SetPermissions(refs)
	}
	MergeAnnotationsFromDto(v.GetAnnotations(), o.Annotations)
	return v, nil
}

func nilsafeGetPermissionSpec(m model.Model, id uuid.UUID) iam.PermissionSpec {
	if m == nil {
		return nil
	}
	return m.GetPermissionSpecById(id)
}

func RoleSpecToDto(v iam.RoleSpec) RoleSpec {
	if v == nil {
		return RoleSpec{}
	}
	desc := v.GetDescription()
	return RoleSpec{
		RoleSpecId:  uuidToOpenAPI(v.GetRoleSpecId()),
		DisplayName: v.GetDisplayName(),
		Description: &desc,
		Permissions: iamPermissionSpecRefsToOpenAPIUUIDs(v.GetPermissions()),
		Annotations: AnnotationsToDto(v.GetAnnotations()),
	}
}

// PermissionFromDto builds a domain Permission from a wire DTO.
func PermissionFromDto(m model.Model, o *Permission) (iam.Permission, error) {
	if o == nil {
		return nil, fmt.Errorf("nil permission")
	}
	id := uuid.UUID(o.PermissionId)
	v := iam.NewPermission(id)
	v.SetDisplayName(o.DisplayName)
	if o.Description != nil {
		v.SetDescription(*o.Description)
	}
	v.SetPermissionSpecById(uuid.UUID(o.Spec))
	MergeAnnotationsFromDto(v.GetAnnotations(), o.Annotations)
	return v, nil
}

func PermissionToDto(v iam.Permission) Permission {
	if v == nil {
		return Permission{}
	}
	desc := v.GetDescription()
	return Permission{
		PermissionId: uuidToOpenAPI(v.GetPermissionId()),
		DisplayName:  v.GetDisplayName(),
		Description:  &desc,
		Spec:         openapi_types.UUID(v.GetPermissionSpecId()),
		Annotations:  AnnotationsToDto(v.GetAnnotations()),
	}
}

// RoleFromDto builds a domain Role from a wire DTO.
func RoleFromDto(m model.Model, o *Role) (iam.Role, error) {
	if o == nil {
		return nil, fmt.Errorf("nil role")
	}
	id := uuid.UUID(o.RoleId)
	v := iam.NewRole(id)
	v.SetDisplayName(o.DisplayName)
	if o.Description != nil {
		v.SetDescription(*o.Description)
	}
	v.SetRoleSpecById(uuid.UUID(o.Spec))
	if o.Permissions != nil {
		refs := make([]*iam.PermissionRef, 0, len(*o.Permissions))
		for _, pid := range *o.Permissions {
			u := uuid.UUID(pid)
			refs = append(refs, &iam.PermissionRef{
				PermissionId: u,
				Permission:   nilsafeGetPermission(m, u),
			})
		}
		v.SetPermissions(refs)
	}
	if o.Resources != nil {
		res := make([]*common.ResourceRef, 0, len(*o.Resources))
		for i := range *o.Resources {
			rr := (*o.Resources)[i]
			res = append(res, &common.ResourceRef{
				ResourceId:   uuid.UUID(rr.ResourceId),
				ResourceType: ResourceTypeFromWireField(rr.ResourceType),
			})
		}
		v.SetResources(res)
	}
	v.SetContextRef(&mdlctx.ContextRef{
		ContextId: uuid.UUID(o.Context),
		Context:   nilsafeGetContext(m, uuid.UUID(o.Context)),
	})
	MergeAnnotationsFromDto(v.GetAnnotations(), o.Annotations)
	return v, nil
}

func nilsafeGetPermission(m model.Model, id uuid.UUID) iam.Permission {
	if m == nil {
		return nil
	}
	return m.GetPermissionById(id)
}

func nilsafeGetContext(m model.Model, id uuid.UUID) mdlctx.Context {
	if m == nil {
		return nil
	}
	return m.GetContextById(id)
}

func RoleToDto(v iam.Role) Role {
	if v == nil {
		return Role{}
	}
	desc := v.GetDescription()
	out := Role{
		RoleId:      uuidToOpenAPI(v.GetRoleId()),
		DisplayName: v.GetDisplayName(),
		Description: &desc,
		Spec:        openapi_types.UUID(v.GetRoleSpecId()),
		Permissions: iamPermissionRefsToOpenAPIUUIDs(v.GetPermissions()),
		Resources:   iamResourceRefsOptional(v.GetResources()),
		Annotations: AnnotationsToDto(v.GetAnnotations()),
	}
	if cr := v.GetContextRef(); cr != nil {
		out.Context = openapi_types.UUID(cr.EffectiveParentContextID())
	}
	return out
}

// BindingFromDto builds a domain Binding from a wire DTO.
func BindingFromDto(m model.Model, o *Binding) (iam.Binding, error) {
	if o == nil {
		return nil, fmt.Errorf("nil binding")
	}
	id := uuid.UUID(o.BindingId)
	v := iam.NewBinding(id)
	v.SetDisplayName(o.DisplayName)
	if o.Description != nil {
		v.SetDescription(*o.Description)
	}
	rid := uuid.UUID(o.Role)
	v.SetRole(&iam.RoleRef{RoleId: rid, Role: nilsafeGetRole(m, rid)})
	sub := &iam.SubjectRef{}
	if o.Subject.GroupId != nil {
		gid := uuid.UUID(*o.Subject.GroupId)
		sub.Group = &iam.GroupRef{GroupId: gid, Group: nilsafeGetGroup(m, gid)}
	}
	if o.Subject.IdentityId != nil {
		iid := uuid.UUID(*o.Subject.IdentityId)
		sub.Identity = &iam.IdentityRef{IdentityId: iid, Identity: nilsafeGetIdentity(m, iid)}
	}
	if sub.EffectiveKind() == iam.SubjectNone {
		return nil, fmt.Errorf("binding %s: subject must set exactly one of groupId or identityId", id)
	}
	v.SetSubject(sub)
	MergeAnnotationsFromDto(v.GetAnnotations(), o.Annotations)
	return v, nil
}

func nilsafeGetRole(m model.Model, id uuid.UUID) iam.Role {
	if m == nil {
		return nil
	}
	return m.GetRoleById(id)
}

func nilsafeGetGroup(m model.Model, id uuid.UUID) iam.Group {
	if m == nil {
		return nil
	}
	return m.GetGroupById(id)
}

func nilsafeGetIdentity(m model.Model, id uuid.UUID) iam.Identity {
	if m == nil {
		return nil
	}
	return m.GetIdentityById(id)
}

func BindingToDto(v iam.Binding) Binding {
	if v == nil {
		return Binding{}
	}
	desc := v.GetDescription()
	out := Binding{
		BindingId:   uuidToOpenAPI(v.GetBindingId()),
		DisplayName: v.GetDisplayName(),
		Description: &desc,
		Annotations: AnnotationsToDto(v.GetAnnotations()),
	}
	if rr := v.GetRole(); rr != nil {
		out.Role = openapi_types.UUID(rr.EffectiveRoleID())
	}
	out.Subject = iamSubjectToAPISubjectRef(v.GetSubject())
	return out
}

// FilterRuleFromDto builds a FilterRule from a wire DTO.
func FilterRuleFromDto(m model.Model, o *FilterRule) (mdlfilterrule.FilterRule, error) {
	if o == nil {
		return nil, fmt.Errorf("nil filter rule")
	}
	id := uuid.UUID(o.RuleId)
	fr := mdlfilterrule.NewFilterRule(id)
	fr.SetDisplayName(o.DisplayName)
	if o.Description != nil {
		fr.SetDescription(*o.Description)
	}
	return fr, nil
}

func FilterRuleToDto(v mdlfilterrule.FilterRule) FilterRule {
	if v == nil {
		return FilterRule{}
	}
	out := FilterRule{
		RuleId:      uuidToOpenAPI(v.GetRuleId()),
		DisplayName: v.GetDisplayName(),
	}
	if desc := v.GetDescription(); desc != "" {
		out.Description = &desc
	}
	return out
}

// MergeRuleFromDto builds a MergeRule from a wire DTO.
func MergeRuleFromDto(m model.Model, o *MergeRule) (mdlmergerule.MergeRule, error) {
	if o == nil {
		return nil, fmt.Errorf("nil merge rule")
	}
	id := uuid.UUID(o.RuleId)
	mr := mdlmergerule.NewMergeRule(id)
	mr.SetDisplayName(o.DisplayName)
	if o.Description != nil {
		mr.SetDescription(*o.Description)
	}
	return mr, nil
}

func MergeRuleToDto(v mdlmergerule.MergeRule) MergeRule {
	if v == nil {
		return MergeRule{}
	}
	out := MergeRule{
		RuleId:      uuidToOpenAPI(v.GetRuleId()),
		DisplayName: v.GetDisplayName(),
	}
	if desc := v.GetDescription(); desc != "" {
		out.Description = &desc
	}
	return out
}

// CapabilityFromDto builds a Capability from a wire DTO.
func CapabilityFromDto(m model.Model, o *Capability) (mdlcapability.Capability, error) {
	if o == nil {
		return nil, fmt.Errorf("nil capability")
	}
	id := uuid.UUID(o.CapabilityId)
	c := mdlcapability.NewCapability(id)
	c.SetDisplayName(o.DisplayName)
	if o.Versions != nil {
		refs := make([]mdlcapability.CapabilityVersionRef, len(*o.Versions))
		for i, v := range *o.Versions {
			refs[i] = mdlcapability.CapabilityVersionRef{CapabilityVersionId: uuid.UUID(v.CapabilityVersionId)}
		}
		c.SetVersions(refs)
	}
	if o.Annotations != nil {
		MergeAnnotationsFromDto(c.GetAnnotations(), o.Annotations)
	}
	return c, nil
}

func CapabilityToDto(v mdlcapability.Capability) Capability {
	if v == nil {
		return Capability{}
	}
	out := Capability{
		CapabilityId: uuidToOpenAPI(v.GetCapabilityId()),
		DisplayName:  v.GetDisplayName(),
	}
	if vers := v.GetVersions(); len(vers) > 0 {
		refs := make([]CapabilityVersionRef, len(vers))
		for i, r := range vers {
			refs[i] = CapabilityVersionRef{CapabilityVersionId: uuidToOpenAPI(r.CapabilityVersionId)}
		}
		out.Versions = &refs
	}
	out.Annotations = AnnotationsToDto(v.GetAnnotations())
	return out
}

// ParameterFromDto builds a Parameter from a wire DTO.
func ParameterFromDto(m model.Model, o *Parameter) (mdlparameter.Parameter, error) {
	if o == nil {
		return nil, fmt.Errorf("nil parameter")
	}
	id := uuid.UUID(o.ParameterId)
	param := mdlparameter.NewParameter(id)
	param.SetDisplayName(o.DisplayName)
	if o.Values != nil {
		param.SetValues(*o.Values)
	}
	if o.Annotations != nil {
		MergeAnnotationsFromDto(param.GetAnnotations(), o.Annotations)
	}
	return param, nil
}

func ParameterToDto(v mdlparameter.Parameter) Parameter {
	if v == nil {
		return Parameter{}
	}
	out := Parameter{
		ParameterId: uuidToOpenAPI(v.GetParameterId()),
		DisplayName: v.GetDisplayName(),
	}
	if vals := v.GetValues(); len(vals) > 0 {
		out.Values = &vals
	}
	out.Annotations = AnnotationsToDto(v.GetAnnotations())
	return out
}
