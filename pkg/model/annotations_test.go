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

func TestAnnotationsOperations(t *testing.T) {
	sink := events.NewListSink()

	testModel, err := model.NewModel(sink)
	assert.NoError(t, err)

	annotations := model.NewAnnotations(testModel, sink)

	annotations.Add("key1", "value1")
	assert.Equal(t, "value1", annotations.GetValue("key1"))

	annotations.Add("key2", "value2")
	assert.Equal(t, "value2", annotations.GetValue("key2"))

	annotations.Add("key1", "value3") // update existing key
	assert.Equal(t, "value3", annotations.GetValue("key1"))

	annotations.Delete("key2")
	assert.Equal(t, "", annotations.GetValue("key2"))

	keys := slices.Collect(annotations.GetKeys())
	assert.Contains(t, keys, "key1")
	assert.NotContains(t, keys, "key2")

	eventList := sink.GetList()
	assert.Equal(t, 4, len(eventList))
	assert.True(t, strings.HasPrefix(eventList[0], fmt.Sprintf("CreateOperation: Annotations %s: [map[key1:value1]]", uuid.Nil.String())))
	assert.True(t, strings.HasPrefix(eventList[1], fmt.Sprintf("CreateOperation: Annotations %s: [map[key2:value2]]", uuid.Nil.String())))
	assert.True(t, strings.HasPrefix(eventList[2], fmt.Sprintf("UpdateOperation: Annotations %s: [map[key1:value3]]", uuid.Nil.String())))
	assert.True(t, strings.HasPrefix(eventList[3], fmt.Sprintf("DeleteOperation: Annotations %s", uuid.Nil.String())))
}
