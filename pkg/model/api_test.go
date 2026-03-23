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

var _ = Describe("API functionalities", func() {
	var (
		apiId     uuid.UUID
		sinkMock  *mocks.MockEventSink
		testModel model.Model
		api       model.API
	)

	BeforeEach(func() {
		apiId = uuid.New()

		sinkMock = mocks.NewMockEventSink(gomock.NewController(GinkgoT()))
		m, err := model.NewModel(sinkMock)
		Expect(err).NotTo(HaveOccurred())
		testModel = m
		api = model.NewAPI(testModel, apiId)
	})

	When("API is created", func() {
		It("must not be nil", func() {
			Expect(api).NotTo(BeNil())
		})

		It("has the provided UUID", func() {
			Expect(api.GetApiId()).To(Equal(apiId))
		})

		It("has annotations set", func() {
			Expect(api.GetAnnotations()).NotTo(BeNil())
		})
	})

	When("API is updated", func() {
		Context("API is not registered", func() {

			When("Display name gets updated", func() {
				It("updates the display name", func() {
					displayName := "API-007"
					api.SetDisplayName(displayName)

					Expect(api.GetDisplayName()).To(Equal(displayName))
				})
			})

			When("Description gets updated", func() {
				It("updates the description", func() {
					description := "This is a test API."
					api.SetDescription(description)

					Expect(api.GetDescription()).To(Equal(description))
				})
			})

			When("Version gets updated", func() {
				It("updates the version", func() {
					newVersion := model.Version{Version: "2.0.0"}
					api.SetVersion(newVersion)

					Expect(api.GetVersion()).To(Equal(newVersion))
				})
			})

			When("Type gets updated", func() {
				It("updates the type", func() {
					api.SetType(model.GraphQL)

					Expect(api.GetType()).To(Equal(model.GraphQL))
				})
			})

			When("System gets updated", func() {
				It("updates the system by ref", func() {
					system := model.NewSystem(sinkMock, uuid.New())
					api.SetSystemByRef(system)

					Expect(api.GetSystem()).NotTo(BeNil())
					Expect(api.GetSystem().System).To(Equal(system))
					Expect(api.GetSystem().SystemId).To(Equal(system.GetSystemId()))
				})
			})
		})
	})

	When("API is updated", func() {
		Context("API is registered", func() {

			BeforeEach(func() {
				api.Register()
			})

			When("DisplayName gets updated", func() {
				It("emits an event and calls Receive", func() {
					expectedEventType := events.UpdateOperation
					expectedResourceType := events.APIResource

					sinkMock.EXPECT().Receive(expectedResourceType, expectedEventType, api.GetApiId(), gomock.Any())

					api.SetDisplayName("FooBar")
				})
			})
		})
	})
})
