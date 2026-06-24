package client_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/model/common"
)

func TestListFinding(t *testing.T) {
	c, m := setupTestServer(t)
	loadTestModel(t, m)

	list, err := c.GetFindings()
	require.NoError(t, err)
	require.NotNil(t, list)
	assert.Greater(t, len(list), 0, "Finding list should not be empty")
}

func TestGetByIdFinding(t *testing.T) {
	c, m := setupTestServer(t)
	loadTestModel(t, m)

	_, err := c.GetFindingById(uuid.New())
	assert.ErrorIs(t, err, common.ErrFindingNotFound)

	got, err := c.GetFindingById(testIDs["Finding"])
	require.NoError(t, err)
	assert.Equal(t, testIDs["Finding"], uuid.UUID(got.Id))
	assert.Equal(t, "Test Finding", got.DisplayName)
}
