package iam_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/iam"
)

func TestRoleSpecBasic(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	roleSpecId := uuid.New()
	roleSpec := iam.NewRoleSpec(testModel.GetSink(), roleSpecId)

	roleSpec.SetDisplayName("Test RoleSpec")
	assert.Equal(t, "Test RoleSpec", roleSpec.GetDisplayName())

	roleSpec.SetDescription("a test role spec")
	assert.Equal(t, "a test role spec", roleSpec.GetDescription())

	assert.Nil(t, testModel.GetRoleSpecById(roleSpecId))

	err = testModel.AddRoleSpec(roleSpec)
	assert.NoError(t, err)

	assert.Same(t, roleSpec, testModel.GetRoleSpecById(roleSpecId))

	roleSpec.SetDisplayName("the real test role spec")
	roleSpec.SetDescription("a test role spec, but with more bla bla")

	roleSpec2 := iam.NewRoleSpec(testModel.GetSink(), roleSpecId)
	roleSpec2.SetDisplayName("The other Test RoleSpec")
	roleSpec2.SetDescription("a different test role spec, but same Id")

	err = testModel.AddRoleSpec(roleSpec2)
	assert.NoError(t, err)

	err = testModel.DeleteRoleSpec(roleSpecId)
	assert.NoError(t, err)

	assert.Nil(t, testModel.GetRoleSpecById(roleSpecId))

	expectedEvents := []struct {
		resource   events.ResourceType
		operation  events.Operation
		resourceId uuid.UUID
	}{
		{resource: events.RoleSpecResource, operation: events.CreateOperation, resourceId: roleSpecId},
		{resource: events.RoleSpecResource, operation: events.UpdateOperation, resourceId: roleSpecId},
		{resource: events.RoleSpecResource, operation: events.UpdateOperation, resourceId: roleSpecId},
		{resource: events.RoleSpecResource, operation: events.UpdateOperation, resourceId: roleSpecId},
		{resource: events.RoleSpecResource, operation: events.DeleteOperation, resourceId: roleSpecId},
	}

	eventsList := sink.GetEvents()
	assert.Equal(t, len(expectedEvents), len(eventsList), "expected %d events in sink, got %d", len(expectedEvents), len(eventsList))

	for i, expectedEvent := range expectedEvents {
		assert.Equal(t, expectedEvent.resource, eventsList[i].ResourceType, "event %d: expected resource type %v, got %v", i+1, expectedEvent.resource, eventsList[i].ResourceType)
		assert.Equal(t, expectedEvent.operation, eventsList[i].Operation, "event %d: expected operation %v, got %v", i+1, expectedEvent.operation, eventsList[i].Operation)
		assert.Equal(t, expectedEvent.resourceId, eventsList[i].ResourceId, "event %d: expected resource ID %v, got %v", i+1, expectedEvent.resourceId, eventsList[i].ResourceId)
	}
}
