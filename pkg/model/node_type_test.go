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

var _ = Describe("NodeType functionalities", func() {
	var (
		nodeTypeId uuid.UUID
		sinkMock   *mocks.MockEventSink
		nodeType   model.NodeType
	)

	BeforeEach(func() {
		nodeTypeId = uuid.New()
		sinkMock = mocks.NewMockEventSink(gomock.NewController(GinkgoT()))
		nodeType = model.NewNodeType(sinkMock, nodeTypeId)
	})

	When("NodeType is created", func() {
		It("must not be nil", func() {
			Expect(nodeType).NotTo(BeNil())
		})

		It("has the provided UUID", func() {
			Expect(nodeType.GetNodeTypeId()).To(Equal(nodeTypeId))
		})

		It("has annotations set", func() {
			Expect(nodeType.GetAnnotations()).NotTo(BeNil())
		})
	})

	When("NodeType is updated", func() {
		Context("NodeType is not registered", func() {
			When("Display name gets updated", func() {
				It("updates the display name without emitting events", func() {
					nodeType.SetDisplayName("Test Node Type")
					Expect(nodeType.GetDisplayName()).To(Equal("Test Node Type"))
				})
			})

			When("Description gets updated", func() {
				It("updates the description without emitting events", func() {
					nodeType.SetDescription("a test node type")
					Expect(nodeType.GetDescription()).To(Equal("a test node type"))
				})
			})
		})
	})

	When("NodeType is updated", func() {
		Context("NodeType is registered", func() {
			BeforeEach(func() {
				nodeType.Register()
			})

			When("DisplayName gets updated", func() {
				It("emits an event and calls Receive", func() {
					expectedEventType := events.UpdateOperation
					expectedResourceType := events.NodeTypeResource

					sinkMock.EXPECT().Receive(expectedResourceType, expectedEventType, nodeType.GetNodeTypeId(), gomock.Any())

					nodeType.SetDisplayName("FooBar")
				})
			})

			When("Description gets updated", func() {
				It("emits an event and calls Receive", func() {
					sinkMock.EXPECT().Receive(events.NodeTypeResource, events.UpdateOperation, nodeType.GetNodeTypeId(), gomock.Any())

					nodeType.SetDescription("a test description")
				})
			})
		})
	})
})

// var _ = Describe("NodeType operations with model", func() {
// 	var (
// 		sink      *events.ListSink
// 		testModel model.Model
// 	)

// 	BeforeEach(func() {
// 		var err error
// 		sink = events.NewListSink()
// 		testModel, err = model.NewModel(sink)
// 		Expect(err).NotTo(HaveOccurred())
// 	})

// 	It("supports full CRUD lifecycle with correct event sequence", func() {
// 		nodeTypeId := uuid.New()
// 		nodeType := model.NewNodeTypeForModel(testModel, nodeTypeId)

// 		nodeType.SetDisplayName("Test Node Type")
// 		nodeType.SetDescription("a test node type")
// 		Expect(testModel.GetNodeTypeById(nodeTypeId)).To(BeNil())

// 		err := testModel.AddNodeType(nodeType)
// 		Expect(err).NotTo(HaveOccurred())
// 		Expect(testModel.GetNodeTypeById(nodeTypeId)).To(Equal(nodeType))

// 		nodeType.SetDisplayName("the real test node type")
// 		nodeType.SetDescription("a test node type, but with more bla bla")

// 		nodeType2 := model.NewNodeTypeForModel(testModel, nodeTypeId)
// 		nodeType2.SetDisplayName("The other Test Node Type")
// 		nodeType2.SetDescription("a different test node type, but same Id")
// 		err = testModel.AddNodeType(nodeType2)
// 		Expect(err).NotTo(HaveOccurred())

// 		err = testModel.DeleteNodeTypeById(nodeTypeId)
// 		Expect(err).NotTo(HaveOccurred())
// 		Expect(testModel.GetNodeTypeById(nodeTypeId)).To(BeNil())

// 		expectedEvents := []struct {
// 			resourceType events.ResourceType
// 			operation    events.Operation
// 			resourceId   uuid.UUID
// 		}{
// 			{events.NodeTypeResource, events.CreateOperation, nodeTypeId},
// 			{events.NodeTypeResource, events.UpdateOperation, nodeTypeId},
// 			{events.NodeTypeResource, events.UpdateOperation, nodeTypeId},
// 			{events.NodeTypeResource, events.UpdateOperation, nodeTypeId},
// 			{events.NodeTypeResource, events.DeleteOperation, nodeTypeId},
// 		}

// 		actualEvents := sink.GetEvents()
// 		Expect(actualEvents).To(HaveLen(len(expectedEvents)))
// 		for i, expected := range expectedEvents {
// 			Expect(actualEvents[i].ResourceType).To(Equal(expected.resourceType))
// 			Expect(actualEvents[i].Operation).To(Equal(expected.operation))
// 			Expect(actualEvents[i].ResourceId).To(Equal(expected.resourceId))
// 		}
// 	})
// })
