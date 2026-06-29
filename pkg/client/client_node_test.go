package client_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/model/common"
)

func TestListNode(t *testing.T) {
	c, m := setupTestServer(t)
	loadTestModel(t, m)

	list, err := c.GetNodes()
	require.NoError(t, err)
	require.NotNil(t, list)
	assert.Greater(t, len(list), 0, "Node list should not be empty")
	assert.Equal(t, testIDs["Node"], uuid.UUID(list[0].NodeId))
	assert.Equal(t, "Test Node", list[0].DisplayName)
	assert.Equal(t, oapi.NodeTypeViewResourceNodeType, list[0].NodeType.Resource)
}

func TestGetByIdNode(t *testing.T) {
	c, m := setupTestServer(t)
	loadTestModel(t, m)

	_, err := c.GetNodeById(uuid.New())
	assert.ErrorIs(t, err, common.ErrNodeNotFound)

	got, err := c.GetNodeById(testIDs["Node"])
	require.NoError(t, err)
	assert.Equal(t, testIDs["Node"], uuid.UUID(got.NodeId))
	assert.Equal(t, "Test Node", got.DisplayName)
}
