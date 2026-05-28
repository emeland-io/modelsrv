package iam_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/iam"
)

func TestBindingBasic(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	bindingId := uuid.New()
	binding := iam.NewBinding(testModel.GetSink(), bindingId)

	binding.SetDisplayName("Test Binding")
	assert.Equal(t, "Test Binding", binding.GetDisplayName())

	binding.SetDescription("a test binding")
	assert.Equal(t, "a test binding", binding.GetDescription())

	assert.Nil(t, testModel.GetBindingById(bindingId))

	err = testModel.AddBinding(binding)
	assert.NoError(t, err)

	assert.Same(t, binding, testModel.GetBindingById(bindingId))

	binding.SetDisplayName("the real test binding")
	binding.SetDescription("a test binding, but with more bla bla")

	binding2 := iam.NewBinding(testModel.GetSink(), bindingId)
	binding2.SetDisplayName("The other Test Binding")
	binding2.SetDescription("a different test binding, but same Id")

	err = testModel.AddBinding(binding2)
	assert.NoError(t, err)

	err = testModel.DeleteBinding(bindingId)
	assert.NoError(t, err)

	assert.Nil(t, testModel.GetBindingById(bindingId))

	expectedEvents := []struct {
		resource   events.ResourceType
		operation  events.Operation
		resourceId uuid.UUID
	}{
		{resource: events.BindingResource, operation: events.CreateOperation, resourceId: bindingId},
		{resource: events.BindingResource, operation: events.UpdateOperation, resourceId: bindingId},
		{resource: events.BindingResource, operation: events.UpdateOperation, resourceId: bindingId},
		{resource: events.BindingResource, operation: events.UpdateOperation, resourceId: bindingId},
		{resource: events.BindingResource, operation: events.DeleteOperation, resourceId: bindingId},
	}

	eventsList := sink.GetEvents()
	assert.Equal(t, len(expectedEvents), len(eventsList), "expected %d events in sink, got %d", len(expectedEvents), len(eventsList))

	for i, expectedEvent := range expectedEvents {
		assert.Equal(t, expectedEvent.resource, eventsList[i].ResourceType, "event %d: expected resource type %v, got %v", i+1, expectedEvent.resource, eventsList[i].ResourceType)
		assert.Equal(t, expectedEvent.operation, eventsList[i].Operation, "event %d: expected operation %v, got %v", i+1, expectedEvent.operation, eventsList[i].Operation)
		assert.Equal(t, expectedEvent.resourceId, eventsList[i].ResourceId, "event %d: expected resource ID %v, got %v", i+1, expectedEvent.resourceId, eventsList[i].ResourceId)
	}
}
