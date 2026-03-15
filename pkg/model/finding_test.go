package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

func TestFindingBasic(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	// Add Finding Type for later use
	// Event: 1: create (FindingType)
	findingTypeId := uuid.New()
	findingType := model.NewFindingType(testModel, findingTypeId)
	testModel.AddFindingType(findingType)

	findingId := uuid.New()
	finding := model.NewFinding(testModel, findingId)

	// this must not create an event, as the finding has not been registered with the system
	finding.SetDisplayName("Test Finding")
	assert.Equal(t, "Test Finding", finding.GetDisplayName())

	// this MUST NOT create an event, as the finding has not been registered with the system
	finding.SetDescription("a test finding")
	assert.Equal(t, "a test finding", finding.GetDescription())

	// Test getting non-existent Finding
	assert.Nil(t, testModel.GetFindingById(findingId))

	// Add Finding and verify it exists
	// Event: 2: create
	err = testModel.AddFinding(finding)
	assert.NoError(t, err)

	// Verify retrieval by ID
	assert.NotNil(t, testModel.GetFindingById(findingId))
	assert.Same(t, finding, testModel.GetFindingById(findingId))

	// update the DisplayName. This MUST create an event, after the object has been registered
	// Event 3: update
	finding.SetDisplayName("the real test finding type")

	// update the description. This MUST create an event, after the object has been registered
	// with the model.
	// Event 4: update
	finding.SetDescription("a test finding type, but with more bla bla")

	// set the finding type
	// Event 5: update
	finding.SetFindingTypeByRef(findingType)

	// create a new go object and re-submit under the same UUID, but with other values
	finding2 := model.NewFinding(testModel, findingId)
	finding2.SetDisplayName("The other Test Finding")
	finding2.SetDescription("a different test finding, but same Id")

	//only when the object is added, it should trigger an event.
	// Event 6: update
	err = testModel.AddFinding(finding2)
	assert.NoError(t, err)

	// delete finding from model
	// Event 7: delete
	err = testModel.DeleteFindingById(findingId)
	assert.NoError(t, err)

	expectedEvents := []expectedEvent{
		{resourceType: events.FindingTypeResource, operation: events.CreateOperation, resourceId: findingTypeId},
		{resourceType: events.FindingResource, operation: events.CreateOperation, resourceId: findingId},
		{resourceType: events.FindingResource, operation: events.UpdateOperation, resourceId: findingId},
		{resourceType: events.FindingResource, operation: events.UpdateOperation, resourceId: findingId},
		{resourceType: events.FindingResource, operation: events.UpdateOperation, resourceId: findingId},
		{resourceType: events.FindingResource, operation: events.UpdateOperation, resourceId: findingId},
		{resourceType: events.FindingResource, operation: events.DeleteOperation, resourceId: findingId},
	}

	// Verify events in sink
	checkEvents(t, sink.GetEvents(), expectedEvents)
}

func TestFindingResources(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	findingId := uuid.New()
	finding := model.NewFinding(testModel, findingId)

	// Add Finding in order to register it and be able to track events for it
	// Event 1: create
	err = testModel.AddFinding(finding)
	assert.NoError(t, err)

	resType := events.APIInstanceResource
	resId := uuid.New()

	// add resource to finding
	// Event 2: update
	finding.AddResource(resType, resId)

	// verify resource is in finding
	resources := finding.GetResources()
	assert.Len(t, resources, 1)
	assert.Equal(t, resType, resources[0].ResourceType)
	assert.Equal(t, resId, resources[0].ResourceId)

	// remove resource from finding
	// Event 3: update
	err = finding.RemoveResourceById(resId)
	assert.NoError(t, err)

	// verify resource is removed
	resources = finding.GetResources()
	assert.Len(t, resources, 0)

	expectedEvents := []expectedEvent{
		{resourceType: events.FindingResource, operation: events.CreateOperation, resourceId: findingId},
		{resourceType: events.FindingResource, operation: events.UpdateOperation, resourceId: findingId},
		{resourceType: events.FindingResource, operation: events.UpdateOperation, resourceId: findingId},
	}

	// Verify events in sink
	checkEvents(t, sink.GetEvents(), expectedEvents)
}
