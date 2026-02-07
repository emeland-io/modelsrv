package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

func TestNodeTypeOperations(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	nodeTypeId := uuid.New()
	nodeType := model.NewNodeType(testModel, nodeTypeId)

	// this must not create an event, as the node type has not been registered with the system
	nodeType.SetDisplayName("Test Node Type")
	assert.Equal(t, "Test Node Type", nodeType.GetDisplayName())

	// this must not create an event, as the node type has not been registered with the system
	nodeType.SetDescription("a test node type")
	assert.Equal(t, "a test node type", nodeType.GetDescription())

	// Test getting non-existent NodeType
	assert.Nil(t, testModel.GetNodeTypeById(nodeTypeId))

	// Add NodeType and verify it exists
	// Event: 1: create
	err = testModel.AddNodeType(nodeType)
	assert.NoError(t, err)

	// Verify retrieval by ID
	assert.Same(t, nodeType, testModel.GetNodeTypeById(nodeTypeId))

	// update the DisplayName. This MUST create an event, after the object has been registered
	// Event 2: update
	nodeType.SetDisplayName("the real test node type")

	// update the description. This MUST create an event, after the object has been registered
	// with the model.
	// Event 3: update
	nodeType.SetDescription("a test node type, but with more bla bla")

	// create a new go object and re-submit under the same UUID, but with other values
	nodeType2 := model.NewNodeType(testModel, nodeTypeId)
	nodeType2.SetDisplayName("The other Test Node Type")
	nodeType2.SetDescription("a different test node type, but same Id")

	//only when the object is added, it should trigger an event.
	// Event 4: update
	err = testModel.AddNodeType(nodeType2)
	assert.NoError(t, err)

	// delete nodeType from model
	// Event 5: delete
	err = testModel.DeleteNodeTypeById(nodeTypeId)
	assert.NoError(t, err)

	expectedEvents := []struct {
		resType    events.ResourceType
		operation  events.Operation
		resourceId uuid.UUID
	}{
		{resType: events.NodeTypeResource, operation: events.CreateOperation, resourceId: nodeTypeId},
		{resType: events.NodeTypeResource, operation: events.UpdateOperation, resourceId: nodeTypeId},
		{resType: events.NodeTypeResource, operation: events.UpdateOperation, resourceId: nodeTypeId},
		{resType: events.NodeTypeResource, operation: events.UpdateOperation, resourceId: nodeTypeId},
		{resType: events.NodeTypeResource, operation: events.DeleteOperation, resourceId: nodeTypeId},
	}

	actualEvents := sink.GetEvents()
	assert.Len(t, actualEvents, len(expectedEvents))
	for i, expected := range expectedEvents {
		assert.Equal(t, expected.resType, actualEvents[i].ResourceType)
		assert.Equal(t, expected.operation, actualEvents[i].Operation)
		assert.Equal(t, expected.resourceId, actualEvents[i].ResourceId)
	}
}
