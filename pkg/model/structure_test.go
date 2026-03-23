package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

func TestVersion(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)

	version := model.Version{
		Version:        "1.0.0",
		AvailableFrom:  &now,
		DeprecatedFrom: &future,
		TerminatedFrom: nil,
	}

	assert.Equal(t, "1.0.0", version.Version)
	assert.Equal(t, now, *version.AvailableFrom)
	assert.Equal(t, future, *version.DeprecatedFrom)
	assert.Nil(t, version.TerminatedFrom)
}

func TestEntityVersion(t *testing.T) {
	ev := model.EntityVersion{
		Name:    "test-entity",
		Version: "1.0.0",
	}

	assert.Equal(t, "test-entity", ev.Name)
	assert.Equal(t, "1.0.0", ev.Version)
}

func TestApiType(t *testing.T) {
	tests := []struct {
		apiType  model.ApiType
		expected string
	}{
		{model.Unknown, "Unknown"},
		{model.OpenAPI, "OpenAPI"},
		{model.GraphQL, "GraphQL"},
		{model.GRPC, "GRPC"},
		{model.Other, "Other"},
		{model.ApiType(99), "Unknown"}, // Invalid value should return Unknown
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.apiType.String(),
			"ApiType.String() should return correct string representation")
	}
}

func TestAPI(t *testing.T) {
	apiId := uuid.New()
	version := model.Version{Version: "1.0.0"}
	testModel, err := model.NewModel(events.NewListSink())
	assert.NoError(t, err)
	systemId, _ := uuid.NewUUID()
	system := &model.SystemRef{System: model.MakeTestSystem(testModel, systemId, "a test system", model.Version{})}

	api := model.NewAPI(testModel, apiId)
	api.SetDisplayName("test-api")
	api.SetDescription("Test API Description")
	api.SetVersion(version)
	api.SetType(model.OpenAPI)
	api.SetSystem(system)
	api.GetAnnotations().Add("key", "value")

	assert.Equal(t, "test-api", api.GetDisplayName())
	assert.Equal(t, "Test API Description", api.GetDescription())
	assert.Equal(t, apiId, api.GetApiId())
	assert.Equal(t, version, api.GetVersion())
	assert.Equal(t, model.OpenAPI, api.GetType())
	assert.Equal(t, system, api.GetSystem())
	assert.Equal(t, "value", api.GetAnnotations().GetValue("key"))
}

func TestComponent(t *testing.T) {
	componentId := uuid.New()
	version := model.Version{Version: "1.0.0"}
	testModel, err := model.NewModel(events.NewListSink())
	assert.NoError(t, err)
	
	apiId := uuid.New()
	api := model.NewAPI(testModel, apiId)
	api.SetDisplayName("test-api")
	apiRef := model.ApiRef{API: api}

	component := model.NewComponent(testModel, componentId)
	component.SetDisplayName("test-component")
	component.SetDescription("Test Component Description")
	component.SetVersion(version)
	component.SetConsumes([]model.ApiRef{apiRef})
	component.SetProvides([]model.ApiRef{apiRef})
	component.GetAnnotations().Add("key", "value")

	assert.Equal(t, "test-component", component.GetDisplayName())
	assert.Equal(t, "Test Component Description", component.GetDescription())
	assert.Equal(t, componentId, component.GetComponentId())
	assert.Equal(t, version, component.GetVersion())
	assert.Len(t, component.GetConsumes(), 1)
	assert.Equal(t, apiRef, component.GetConsumes()[0])
	assert.Len(t, component.GetProvides(), 1)
	assert.Equal(t, apiRef, component.GetProvides()[0])
	assert.Equal(t, "value", component.GetAnnotations().GetValue("key"))
}

func TestSystemInstance(t *testing.T) {
	testModel, err := model.NewModel(events.NewListSink())
	assert.NoError(t, err)
	instanceId := uuid.New()
	systemId, _ := uuid.NewUUID()
	system := model.MakeTestSystem(testModel, systemId, "test-system", model.Version{})

	sysRef := &model.SystemRef{System: system}

	instance := model.NewSystemInstance(testModel, instanceId)
	instance.SetDisplayName("test-instance")
	instance.SetSystemRef(sysRef)
	instance.GetAnnotations().Add("key", "value")

	assert.Equal(t, "test-instance", instance.GetDisplayName())
	assert.Equal(t, instanceId, instance.GetInstanceId())
	assert.Equal(t, sysRef, instance.GetSystemRef())
	assert.Equal(t, "value", instance.GetAnnotations().GetValue("key"))
}

func TestDeleteSystemById(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)
	systemId := uuid.New()

	// Test deleting non-existent system
	err = testModel.DeleteSystemById(uuid.New())
	assert.Equal(t, model.SystemNotFoundError, err)

	// Add a system and verify it exists
	sys := model.MakeTestSystem(testModel, systemId, "test-system", model.Version{})

	err = testModel.AddSystem(sys)
	assert.NoError(t, err)
	assert.NotNil(t, testModel.GetSystemById(systemId))

	// Delete the system
	err = testModel.DeleteSystemById(systemId)
	assert.NoError(t, err)

	// Verify system was deleted
	assert.Nil(t, testModel.GetSystemById(systemId))

	// Try deleting again should return error
	err = testModel.DeleteSystemById(systemId)
	assert.Equal(t, model.SystemNotFoundError, err)
}

func TestGetSystemBySystemId(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	sysId := uuid.New()
	sys := model.MakeTestSystem(testModel, sysId, "test-system", model.Version{})

	// Test getting non-existent system
	assert.Nil(t, testModel.GetSystemById(sysId))

	// Add system and verify it can be retrieved by UUID
	err = testModel.AddSystem(sys)
	assert.NoError(t, err)

	retrieved := testModel.GetSystemById(sysId)
	assert.NotNil(t, retrieved)
	assert.Equal(t, sys, retrieved)
}

func TestAPIOperations(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	apiId := uuid.New()
	api := model.NewAPI(testModel, apiId)
	api.SetDisplayName("test-api")
	api.SetType(model.OpenAPI)

	// Test getting non-existent API
	assert.Nil(t, testModel.GetApiById(apiId))

	// Add API and verify it exists
	err = testModel.AddApi(api)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, api, testModel.GetApiById(apiId))

	// Delete API and verify it's gone
	err = testModel.DeleteApiById(apiId)
	assert.NoError(t, err)
	assert.Nil(t, testModel.GetApiById(apiId))
}

func TestComponentOperations(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	componentId := uuid.New()
	component := model.NewComponent(testModel, componentId)
	component.SetDisplayName("test-component")

	// Test getting non-existent component
	assert.Nil(t, testModel.GetComponentById(componentId))

	// Add component and verify it exists
	err = testModel.AddComponent(component)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, component, testModel.GetComponentById(componentId))

	// Delete component and verify it's gone
	err = testModel.DeleteComponentById(componentId)
	assert.NoError(t, err)
	assert.Nil(t, testModel.GetComponentById(componentId))
}

func TestSystemInstanceOperations(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	instanceId := uuid.New()
	systemId := uuid.New()
	sysRef := &model.SystemRef{System: model.MakeTestSystem(testModel, systemId, "test-system", model.Version{})}

	instance := model.NewSystemInstance(testModel, instanceId)
	instance.SetDisplayName("test-instance")
	instance.SetSystemRef(sysRef)

	// Test getting non-existent instance
	assert.Nil(t, testModel.GetSystemInstanceById(instanceId))

	// Add instance and verify it exists
	err = testModel.AddSystemInstance(instance)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, instance, testModel.GetSystemInstanceById(instanceId))

	// Delete instance and verify it's gone
	err = testModel.DeleteSystemInstanceById(instanceId)
	assert.NoError(t, err)
	assert.Nil(t, testModel.GetSystemInstanceById(instanceId))
}

func TestComponentInstanceOperations(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	instanceId := uuid.New()
	componentRef := &model.ComponentRef{
		Component:   nil,
		ComponentId: uuid.New(),
	}
	instance := model.NewComponentInstance(testModel, instanceId)
	instance.SetDisplayName("test-instance")
	instance.SetComponentRef(componentRef)

	// Test getting non-existent instance
	assert.Nil(t, testModel.GetComponentInstanceById(instanceId))

	// Add instance and verify it exists
	err = testModel.AddComponentInstance(instance)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, instance, testModel.GetComponentInstanceById(instanceId))

	// Delete instance and verify it's gone
	err = testModel.DeleteComponentInstanceById(instanceId)
	assert.NoError(t, err)
	assert.Nil(t, testModel.GetComponentInstanceById(instanceId))
}

func TestApiRef(t *testing.T) {
	testModel, err := model.NewModel(events.NewListSink())
	assert.NoError(t, err)
	
	apiId := uuid.New()
	api := model.NewAPI(testModel, apiId)
	api.SetDisplayName("test-api")
	ev := &model.EntityVersion{Name: "test-api", Version: "1.0.0"}

	apiRef := model.ApiRef{
		API:    api,
		ApiID:  apiId,
		ApiRef: ev,
	}

	assert.Equal(t, api, apiRef.API)
	assert.Equal(t, apiId, apiRef.ApiID)
	assert.Equal(t, ev, apiRef.ApiRef)
}
