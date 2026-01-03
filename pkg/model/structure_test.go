package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gitlab.com/emeland/modelsrv/pkg/events"
	"gitlab.com/emeland/modelsrv/pkg/model"
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

func TestSystem(t *testing.T) {
	sysId := uuid.New()
	version := model.Version{Version: "1.0.0"}

	system := model.System{
		DisplayName: "test-system",
		Description: "Test System Description",
		SystemId:    sysId,
		Version:     version,
		Abstract:    false,
		Annotations: map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-system", system.DisplayName)
	assert.Equal(t, "Test System Description", system.Description)
	assert.Equal(t, sysId, system.SystemId)
	assert.Equal(t, version, system.Version)
	assert.False(t, system.Abstract)
	assert.Equal(t, "value", system.Annotations["key"])
}

func TestSystemRef(t *testing.T) {
	sysId := uuid.New()
	system := &model.System{DisplayName: "test-system"}
	ev := &model.EntityVersion{Name: "test-system", Version: "1.0.0"}

	sysRef := model.SystemRef{
		System:    system,
		SystemId:  sysId,
		SystemRef: ev,
	}

	assert.Equal(t, system, sysRef.System)
	assert.Equal(t, sysId, sysRef.SystemId)
	assert.Equal(t, ev, sysRef.SystemRef)
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
	system := &model.SystemRef{System: &model.System{DisplayName: "test-system"}}

	api := model.API{
		DisplayName: "test-api",
		Description: "Test API Description",
		ApiId:       apiId,
		Version:     version,
		Type:        model.OpenAPI,
		System:      system,
		Annotations: map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-api", api.DisplayName)
	assert.Equal(t, "Test API Description", api.Description)
	assert.Equal(t, apiId, api.ApiId)
	assert.Equal(t, version, api.Version)
	assert.Equal(t, model.OpenAPI, api.Type)
	assert.Equal(t, system, api.System)
	assert.Equal(t, "value", api.Annotations["key"])
}

func TestComponent(t *testing.T) {
	componentId := uuid.New()
	version := model.Version{Version: "1.0.0"}
	apiRef := model.ApiRef{API: &model.API{DisplayName: "test-api"}}

	component := model.Component{
		DisplayName: "test-component",
		Description: "Test Component Description",
		ComponentId: componentId,
		Version:     version,
		Consumes:    []model.ApiRef{apiRef},
		Provides:    []model.ApiRef{apiRef},
		Annotations: map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-component", component.DisplayName)
	assert.Equal(t, "Test Component Description", component.Description)
	assert.Equal(t, componentId, component.ComponentId)
	assert.Equal(t, version, component.Version)
	assert.Len(t, component.Consumes, 1)
	assert.Equal(t, apiRef, component.Consumes[0])
	assert.Len(t, component.Provides, 1)
	assert.Equal(t, apiRef, component.Provides[0])
	assert.Equal(t, "value", component.Annotations["key"])
}

func TestSystemInstance(t *testing.T) {
	instanceId := uuid.New()
	system := &model.System{DisplayName: "test-system"}
	sysRef := &model.SystemRef{System: system}

	instance := model.SystemInstance{
		DisplayName: "test-instance",
		InstanceId:  instanceId,
		SystemRef:   sysRef,
		Annotations: map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-instance", instance.DisplayName)
	assert.Equal(t, instanceId, instance.InstanceId)
	assert.Equal(t, sysRef, instance.SystemRef)
	assert.Equal(t, "value", instance.Annotations["key"])
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
	sys := &model.System{
		SystemId:    systemId,
		DisplayName: "test-system",
	}
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
	sys := &model.System{
		DisplayName: "test-system",
		SystemId:    sysId,
	}

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
	api := &model.API{
		DisplayName: "test-api",
		ApiId:       apiId,
		Type:        model.OpenAPI,
	}

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
	component := &model.Component{
		DisplayName: "test-component",
		ComponentId: componentId,
	}

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
	sysRef := model.SystemRef{System: &model.System{DisplayName: "test-system"}}
	instance := &model.SystemInstance{
		DisplayName: "test-instance",
		InstanceId:  instanceId,
		SystemRef:   &sysRef,
	}

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

func TestAPIInstanceOperations(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	instanceId := uuid.New()
	apiRef := model.ApiRef{API: &model.API{DisplayName: "test-api"}}
	instance := &model.ApiInstance{
		DisplayName: "test-instance",
		InstanceId:  instanceId,
		ApiRef:      &apiRef,
	}

	// Test getting non-existent instance
	assert.Nil(t, testModel.GetApiInstanceById(instanceId))

	// Add instance and verify it exists
	err = testModel.AddApiInstance(instance)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, instance, testModel.GetApiInstanceById(instanceId))

	// Delete instance and verify it's gone
	err = testModel.DeleteApiInstanceById(instanceId)
	assert.NoError(t, err)
	assert.Nil(t, testModel.GetApiInstanceById(instanceId))
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
	instance := &model.ComponentInstance{
		DisplayName:  "test-instance",
		InstanceId:   instanceId,
		ComponentRef: componentRef,
	}

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
	apiId := uuid.New()
	api := &model.API{DisplayName: "test-api"}
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
