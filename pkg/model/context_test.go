package model_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gitlab.com/emeland/modelsrv/pkg/events"
	"gitlab.com/emeland/modelsrv/pkg/model"
)

func TestContextOperations(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	contextId := uuid.New()
	context := model.NewContext(testModel, contextId)

	// this must not create an event, as the context has not been registered with the system
	context.SetDisplayName("Test Context")
	assert.Equal(t, "Test Context", context.GetDisplayName())

	// this must not create an event, as the context has not been registered with the system
	context.SetDescription("a test context")
	assert.Equal(t, "a test context", context.GetDescription())

	// Test getting non-existent API
	assert.Nil(t, testModel.GetContextById(contextId))

	// Add Context and verify it exists
	// Event: 1: create
	err = testModel.AddContext(context)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Same(t, context, testModel.GetContextById(contextId))

	// update the DisplayName. This MUST create an event, after the object has been registered
	// with the model.
	// Event 2: update
	context.SetDisplayName("the real test context")

	// update the description. This MUST create an event, after the object has been registered
	// with the model.
	// Event 3: update
	context.SetDescription("a test context, but with more bla bla")

	// create a new go object and re-submit under the same UUID, but with other values
	context2 := model.NewContext(testModel, contextId)
	context2.SetDisplayName("The other Test Context")
	context2.SetDescription("a different test context, but same Id")

	//only when the object is added, it should trigger an event.
	// Event 4: update
	err = testModel.AddContext(context2)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Same(t, context2, testModel.GetContextById(contextId))

	// Delete API and verify it's gone
	// Event: 5: delete
	err = testModel.DeleteContextById(contextId)
	assert.NoError(t, err)
	assert.Nil(t, testModel.GetContextById(contextId))

	eventList := sink.GetList()
	assert.Equal(t, 5, len(eventList))
	assert.True(t, strings.HasPrefix(eventList[0], fmt.Sprintf("CreateOperation: Context %s", contextId.String())))
	assert.True(t, strings.HasPrefix(eventList[1], fmt.Sprintf("UpdateOperation: Context %s", contextId.String())))
	assert.True(t, strings.HasPrefix(eventList[2], fmt.Sprintf("UpdateOperation: Context %s", contextId.String())))
	assert.True(t, strings.HasPrefix(eventList[3], fmt.Sprintf("UpdateOperation: Context %s", contextId.String())))
	assert.True(t, strings.HasPrefix(eventList[4], fmt.Sprintf("DeleteOperation: Context %s", contextId.String())))
}

func TestContextAnnotations(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	contextId := uuid.New()
	context := model.NewContext(testModel, contextId)
	context.SetDisplayName("Test Context")
	context.SetDescription("a test context")

	// Event 1: create
	err = testModel.AddContext(context)
	assert.NoError(t, err)

	// Add annotations
	annotations := context.GetAnnotations()
	assert.NotNil(t, annotations)
	// Event 2,3: add keys -> update context
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
	// Event 4: delete annotation -> update context
	annotations.Delete("key1")
	keys = slices.Collect(annotations.GetKeys())
	assert.NotContains(t, keys, "key1")
	assert.Contains(t, keys, "key2")

	value1 = annotations.GetValue("key1")
	assert.Equal(t, "", value1) // should return empty string for non-existent key

	eventList := sink.GetList()
	assert.Equal(t, 4, len(eventList))
	assert.True(t, strings.HasPrefix(eventList[0], fmt.Sprintf("CreateOperation: Context %s", contextId.String())))
	assert.True(t, strings.HasPrefix(eventList[1], fmt.Sprintf("UpdateOperation: Context %s", contextId.String())))
	assert.True(t, strings.HasPrefix(eventList[2], fmt.Sprintf("UpdateOperation: Context %s", contextId.String())))
	assert.True(t, strings.HasPrefix(eventList[3], fmt.Sprintf("UpdateOperation: Context %s", contextId.String())))

}

func TestContextSetParent(t *testing.T) {
	sink := events.NewListSink()
	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	parentId := uuid.New()
	parent := model.NewContext(testModel, parentId)
	parent.SetDisplayName("Parent Context")
	parent.SetDescription("a parent context")

	contextId := uuid.New()
	context := model.NewContext(testModel, contextId)
	context.SetDisplayName("Test Context")
	context.SetDescription("a test context")

	// Event 1: create parent
	err = testModel.AddContext(parent)
	assert.NoError(t, err)

	// Event 2: create context
	err = testModel.AddContext(context)
	assert.NoError(t, err)

	// Set parent
	// Event 3: update context
	context.SetParentById(parentId)

	parentRetrieved, err := context.GetParent()
	assert.NoError(t, err)
	assert.Same(t, parent, parentRetrieved)

	eventList := sink.GetList()
	assert.Equal(t, 4, len(eventList))
	assert.True(t, strings.HasPrefix(eventList[0], fmt.Sprintf("CreateOperation: Context %s", parentId.String())))
	assert.True(t, strings.HasPrefix(eventList[1], fmt.Sprintf("CreateOperation: Context %s", contextId.String())))

}
