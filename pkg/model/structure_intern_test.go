package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
)

// These are tests that are only possible within the package itself

func TestNewModel(t *testing.T) {
	sink := events.NewDummySink()
	model, err := NewModel(sink)
	assert.NoError(t, err, "NewModel should not return an error")
	assert.NotNil(t, model, "NewModel should return a non-nil model")

	// Verify the fields are set correctly
	assert.Same(t, sink, model.GetSink())

	// Verify all maps are initialized
	assert.NotNil(t, model.systemsByUUID, "SystemsByUUID map should be initialized")
	assert.NotNil(t, model.apisByUUID, "APIsByUUID map should be initialized")
	assert.NotNil(t, model.componentsByUUID, "ComponentsByUUID map should be initialized")
	assert.NotNil(t, model.systemInstancesByUUID, "SystemInstances map should be initialized")
	assert.NotNil(t, model.apiInstancesByUUID, "APIInstances map should be initialized")
	assert.NotNil(t, model.componentInstancesByUUID, "ComponentInstances map should be initialized")
}

func TestGetCurrentEventSequenceId(t *testing.T) {
	sink := events.NewDummySink()
	m, err := NewModel(sink)
	require.NoError(t, err)

	seq, err := m.GetCurrentEventSequenceId(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "forty-two", seq)
}
