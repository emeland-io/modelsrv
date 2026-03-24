// pkg/model/context_test.go
package model_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/mocks"
	"go.emeland.io/modelsrv/pkg/model"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Context functionalities", func() {
	var (
		contextId uuid.UUID
		sinkMock  *mocks.MockEventSink
		ctx       model.Context
	)

	BeforeEach(func() {
		contextId = uuid.New()

		sinkMock = mocks.NewMockEventSink(gomock.NewController(GinkgoT()))
		ctx = model.NewContext(sinkMock, contextId)
	})

	When("Context is created", func() {
		It("must not be nil", func() {
			Expect(ctx).NotTo(BeNil())
		})

		It("has the provided UUID", func() {
			Expect(ctx.GetContextId()).To(Equal(contextId))
		})

		It("has annotations set", func() {
			Expect(ctx.GetAnnotations()).NotTo(BeNil())
		})
	})

	When("Context is updated", func() {
		Context("Context is not registered", func() {

			When("Display name gets updated", func() {
				It("updates the display name", func() {
					displayName := "Context-007"
					ctx.SetDisplayName(displayName)

					Expect(ctx.GetDisplayName()).To(Equal(displayName))
				})
			})

			When("Description gets updated", func() {
				It("updates the description", func() {
					description := "This is a test context."
					ctx.SetDescription(description)

					Expect(ctx.GetDescription()).To(Equal(description))
				})
			})

			When("Parent gets updated", func() {
				It("updates the parent by ref", func() {
					parentContext := model.NewContext(sinkMock, uuid.New())
					ctx.SetParentByRef(parentContext)

					retrievedParent, err := ctx.GetParent()
					Expect(err).NotTo(HaveOccurred())
					Expect(retrievedParent).NotTo(BeNil())
					Expect(retrievedParent.GetContextId()).To(Equal(parentContext.GetContextId()))
				})
			})
		})
	})

	When("Context is updated", func() {
		Context("Context is registered", func() {

			BeforeEach(func() {
				ctx.Register()
			})

			When("DisplayName gets updated", func() {
				It("emits an event and calls Receive", func() {
					expectedEventType := events.UpdateOperation
					expectedResourceType := events.ContextResource

					sinkMock.EXPECT().Receive(expectedResourceType, expectedEventType, ctx.GetContextId(), gomock.Any())

					ctx.SetDisplayName("FooBar")
				})
			})
		})
	})
})
