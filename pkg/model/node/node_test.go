package node_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/mocks"
	"go.emeland.io/modelsrv/pkg/model"
	nodemdl "go.emeland.io/modelsrv/pkg/model/node"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Node functionalities", func() {
	var (
		nodeId   uuid.UUID
		sinkMock *mocks.MockEventSink
		node     nodemdl.Node
	)

	BeforeEach(func() {
		nodeId = uuid.New()
		sinkMock = mocks.NewMockEventSink(gomock.NewController(GinkgoT()))
		node = nodemdl.NewNode(sinkMock, nodeId)
	})

	When("Node is created", func() {
		It("must not be nil", func() {
			Expect(node).NotTo(BeNil())
		})

		It("has the provided UUID", func() {
			Expect(node.GetNodeId()).To(Equal(nodeId))
		})

		It("has annotations set", func() {
			Expect(node.GetAnnotations()).NotTo(BeNil())
		})
	})

	When("Node is updated", func() {
		Context("Node is not registered", func() {
			When("Display name gets updated", func() {
				It("updates the display name without emitting events", func() {
					node.SetDisplayName("Test Node")
					Expect(node.GetDisplayName()).To(Equal("Test Node"))
				})
			})

			When("Description gets updated", func() {
				It("updates the description without emitting events", func() {
					node.SetDescription("a test node")
					Expect(node.GetDescription()).To(Equal("a test node"))
				})
			})

			When("Node type gets updated", func() {
				It("updates the node type by ref", func() {
					nodeType := nodemdl.NewNodeType(sinkMock, uuid.New())
					node.SetNodeTypeByRef(nodeType)

					retrievedType, err := node.GetNodeType()
					Expect(err).NotTo(HaveOccurred())
					Expect(retrievedType).NotTo(BeNil())
					Expect(retrievedType.GetNodeTypeId()).To(Equal(nodeType.GetNodeTypeId()))
				})
			})
		})
	})

	When("Node is updated", func() {
		Context("Node is registered", func() {
			BeforeEach(func() {
				node.Register()
			})

			When("DisplayName gets updated", func() {
				It("emits an event and calls Receive", func() {
					expectedEventType := events.UpdateOperation
					expectedResourceType := events.NodeResource

					sinkMock.EXPECT().Receive(expectedResourceType, expectedEventType, node.GetNodeId(), gomock.Any())

					node.SetDisplayName("FooBar")
				})
			})

			When("Description gets updated", func() {
				It("emits an event and calls Receive", func() {
					sinkMock.EXPECT().Receive(events.NodeResource, events.UpdateOperation, node.GetNodeId(), gomock.Any())

					node.SetDescription("a test description")
				})
			})
		})
	})
})

var _ = Describe("Node operations with model", func() {
	var (
		sink      *events.ListSink
		testModel model.Model
	)

	BeforeEach(func() {
		var err error
		sink = events.NewListSink()
		testModel, err = model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())
	})

	It("supports full CRUD lifecycle with correct event sequence", func() {
		nodeId := uuid.New()
		n1 := nodemdl.NewNode(testModel.GetSink(), nodeId)

		n1.SetDisplayName("Test Node")
		n1.SetDescription("a test node")
		Expect(testModel.GetNodeById(nodeId)).To(BeNil())

		err := testModel.AddNode(n1)
		Expect(err).NotTo(HaveOccurred())
		Expect(testModel.GetNodeById(nodeId)).To(Equal(n1))

		n1.SetDisplayName("the real test node")
		n1.SetDescription("a test node, but with more bla bla")

		node2 := nodemdl.NewNode(testModel.GetSink(), nodeId)
		node2.SetDisplayName("The other Test Node")
		node2.SetDescription("a different test node, but same Id")
		err = testModel.AddNode(node2)
		Expect(err).NotTo(HaveOccurred())

		err = testModel.DeleteNodeById(nodeId)
		Expect(err).NotTo(HaveOccurred())

		expectedEvents := []struct {
			resourceType events.ResourceType
			operation    events.Operation
			resourceId   uuid.UUID
		}{
			{events.NodeResource, events.CreateOperation, nodeId},
			{events.NodeResource, events.UpdateOperation, nodeId},
			{events.NodeResource, events.UpdateOperation, nodeId},
			{events.NodeResource, events.UpdateOperation, nodeId},
			{events.NodeResource, events.DeleteOperation, nodeId},
		}

		actualEvents := sink.GetEvents()
		Expect(actualEvents).To(HaveLen(len(expectedEvents)))
		for i, expected := range expectedEvents {
			Expect(actualEvents[i].ResourceType).To(Equal(expected.resourceType))
			Expect(actualEvents[i].Operation).To(Equal(expected.operation))
			Expect(actualEvents[i].ResourceId).To(Equal(expected.resourceId))
		}
	})
})
