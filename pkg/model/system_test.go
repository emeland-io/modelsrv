package model_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

func TestSystemOperations(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	sysId := uuid.New()
	version := model.Version{Version: "1.0.0"}

	system := model.NewSystem(testModel, sysId)
	assert.NotNil(t, system)

	system.SetDisplayName("Test System")
	assert.Equal(t, "Test System", system.GetDisplayName())

	system.SetDescription("a test system")
	assert.Equal(t, "a test system", system.GetDescription())

	system.SetVersion(version)
	assert.Equal(t, version, system.GetVersion())

	// Test getting non-existent System
	assert.Nil(t, testModel.GetSystemById(sysId))

	// Add System and verify it exists
	// Event: 1: create
	err = testModel.AddSystem(system)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Same(t, system, testModel.GetSystemById(sysId))

	// update the DisplayName. This MUST create an event, after the object has been registered
	// with the model.
	// Event 2: update
	system.SetDisplayName("the real test system")

	// update the description. This MUST create an event, after the object has been registered
	// with the model.
	// Event 3: update
	system.SetDescription("a test system, but with more bla bla")

	// create a new go object and re-submit under the same UUID, but with other values
	system2 := model.NewSystem(testModel, sysId)
	system2.SetDisplayName("The other Test System")
	system2.SetDescription("a different test system, but same Id")

	//only when the object is added, it should trigger an event.
	// Event 4: update
	err = testModel.AddSystem(system2)
	assert.NoError(t, err)

	eventList := sink.GetList()
	assert.Equal(t, 4, len(eventList))
	assert.True(t, strings.HasPrefix(eventList[0], fmt.Sprintf("CreateOperation: System %s", sysId.String())))
	assert.True(t, strings.HasPrefix(eventList[1], fmt.Sprintf("UpdateOperation: System %s", sysId.String())))
	assert.True(t, strings.HasPrefix(eventList[2], fmt.Sprintf("UpdateOperation: System %s", sysId.String())))
	assert.True(t, strings.HasPrefix(eventList[3], fmt.Sprintf("UpdateOperation: System %s", sysId.String())))
}
