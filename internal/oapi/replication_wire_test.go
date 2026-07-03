package oapi_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/eventfilter/phase0"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	mdlcap "go.emeland.io/modelsrv/pkg/model/capacity"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.emeland.io/modelsrv/pkg/model/iam"
	"go.emeland.io/modelsrv/pkg/model/node"
	"go.emeland.io/modelsrv/pkg/model/system"
)

func replicationTestModel() model.Model {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	Expect(err).NotTo(HaveOccurred())
	return m
}

func replicationFindingsOfKind(m model.Model, kind finding.FindingKind) []finding.Finding {
	typeID := finding.TypeIDForKind(kind)
	all, err := m.GetFindings()
	Expect(err).NotTo(HaveOccurred())
	var out []finding.Finding
	for _, f := range all {
		if f.GetFindingTypeId() == typeID {
			out = append(out, f)
		}
	}
	return out
}

var _ = Describe("replication wire: PushWireEventFromDomain (encode)", func() {
	It("returns an error for a nil event", func() {
		_, err := oapi.PushWireEventFromDomain(nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("nil event"))
	})

	It("returns an error when create/update has no objects", func() {
		ev := &events.Event{
			ResourceType: events.SystemResource,
			Operation:    events.CreateOperation,
			ResourceId:   uuid.New(),
			Objects:      nil,
		}
		_, err := oapi.PushWireEventFromDomain(ev)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing resource object"))
	})

	It("returns an error for standalone Annotations replication events", func() {
		ev := &events.Event{
			ResourceType: events.AnnotationsResource,
			Operation:    events.CreateOperation,
			ResourceId:   uuid.Nil,
			Objects:      []any{map[string]string{"k": "v"}},
		}
		_, err := oapi.PushWireEventFromDomain(ev)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not replicated as standalone"))
	})

	It("builds a delete wire event with kind, operation, and resourceId", func() {
		rid := uuid.New()
		ev := &events.Event{
			ResourceType: events.SystemResource,
			Operation:    events.DeleteOperation,
			ResourceId:   rid,
		}
		wire, err := oapi.PushWireEventFromDomain(ev)
		Expect(err).NotTo(HaveOccurred())
		Expect(wire.Kind).To(Equal("System"))
		Expect(wire.Operation).To(Equal("Delete"))
		Expect(wire.ResourceId).NotTo(BeNil())
		Expect(uuid.UUID(*wire.ResourceId)).To(Equal(rid))
		Expect(wire.Resource).To(BeNil())
	})

	It("embeds OpenAPI-shaped system fields in Resource for create", func() {
		sysID := uuid.New()
		sys := system.NewSystem(sysID)
		sys.SetDisplayName("push-name")
		ev := &events.Event{
			ResourceType: events.SystemResource,
			Operation:    events.CreateOperation,
			ResourceId:   sysID,
			Objects:      []any{sys},
		}
		wire, err := oapi.PushWireEventFromDomain(ev)
		Expect(err).NotTo(HaveOccurred())
		Expect(wire.Resource).NotTo(BeNil())
		Expect(*wire.Resource).To(HaveKeyWithValue("displayName", "push-name"))
	})

	It("includes context annotations in the wire resource payload", func() {
		m := replicationTestModel()
		ctid := uuid.New()
		cid := uuid.New()
		ct := mdlctx.NewContextType(ctid)
		ct.SetDisplayName("env-type")
		Expect(m.AddContextType(ct)).To(Succeed())

		c := mdlctx.NewContext(cid)
		c.SetDisplayName("Production")
		c.SetContextTypeById(ctid)
		c.GetAnnotations().Add("env", "prod")

		wire, err := oapi.PushWireEventFromDomain(&events.Event{
			ResourceType: events.ContextResource,
			Operation:    events.CreateOperation,
			ResourceId:   cid,
			Objects:      []any{c},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(wire.Resource).NotTo(BeNil())
		anns, ok := (*wire.Resource)["annotations"].([]interface{})
		Expect(ok).To(BeTrue())
		Expect(anns).To(ContainElement(map[string]interface{}{"key": "env", "value": "prod"}))

		back, err := oapi.ReplicationEventFromWire(m, &wire)
		Expect(err).NotTo(HaveOccurred())
		ctxOut, ok := back.Objects[0].(mdlctx.Context)
		Expect(ok).To(BeTrue())
		Expect(ctxOut.GetAnnotations().GetValue("env")).To(Equal("prod"))
	})
})

var _ = Describe("replication wire: ReplicationEventFromWire (decode + normalize)", func() {
	It("returns an error when create is missing resource", func() {
		m := replicationTestModel()
		ev := oapi.Event{
			Kind:      "System",
			Operation: "Create",
		}
		_, err := oapi.ReplicationEventFromWire(m, &ev)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing resource payload"))
	})

	It("returns an error for unknown replication kind Annotations", func() {
		m := replicationTestModel()
		res := map[string]interface{}{"k": "v"}
		ev := oapi.Event{
			Kind:      "Annotations",
			Operation: "Create",
			Resource:  &res,
		}
		_, err := oapi.ReplicationEventFromWire(m, &ev)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unknown event kind"))
	})

	It("decodes a system after stripping an empty annotations object", func() {
		m := replicationTestModel()
		sysID := uuid.New()
		res := map[string]interface{}{
			"systemId":    sysID.String(),
			"displayName": "ann-strip",
			"annotations": map[string]interface{}{},
		}
		ev := oapi.Event{
			Kind:      "System",
			Operation: "Create",
			Resource:  &res,
		}
		out, err := oapi.ReplicationEventFromWire(m, &ev)
		Expect(err).NotTo(HaveOccurred())
		Expect(out.ResourceId).To(Equal(sysID))
		sys, ok := out.Objects[0].(system.System)
		Expect(ok).To(BeTrue())
		Expect(sys.GetDisplayName()).To(Equal("ann-strip"))
	})

	It("coalesces a nested parent ref map then decodes the system", func() {
		m := replicationTestModel()
		parentID := uuid.New()
		parent := system.NewSystem(parentID)
		parent.SetDisplayName("parent-sys")
		Expect(m.AddSystem(parent)).To(Succeed())

		sysID := uuid.New()
		res := map[string]interface{}{
			"systemId":    sysID.String(),
			"displayName": "child",
			"parent":      map[string]interface{}{"systemId": parentID.String()},
		}
		ev := oapi.Event{
			Kind:      "System",
			Operation: "Update",
			Resource:  &res,
		}
		out, err := oapi.ReplicationEventFromWire(m, &ev)
		Expect(err).NotTo(HaveOccurred())
		sys, ok := out.Objects[0].(system.System)
		Expect(ok).To(BeTrue())
		p, err := sys.GetParent()
		Expect(err).NotTo(HaveOccurred())
		Expect(p).NotTo(BeNil())
		Expect(p.GetSystemId()).To(Equal(parentID))
		Expect(p.GetDisplayName()).To(Equal("parent-sys"))
	})

	It("coalesces SystemInstance nested system and context refs", func() {
		m := replicationTestModel()
		sid := uuid.New()
		cid := uuid.New()
		iid := uuid.New()
		res := map[string]interface{}{
			"systemInstanceId": iid.String(),
			"displayName":      "si",
			"System":           map[string]interface{}{"systemId": sid.String()},
			"Context":          map[string]interface{}{"contextId": cid.String()},
		}
		ev := oapi.Event{
			Kind:      "SystemInstance",
			Operation: "Create",
			Resource:  &res,
		}
		out, err := oapi.ReplicationEventFromWire(m, &ev)
		Expect(err).NotTo(HaveOccurred())
		si, ok := out.Objects[0].(system.SystemInstance)
		Expect(ok).To(BeTrue())
		Expect(si.GetSystemRef().SystemId).To(Equal(sid))
		Expect(si.GetContextRef()).NotTo(BeNil())
		Expect(si.GetContextRef().ContextId).To(Equal(cid))
	})

	It("coalesces Node nested nodeType ref", func() {
		m := replicationTestModel()
		ntid := uuid.New()
		nt := node.NewNodeType(ntid)
		nt.SetDisplayName("nt")
		Expect(m.AddNodeType(nt)).To(Succeed())

		nid := uuid.New()
		res := map[string]interface{}{
			"nodeId":      nid.String(),
			"displayName": "n1",
			"NodeType":    map[string]interface{}{"nodeTypeId": ntid.String()},
		}
		ev := oapi.Event{
			Kind:      "Node",
			Operation: "Create",
			Resource:  &res,
		}
		out, err := oapi.ReplicationEventFromWire(m, &ev)
		Expect(err).NotTo(HaveOccurred())
		n, ok := out.Objects[0].(node.Node)
		Expect(ok).To(BeTrue())
		ntOut, err := n.GetNodeType()
		Expect(err).NotTo(HaveOccurred())
		Expect(ntOut).NotTo(BeNil())
		Expect(ntOut.GetNodeTypeId()).To(Equal(ntid))
	})

	It("coalesces Context TypeRef and Parent (domain-shaped) into type and parent scalars", func() {
		m := replicationTestModel()
		ctid := uuid.New()
		pid := uuid.New()
		cid := uuid.New()
		res := map[string]interface{}{
			"contextId":   cid.String(),
			"displayName": "ctx1",
			"TypeRef":     map[string]interface{}{"contextTypeId": ctid.String()},
			"Parent":      map[string]interface{}{"contextId": pid.String()},
			"annotations": []interface{}{},
		}
		ev := oapi.Event{
			Kind:      "Context",
			Operation: "Create",
			Resource:  &res,
		}
		out, err := oapi.ReplicationEventFromWire(m, &ev)
		Expect(err).NotTo(HaveOccurred())
		ctx, ok := out.Objects[0].(mdlctx.Context)
		Expect(ok).To(BeTrue())
		Expect(ctx.GetContextTypeId()).To(Equal(ctid))
		Expect(ctx.GetParentId()).To(Equal(pid))
	})

	It("coalesces Node TypeRef (domain-shaped from json.Marshal) into nodeType", func() {
		m := replicationTestModel()
		ntid := uuid.New()
		nid := uuid.New()
		nt := node.NewNodeType(ntid)
		nt.SetDisplayName("nt")
		Expect(m.AddNodeType(nt)).To(Succeed())
		res := map[string]interface{}{
			"nodeId":      nid.String(),
			"displayName": "n1",
			"TypeRef":     map[string]interface{}{"nodeTypeId": ntid.String()},
		}
		ev := oapi.Event{
			Kind:      "Node",
			Operation: "Create",
			Resource:  &res,
		}
		out, err := oapi.ReplicationEventFromWire(m, &ev)
		Expect(err).NotTo(HaveOccurred())
		n, ok := out.Objects[0].(node.Node)
		Expect(ok).To(BeTrue())
		ntOut, err := n.GetNodeType()
		Expect(err).NotTo(HaveOccurred())
		Expect(ntOut).NotTo(BeNil())
		Expect(ntOut.GetNodeTypeId()).To(Equal(ntid))
	})

	It("coalesces Finding nested type ref", func() {
		m := replicationTestModel()
		fid := uuid.New()
		ftid := uuid.New()
		res := map[string]interface{}{
			"findingId":   fid.String(),
			"displayName": "name",
			"resources":   []interface{}{},
			"Type":        map[string]interface{}{"findingTypeId": ftid.String()},
		}
		ev := oapi.Event{
			Kind:      "Finding",
			Operation: "Create",
			Resource:  &res,
		}
		out, err := oapi.ReplicationEventFromWire(m, &ev)
		Expect(err).NotTo(HaveOccurred())
		f, ok := out.Objects[0].(finding.Finding)
		Expect(ok).To(BeTrue())
		Expect(f.GetFindingTypeId()).To(Equal(ftid))
	})

	It("coalesces Finding TypeRef from domain encode shape", func() {
		m := replicationTestModel()
		fid := uuid.New()
		ftid := finding.TypeIDForKind(finding.ContextParentNotFound)
		res := map[string]interface{}{
			"findingId":   fid.String(),
			"displayName": "Phase 0 Integrity check",
			"resources":   []interface{}{},
			"TypeRef":     map[string]interface{}{"FindingTypeId": ftid.String()},
		}
		ev := oapi.Event{
			Kind:      "Finding",
			Operation: "Create",
			Resource:  &res,
		}
		out, err := oapi.ReplicationEventFromWire(m, &ev)
		Expect(err).NotTo(HaveOccurred())
		f, ok := out.Objects[0].(finding.Finding)
		Expect(ok).To(BeTrue())
		Expect(f.GetFindingTypeId()).To(Equal(ftid))
	})
})

var _ = Describe("replication wire: encode then decode round-trip", func() {
	It("preserves phase0 finding type ref for read API display name resolution", func() {
		m := replicationTestModel()
		phase0.EnsureWellKnownFindingTypes(m)

		fid := uuid.New()
		kind := finding.ContextTypeMissing
		ftid := finding.TypeIDForKind(kind)
		f := finding.NewFinding(fid)
		f.SetDisplayName("Phase 0 Integrity check")
		f.SetFindingTypeById(ftid)
		f.SetDescription("ContextTypeMissing: context cccc0002-0000-4000-8000-000000000002 has no type assigned")

		wire, err := oapi.PushWireEventFromDomain(&events.Event{
			ResourceType: events.FindingResource,
			Operation:    events.CreateOperation,
			ResourceId:   fid,
			Objects:      []any{f},
		})
		Expect(err).NotTo(HaveOccurred())

		back, err := oapi.ReplicationEventFromWire(m, &wire)
		Expect(err).NotTo(HaveOccurred())
		fOut, ok := back.Objects[0].(finding.Finding)
		Expect(ok).To(BeTrue())
		Expect(fOut.GetFindingTypeId()).To(Equal(ftid))

		Expect(m.Apply(back)).To(Succeed())
		stored := m.GetFindingById(fid)
		Expect(stored.GetFindingTypeId()).To(Equal(ftid))
		ft := m.GetFindingTypeById(ftid)
		Expect(ft).NotTo(BeNil())
		Expect(ft.GetDisplayName()).To(Equal(string(kind)))
	})

	It("preserves system id and display name", func() {
		m := replicationTestModel()
		sysID := uuid.New()
		sys := system.NewSystem(sysID)
		sys.SetDisplayName("round-trip")

		dom := &events.Event{
			ResourceType: events.SystemResource,
			Operation:    events.CreateOperation,
			ResourceId:   sysID,
			Objects:      []any{sys},
		}
		wire, err := oapi.PushWireEventFromDomain(dom)
		Expect(err).NotTo(HaveOccurred())

		back, err := oapi.ReplicationEventFromWire(m, &wire)
		Expect(err).NotTo(HaveOccurred())
		Expect(back.ResourceType).To(Equal(events.SystemResource))
		Expect(back.Operation).To(Equal(events.CreateOperation))
		Expect(back.ResourceId).To(Equal(sysID))
		Expect(back.Objects).To(HaveLen(1))
		out, ok := back.Objects[0].(system.System)
		Expect(ok).To(BeTrue())
		Expect(out.GetSystemId()).To(Equal(sysID))
		Expect(out.GetDisplayName()).To(Equal("round-trip"))
	})

	It("preserves context id, context type id, and parent id", func() {
		m := replicationTestModel()
		ctid := uuid.New()
		parentID := uuid.New()
		cid := uuid.New()
		ct := mdlctx.NewContextType(ctid)
		ct.SetDisplayName("ct")
		Expect(m.AddContextType(ct)).To(Succeed())
		parent := mdlctx.NewContext(parentID)
		parent.SetDisplayName("parent-ctx")
		Expect(m.AddContext(parent)).To(Succeed())

		c := mdlctx.NewContext(cid)
		c.SetDisplayName("child-ctx")
		c.SetContextTypeById(ctid)
		c.SetParentById(parentID)

		dom := &events.Event{
			ResourceType: events.ContextResource,
			Operation:    events.CreateOperation,
			ResourceId:   cid,
			Objects:      []any{c},
		}
		wire, err := oapi.PushWireEventFromDomain(dom)
		Expect(err).NotTo(HaveOccurred())

		back, err := oapi.ReplicationEventFromWire(m, &wire)
		Expect(err).NotTo(HaveOccurred())
		ctxOut, ok := back.Objects[0].(mdlctx.Context)
		Expect(ok).To(BeTrue())
		Expect(ctxOut.GetContextId()).To(Equal(cid))
		Expect(ctxOut.GetContextTypeId()).To(Equal(ctid))
		Expect(ctxOut.GetParentId()).To(Equal(parentID))
	})

	It("preserves node id and node type id", func() {
		m := replicationTestModel()
		ntid := uuid.New()
		nid := uuid.New()
		nt := node.NewNodeType(ntid)
		nt.SetDisplayName("nt")
		Expect(m.AddNodeType(nt)).To(Succeed())
		n := node.NewNode(nid)
		n.SetDisplayName("round-trip-node")
		n.SetNodeTypeByRef(nt)

		dom := &events.Event{
			ResourceType: events.NodeResource,
			Operation:    events.CreateOperation,
			ResourceId:   nid,
			Objects:      []any{n},
		}
		wire, err := oapi.PushWireEventFromDomain(dom)
		Expect(err).NotTo(HaveOccurred())

		back, err := oapi.ReplicationEventFromWire(m, &wire)
		Expect(err).NotTo(HaveOccurred())
		nOut, ok := back.Objects[0].(node.Node)
		Expect(ok).To(BeTrue())
		Expect(nOut.GetNodeId()).To(Equal(nid))
		ntOut, err := nOut.GetNodeType()
		Expect(err).NotTo(HaveOccurred())
		Expect(ntOut).NotTo(BeNil())
		Expect(ntOut.GetNodeTypeId()).To(Equal(ntid))
	})

	It("normalizes IAM Role nested spec, permissions, and context refs for OpenAPI decode", func() {
		m := replicationTestModel()
		rsid := uuid.New()
		pid := uuid.New()
		cid := uuid.New()
		rid := uuid.New()
		res := map[string]interface{}{
			"roleId":      rid.String(),
			"displayName": "r1",
			"spec":        map[string]interface{}{"roleSpecId": rsid.String()},
			"permissions": []interface{}{
				map[string]interface{}{"permissionId": pid.String()},
			},
			"resources": []interface{}{},
			"context":   map[string]interface{}{"contextId": cid.String()},
		}
		ev := oapi.Event{
			Kind:      "Role",
			Operation: "Create",
			Resource:  &res,
		}
		out, err := oapi.ReplicationEventFromWire(m, &ev)
		Expect(err).NotTo(HaveOccurred())
		r, ok := out.Objects[0].(iam.Role)
		Expect(ok).To(BeTrue())
		Expect(r.GetRoleSpecId()).To(Equal(rsid))
		Expect(r.GetPermissions()).To(HaveLen(1))
		Expect(r.GetPermissions()[0].EffectivePermissionID()).To(Equal(pid))
		Expect(r.GetContextRef()).NotTo(BeNil())
		Expect(r.GetContextRef().EffectiveParentContextID()).To(Equal(cid))
	})

	It("normalizes IAM Binding role object and nested subject group ref", func() {
		m := replicationTestModel()
		bindID := uuid.New()
		rid := uuid.New()
		gid := uuid.New()
		res := map[string]interface{}{
			"bindingId":   bindID.String(),
			"displayName": "b1",
			"role":        map[string]interface{}{"roleId": rid.String()},
			"subject": map[string]interface{}{
				"Group": map[string]interface{}{"groupId": gid.String()},
			},
		}
		ev := oapi.Event{
			Kind:      "Binding",
			Operation: "Create",
			Resource:  &res,
		}
		out, err := oapi.ReplicationEventFromWire(m, &ev)
		Expect(err).NotTo(HaveOccurred())
		b, ok := out.Objects[0].(iam.Binding)
		Expect(ok).To(BeTrue())
		Expect(b.GetRole().EffectiveRoleID()).To(Equal(rid))
		Expect(b.GetSubject()).NotTo(BeNil())
		Expect(b.GetSubject().EffectiveKind()).To(Equal(iam.SubjectKindGroup))
		Expect(b.GetSubject().EffectiveGroupID()).To(Equal(gid))
	})

	// Full push→normalize→decode roundtrip tests: these use PushWireEventFromDomain so they catch
	// normalization bugs caused by the domain's unexported JSON keys (no struct tags on generated
	// types, so encoding/json uses capitalized field names).

	It("round-trips an IAM Role via PushWireEventFromDomain", func() {
		m := replicationTestModel()
		rsid := uuid.New()
		pid := uuid.New()
		cid := uuid.New()
		rid := uuid.New()

		r := iam.NewRole(rid)
		r.SetDisplayName("rt-role")
		r.SetRoleSpecById(rsid)
		r.SetPermissions([]*iam.PermissionRef{{PermissionId: pid}})
		r.SetContextRef(&mdlctx.ContextRef{ContextId: cid})

		wire, err := oapi.PushWireEventFromDomain(&events.Event{
			ResourceType: events.RoleResource,
			Operation:    events.CreateOperation,
			ResourceId:   rid,
			Objects:      []any{r},
		})
		Expect(err).NotTo(HaveOccurred())

		back, err := oapi.ReplicationEventFromWire(m, &wire)
		Expect(err).NotTo(HaveOccurred())
		rOut, ok := back.Objects[0].(iam.Role)
		Expect(ok).To(BeTrue())
		Expect(rOut.GetRoleId()).To(Equal(rid))
		Expect(rOut.GetRoleSpecId()).To(Equal(rsid))
		Expect(rOut.GetPermissions()).To(HaveLen(1))
		Expect(rOut.GetPermissions()[0].EffectivePermissionID()).To(Equal(pid))
		Expect(rOut.GetContextRef()).NotTo(BeNil())
		Expect(rOut.GetContextRef().EffectiveParentContextID()).To(Equal(cid))
	})

	It("round-trips an IAM RoleSpec via PushWireEventFromDomain", func() {
		m := replicationTestModel()
		rsid := uuid.New()
		psid := uuid.New()

		rs := iam.NewRoleSpec(rsid)
		rs.SetDisplayName("rt-rolespec")
		rs.SetPermissions([]*iam.PermissionSpecRef{{PermissionSpecId: psid}})

		wire, err := oapi.PushWireEventFromDomain(&events.Event{
			ResourceType: events.RoleSpecResource,
			Operation:    events.CreateOperation,
			ResourceId:   rsid,
			Objects:      []any{rs},
		})
		Expect(err).NotTo(HaveOccurred())

		back, err := oapi.ReplicationEventFromWire(m, &wire)
		Expect(err).NotTo(HaveOccurred())
		rsOut, ok := back.Objects[0].(iam.RoleSpec)
		Expect(ok).To(BeTrue())
		Expect(rsOut.GetRoleSpecId()).To(Equal(rsid))
		Expect(rsOut.GetPermissions()).To(HaveLen(1))
		Expect(rsOut.GetPermissions()[0].EffectivePermissionSpecID()).To(Equal(psid))
	})

	It("round-trips an IAM Binding (group subject) via PushWireEventFromDomain", func() {
		m := replicationTestModel()
		bindID := uuid.New()
		rid := uuid.New()
		gid := uuid.New()

		b := iam.NewBinding(bindID)
		b.SetDisplayName("rt-binding")
		b.SetRole(&iam.RoleRef{RoleId: rid})
		b.SetSubject(&iam.SubjectRef{Group: &iam.GroupRef{GroupId: gid}})

		wire, err := oapi.PushWireEventFromDomain(&events.Event{
			ResourceType: events.BindingResource,
			Operation:    events.CreateOperation,
			ResourceId:   bindID,
			Objects:      []any{b},
		})
		Expect(err).NotTo(HaveOccurred())

		back, err := oapi.ReplicationEventFromWire(m, &wire)
		Expect(err).NotTo(HaveOccurred())
		bOut, ok := back.Objects[0].(iam.Binding)
		Expect(ok).To(BeTrue())
		Expect(bOut.GetBindingId()).To(Equal(bindID))
		Expect(bOut.GetRole().EffectiveRoleID()).To(Equal(rid))
		Expect(bOut.GetSubject().EffectiveKind()).To(Equal(iam.SubjectKindGroup))
		Expect(bOut.GetSubject().EffectiveGroupID()).To(Equal(gid))
	})

	It("round-trips CapacityResourceType via PushWireEventFromDomain", func() {
		m := replicationTestModel()
		crtID := uuid.New()

		crt := mdlcap.NewCapacityResourceType(crtID)
		crt.SetDisplayName("CPU cores")
		crt.SetDescription("Virtual CPU cores")
		crt.SetUnit("cores")

		wire, err := oapi.PushWireEventFromDomain(&events.Event{
			ResourceType: events.CapacityResourceTypeResource,
			Operation:    events.CreateOperation,
			ResourceId:   crtID,
			Objects:      []any{crt},
		})
		Expect(err).NotTo(HaveOccurred())

		back, err := oapi.ReplicationEventFromWire(m, &wire)
		Expect(err).NotTo(HaveOccurred())
		out, ok := back.Objects[0].(mdlcap.CapacityResourceType)
		Expect(ok).To(BeTrue())
		Expect(out.GetCapacityResourceTypeId()).To(Equal(crtID))
		Expect(out.GetDisplayName()).To(Equal("CPU cores"))
		Expect(out.GetUnit()).To(Equal("cores"))

		Expect(m.Apply(back)).To(Succeed())
		stored := m.GetCapacityResourceTypeById(crtID)
		Expect(stored).NotTo(BeNil())
		Expect(stored.GetUnit()).To(Equal("cores"))

		crt.SetUnit("vCPU")
		wireUpdate, err := oapi.PushWireEventFromDomain(&events.Event{
			ResourceType: events.CapacityResourceTypeResource,
			Operation:    events.UpdateOperation,
			ResourceId:   crtID,
			Objects:      []any{crt},
		})
		Expect(err).NotTo(HaveOccurred())

		backUpdate, err := oapi.ReplicationEventFromWire(m, &wireUpdate)
		Expect(err).NotTo(HaveOccurred())
		Expect(m.Apply(backUpdate)).To(Succeed())
		Expect(m.GetCapacityResourceTypeById(crtID).GetUnit()).To(Equal("vCPU"))

		wireDelete, err := oapi.PushWireEventFromDomain(&events.Event{
			ResourceType: events.CapacityResourceTypeResource,
			Operation:    events.DeleteOperation,
			ResourceId:   crtID,
		})
		Expect(err).NotTo(HaveOccurred())
		backDelete, err := oapi.ReplicationEventFromWire(m, &wireDelete)
		Expect(err).NotTo(HaveOccurred())
		Expect(m.Apply(backDelete)).To(Succeed())
		Expect(m.GetCapacityResourceTypeById(crtID)).To(BeNil())
	})

	It("round-trips Capacity via PushWireEventFromDomain", func() {
		m := replicationTestModel()
		crtID := uuid.New()
		ctxID := uuid.New()
		capID := uuid.New()

		crt := mdlcap.NewCapacityResourceType(crtID)
		crt.SetDisplayName("CPU")
		crt.SetUnit("cores")
		Expect(m.AddCapacityResourceType(crt)).To(Succeed())

		ct := mdlctx.NewContextType(uuid.New())
		ct.SetDisplayName("Environment")
		Expect(m.AddContextType(ct)).To(Succeed())

		ctx := mdlctx.NewContext(ctxID)
		ctx.SetDisplayName("Production")
		ctx.SetContextTypeById(ct.GetContextTypeId())
		Expect(m.AddContext(ctx)).To(Succeed())

		cap := mdlcap.NewCapacity(capID)
		cap.SetDisplayName("Production CPU provided")
		cap.SetCapacityResourceTypeById(crtID)
		cap.SetContextById(ctxID)
		cap.SetCategory(mdlcap.CategoryProvided)
		cap.SetAmount(mdlcap.Amount("64"))

		wire, err := oapi.PushWireEventFromDomain(&events.Event{
			ResourceType: events.CapacityResource,
			Operation:    events.CreateOperation,
			ResourceId:   capID,
			Objects:      []any{cap},
		})
		Expect(err).NotTo(HaveOccurred())

		back, err := oapi.ReplicationEventFromWire(m, &wire)
		Expect(err).NotTo(HaveOccurred())
		out, ok := back.Objects[0].(mdlcap.Capacity)
		Expect(ok).To(BeTrue())
		Expect(out.GetCapacityId()).To(Equal(capID))
		Expect(out.GetCapacityResourceTypeId()).To(Equal(crtID))
		Expect(out.GetContextId()).To(Equal(ctxID))
		Expect(string(out.GetCategory())).To(Equal("provided"))
		Expect(string(out.GetAmount())).To(Equal("64"))
	})
})

var _ = Describe("replication wire: phase0 after push/decode", func() {
	It("does not emit ContextTypeMissing or NodeTypeMissing for clean domain refs", func() {
		m := replicationTestModel()
		ctid := uuid.New()
		ntid := uuid.New()
		ct := mdlctx.NewContextType(ctid)
		ct.SetDisplayName("ct")
		Expect(m.AddContextType(ct)).To(Succeed())
		nt := node.NewNodeType(ntid)
		nt.SetDisplayName("nt")
		Expect(m.AddNodeType(nt)).To(Succeed())
		parentID := uuid.New()
		parent := mdlctx.NewContext(parentID)
		parent.SetDisplayName("p")
		Expect(m.AddContext(parent)).To(Succeed())

		cid := uuid.New()
		c := mdlctx.NewContext(cid)
		c.SetContextTypeById(ctid)
		c.SetParentById(parentID)
		wireCtx, err := oapi.PushWireEventFromDomain(&events.Event{
			ResourceType: events.ContextResource,
			Operation:    events.CreateOperation,
			ResourceId:   cid,
			Objects:      []any{c},
		})
		Expect(err).NotTo(HaveOccurred())
		evCtx, err := oapi.ReplicationEventFromWire(m, &wireCtx)
		Expect(err).NotTo(HaveOccurred())

		nid := uuid.New()
		n := node.NewNode(nid)
		n.SetNodeTypeByRef(nt)
		wireNode, err := oapi.PushWireEventFromDomain(&events.Event{
			ResourceType: events.NodeResource,
			Operation:    events.CreateOperation,
			ResourceId:   nid,
			Objects:      []any{n},
		})
		Expect(err).NotTo(HaveOccurred())
		evNode, err := oapi.ReplicationEventFromWire(m, &wireNode)
		Expect(err).NotTo(HaveOccurred())

		fn := phase0.NewFilterFunc()
		fn(m, evCtx)
		fn(m, evNode)

		Expect(replicationFindingsOfKind(m, finding.ContextTypeMissing)).To(BeEmpty())
		Expect(replicationFindingsOfKind(m, finding.NodeTypeMissing)).To(BeEmpty())
	})
})
