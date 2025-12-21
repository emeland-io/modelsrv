package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// These are tests that are only possible within the package itself

func TestNewModel(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err, "NewModel should not return an error")
	assert.NotNil(t, model, "NewModel should return a non-nil model")

	// Verify all maps are initialized
	assert.NotNil(t, model.systemsByUUID, "SystemsByUUID map should be initialized")
	assert.NotNil(t, model.apisByUUID, "APIsByUUID map should be initialized")
	assert.NotNil(t, model.componentsByUUID, "ComponentsByUUID map should be initialized")
	assert.NotNil(t, model.systemInstancesByUUID, "SystemInstances map should be initialized")
	assert.NotNil(t, model.apiInstancesByUUID, "APIInstances map should be initialized")
	assert.NotNil(t, model.componentInstancesByUUID, "ComponentInstances map should be initialized")
}
