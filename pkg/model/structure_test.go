package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewModel(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err, "NewModel should not return an error")
	assert.NotNil(t, model, "NewModel should return a non-nil model")

	// Verify all maps are initialized
	assert.NotNil(t, model.SystemsByName, "SystemsByName map should be initialized")
	assert.NotNil(t, model.APIsByName, "APIsByName map should be initialized")
	assert.NotNil(t, model.ComponentsByName, "ComponentsByName map should be initialized")
	assert.NotNil(t, model.SystemsByUUID, "SystemsByUUID map should be initialized")
	assert.NotNil(t, model.APIsByUUID, "APIsByUUID map should be initialized")
	assert.NotNil(t, model.ComponentsByUUID, "ComponentsByUUID map should be initialized")
	assert.NotNil(t, model.SystemInstancesByUUID, "SystemInstances map should be initialized")
	assert.NotNil(t, model.APIInstancesByUUID, "APIInstances map should be initialized")
	assert.NotNil(t, model.ComponentInstancesByUUID, "ComponentInstances map should be initialized")
}

func TestVersion(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)

	version := Version{
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
	ev := EntityVersion{
		Name:    "test-entity",
		Version: "1.0.0",
	}

	assert.Equal(t, "test-entity", ev.Name)
	assert.Equal(t, "1.0.0", ev.Version)
}

func TestSystem(t *testing.T) {
	sysId := uuid.New()
	version := Version{Version: "1.0.0"}

	system := System{
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
	system := &System{DisplayName: "test-system"}
	ev := &EntityVersion{Name: "test-system", Version: "1.0.0"}

	sysRef := SystemRef{
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
		apiType  ApiType
		expected string
	}{
		{Unknown, "Unknown"},
		{OpenAPI, "OpenAPI"},
		{GraphQL, "GraphQL"},
		{gRPC, "gRPC"},
		{Other, "Other"},
		{ApiType(99), "Unknown"}, // Invalid value should return Unknown
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.apiType.String(),
			"ApiType.String() should return correct string representation")
	}
}

func TestAPI(t *testing.T) {
	apiId := uuid.New()
	version := Version{Version: "1.0.0"}
	system := &SystemRef{System: &System{DisplayName: "test-system"}}

	api := API{
		DisplayName: "test-api",
		Description: "Test API Description",
		ApiId:       apiId,
		Version:     version,
		Type:        OpenAPI,
		System:      system,
		Annotations: map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-api", api.DisplayName)
	assert.Equal(t, "Test API Description", api.Description)
	assert.Equal(t, apiId, api.ApiId)
	assert.Equal(t, version, api.Version)
	assert.Equal(t, OpenAPI, api.Type)
	assert.Equal(t, system, api.System)
	assert.Equal(t, "value", api.Annotations["key"])
}

func TestComponent(t *testing.T) {
	componentId := uuid.New()
	version := Version{Version: "1.0.0"}
	apiRef := ApiRef{API: &API{DisplayName: "test-api"}}

	component := Component{
		DisplayName: "test-component",
		Description: "Test Component Description",
		ComponentId: componentId,
		Version:     version,
		Consumes:    []ApiRef{apiRef},
		Provides:    []ApiRef{apiRef},
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
	system := &System{DisplayName: "test-system"}
	sysRef := &SystemRef{System: system}

	instance := SystemInstance{
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

func TestDeleteSystemByResourceName(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	// Test deleting non-existent system
	err = model.DeleteSystemByResourceName("non-existent")
	assert.Equal(t, SystemNotFoundError, err)

	// Add a system and verify it exists
	sys := &System{DisplayName: "test-system"}
	err = model.AddSystem(sys, "test-system", nil)
	assert.NoError(t, err)
	assert.NotNil(t, model.GetSystemByResourceName("test-system"))

	// Delete the system
	err = model.DeleteSystemByResourceName("test-system")
	assert.NoError(t, err)

	// Verify system was deleted
	assert.Nil(t, model.GetSystemByResourceName("test-system"))

	// Try deleting again should return error
	err = model.DeleteSystemByResourceName("test-system")
	assert.Equal(t, SystemNotFoundError, err)
}

func TestGetSystemBySystemId(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	sysId := uuid.New()
	sys := &System{
		DisplayName: "test-system",
		SystemId:    sysId,
	}

	// Test getting non-existent system
	assert.Nil(t, model.GetSystemById(sysId))

	// Add system and verify it can be retrieved by UUID
	err = model.AddSystem(sys, "test-system", nil)
	assert.NoError(t, err)

	retrieved := model.GetSystemById(sysId)
	assert.NotNil(t, retrieved)
	assert.Equal(t, sys, retrieved)
}

func TestAPIOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	apiId := uuid.New()
	api := &API{
		DisplayName: "test-api",
		ApiId:       apiId,
		Type:        OpenAPI,
	}

	// Test getting non-existent API
	assert.Nil(t, model.GetApiByResourceName("test-api"))
	assert.Nil(t, model.GetApiById(apiId))

	// Add API and verify it exists
	err = model.AddApi(api, "test-api", nil)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, api, model.GetApiByResourceName("test-api"))
	assert.Equal(t, api, model.GetApiById(apiId))

	// Delete API and verify it's gone
	err = model.DeleteApiByResourceName("test-api")
	assert.NoError(t, err)
	assert.Nil(t, model.GetApiByResourceName("test-api"))
	assert.Nil(t, model.GetApiById(apiId))
}

func TestComponentOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	componentId := uuid.New()
	component := &Component{
		DisplayName: "test-component",
		ComponentId: componentId,
	}

	// Test getting non-existent component
	assert.Nil(t, model.GetComponentByResourceName("test-component"))
	assert.Nil(t, model.GetComponentById(componentId))

	// Add component and verify it exists
	err = model.AddComponent(component, "test-component", nil)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, component, model.GetComponentByResourceName("test-component"))
	assert.Equal(t, component, model.GetComponentById(componentId))

	// Delete component and verify it's gone
	err = model.DeleteComponentByResourceName("test-component")
	assert.NoError(t, err)
	assert.Nil(t, model.GetComponentByResourceName("test-component"))
	assert.Nil(t, model.GetComponentById(componentId))
}

func TestSystemInstanceOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	instanceId := uuid.New()
	sysRef := SystemRef{System: &System{DisplayName: "test-system"}}
	instance := &SystemInstance{
		DisplayName: "test-instance",
		InstanceId:  instanceId,
		SystemRef:   &sysRef,
	}

	// Test getting non-existent instance
	assert.Nil(t, model.GetSystemInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetSystemInstanceById(instanceId))

	// Add instance and verify it exists
	err = model.AddSystemInstance(instance, "test-instance", nil)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, instance, model.GetSystemInstanceByResourceName("test-instance"))
	assert.Equal(t, instance, model.GetSystemInstanceById(instanceId))

	// Delete instance and verify it's gone
	err = model.DeleteSystemInstanceByResourceName("test-instance")
	assert.NoError(t, err)
	assert.Nil(t, model.GetSystemInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetSystemInstanceById(instanceId))
}

func TestAPIInstanceOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	instanceId := uuid.New()
	apiRef := ApiRef{API: &API{DisplayName: "test-api"}}
	instance := &APIInstance{
		DisplayName: "test-instance",
		InstanceId:  instanceId,
		ApiRef:      apiRef,
	}

	// Test getting non-existent instance
	assert.Nil(t, model.GetApiInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetApiInstanceById(instanceId))

	// Add instance and verify it exists
	err = model.AddApiInstance(instance, "test-instance", nil)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, instance, model.GetApiInstanceByResourceName("test-instance"))
	assert.Equal(t, instance, model.GetApiInstanceById(instanceId))

	// Delete instance and verify it's gone
	err = model.DeleteApiInstanceByResourceName("test-instance")
	assert.NoError(t, err)
	assert.Nil(t, model.GetApiInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetApiInstanceById(instanceId))
}

func TestComponentInstanceOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	instanceId := uuid.New()
	componentRef := EntityVersion{Name: "test-component", Version: "1.0.0"}
	instance := &ComponentInstance{
		DisplayName:  "test-instance",
		InstanceId:   instanceId,
		ComponentRef: componentRef,
	}

	// Test getting non-existent instance
	assert.Nil(t, model.GetComponentInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetComponentInstanceById(instanceId))

	// Add instance and verify it exists
	err = model.AddComponentInstance(instance, "test-instance", nil)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, instance, model.GetComponentInstanceByResourceName("test-instance"))
	assert.Equal(t, instance, model.GetComponentInstanceById(instanceId))

	// Delete instance and verify it's gone
	err = model.DeleteComponentInstanceByResourceName("test-instance")
	assert.NoError(t, err)
	assert.Nil(t, model.GetComponentInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetComponentInstanceById(instanceId))
}

func TestApiRef(t *testing.T) {
	apiId := uuid.New()
	api := &API{DisplayName: "test-api"}
	ev := &EntityVersion{Name: "test-api", Version: "1.0.0"}

	apiRef := ApiRef{
		API:    api,
		ApiID:  apiId,
		ApiRef: ev,
	}

	assert.Equal(t, api, apiRef.API)
	assert.Equal(t, apiId, apiRef.ApiID)
	assert.Equal(t, ev, apiRef.ApiRef)
}
