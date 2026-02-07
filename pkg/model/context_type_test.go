package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

func TestContextTypeOperations(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	contextTypeId := uuid.New()
	contextType := model.NewContextType(testModel, contextTypeId)

	// this must not create an event, as the context type has not been registered with the system
	contextType.SetDisplayName("Test Context Type")
	assert.Equal(t, "Test Context Type", contextType.GetDisplayName())

	// this must not create an event, as the context type has not been registered with the system
	contextType.SetDescription("a test context type")
	assert.Equal(t, "a test context type", contextType.GetDescription())

	// Test getting non-existent ContextType
	assert.Nil(t, testModel.GetContextTypeById(contextTypeId))

	// Add ContextType and verify it exists
	// Event: 1: create
	err = testModel.AddContextType(contextType)
	assert.NoError(t, err)

	// Verify retrieval by ID
	assert.Same(t, contextType, testModel.GetContextTypeById(contextTypeId))

	// update the DisplayName. This MUST create an event, after the object has been registered
	// Event 2: update
	contextType.SetDisplayName("the real test context type")

	// update the description. This MUST create an event, after the object has been registered
	// with the model.
	// Event 3: update
	contextType.SetDescription("a test context type, but with more bla bla")

	// create a new go object and re-submit under the same UUID, but with other values
	contextType2 := model.NewContextType(testModel, contextTypeId)
	contextType2.SetDisplayName("The other Test Context Type")
	contextType2.SetDescription("a different test context type, but same Id")

	//only when the object is added, it should trigger an event.
	// Event 4: update
	err = testModel.AddContextType(contextType2)
	assert.NoError(t, err)

	// delete contextType from model
	// Event 5: delete
	err = testModel.DeleteContextTypeById(contextTypeId)
	assert.NoError(t, err)
	// Test contextType is gone
	assert.Nil(t, testModel.GetContextTypeById(contextTypeId))

	// Verify events in sink
	eventsList := sink.GetEvents()
	assert.Equal(t, 5, len(eventsList), "expected 5 events in sink")

	expectedEvents := []struct {
		resType    events.ResourceType
		operation  events.Operation
		resourceId uuid.UUID
	}{
		{events.ContextTypeResource, events.CreateOperation, contextTypeId},
		{events.ContextTypeResource, events.UpdateOperation, contextTypeId},
		{events.ContextTypeResource, events.UpdateOperation, contextTypeId},
		{events.ContextTypeResource, events.UpdateOperation, contextTypeId},
		{events.ContextTypeResource, events.DeleteOperation, contextTypeId},
	}

	for i, expected := range expectedEvents {
		actual := eventsList[i]
		assert.Equal(t, expected.resType, actual.ResourceType, "event %d: resource type mismatch", i+1)
		assert.Equal(t, expected.operation, actual.Operation, "event %d: operation mismatch", i+1)
		assert.Equal(t, expected.resourceId, actual.ResourceId, "event %d: resource ID mismatch", i+1)
	}
}
