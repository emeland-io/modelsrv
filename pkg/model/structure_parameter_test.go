package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlparameter "go.emeland.io/modelsrv/pkg/model/parameter"
)

func seedParameter(t *testing.T, m model.Model) uuid.UUID {
	t.Helper()
	paramID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	param := mdlparameter.NewParameter(paramID)
	param.SetDisplayName("Region")
	require.NoError(t, m.AddParameter(param))
	return paramID
}

func newValidValue(id, paramID uuid.UUID, name string) mdlparameter.ValidValue {
	vv := mdlparameter.NewValidValue(id)
	vv.SetDisplayName(name)
	vv.SetParameterById(paramID)
	return vv
}

func TestValidValueUniqueness(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	paramID := seedParameter(t, m)
	vvID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	require.NoError(t, m.AddValidValue(newValidValue(vvID, paramID, "eu-west-1")))

	updated := newValidValue(vvID, paramID, "eu-west-1")
	require.NoError(t, m.AddValidValue(updated))

	conflict := newValidValue(uuid.New(), paramID, "eu-west-1")
	assert.ErrorIs(t, m.AddValidValue(conflict), common.ErrValidValueConflict)
	assert.Nil(t, m.GetValidValueById(conflict.GetValidValueId()))
}

func TestAddValidValueRejectsMissingParameter(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	vv := newValidValue(uuid.New(), uuid.New(), "orphan")
	assert.ErrorIs(t, m.AddValidValue(vv), common.ErrParameterNotFound)
}

func TestAddParameterRejectsUnknownValidValue(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	paramID := uuid.New()
	param := mdlparameter.NewParameter(paramID)
	param.SetDisplayName("Region")
	param.SetValues([]uuid.UUID{uuid.New()})
	assert.ErrorIs(t, m.AddParameter(param), common.ErrValidValueNotFound)
}

func TestAddParameterRejectsValidValueFromOtherParameter(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	paramA := seedParameter(t, m)
	vvID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	require.NoError(t, m.AddValidValue(newValidValue(vvID, paramA, "eu-west-1")))

	paramB := mdlparameter.NewParameter(uuid.New())
	paramB.SetDisplayName("Other")
	require.NoError(t, m.AddParameter(paramB))

	paramB.SetValues([]uuid.UUID{vvID})
	assert.Error(t, m.AddParameter(paramB))
}

func TestAddParameterAcceptsOwnValidValues(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	paramID := seedParameter(t, m)
	vvID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	require.NoError(t, m.AddValidValue(newValidValue(vvID, paramID, "eu-west-1")))

	param := m.GetParameterById(paramID)
	require.NotNil(t, param)
	param.SetValues([]uuid.UUID{vvID})
	require.NoError(t, m.AddParameter(param))
	assert.Equal(t, []uuid.UUID{vvID}, m.GetParameterById(paramID).GetValues())
}
