package iam_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/iam"
)

func TestOrgUnitBasic(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	orgUnitId := uuid.New()
	orgUnit := iam.NewOrgUnit(testModel.GetSink(), orgUnitId)

	// this must not create an event, as the org unit has not been registered with the system
	orgUnit.SetDisplayName("Test Org Unit")
	assert.Equal(t, "Test Org Unit", orgUnit.GetDisplayName())

	// this MUST NOT create an event, as the org unit has not been registered with the system
	orgUnit.SetDescription("a test org unit")
	assert.Equal(t, "a test org unit", orgUnit.GetDescription())

	// Test getting non-existent OrgUnit
	assert.Nil(t, testModel.GetOrgUnitById(orgUnitId))

	// Add OrgUnit and verify it exists
	// Event: 1: create
	err = testModel.AddOrgUnit(orgUnit)
	assert.NoError(t, err)

	// Verify retrieval by ID
	assert.Same(t, orgUnit, testModel.GetOrgUnitById(orgUnitId))

	// update the DisplayName. This MUST create an event, after the object has been registered
	// Event 2: update
	orgUnit.SetDisplayName("the real test org unit")

	// update the description. This MUST create an event, after the object has been registered
	// with the model.
	// Event 3: update
	orgUnit.SetDescription("a test org unit, but with more bla bla")

	// create a new go object and re-submit under the same UUID, but with other values
	orgUnit2 := iam.NewOrgUnit(testModel.GetSink(), orgUnitId)
	orgUnit2.SetDisplayName("The other Test Org Unit")
	orgUnit2.SetDescription("a different test org unit, but same Id")

	//only when the object is added, it should trigger an event.
	// Event 4: update
	err = testModel.AddOrgUnit(orgUnit2)
	assert.NoError(t, err)

	// delete orgUnit from model
	// Event 5: delete
	err = testModel.DeleteOrgUnit(orgUnitId)
	assert.NoError(t, err)

	// verify deletion
	assert.Nil(t, testModel.GetOrgUnitById(orgUnitId))

	expectedEvents := []struct {
		resource   events.ResourceType
		operation  events.Operation
		resourceId uuid.UUID
	}{
		{resource: events.OrgUnitResource, operation: events.CreateOperation, resourceId: orgUnitId},
		{resource: events.OrgUnitResource, operation: events.UpdateOperation, resourceId: orgUnitId},
		{resource: events.OrgUnitResource, operation: events.UpdateOperation, resourceId: orgUnitId},
		{resource: events.OrgUnitResource, operation: events.UpdateOperation, resourceId: orgUnitId},
		{resource: events.OrgUnitResource, operation: events.DeleteOperation, resourceId: orgUnitId},
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
