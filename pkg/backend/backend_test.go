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

			sys := model.MakeTestSystem(b.GetModel().GetSink(), uuid.New(), "test-sys", common.Version{})
			Expect(b.GetModel().AddSystem(sys)).To(Succeed())

			Expect(intercepted).NotTo(BeEmpty())
			Expect(intercepted[0].ResourceType).To(Equal(events.SystemResource))
		})

		It("the chain receives the same Model instance that Backend constructed", func() {
			b, err := backend.New()
			Expect(err).NotTo(HaveOccurred())

			var receivedModel model.Model
			b.GetChain().Register(func(m model.Model, ev events.Event) []events.Event {
				receivedModel = m
				return []events.Event{ev}
			})

			sys := model.MakeTestSystem(b.GetModel().GetSink(), uuid.New(), "check-model", common.Version{})
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

			sys := model.MakeTestSystem(b.GetModel().GetSink(), uuid.New(), "expand-sys", common.Version{})
			Expect(b.GetModel().AddSystem(sys)).To(Succeed())

			Expect(count).To(BeNumerically(">", 0))
			seqID, err := b.GetEventManager().GetCurrentSequenceId(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(seqID).To(Equal(uint64(2))) // 1 for the system creation, 1 for the finding creation
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
