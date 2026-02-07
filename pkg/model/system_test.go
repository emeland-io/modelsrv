package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

func TestSystemOperations(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	sysId := uuid.New()
	version := model.Version{Version: "1.0.0"}

	system := model.NewSystem(testModel, sysId)
	assert.NotNil(t, system)

	system.SetDisplayName("Test System")
	assert.Equal(t, "Test System", system.GetDisplayName())

	system.SetDescription("a test system")
	assert.Equal(t, "a test system", system.GetDescription())

	system.SetVersion(version)
	assert.Equal(t, version, system.GetVersion())

	// Test getting non-existent System
	assert.Nil(t, testModel.GetSystemById(sysId))

	// Add System and verify it exists
	// Event: 1: create
	err = testModel.AddSystem(system)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Same(t, system, testModel.GetSystemById(sysId))

	// update the DisplayName. This MUST create an event, after the object has been registered
	// with the model.
	// Event 2: update
	system.SetDisplayName("the real test system")

	// update the description. This MUST create an event, after the object has been registered
	// with the model.
	// Event 3: update
	system.SetDescription("a test system, but with more bla bla")

	// create a new go object and re-submit under the same UUID, but with other values
	system2 := model.NewSystem(testModel, sysId)
	system2.SetDisplayName("The other Test System")
	system2.SetDescription("a different test system, but same Id")

	//only when the object is added, it should trigger an event.
	// Event 4: update
	err = testModel.AddSystem(system2)
	assert.NoError(t, err)

	// delete system from model
	// Event 5: delete
	err = testModel.DeleteSystemById(sysId)
	assert.NoError(t, err)

	expectedEvents := []struct {
		resType    events.ResourceType
		operation  events.Operation
		resourceId uuid.UUID
	}{
		{events.SystemResource, events.CreateOperation, sysId},
		{events.SystemResource, events.UpdateOperation, sysId},
		{events.SystemResource, events.UpdateOperation, sysId},
		{events.SystemResource, events.UpdateOperation, sysId},
		{events.SystemResource, events.DeleteOperation, sysId},
	}

	for i, expected := range expectedEvents {
		event := sink.GetEvents()[i]
		assert.Equal(t, expected.resType, event.ResourceType, "event %d: resource type mismatch", i)
		assert.Equal(t, expected.operation, event.Operation, "event %d: operation mismatch", i)
		assert.Equal(t, expected.resourceId, event.ResourceId, "event %d: resource ID mismatch", i)
	}
}
