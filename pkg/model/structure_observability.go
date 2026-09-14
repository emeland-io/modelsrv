package model

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlobs "go.emeland.io/modelsrv/pkg/model/observability"
)

// Observability resources follow the same Add posture as the other instance
// types (ApiInstance/SystemInstance): no Add-time reference validation. A
// MetricValue or Threshold may reference a MetricInstance that is not yet
// present (e.g. delivered earlier in a replication or snapshot stream), and a
// MetricInstance may reference an absent Metric. This keeps event apply
// order-tolerant; dangling references can be surfaced later via findings.

// AddMetric implements [Model].
func (m *modelData) AddMetric(metric mdlobs.Metric) error {
	return addEventEnabled(m, metric, mdlobs.Metric.GetMetricId,
		func(x mdlobs.Metric, s events.EventSink) { x.Register(s) },
		m.metricsByUUID, events.MetricResource)
}

// DeleteMetricById implements [Model].
func (m *modelData) DeleteMetricById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.metricsByUUID, events.MetricResource, common.ErrMetricNotFound)
}

// GetMetricById implements [Model].
func (m *modelData) GetMetricById(id uuid.UUID) mdlobs.Metric {
	return getEventEnabled(m, id, m.metricsByUUID)
}

// GetMetrics implements [Model].
func (m *modelData) GetMetrics() ([]mdlobs.Metric, error) {
	return getAllEventEnabled(m, m.metricsByUUID)
}

// AddThreshold implements [Model].
func (m *modelData) AddThreshold(threshold mdlobs.Threshold) error {
	return addEventEnabled(m, threshold, mdlobs.Threshold.GetThresholdId,
		func(x mdlobs.Threshold, s events.EventSink) { x.Register(s) },
		m.thresholdsByUUID, events.ThresholdResource)
}

// DeleteThresholdById implements [Model].
func (m *modelData) DeleteThresholdById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.thresholdsByUUID, events.ThresholdResource, common.ErrThresholdNotFound)
}

// GetThresholdById implements [Model].
func (m *modelData) GetThresholdById(id uuid.UUID) mdlobs.Threshold {
	return getEventEnabled(m, id, m.thresholdsByUUID)
}

// GetThresholds implements [Model].
func (m *modelData) GetThresholds() ([]mdlobs.Threshold, error) {
	return getAllEventEnabled(m, m.thresholdsByUUID)
}

// AddMetricInstance implements [Model].
func (m *modelData) AddMetricInstance(metricInstance mdlobs.MetricInstance) error {
	return addEventEnabled(m, metricInstance, mdlobs.MetricInstance.GetMetricInstanceId,
		func(x mdlobs.MetricInstance, s events.EventSink) { x.Register(s) },
		m.metricInstancesByUUID, events.MetricInstanceResource)
}

// DeleteMetricInstanceById implements [Model].
func (m *modelData) DeleteMetricInstanceById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.metricInstancesByUUID, events.MetricInstanceResource, common.ErrMetricInstanceNotFound)
}

// GetMetricInstanceById implements [Model].
func (m *modelData) GetMetricInstanceById(id uuid.UUID) mdlobs.MetricInstance {
	return getEventEnabled(m, id, m.metricInstancesByUUID)
}

// GetMetricInstances implements [Model].
func (m *modelData) GetMetricInstances() ([]mdlobs.MetricInstance, error) {
	return getAllEventEnabled(m, m.metricInstancesByUUID)
}

// AddMetricValue implements [Model].
func (m *modelData) AddMetricValue(metricValue mdlobs.MetricValue) error {
	return addEventEnabled(m, metricValue, mdlobs.MetricValue.GetMetricValueId,
		func(x mdlobs.MetricValue, s events.EventSink) { x.Register(s) },
		m.metricValuesByUUID, events.MetricValueResource)
}

// DeleteMetricValueById implements [Model].
func (m *modelData) DeleteMetricValueById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.metricValuesByUUID, events.MetricValueResource, common.ErrMetricValueNotFound)
}

// GetMetricValueById implements [Model].
func (m *modelData) GetMetricValueById(id uuid.UUID) mdlobs.MetricValue {
	return getEventEnabled(m, id, m.metricValuesByUUID)
}

// GetMetricValues implements [Model].
func (m *modelData) GetMetricValues() ([]mdlobs.MetricValue, error) {
	return getAllEventEnabled(m, m.metricValuesByUUID)
}
