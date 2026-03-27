package context_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/mocks"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Context functionalities", func() {
	var (
		contextId uuid.UUID
		sinkMock  *mocks.MockEventSink
		testCtx   mdlctx.Context
	)

	BeforeEach(func() {
		contextId = uuid.New()

		sinkMock = mocks.NewMockEventSink(gomock.NewController(GinkgoT()))
		testCtx = mdlctx.NewContext(sinkMock, contextId)
	})

	When("Context is created", func() {
		It("must not be nil", func() {
			Expect(testCtx).NotTo(BeNil())
		})

		It("has the provided UUID", func() {
			Expect(testCtx.GetContextId()).To(Equal(contextId))
		})

		It("has annotations set", func() {
			Expect(testCtx.GetAnnotations()).NotTo(BeNil())
		})
	})

	When("Context is updated", func() {
		Context("Context is not registered", func() {

			When("Display name gets updated", func() {
				It("updates the display name", func() {
					displayName := "Context-007"
					testCtx.SetDisplayName(displayName)

					Expect(testCtx.GetDisplayName()).To(Equal(displayName))
				})
			})

			When("Description gets updated", func() {
				It("updates the description", func() {
					description := "This is a test context."
					testCtx.SetDescription(description)

					Expect(testCtx.GetDescription()).To(Equal(description))
				})
			})

			When("Parent gets updated", func() {
				It("updates the parent by ref", func() {
					parentContext := mdlctx.NewContext(sinkMock, uuid.New())
					testCtx.SetParentByRef(parentContext)

					retrievedParent, err := testCtx.GetParent()
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
				testCtx.Register()
			})

			When("DisplayName gets updated", func() {
				It("emits an event and calls Receive", func() {
					expectedEventType := events.UpdateOperation
					expectedResourceType := events.ContextResource

					sinkMock.EXPECT().Receive(expectedResourceType, expectedEventType, testCtx.GetContextId(), gomock.Any())

					testCtx.SetDisplayName("FooBar")
				})
			})
		})
	})
})
