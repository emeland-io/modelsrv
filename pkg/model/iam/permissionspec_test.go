package iam_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/iam"
)

func TestPermissionSpecBasic(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	permissionSpecId := uuid.New()
	permissionSpec := iam.NewPermissionSpec(testModel.GetSink(), permissionSpecId)

	permissionSpec.SetDisplayName("Test PermissionSpec")
	assert.Equal(t, "Test PermissionSpec", permissionSpec.GetDisplayName())

	permissionSpec.SetDescription("a test permission spec")
	assert.Equal(t, "a test permission spec", permissionSpec.GetDescription())

	assert.Nil(t, testModel.GetPermissionSpecById(permissionSpecId))

	err = testModel.AddPermissionSpec(permissionSpec)
	assert.NoError(t, err)

	assert.Same(t, permissionSpec, testModel.GetPermissionSpecById(permissionSpecId))

	permissionSpec.SetDisplayName("the real test permission spec")
	permissionSpec.SetDescription("a test permission spec, but with more bla bla")

	permissionSpec2 := iam.NewPermissionSpec(testModel.GetSink(), permissionSpecId)
	permissionSpec2.SetDisplayName("The other Test PermissionSpec")
	permissionSpec2.SetDescription("a different test permission spec, but same Id")

	err = testModel.AddPermissionSpec(permissionSpec2)
	assert.NoError(t, err)

	err = testModel.DeletePermissionSpec(permissionSpecId)
	assert.NoError(t, err)

	assert.Nil(t, testModel.GetPermissionSpecById(permissionSpecId))

	expectedEvents := []struct {
		resource   events.ResourceType
		operation  events.Operation
		resourceId uuid.UUID
	}{
		{resource: events.PermissionSpecResource, operation: events.CreateOperation, resourceId: permissionSpecId},
		{resource: events.PermissionSpecResource, operation: events.UpdateOperation, resourceId: permissionSpecId},
		{resource: events.PermissionSpecResource, operation: events.UpdateOperation, resourceId: permissionSpecId},
		{resource: events.PermissionSpecResource, operation: events.UpdateOperation, resourceId: permissionSpecId},
		{resource: events.PermissionSpecResource, operation: events.DeleteOperation, resourceId: permissionSpecId},
	}

	eventsList := sink.GetEvents()
	assert.Equal(t, len(expectedEvents), len(eventsList), "expected %d events in sink, got %d", len(expectedEvents), len(eventsList))

	for i, expectedEvent := range expectedEvents {
		assert.Equal(t, expectedEvent.resource, eventsList[i].ResourceType, "event %d: expected resource type %v, got %v", i+1, expectedEvent.resource, eventsList[i].ResourceType)
		assert.Equal(t, expectedEvent.operation, eventsList[i].Operation, "event %d: expected operation %v, got %v", i+1, expectedEvent.operation, eventsList[i].Operation)
		assert.Equal(t, expectedEvent.resourceId, eventsList[i].ResourceId, "event %d: expected resource ID %v, got %v", i+1, expectedEvent.resourceId, eventsList[i].ResourceId)
	}
}
