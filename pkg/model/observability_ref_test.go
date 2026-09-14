package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
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

func TestAddThresholdAcceptsUnresolvedMetricInstanceRef(t *testing.T) {
	// References are not existence-checked on Add, so event apply stays
	// order-tolerant (a Threshold may arrive before its MetricInstance).
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	th := mdlobs.NewThreshold(uuid.New())
	th.SetDisplayName("dangling threshold")
	th.SetMetricInstanceById(uuid.New())

	assert.NoError(t, m.AddThreshold(th))
}

func TestAddThresholdAcceptsNilMetricInstanceRef(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	th := mdlobs.NewThreshold(uuid.New())
	th.SetDisplayName("no metric instance")

	assert.NoError(t, m.AddThreshold(th))
}

func seedMetricInstance(t *testing.T, m model.Model, metricID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	mi := mdlobs.NewMetricInstance(id)
	mi.SetDisplayName("p99 API latency for orders-api")
	mi.SetMetricById(metricID)
	require.NoError(t, m.AddMetricInstance(mi))
	return id
}

func TestAddMetricInstanceAcceptsUnresolvedMetricRef(t *testing.T) {
	// The Metric reference is optional and not existence-checked: a
	// MetricInstance may reference a Metric that is not (yet) in the model.
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	mi := mdlobs.NewMetricInstance(uuid.New())
	mi.SetDisplayName("dangling metric ref")
	mi.SetMetricById(uuid.New())

	assert.NoError(t, m.AddMetricInstance(mi))
}

func TestAddMetricInstanceAcceptsNilMetricRef(t *testing.T) {
	// A MetricInstance is a concrete measured thing; the abstract Metric is an
	// optional organizational grouping, so no MetricRef is required. Scraped
	// instances (e.g. from AlertManager) have no reliable parent Metric.
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	mi := mdlobs.NewMetricInstance(uuid.New())
	mi.SetDisplayName("no metric")

	assert.NoError(t, m.AddMetricInstance(mi))
}

func TestAddMetricValueAcceptsUnresolvedMetricInstanceRef(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	mv := mdlobs.NewMetricValue(uuid.New())
	mv.SetDisplayName("dangling value")
	mv.SetMetricInstanceById(uuid.New())
	mv.SetValue("412")

	assert.NoError(t, m.AddMetricValue(mv))
}

func TestAddMetricValueAcceptsNilMetricInstanceRef(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	mv := mdlobs.NewMetricValue(uuid.New())
	mv.SetDisplayName("no instance")
	mv.SetValue("412")

	assert.NoError(t, m.AddMetricValue(mv))
}

func TestAddThresholdAndMetricValueWithValidMetric(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	metricID := seedMetric(t, m)
	miID := seedMetricInstance(t, m, metricID)

	thID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	th := mdlobs.NewThreshold(thID)
	th.SetDisplayName("Latency SLO breach")
	th.SetMetricInstanceById(miID)
	require.NoError(t, m.AddThreshold(th))

	mvID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	mv := mdlobs.NewMetricValue(mvID)
	mv.SetDisplayName("Current p99 latency")
	mv.SetMetricInstanceById(miID)
	mv.SetValue("412")
	require.NoError(t, m.AddMetricValue(mv))

	gotTh := m.GetThresholdById(thID)
	require.NotNil(t, gotTh)
	assert.Equal(t, miID, gotTh.GetMetricInstanceId())

	gotMi := m.GetMetricInstanceById(miID)
	require.NotNil(t, gotMi)
	assert.Equal(t, metricID, gotMi.GetMetricId())

	gotMv := m.GetMetricValueById(mvID)
	require.NotNil(t, gotMv)
	assert.Equal(t, miID, gotMv.GetMetricInstanceId())
	assert.Equal(t, "412", gotMv.GetValue())
}
