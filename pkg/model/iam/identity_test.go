package iam_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/iam"
)

func TestIdentityBasic(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	identityId := uuid.New()
	identity := iam.NewIdentity(testModel.GetSink(), identityId)

	// this must not create an event, as the identity has not been registered with the system
	identity.SetDisplayName("Test Identity")
	assert.Equal(t, "Test Identity", identity.GetDisplayName())

	// this MUST NOT create an event, as the identity has not been registered with the system
	identity.SetDescription("a test identity")
	assert.Equal(t, "a test identity", identity.GetDescription())

	// Test getting non-existent Identity
	assert.Nil(t, testModel.GetIdentityById(identityId))

	// Add Identity and verify it exists
	// Event: 1: create
	err = testModel.AddIdentity(identity)
	assert.NoError(t, err)

	// Verify retrieval by ID
	assert.Same(t, identity, testModel.GetIdentityById(identityId))

	// update the DisplayName. This MUST create an event, after the object has been registered
	// Event 2: update
	identity.SetDisplayName("the real test identity")

	// update the description. This MUST create an event, after the object has been registered
	// with the model.
	// Event 3: update
	identity.SetDescription("a test identity, but with more bla bla")

	// create a new go object and re-submit under the same UUID, but with other values
	identity2 := iam.NewIdentity(testModel.GetSink(), identityId)
	identity2.SetDisplayName("The other Test Identity")
	identity2.SetDescription("a different test identity, but same Id")

	//only when the object is added, it should trigger an event.
	// Event 4: update
	err = testModel.AddIdentity(identity2)
	assert.NoError(t, err)

	// delete identity from model
	// Event 5: delete
	err = testModel.DeleteIdentity(identityId)
	assert.NoError(t, err)

	// verify deletion
	assert.Nil(t, testModel.GetIdentityById(identityId))

	expectedEvents := []struct {
		resource   events.ResourceType
		operation  events.Operation
		resourceId uuid.UUID
	}{
		{resource: events.IdentityResource, operation: events.CreateOperation, resourceId: identityId},
		{resource: events.IdentityResource, operation: events.UpdateOperation, resourceId: identityId},
		{resource: events.IdentityResource, operation: events.UpdateOperation, resourceId: identityId},
		{resource: events.IdentityResource, operation: events.UpdateOperation, resourceId: identityId},
		{resource: events.IdentityResource, operation: events.DeleteOperation, resourceId: identityId},
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
