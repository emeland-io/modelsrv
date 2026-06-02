package iam_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/iam"
)

func TestRoleBasic(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	roleId := uuid.New()
	role := iam.NewRole(roleId)

	role.SetDisplayName("Test Role")
	assert.Equal(t, "Test Role", role.GetDisplayName())

	role.SetDescription("a test role")
	assert.Equal(t, "a test role", role.GetDescription())

	assert.Nil(t, testModel.GetRoleById(roleId))

	err = testModel.AddRole(role)
	assert.NoError(t, err)

	assert.Same(t, role, testModel.GetRoleById(roleId))

	role.SetDisplayName("the real test role")
	role.SetDescription("a test role, but with more bla bla")

	role2 := iam.NewRole(roleId)
	role2.SetDisplayName("The other Test Role")
	role2.SetDescription("a different test role, but same Id")

	err = testModel.AddRole(role2)
	assert.NoError(t, err)

	err = testModel.DeleteRole(roleId)
	assert.NoError(t, err)

	assert.Nil(t, testModel.GetRoleById(roleId))

	expectedEvents := []struct {
		resource   events.ResourceType
		operation  events.Operation
		resourceId uuid.UUID
	}{
		{resource: events.RoleResource, operation: events.CreateOperation, resourceId: roleId},
		{resource: events.RoleResource, operation: events.UpdateOperation, resourceId: roleId},
		{resource: events.RoleResource, operation: events.UpdateOperation, resourceId: roleId},
		{resource: events.RoleResource, operation: events.UpdateOperation, resourceId: roleId},
		{resource: events.RoleResource, operation: events.DeleteOperation, resourceId: roleId},
	}

	eventsList := sink.GetEvents()
	assert.Equal(t, len(expectedEvents), len(eventsList), "expected %d events in sink, got %d", len(expectedEvents), len(eventsList))

	for i, expectedEvent := range expectedEvents {
		assert.Equal(t, expectedEvent.resource, eventsList[i].ResourceType, "event %d: expected resource type %v, got %v", i+1, expectedEvent.resource, eventsList[i].ResourceType)
		assert.Equal(t, expectedEvent.operation, eventsList[i].Operation, "event %d: expected operation %v, got %v", i+1, expectedEvent.operation, eventsList[i].Operation)
		assert.Equal(t, expectedEvent.resourceId, eventsList[i].ResourceId, "event %d: expected resource ID %v, got %v", i+1, expectedEvent.resourceId, eventsList[i].ResourceId)
	}
}
