package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

func TestFindingTypeBasic(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	findingTypeId := uuid.New()
	findingType := model.NewFindingType(testModel, findingTypeId)

	// this must not create an event, as the finding type has not been registered with the system
	findingType.SetDisplayName("Test Finding Type")
	assert.Equal(t, "Test Finding Type", findingType.GetDisplayName())

	// this MUST NOT create an event, as the finding type has not been registered with the system
	findingType.SetDescription("a test finding type")
	assert.Equal(t, "a test finding type", findingType.GetDescription())

	// Test getting non-existent FindingType
	assert.Nil(t, testModel.GetFindingTypeById(findingTypeId))

	// Add FindingType and verify it exists
	// Event: 1: create
	err = testModel.AddFindingType(findingType)
	assert.NoError(t, err)

	// Verify retrieval by ID
	assert.Same(t, findingType, testModel.GetFindingTypeById(findingTypeId))

	// update the DisplayName. This MUST create an event, after the object has been registered
	// Event 2: update
	findingType.SetDisplayName("the real test finding type")

	// update the description. This MUST create an event, after the object has been registered
	// with the model.
	// Event 3: update
	findingType.SetDescription("a test finding type, but with more bla bla")

	// create a new go object and re-submit under the same UUID, but with other values
	findingType2 := model.NewFindingType(testModel, findingTypeId)
	findingType2.SetDisplayName("The other Test Finding Type")
	findingType2.SetDescription("a different test finding type, but same Id")

	//only when the object is added, it should trigger an event.
	// Event 4: update
	err = testModel.AddFindingType(findingType2)
	assert.NoError(t, err)

	// delete findingType from model
	// Event 5: delete
	err = testModel.DeleteFindingTypeById(findingTypeId)
	assert.NoError(t, err)

	expectedEvents := []expectedEvent{
		{resourceType: events.FindingTypeResource, operation: events.CreateOperation, resourceId: findingTypeId},
		{resourceType: events.FindingTypeResource, operation: events.UpdateOperation, resourceId: findingTypeId},
		{resourceType: events.FindingTypeResource, operation: events.UpdateOperation, resourceId: findingTypeId},
		{resourceType: events.FindingTypeResource, operation: events.UpdateOperation, resourceId: findingTypeId},
		{resourceType: events.FindingTypeResource, operation: events.DeleteOperation, resourceId: findingTypeId},
	}

	// Verify events in sink
	checkEvents(t, sink.GetEvents(), expectedEvents)
}
