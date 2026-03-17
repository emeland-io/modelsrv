package eventforwarder

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

var _ = Describe("Event Forwarder", func() {
	var (
		forwarder *eventForwarder
		testModel model.Model
		contextId uuid.UUID
		ctx       model.Context
	)

	BeforeEach(func() {
		var err error
		forwarder = NewEventForwarder(4)
		testModel, err = model.NewModel(forwarder)
		Expect(err).NotTo(HaveOccurred())

		contextId = uuid.New()
		ctx = model.NewContext(forwarder, contextId)
		ctx.SetDisplayName("Test Context")
		ctx.SetDescription("a test context")
		ctx.GetAnnotations().Add("a key", "a value")
	})

	When("a context is added to the model", func() {
		BeforeEach(func() {
			err := testModel.AddContext(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("enqueues a CreateOperation event", func() {
			event, err := forwarder.queue.Dequeue()
			Expect(err).NotTo(HaveOccurred())

			Expect(event.resourceType).To(Equal(events.ContextResource))
			Expect(event.operation).To(Equal(events.CreateOperation))
			Expect(event.resourceId).To(Equal(contextId))
		})

		It("serializes the context in the event payload", func() {
			event, err := forwarder.queue.Dequeue()
			Expect(err).NotTo(HaveOccurred())

			Expect(event.objectJson).To(HaveLen(1))
			Expect(event.objectJson[0]).To(ContainSubstring(`"contextId":"` + contextId.String() + `"`))
			Expect(event.objectJson[0]).To(ContainSubstring(`"displayName":"Test Context"`))
			Expect(event.objectJson[0]).To(ContainSubstring(`"description":"a test context"`))
			Expect(event.objectJson[0]).To(ContainSubstring(`"annotations":[{"key":"a key","value":"a value"}]`))
		})
	})

	When("Receive is called with invalid or unsupported input", func() {
		It("returns error for unknown type", func() {
			err := forwarder.Receive(events.ContextResource, events.CreateOperation, contextId, 42)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown type"))
		})

		It("returns error when object is invalid JSON passed as non-string type", func() {
			err := forwarder.Receive(events.ContextResource, events.CreateOperation, contextId, struct{ X int }{1})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown type"))
		})

		It("passes through plain JSON string without validation", func() {
			plainJSON := `{"plain": json}`
			err := forwarder.Receive(events.ContextResource, events.CreateOperation, contextId, plainJSON)
			Expect(err).NotTo(HaveOccurred())
			event, err := forwarder.queue.Dequeue()
			Expect(err).NotTo(HaveOccurred())
			Expect(event.objectJson).To(HaveLen(1))
			Expect(event.objectJson[0]).To(Equal(plainJSON))
		})
	})
})
