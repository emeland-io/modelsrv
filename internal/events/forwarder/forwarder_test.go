package eventforwarder

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

func TestEventForwarder(t *testing.T) {

	forwarder := NewEventForwarder(4)
	testModel, err := model.NewModel(forwarder)
	assert.NoError(t, err)

	contextId := uuid.New()
	context := model.NewContext(testModel, contextId)
	context.SetDisplayName("Test Context")
	context.SetDescription("a test context")
	context.GetAnnotations().Add("a key", "a value")

	err = forwarder.Receive(events.ContextResource, events.CreateOperation, contextId, context)
	assert.NoError(t, err)

	event, err := forwarder.queue.Dequeue()
	assert.NoError(t, err)

	assert.Equal(t, events.ContextResource, event.resourceType)
	assert.Equal(t, events.CreateOperation, event.operation)
	assert.Equal(t, contextId, event.resourceId)
	assert.Len(t, event.objectJson, 1)
	assert.Contains(t, event.objectJson[0], `"contextId":"`+contextId.String()+`"`)
	assert.Contains(t, event.objectJson[0], `"displayName":"Test Context"`)
	assert.Contains(t, event.objectJson[0], `"description":"a test context"`)
	assert.Contains(t, event.objectJson[0], `"annotations":[{"key":"a key","value":"a value"}]`)

}
