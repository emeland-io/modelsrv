package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/component"
	"go.emeland.io/modelsrv/pkg/model/system"
)

func TestVersion(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)

	version := common.Version{
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
	ev := common.EntityVersion{
		Name:    "test-entity",
		Version: "1.0.0",
	}

	assert.Equal(t, "test-entity", ev.Name)
	assert.Equal(t, "1.0.0", ev.Version)
}

func TestApiType(t *testing.T) {
	tests := []struct {
		apiType  mdlapi.ApiType
		expected string
	}{
		{mdlapi.Unknown, "Unknown"},
		{mdlapi.OpenAPI, "OpenAPI"},
		{mdlapi.GraphQL, "GraphQL"},
		{mdlapi.GRPC, "GRPC"},
		{mdlapi.Other, "Other"},
		{mdlapi.ApiType(99), "Unknown"}, // Invalid value should return Unknown
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.apiType.String(),
			"ApiType.String() should return correct string representation")
	}
}

func TestAPI(t *testing.T) {
	apiId := uuid.New()
	version := common.Version{Version: "1.0.0"}
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)
	systemId, _ := uuid.NewUUID()
	sys := model.MakeTestSystem(sink, systemId, "a test system", common.Version{})

	api := mdlapi.NewAPI(testModel.GetSink(), apiId)
	api.SetDisplayName("test-api")
	api.SetDescription("Test API Description")
	api.SetVersion(version)
	api.SetType(mdlapi.OpenAPI)
	api.SetSystemByRef(sys)
	api.GetAnnotations().Add("key", "value")

	assert.Equal(t, "test-api", api.GetDisplayName())
	assert.Equal(t, "Test API Description", api.GetDescription())
	assert.Equal(t, apiId, api.GetApiId())
	assert.Equal(t, version, api.GetVersion())
	assert.Equal(t, mdlapi.OpenAPI, api.GetType())
	assert.Equal(t, sys, api.GetSystem().System)
	assert.Equal(t, "value", api.GetAnnotations().GetValue("key"))
}

func TestComponent(t *testing.T) {
	componentId := uuid.New()
	version := common.Version{Version: "1.0.0"}
	testModel, err := model.NewModel(events.NewListSink())
	assert.NoError(t, err)

	apiId := uuid.New()
	a := mdlapi.NewAPI(testModel.GetSink(), apiId)
	a.SetDisplayName("test-api")
	apiRef := mdlapi.ApiRef{API: a}

	comp := component.NewComponent(testModel.GetSink(), componentId)
	comp.SetDisplayName("test-component")
	comp.SetDescription("Test Component Description")
	comp.SetVersion(version)
	comp.SetConsumes([]mdlapi.ApiRef{apiRef})
	comp.SetProvides([]mdlapi.ApiRef{apiRef})
	comp.GetAnnotations().Add("key", "value")

	assert.Equal(t, "test-component", comp.GetDisplayName())
	assert.Equal(t, "Test Component Description", comp.GetDescription())
	assert.Equal(t, componentId, comp.GetComponentId())
	assert.Equal(t, version, comp.GetVersion())
	assert.Len(t, comp.GetConsumes(), 1)
	assert.Equal(t, apiRef, comp.GetConsumes()[0])
	assert.Len(t, comp.GetProvides(), 1)
	assert.Equal(t, apiRef, comp.GetProvides()[0])
	assert.Equal(t, "value", comp.GetAnnotations().GetValue("key"))
}

func TestSystemInstance(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)
	instanceId := uuid.New()
	systemId, _ := uuid.NewUUID()
	sys := model.MakeTestSystem(sink, systemId, "test-system", common.Version{})

	sysRef := &system.SystemRef{System: sys}

	instance := system.NewSystemInstance(testModel.GetSink(), instanceId)
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
	assert.Equal(t, common.ErrSystemNotFound, err)

	// Add a system and verify it exists
	sys := model.MakeTestSystem(sink, systemId, "test-system", common.Version{})

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
	assert.Equal(t, common.ErrSystemNotFound, err)
}

func TestGetSystemBySystemId(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	sysId := uuid.New()
	sys := model.MakeTestSystem(sink, sysId, "test-system", common.Version{})

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
	a := mdlapi.NewAPI(testModel.GetSink(), apiId)
	a.SetDisplayName("test-api")
	a.SetType(mdlapi.OpenAPI)

	// Test getting non-existent API
	assert.Nil(t, testModel.GetApiById(apiId))

	// Add API and verify it exists
	err = testModel.AddApi(a)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	retrieved := testModel.GetApiById(apiId)
	assert.NotNil(t, retrieved)
	assert.Equal(t, apiId, retrieved.GetApiId())
	assert.Equal(t, "test-api", retrieved.GetDisplayName())

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
	comp := component.NewComponent(testModel.GetSink(), componentId)
	comp.SetDisplayName("test-component")

	// Test getting non-existent component
	assert.Nil(t, testModel.GetComponentById(componentId))

	// Add component and verify it exists
	err = testModel.AddComponent(comp)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, comp, testModel.GetComponentById(componentId))

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
	sysRef := &system.SystemRef{System: model.MakeTestSystem(sink, systemId, "test-system", common.Version{})}

	instance := system.NewSystemInstance(testModel.GetSink(), instanceId)
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
	componentRef := &component.ComponentRef{
		Component:   nil,
		ComponentId: uuid.New(),
	}
	instance := component.NewComponentInstance(testModel.GetSink(), instanceId)
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
	a := mdlapi.NewAPI(testModel.GetSink(), apiId)
	a.SetDisplayName("test-api")
	ev := &common.EntityVersion{Name: "test-api", Version: "1.0.0"}

	apiRef := mdlapi.ApiRef{
		API:    a,
		ApiID:  apiId,
		ApiRef: ev,
	}

	assert.Equal(t, a, apiRef.API)
	assert.Equal(t, apiId, apiRef.ApiID)
	assert.Equal(t, ev, apiRef.ApiRef)
}
