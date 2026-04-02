// pkg/model/system_test.go
package system_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/mocks"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/system"
	"go.uber.org/mock/gomock"
)

var _ = Describe("System functionalities", func() {
	var (
		sysId    uuid.UUID
		sinkMock *mocks.MockEventSink
		sys      system.System
	)

	BeforeEach(func() {
		sysId = uuid.New()

		sinkMock = mocks.NewMockEventSink(gomock.NewController(GinkgoT()))
		sys = system.NewSystem(sinkMock, sysId)
	})

	When("System is created", func() {
		It("must not be nil", func() {
			Expect(sys).NotTo(BeNil())
		})

		It("has the provided UUID", func() {
			Expect(sys.GetSystemId()).To(Equal(sysId))
		})

		It("has annotations set", func() {
			Expect(sys.GetAnnotations()).NotTo(BeNil())
		})
	})

	When("System is updated", func() {
		Context("System is not registered", func() {

			When("Display name gets updated", func() {

				It("updates the display name", func() {
					displayName := "System-007"
					sys.SetDisplayName(displayName)

					Expect(sys.GetDisplayName()).To(Equal(displayName))
				})
			})

			When("Description gets updated", func() {

				It("updates the description", func() {
					description := "This is a test system."
					sys.SetDescription(description)

					Expect(sys.GetDescription()).To(Equal(description))
				})
			})

			When("Version gets updated", func() {
				It("updates the version", func() {
					newPatchVersion := common.Version{Version: "1.0.1"}
					sys.SetVersion(newPatchVersion)

					Expect(sys.GetVersion()).To(Equal(newPatchVersion))
				})
			})

			When("Abstract gets updated", func() {
				It("updates the abstract", func() {
					abs := true
					sys.SetAbstract(abs)

					Expect(sys.GetAbstract()).To(Equal(abs))
				})
			})

			When("Parent gets updated", func() {
				It("updates the parent by ref", func() {
					parentSystem := system.NewSystem(sinkMock, uuid.New())
					sys.SetParentByRef(parentSystem)

					retrievedParent, err := sys.GetParent()
					Expect(err).NotTo(HaveOccurred())
					Expect(retrievedParent).NotTo(BeNil())
					Expect(retrievedParent.GetSystemId()).To(Equal(parentSystem.GetSystemId()))
				})
			})
		})
	})

	When("System is updated", func() {

		Context("System is registered", func() {

			BeforeEach(func() {
				sys.Register()
			})

			When("DisplayName gets updated", func() {

				It("emits an event and calls Receive", func() {
					expectedEventType := events.UpdateOperation
					expectedResourceType := events.SystemResource

					sinkMock.EXPECT().Receive(expectedResourceType, expectedEventType, sys.GetSystemId(), gomock.Any())

					sys.SetDisplayName("FooBar")
					// Update already tested above
				})
			})
		})
	})
})
