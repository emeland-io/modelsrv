package oapi_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.emeland.io/modelsrv/pkg/model/node"
	"go.emeland.io/modelsrv/pkg/model/system"
)

func replicationTestModel() model.Model {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	Expect(err).NotTo(HaveOccurred())
	return m
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

	It("embeds the JSON-marshaled system fields in Resource for create", func() {
		m := replicationTestModel()
		sysID := uuid.New()
		sys := system.NewSystem(m.GetSink(), sysID)
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
		Expect(*wire.Resource).To(HaveKeyWithValue("DisplayName", "push-name"))
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
		parent := system.NewSystem(m.GetSink(), parentID)
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
		nt := node.NewNodeType(m.GetSink(), ntid)
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

	It("coalesces Finding nested type ref", func() {
		m := replicationTestModel()
		fid := uuid.New()
		ftid := uuid.New()
		res := map[string]interface{}{
			"findingId": fid.String(),
			"summary":   "sum",
			"resources": []interface{}{},
			"Type":      map[string]interface{}{"findingTypeId": ftid.String()},
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
	It("preserves system id and display name", func() {
		m := replicationTestModel()
		sysID := uuid.New()
		sys := system.NewSystem(m.GetSink(), sysID)
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
})
