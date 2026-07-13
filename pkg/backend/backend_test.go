package backend_test

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.emeland.io/modelsrv/pkg/backend"
	"go.emeland.io/modelsrv/pkg/eventfilter"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.emeland.io/modelsrv/pkg/model/node"
)

var _ = Describe("Backend", func() {
	Describe("New", func() {
		It("constructs without error", func() {
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())
			Expect(b).NotTo(BeNil())
		})

		It("exposes a non-nil Model", func() {
			b, _ := backend.New()
			Expect(b.GetModel()).NotTo(BeNil())
		})

		It("exposes a non-nil Chain", func() {
			b, _ := backend.New()
			Expect(b.GetChain()).NotTo(BeNil())
		})

		It("exposes a non-nil EventManager", func() {
			b, _ := backend.New()
			Expect(b.GetEventManager()).NotTo(BeNil())
		})

		It("registers well-known FindingTypes with descriptions at startup", func() {
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())

			ft := b.GetModel().GetFindingTypeById(finding.TypeIDForKind(finding.NodeTypeMissing))
			Expect(ft).NotTo(BeNil())
			Expect(ft.GetDescription()).To(Equal(finding.DescriptionForKind(finding.NodeTypeMissing)))
		})

		It("registers phase0 as a discoverable FilterRule", func() {
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())

			rules, err := b.GetModel().GetFilterRules()
			Expect(err).NotTo(HaveOccurred())
			Expect(rules).NotTo(BeEmpty())

			var phase0Rule bool
			for _, rule := range rules {
				if rule.GetDisplayName() == "Phase 0 referential integrity" {
					phase0Rule = true
					Expect(rule.GetDescription()).To(ContainSubstring("referential integrity"))
				}
			}
			Expect(phase0Rule).To(BeTrue())
		})

		It("registers static MergeRules", func() {
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())

			rules, err := b.GetModel().GetMergeRules()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(rules)).To(BeNumerically(">=", 2))
		})
	})

	Describe("wiring: model mutations flow through the filter chain", func() {
		It("a registered FilterFunc receives events emitted by model mutations", func() {
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())

			var intercepted []events.Event
			b.GetChain().Register(func(_ model.Model, ev events.Event) []events.Event {
				intercepted = append(intercepted, ev)
				return []events.Event{ev}
			})

			sys := model.MakeTestSystem(uuid.New(), "test-sys", common.Version{})
			Expect(b.GetModel().AddSystem(sys)).To(Succeed())

			var sawSystem bool
			for _, ev := range intercepted {
				if ev.ResourceType == events.SystemResource {
					sawSystem = true
					break
				}
			}
			Expect(sawSystem).To(BeTrue())
		})

		It("the chain receives the same Model instance that Backend constructed", func() {
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())

			var receivedModel model.Model
			b.GetChain().Register(func(m model.Model, ev events.Event) []events.Event {
				receivedModel = m
				return []events.Event{ev}
			})

			sys := model.MakeTestSystem(uuid.New(), "check-model", common.Version{})
			Expect(b.GetModel().AddSystem(sys)).To(Succeed())

			Expect(receivedModel).To(BeIdenticalTo(b.GetModel()))
		})

		It("a filter can expand model events with additional events", func() {
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())

			var count int
			b.GetChain().Register(func(_ model.Model, ev events.Event) []events.Event {
				count++
				extra := events.Event{
					ResourceType: events.FindingResource,
					Operation:    events.CreateOperation,
					ResourceId:   uuid.New(),
				}
				return []events.Event{ev, extra}
			})

			seqBefore, err := b.GetEventManager().GetCurrentSequenceId(context.Background())
			Expect(err).NotTo(HaveOccurred())

			sys := model.MakeTestSystem(uuid.New(), "expand-sys", common.Version{})
			Expect(b.GetModel().AddSystem(sys)).To(Succeed())

			Expect(count).To(BeNumerically(">", 0))
			seqAfter, err := b.GetEventManager().GetCurrentSequenceId(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(seqAfter - seqBefore).To(Equal(uint64(2))) // system creation + expanded finding
		})
	})

	Describe("phase0 finding negation", func() {
		It("propagates a Finding delete through the chain when a missing ContextType is added", func() {
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())

			var intercepted []events.Event
			b.GetChain().Register(func(_ model.Model, ev events.Event) []events.Event {
				intercepted = append(intercepted, ev)
				return []events.Event{ev}
			})

			m := b.GetModel()
			typeID := uuid.New()
			ctx := mdlctx.NewContext(uuid.New())
			ctx.SetContextTypeById(typeID)
			Expect(m.AddContext(ctx)).To(Succeed())

			ct := mdlctx.NewContextType(typeID)
			ct.SetDisplayName("t")
			Expect(m.AddContextType(ct)).To(Succeed())

			var sawFindingDelete bool
			for _, ev := range intercepted {
				if ev.ResourceType == events.FindingResource && ev.Operation == events.DeleteOperation {
					sawFindingDelete = true
					break
				}
			}
			Expect(sawFindingDelete).To(BeTrue())
		})

		It("propagates a Finding delete through the chain when a missing NodeType is added", func() {
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())

			var intercepted []events.Event
			b.GetChain().Register(func(_ model.Model, ev events.Event) []events.Event {
				intercepted = append(intercepted, ev)
				return []events.Event{ev}
			})

			m := b.GetModel()
			typeID := uuid.New()
			n := node.NewNode(uuid.New())
			n.SetNodeTypeById(typeID)
			Expect(m.AddNode(n)).To(Succeed())

			nt := node.NewNodeType(typeID)
			nt.SetDisplayName("t")
			Expect(m.AddNodeType(nt)).To(Succeed())

			var sawFindingDelete bool
			for _, ev := range intercepted {
				if ev.ResourceType == events.FindingResource && ev.Operation == events.DeleteOperation {
					sawFindingDelete = true
					break
				}
			}
			Expect(sawFindingDelete).To(BeTrue())
		})
	})

	Describe("Chain.SetModel", func() {
		It("updates the model reference used by subsequent Apply calls", func() {
			chain := eventfilter.NewChain(nil)

			var capturedModel model.Model
			chain.Register(func(m model.Model, ev events.Event) []events.Event {
				capturedModel = m
				return []events.Event{ev}
			})

			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())

			chain.SetModel(b.GetModel())
			chain.Apply(events.Event{
				ResourceType: events.SystemResource,
				Operation:    events.CreateOperation,
				ResourceId:   uuid.New(),
			})

			Expect(capturedModel).To(BeIdenticalTo(b.GetModel()))
		})

		It("is safe to call after filters are already registered", func() {
			chain := eventfilter.NewChain(nil)
			chain.Register(func(_ model.Model, ev events.Event) []events.Event {
				return []events.Event{ev}
			})

			b, _ := backend.New()
			Expect(func() { chain.SetModel(b.GetModel()) }).NotTo(Panic())

			result := chain.Apply(events.Event{
				ResourceType: events.NodeResource,
				Operation:    events.CreateOperation,
				ResourceId:   uuid.New(),
			})
			Expect(result).To(HaveLen(1))
		})
	})
})
