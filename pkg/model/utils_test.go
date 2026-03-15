package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/pkg/events"
)

type expectedEvent struct {
	resourceType events.ResourceType
	operation    events.Operation
	resourceId   uuid.UUID
}

func checkEvents(t *testing.T, actual []events.Event, expected []expectedEvent) {
	assert.Len(t, actual, len(expected))
	for i, expected := range expected {
		assert.Equal(t, expected.resourceType, actual[i].ResourceType,
			"resource type mismatch for event %d: expected %s, got %s", i,
			expected.resourceType.String(),
			actual[i].ResourceType.String())
		assert.Equal(t, expected.operation, actual[i].Operation,
			"operation mismatch for event %d: expected %s, got %s", i,
			expected.operation.String(),
			actual[i].Operation.String())
		assert.Equal(t, expected.resourceId, actual[i].ResourceId,
			"resource ID mismatch for event %d: expected %s, got %s", i,
			expected.resourceId.String(),
			actual[i].ResourceId.String())
	}
}
