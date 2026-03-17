package model_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

func TestApiInstanceOperations(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	instanceId := uuid.New()
	instance := model.NewApiInstance(testModel, instanceId)

	// this must not create an event, as the instance has not been registered with the system
	instance.SetDisplayName("Test API Instance")
	assert.Equal(t, "Test API Instance", instance.GetDisplayName())

	// Test getting non-existent instance
	assert.Nil(t, testModel.GetApiInstanceById(instanceId))

	// Add instance and verify it exists
	// Event: 1: create
	err = testModel.AddApiInstance(instance)
	assert.NoError(t, err)

	// Verify retrieval by ID
	assert.Same(t, instance, testModel.GetApiInstanceById(instanceId))

	// update the DisplayName. This MUST create an event, after the object has been registered
	// with the model.
	// Event 2: update
	instance.SetDisplayName("the real test API instance")

	// create a new go object and re-submit under the same UUID, but with other values
	instance2 := model.NewApiInstance(testModel, instanceId)
	instance2.SetDisplayName("The other Test API Instance")

	//only when the object is added, it should trigger an event.
	// Event 3: update
	err = testModel.AddApiInstance(instance2)
	assert.NoError(t, err)

	// Verify retrieval by ID
	assert.Same(t, instance2, testModel.GetApiInstanceById(instanceId))

	// Delete instance and verify it's gone
	// Event: 4: delete
	err = testModel.DeleteApiInstanceById(instanceId)
	assert.NoError(t, err)
	assert.Nil(t, testModel.GetApiInstanceById(instanceId))

	eventList := sink.GetList()
	assert.Equal(t, 4, len(eventList))
	assert.True(t, strings.HasPrefix(eventList[0], fmt.Sprintf("CreateOperation: APIInstance %s", instanceId.String())))
	assert.True(t, strings.HasPrefix(eventList[1], fmt.Sprintf("UpdateOperation: APIInstance %s", instanceId.String())))
	assert.True(t, strings.HasPrefix(eventList[2], fmt.Sprintf("CreateOperation: APIInstance %s", instanceId.String())))
	assert.True(t, strings.HasPrefix(eventList[3], fmt.Sprintf("DeleteOperation: APIInstance %s", instanceId.String())))
}

func TestApiInstanceAnnotations(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	instanceId := uuid.New()
	instance := model.NewApiInstance(testModel, instanceId)
	instance.SetDisplayName("Test API Instance")

	// Event 1: create
	err = testModel.AddApiInstance(instance)
	assert.NoError(t, err)

	// Add annotations
	annotations := instance.GetAnnotations()
	assert.NotNil(t, annotations)
	// Event 2,3: add keys -> update instance
	annotations.Add("key1", "value1")
	annotations.Add("key2", "value2")

	keys := slices.Collect(annotations.GetKeys())
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")

	value1 := annotations.GetValue("key1")
	assert.Equal(t, "value1", value1)

	value2 := annotations.GetValue("key2")
	assert.Equal(t, "value2", value2)

	// Delete an annotation
	// Event 4: delete annotation -> update instance
	annotations.Delete("key1")
	keys = slices.Collect(annotations.GetKeys())
	assert.NotContains(t, keys, "key1")
	assert.Contains(t, keys, "key2")

	value1 = annotations.GetValue("key1")
	assert.Equal(t, "", value1) // should return empty string for non-existent key

	eventList := sink.GetList()
	assert.Equal(t, 4, len(eventList))
	assert.True(t, strings.HasPrefix(eventList[0], fmt.Sprintf("CreateOperation: APIInstance %s", instanceId.String())))
	assert.True(t, strings.HasPrefix(eventList[1], fmt.Sprintf("UpdateOperation: APIInstance %s", instanceId.String())))
	assert.True(t, strings.HasPrefix(eventList[2], fmt.Sprintf("UpdateOperation: APIInstance %s", instanceId.String())))
	assert.True(t, strings.HasPrefix(eventList[3], fmt.Sprintf("UpdateOperation: APIInstance %s", instanceId.String())))
}

func TestApiInstanceSetApiRef(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	apiId := uuid.New()
	api := model.MakeTestAPIForModel(testModel, apiId, "Test API", model.OpenAPI, model.Version{})

	instanceId := uuid.New()
	instance := model.NewApiInstance(testModel, instanceId)
	instance.SetDisplayName("Test API Instance")

	// Add API (emits event - API is now interface with sink)
	// Event 1: create API
	err = testModel.AddApi(api)
	assert.NoError(t, err)

	// Event 2: create instance
	err = testModel.AddApiInstance(instance)
	assert.NoError(t, err)

	// Set API reference by ID
	// Event 3: update instance
	instance.SetApiRefById(apiId)

	apiRef := instance.GetApiRef()
	assert.NotNil(t, apiRef)
	assert.Equal(t, apiId, apiRef.ApiID)
	assert.Equal(t, api, apiRef.API)

	eventList := sink.GetList()
	assert.Equal(t, 3, len(eventList))
	assert.True(t, strings.HasPrefix(eventList[0], fmt.Sprintf("CreateOperation: API %s", apiId.String())))
	assert.True(t, strings.HasPrefix(eventList[1], fmt.Sprintf("CreateOperation: APIInstance %s", instanceId.String())))
	assert.True(t, strings.HasPrefix(eventList[2], fmt.Sprintf("UpdateOperation: APIInstance %s", instanceId.String())))
}

func TestApiInstanceSetSystemInstance(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	systemInstanceId := uuid.New()
	systemInstance := &model.SystemInstance{
		InstanceId:  systemInstanceId,
		DisplayName: "Test System Instance",
	}

	instanceId := uuid.New()
	instance := model.NewApiInstance(testModel, instanceId)
	instance.SetDisplayName("Test API Instance")

	// Add system instance (no event - SystemInstance is still a struct)
	err = testModel.AddSystemInstance(systemInstance)
	assert.NoError(t, err)

	// Event 1: create API instance
	err = testModel.AddApiInstance(instance)
	assert.NoError(t, err)

	// Set system instance reference by ID
	// Event 2: update instance
	instance.SetSystemInstanceById(systemInstanceId)

	sysInstRef := instance.GetSystemInstance()
	assert.NotNil(t, sysInstRef)
	assert.Equal(t, systemInstanceId, sysInstRef.InstanceId)
	assert.Same(t, systemInstance, sysInstRef.SystemInstance)

	eventList := sink.GetList()
	assert.Equal(t, 2, len(eventList))
	assert.True(t, strings.HasPrefix(eventList[0], fmt.Sprintf("CreateOperation: APIInstance %s", instanceId.String())))
	assert.True(t, strings.HasPrefix(eventList[1], fmt.Sprintf("UpdateOperation: APIInstance %s", instanceId.String())))
}
