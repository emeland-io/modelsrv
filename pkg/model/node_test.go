package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

func TestNodeOperations(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	nodeId := uuid.New()
	node := model.NewNode(testModel, nodeId)

	// this must not create an event, as the node has not been registered with the system
	node.SetDisplayName("Test Node")
	assert.Equal(t, "Test Node", node.GetDisplayName())

	// this must not create an event, as the node has not been registered with the system
	node.SetDescription("a test node")
	assert.Equal(t, "a test node", node.GetDescription())

	// Test getting non-existent Node
	assert.Nil(t, testModel.GetNodeById(nodeId))

	// Add Node and verify it exists
	// Event: 1: create
	err = testModel.AddNode(node)
	assert.NoError(t, err)

	// Verify retrieval by ID
	assert.Same(t, node, testModel.GetNodeById(nodeId))

	// update the DisplayName. This MUST create an event, after the object has been registered
	// Event 2: update
	node.SetDisplayName("the real test node")

	// update the description. This MUST create an event, after the object has been registered
	// with the model.
	// Event 3: update
	node.SetDescription("a test node, but with more bla bla")

	// create a new go object and re-submit under the same UUID, but with other values
	node2 := model.NewNode(testModel, nodeId)
	node2.SetDisplayName("The other Test Node")
	node2.SetDescription("a different test node, but same Id")

	//only when the object is added, it should trigger an event.
	// Event 4: update
	err = testModel.AddNode(node2)
	assert.NoError(t, err)

	// delete node from model
	// Event 5: delete
	err = testModel.DeleteNodeById(nodeId)
	assert.NoError(t, err)

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
	assert.Len(t, actualEvents, len(expectedEvents))
	for i, expected := range expectedEvents {
		assert.Equal(t, expected.resourceType, actualEvents[i].ResourceType)
		assert.Equal(t, expected.operation, actualEvents[i].Operation)
		assert.Equal(t, expected.resourceId, actualEvents[i].ResourceId)
	}
}
