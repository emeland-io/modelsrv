package model_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/system"
)

var _ = Describe("EventApplier.Apply (replication)", func() {
	var (
		sink events.EventSink
		m    model.Model
	)

	BeforeEach(func() {
		sink = events.NewListSink()
		md, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())
		m = md
	})

	When("operation is create or update", func() {
		It("adds a system when Objects holds a System", func() {
			sysID := uuid.New()
			sys := system.NewSystem(sink, sysID)
			sys.SetDisplayName("replicated-system")

			err := m.Apply(events.Event{
				ResourceType: events.SystemResource,
				Operation:    events.CreateOperation,
				ResourceId:   sysID,
				Objects:      []any{sys},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(m.GetSystemById(sysID)).NotTo(BeNil())
			Expect(m.GetSystemById(sysID).GetDisplayName()).To(Equal("replicated-system"))
		})

		It("treats update like create for an existing system id", func() {
			sysID := uuid.New()
			first := system.NewSystem(sink, sysID)
			first.SetDisplayName("v1")
			Expect(m.Apply(events.Event{
				ResourceType: events.SystemResource,
				Operation:    events.CreateOperation,
				ResourceId:   sysID,
				Objects:      []any{first},
			})).To(Succeed())

			second := system.NewSystem(sink, sysID)
			second.SetDisplayName("v2")
			err := m.Apply(events.Event{
				ResourceType: events.SystemResource,
				Operation:    events.UpdateOperation,
				ResourceId:   sysID,
				Objects:      []any{second},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(m.GetSystemById(sysID).GetDisplayName()).To(Equal("v2"))
		})

		It("returns an error when Objects is empty", func() {
			err := m.Apply(events.Event{
				ResourceType: events.SystemResource,
				Operation:    events.CreateOperation,
				ResourceId:   uuid.New(),
				Objects:      nil,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing resource object"))
		})

		It("returns an error when the object type does not match the resource type", func() {
			err := m.Apply(events.Event{
				ResourceType: events.SystemResource,
				Operation:    events.CreateOperation,
				ResourceId:   uuid.New(),
				Objects:      []any{"not-a-system"},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("replication object for System"))
		})

		It("returns an error for an unsupported upsert resource type", func() {
			err := m.Apply(events.Event{
				ResourceType: events.NodeResource,
				Operation:    events.CreateOperation,
				ResourceId:   uuid.New(),
				Objects:      []any{struct{}{}},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported resource type for upsert"))
		})
	})

	When("operation is delete", func() {
		It("removes an existing system", func() {
			sysID := uuid.New()
			sys := system.NewSystem(sink, sysID)
			sys.SetDisplayName("to-delete")
			Expect(m.AddSystem(sys)).To(Succeed())

			err := m.Apply(events.Event{
				ResourceType: events.SystemResource,
				Operation:    events.DeleteOperation,
				ResourceId:   sysID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(m.GetSystemById(sysID)).To(BeNil())
		})

		It("succeeds when the resource is already absent (idempotent replication)", func() {
			missing := uuid.New()
			err := m.Apply(events.Event{
				ResourceType: events.SystemResource,
				Operation:    events.DeleteOperation,
				ResourceId:   missing,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error for an unsupported delete resource type", func() {
			err := m.Apply(events.Event{
				ResourceType: events.NodeResource,
				Operation:    events.DeleteOperation,
				ResourceId:   uuid.New(),
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported resource type for delete"))
		})
	})

	When("operation is not supported", func() {
		It("returns an error", func() {
			err := m.Apply(events.Event{
				ResourceType: events.SystemResource,
				Operation:    events.UnknownOperation,
				ResourceId:   uuid.New(),
				Objects:      []any{system.NewSystem(sink, uuid.New())},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported operation"))
		})
	})
})
