package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlobs "go.emeland.io/modelsrv/pkg/model/observability"
)

func seedMetric(t *testing.T, m model.Model) uuid.UUID {
	t.Helper()
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	metric := mdlobs.NewMetric(id)
	metric.SetDisplayName("p99 API latency")
	require.NoError(t, m.AddMetric(metric))
	return id
}

func TestAddThresholdRejectsUnresolvedMetricRef(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	th := mdlobs.NewThreshold(uuid.New())
	th.SetDisplayName("orphan threshold")
	th.SetMetricById(uuid.New())

	assert.ErrorIs(t, m.AddThreshold(th), common.ErrMetricNotFound)
}

func TestAddThresholdRejectsNilMetricRef(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	th := mdlobs.NewThreshold(uuid.New())
	th.SetDisplayName("no metric")

	assert.ErrorContains(t, m.AddThreshold(th), "metricRef is required")
}

func TestAddMetricValueRejectsUnresolvedMetricRef(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	mv := mdlobs.NewMetricValue(uuid.New())
	mv.SetDisplayName("orphan value")
	mv.SetMetricById(uuid.New())
	mv.SetValue("412")

	assert.ErrorIs(t, m.AddMetricValue(mv), common.ErrMetricNotFound)
}

func TestAddThresholdAndMetricValueWithValidMetric(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	metricID := seedMetric(t, m)

	thID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	th := mdlobs.NewThreshold(thID)
	th.SetDisplayName("Latency SLO breach")
	th.SetMetricById(metricID)
	require.NoError(t, m.AddThreshold(th))

	mvID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	mv := mdlobs.NewMetricValue(mvID)
	mv.SetDisplayName("Current p99 latency")
	mv.SetMetricById(metricID)
	mv.SetValue("412")
	require.NoError(t, m.AddMetricValue(mv))

	gotTh := m.GetThresholdById(thID)
	require.NotNil(t, gotTh)
	assert.Equal(t, metricID, gotTh.GetMetricId())

	gotMv := m.GetMetricValueById(mvID)
	require.NotNil(t, gotMv)
	assert.Equal(t, metricID, gotMv.GetMetricId())
	assert.Equal(t, "412", gotMv.GetValue())
}
