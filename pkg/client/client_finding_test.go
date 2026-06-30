package client_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/finding"
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
	assert.Equal(t, testIDs["Finding"], uuid.UUID(got.FindingId))
	assert.Equal(t, "Test Finding", got.DisplayName)
}

func TestGetByIdFindingTypeDescription(t *testing.T) {
	c, m := setupTestServer(t)
	id := uuid.New()
	ft := finding.NewFindingType(id)
	ft.SetDisplayName("Typed finding")
	ft.SetDescription("A detailed finding type description")
	require.NoError(t, m.AddFindingType(ft))

	got, err := c.GetFindingTypeById(id)
	require.NoError(t, err)
	assert.Equal(t, "A detailed finding type description", got.GetDescription())
}

func TestListFindingTypeDescription(t *testing.T) {
	c, m := setupTestServer(t)
	id := uuid.New()
	ft := finding.NewFindingType(id)
	ft.SetDisplayName("Listed finding type")
	ft.SetDescription("Description in list response")
	require.NoError(t, m.AddFindingType(ft))

	list, err := c.GetFindingTypes()
	require.NoError(t, err)
	require.NotEmpty(t, list)

	var found finding.FindingType
	for _, item := range list {
		if item.GetFindingTypeId() == id {
			found = item
			break
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, "Description in list response", found.GetDescription())
}
