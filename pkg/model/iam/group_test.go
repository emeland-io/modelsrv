package iam_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/iam"
)

func TestGroupBasic(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	groupId := uuid.New()
	group := iam.NewGroup(testModel.GetSink(), groupId)

	// this must not create an event, as the group has not been registered with the system
	group.SetDisplayName("Test Group")
	assert.Equal(t, "Test Group", group.GetDisplayName())

	// this MUST NOT create an event, as the group has not been registered with the system
	group.SetDescription("a test group")
	assert.Equal(t, "a test group", group.GetDescription())

	// Test getting non-existent Group
	assert.Nil(t, testModel.GetGroupById(groupId))

	// Add Group and verify it exists
	// Event: 1: create
	err = testModel.AddGroup(group)
	assert.NoError(t, err)

	// Verify retrieval by ID
	assert.Same(t, group, testModel.GetGroupById(groupId))

	// update the DisplayName. This MUST create an event, after the object has been registered
	// Event 2: update
	group.SetDisplayName("the real test group")

	// update the description. This MUST create an event, after the object has been registered
	// with the model.
	// Event 3: update
	group.SetDescription("a test group, but with more bla bla")

	// create a new go object and re-submit under the same UUID, but with other values
	group2 := iam.NewGroup(testModel.GetSink(), groupId)
	group2.SetDisplayName("The other Test Group")
	group2.SetDescription("a different test group, but same Id")

	//only when the object is added, it should trigger an event.
	// Event 4: update
	err = testModel.AddGroup(group2)
	assert.NoError(t, err)

	// delete group from model
	// Event 5: delete
	err = testModel.DeleteGroup(groupId)
	assert.NoError(t, err)

	// verify group is deleted
	assert.Nil(t, testModel.GetGroupById(groupId))

	expectedEvents := []struct {
		resource   events.ResourceType
		operation  events.Operation
		resourceId uuid.UUID
	}{
		{resource: events.GroupResource, operation: events.CreateOperation, resourceId: groupId},
		{resource: events.GroupResource, operation: events.UpdateOperation, resourceId: groupId},
		{resource: events.GroupResource, operation: events.UpdateOperation, resourceId: groupId},
		{resource: events.GroupResource, operation: events.UpdateOperation, resourceId: groupId},
		{resource: events.GroupResource, operation: events.DeleteOperation, resourceId: groupId},
	}

	// Verify events in sink
	eventsList := sink.GetEvents()
	assert.Equal(t, len(expectedEvents), len(eventsList), "expected %d events in sink, got %d", len(expectedEvents), len(eventsList))

	for i, expectedEvent := range expectedEvents {
		assert.Equal(t, expectedEvent.resource, eventsList[i].ResourceType, "event %d: expected resource type %v, got %v", i+1, expectedEvent.resource, eventsList[i].ResourceType)
		assert.Equal(t, expectedEvent.operation, eventsList[i].Operation, "event %d: expected operation %v, got %v", i+1, expectedEvent.operation, eventsList[i].Operation)
		assert.Equal(t, expectedEvent.resourceId, eventsList[i].ResourceId, "event %d: expected resource ID %v, got %v", i+1, expectedEvent.resourceId, eventsList[i].ResourceId)
	}
}
