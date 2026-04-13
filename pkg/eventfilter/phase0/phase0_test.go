package phase0_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.emeland.io/modelsrv/pkg/eventfilter/phase0"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.emeland.io/modelsrv/pkg/model/node"
)

// newModel creates a model backed by a no-op sink, suitable for filter tests
// that drive the FilterFunc directly without a Chain.
func newModel() model.Model {
	m, err := model.NewModel(events.NewDummySink())
	Expect(err).NotTo(HaveOccurred())
	return m
}

// applyFilter calls the phase0 FilterFunc with the given event and returns
// the resulting event slice (always the original event passed through).
func applyFilter(m model.Model, ev events.Event) []events.Event {
	fn := phase0.NewFilterFunc()
	return fn(m, ev)
}

// contextEvent builds a Create event for a Context object.
func contextEvent(ctx mdlctx.Context) events.Event {
	return events.Event{
		ResourceType: events.ContextResource,
		Operation:    events.CreateOperation,
		ResourceId:   ctx.GetContextId(),
		Objects:      []any{ctx},
	}
}

// nodeEvent builds a Create event for a Node object.
func nodeEvent(n node.Node) events.Event {
	return events.Event{
		ResourceType: events.NodeResource,
		Operation:    events.CreateOperation,
		ResourceId:   n.GetNodeId(),
		Objects:      []any{n},
	}
}

// findingsOfKind returns all findings in m whose FindingType UUID matches the
// well-known type for the given FindingKind.
func findingsOfKind(m model.Model, kind finding.FindingKind) []finding.Finding {
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

var _ = Describe("phase0 FilterFunc", func() {
	Describe("Context checks", func() {
		Describe("ContextTypeMissing", func() {
			It("creates a ContextTypeMissing finding when the context has no type set", func() {
				m := newModel()
				ctx := mdlctx.NewContext(events.NewDummySink(), uuid.New())

				applyFilter(m, contextEvent(ctx))

				Expect(findingsOfKind(m, finding.ContextTypeMissing)).To(HaveLen(1))
			})

			It("creates a ContextTypeMissing finding when the referenced ContextType does not exist in the model", func() {
				m := newModel()
				ctx := mdlctx.NewContext(events.NewDummySink(), uuid.New())
				ctx.SetContextTypeById(uuid.New()) // type UUID not in model

				applyFilter(m, contextEvent(ctx))

				Expect(findingsOfKind(m, finding.ContextTypeMissing)).To(HaveLen(1))
			})

			It("does not create a ContextTypeMissing finding when the ContextType exists", func() {
				m := newModel()

				ct := mdlctx.NewContextType(events.NewDummySink(), uuid.New())
				Expect(m.AddContextType(ct)).To(Succeed())

				ctx := mdlctx.NewContext(events.NewDummySink(), uuid.New())
				ctx.SetContextTypeByRef(ct)

				applyFilter(m, contextEvent(ctx))

				Expect(findingsOfKind(m, finding.ContextTypeMissing)).To(BeEmpty())
			})

			It("removes an existing ContextTypeMissing finding when the type is subsequently resolved", func() {
				m := newModel()

				ct := mdlctx.NewContextType(events.NewDummySink(), uuid.New())

				ctx := mdlctx.NewContext(events.NewDummySink(), uuid.New())
				// First event: type not yet in model → finding created
				ctx.SetContextTypeById(ct.GetContextTypeId())
				applyFilter(m, contextEvent(ctx))
				Expect(findingsOfKind(m, finding.ContextTypeMissing)).To(HaveLen(1))

				// Now add the type to the model and re-evaluate via an Update event
				Expect(m.AddContextType(ct)).To(Succeed())
				ctx.SetContextTypeByRef(ct)
				updateEv := events.Event{
					ResourceType: events.ContextResource,
					Operation:    events.UpdateOperation,
					ResourceId:   ctx.GetContextId(),
					Objects:      []any{ctx},
				}
				applyFilter(m, updateEv)
				Expect(findingsOfKind(m, finding.ContextTypeMissing)).To(BeEmpty())
			})

			It("is idempotent: applying the filter twice produces exactly one finding", func() {
				m := newModel()
				ctx := mdlctx.NewContext(events.NewDummySink(), uuid.New())

				applyFilter(m, contextEvent(ctx))
				applyFilter(m, contextEvent(ctx))

				Expect(findingsOfKind(m, finding.ContextTypeMissing)).To(HaveLen(1))
			})
		})

		Describe("ContextParentNotFound", func() {
			It("creates a ContextParentNotFound finding when the referenced parent does not exist", func() {
				m := newModel()
				ctx := mdlctx.NewContext(events.NewDummySink(), uuid.New())
				ctx.SetParentById(uuid.New()) // parent UUID not in model

				applyFilter(m, contextEvent(ctx))

				Expect(findingsOfKind(m, finding.ContextParentNotFound)).To(HaveLen(1))
			})

			It("does not create a ContextParentNotFound finding when no parent is set", func() {
				m := newModel()
				ctx := mdlctx.NewContext(events.NewDummySink(), uuid.New())

				applyFilter(m, contextEvent(ctx))

				Expect(findingsOfKind(m, finding.ContextParentNotFound)).To(BeEmpty())
			})

			It("does not create a ContextParentNotFound finding when the parent exists", func() {
				m := newModel()

				parent := mdlctx.NewContext(events.NewDummySink(), uuid.New())
				Expect(m.AddContext(parent)).To(Succeed())

				ctx := mdlctx.NewContext(events.NewDummySink(), uuid.New())
				ctx.SetParentByRef(parent)

				applyFilter(m, contextEvent(ctx))

				Expect(findingsOfKind(m, finding.ContextParentNotFound)).To(BeEmpty())
			})

			It("removes an existing ContextParentNotFound finding when the parent is subsequently added", func() {
				m := newModel()

				parent := mdlctx.NewContext(events.NewDummySink(), uuid.New())
				ctx := mdlctx.NewContext(events.NewDummySink(), uuid.New())

				// First event: parent not yet in model → finding created
				ctx.SetParentById(parent.GetContextId())
				applyFilter(m, contextEvent(ctx))
				Expect(findingsOfKind(m, finding.ContextParentNotFound)).To(HaveLen(1))

				// Add the parent and re-evaluate
				Expect(m.AddContext(parent)).To(Succeed())
				ctx.SetParentByRef(parent)
				updateEv := events.Event{
					ResourceType: events.ContextResource,
					Operation:    events.UpdateOperation,
					ResourceId:   ctx.GetContextId(),
					Objects:      []any{ctx},
				}
				applyFilter(m, updateEv)
				Expect(findingsOfKind(m, finding.ContextParentNotFound)).To(BeEmpty())
			})

			It("removes an existing ContextParentNotFound finding when the parent is cleared", func() {
				m := newModel()
				ctx := mdlctx.NewContext(events.NewDummySink(), uuid.New())

				// Finding present with a missing parent
				ctx.SetParentById(uuid.New())
				applyFilter(m, contextEvent(ctx))
				Expect(findingsOfKind(m, finding.ContextParentNotFound)).To(HaveLen(1))

				// Clear the parent reference
				ctx.SetParentById(uuid.Nil)
				updateEv := events.Event{
					ResourceType: events.ContextResource,
					Operation:    events.UpdateOperation,
					ResourceId:   ctx.GetContextId(),
					Objects:      []any{ctx},
				}
				applyFilter(m, updateEv)
				Expect(findingsOfKind(m, finding.ContextParentNotFound)).To(BeEmpty())
			})
		})

		Describe("both ContextTypeMissing and ContextParentNotFound", func() {
			It("can coexist simultaneously on the same context without UUID collision", func() {
				m := newModel()
				ctx := mdlctx.NewContext(events.NewDummySink(), uuid.New())
				ctx.SetContextTypeById(uuid.New()) // type not in model
				ctx.SetParentById(uuid.New())      // parent not in model

				applyFilter(m, contextEvent(ctx))

				typeMissing := findingsOfKind(m, finding.ContextTypeMissing)
				parentNotFound := findingsOfKind(m, finding.ContextParentNotFound)

				Expect(typeMissing).To(HaveLen(1))
				Expect(parentNotFound).To(HaveLen(1))
				Expect(typeMissing[0].GetFindingId()).NotTo(Equal(parentNotFound[0].GetFindingId()))
			})
		})
	})

	Describe("Node checks", func() {
		Describe("NodeTypeMissing", func() {
			It("creates a NodeTypeMissing finding when the node has no type assigned", func() {
				m := newModel()
				n := node.NewNode(events.NewDummySink(), uuid.New())

				applyFilter(m, nodeEvent(n))

				Expect(findingsOfKind(m, finding.NodeTypeMissing)).To(HaveLen(1))
			})

			It("does not create a NodeTypeMissing finding when the node has a type", func() {
				m := newModel()

				nt := node.NewNodeType(events.NewDummySink(), uuid.New())
				n := node.NewNode(events.NewDummySink(), uuid.New())
				n.SetNodeTypeByRef(nt)

				applyFilter(m, nodeEvent(n))

				Expect(findingsOfKind(m, finding.NodeTypeMissing)).To(BeEmpty())
			})

			It("removes an existing NodeTypeMissing finding when a type is later assigned", func() {
				m := newModel()

				nt := node.NewNodeType(events.NewDummySink(), uuid.New())
				n := node.NewNode(events.NewDummySink(), uuid.New())

				// First event: no type → finding created
				applyFilter(m, nodeEvent(n))
				Expect(findingsOfKind(m, finding.NodeTypeMissing)).To(HaveLen(1))

				// Assign a type and re-evaluate
				n.SetNodeTypeByRef(nt)
				updateEv := events.Event{
					ResourceType: events.NodeResource,
					Operation:    events.UpdateOperation,
					ResourceId:   n.GetNodeId(),
					Objects:      []any{n},
				}
				applyFilter(m, updateEv)
				Expect(findingsOfKind(m, finding.NodeTypeMissing)).To(BeEmpty())
			})
		})
	})

	Describe("pass-through behaviour", func() {
		It("always returns the original event unchanged", func() {
			m := newModel()
			ctx := mdlctx.NewContext(events.NewDummySink(), uuid.New())
			ev := contextEvent(ctx)

			result := applyFilter(m, ev)

			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(Equal(ev))
		})

		It("ignores Delete events and does not mutate findings", func() {
			m := newModel()
			ev := events.Event{
				ResourceType: events.ContextResource,
				Operation:    events.DeleteOperation,
				ResourceId:   uuid.New(),
			}

			result := applyFilter(m, ev)

			Expect(result).To(HaveLen(1))
			all, _ := m.GetFindings()
			Expect(all).To(BeEmpty())
		})

		It("ignores events for unrelated resource types", func() {
			m := newModel()
			ev := events.Event{
				ResourceType: events.SystemResource,
				Operation:    events.CreateOperation,
				ResourceId:   uuid.New(),
			}

			result := applyFilter(m, ev)

			Expect(result).To(HaveLen(1))
			all, _ := m.GetFindings()
			Expect(all).To(BeEmpty())
		})
	})
})
