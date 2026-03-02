package events_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	ievents "go.emeland.io/modelsrv/internal/events"
	"go.emeland.io/modelsrv/pkg/model"
)

func TestEventForwarder(t *testing.T) {

	sink := ievents.NewEnumeratedListSink()

	ievents.NewEventForwarder("http://localhost:8080", sink)

	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	contextId := uuid.New()
	context := model.NewContext(testModel, contextId)
	context.SetDisplayName("Test Context")
	context.SetDescription("a test context")
	context.GetAnnotations().Add("a key", "a value")

	testModel.AddContext(context)

	// TODO: find out if the event was correctly forwarded to the subscriber URL (e.g. by mocking the subscriber and checking if it received the event)
}
