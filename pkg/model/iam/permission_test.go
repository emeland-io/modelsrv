package iam_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/iam"
)

func TestPermissionBasic(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	permissionId := uuid.New()
	permission := iam.NewPermission(permissionId)

	permission.SetDisplayName("Test Permission")
	assert.Equal(t, "Test Permission", permission.GetDisplayName())

	permission.SetDescription("a test permission")
	assert.Equal(t, "a test permission", permission.GetDescription())

	assert.Nil(t, testModel.GetPermissionById(permissionId))

	err = testModel.AddPermission(permission)
	assert.NoError(t, err)

	assert.Same(t, permission, testModel.GetPermissionById(permissionId))

	permission.SetDisplayName("the real test permission")
	permission.SetDescription("a test permission, but with more bla bla")

	permission2 := iam.NewPermission(permissionId)
	permission2.SetDisplayName("The other Test Permission")
	permission2.SetDescription("a different test permission, but same Id")

	err = testModel.AddPermission(permission2)
	assert.NoError(t, err)

	err = testModel.DeletePermission(permissionId)
	assert.NoError(t, err)

	assert.Nil(t, testModel.GetPermissionById(permissionId))

	expectedEvents := []struct {
		resource   events.ResourceType
		operation  events.Operation
		resourceId uuid.UUID
	}{
		{resource: events.PermissionResource, operation: events.CreateOperation, resourceId: permissionId},
		{resource: events.PermissionResource, operation: events.UpdateOperation, resourceId: permissionId},
		{resource: events.PermissionResource, operation: events.UpdateOperation, resourceId: permissionId},
		{resource: events.PermissionResource, operation: events.UpdateOperation, resourceId: permissionId},
		{resource: events.PermissionResource, operation: events.DeleteOperation, resourceId: permissionId},
	}

	eventsList := sink.GetEvents()
	assert.Equal(t, len(expectedEvents), len(eventsList), "expected %d events in sink, got %d", len(expectedEvents), len(eventsList))

	for i, expectedEvent := range expectedEvents {
		assert.Equal(t, expectedEvent.resource, eventsList[i].ResourceType, "event %d: expected resource type %v, got %v", i+1, expectedEvent.resource, eventsList[i].ResourceType)
		assert.Equal(t, expectedEvent.operation, eventsList[i].Operation, "event %d: expected operation %v, got %v", i+1, expectedEvent.operation, eventsList[i].Operation)
		assert.Equal(t, expectedEvent.resourceId, eventsList[i].ResourceId, "event %d: expected resource ID %v, got %v", i+1, expectedEvent.resourceId, eventsList[i].ResourceId)
	}
}
