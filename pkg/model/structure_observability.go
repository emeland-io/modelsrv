package model

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlobs "go.emeland.io/modelsrv/pkg/model/observability"
)

// validateMetricRef enforces the mandatory Metric reference shared by
// [mdlobs.Threshold] and [mdlobs.MetricValue]: the id must be set and must
// resolve to a Metric already in the model.
func validateMetricRef(metricID uuid.UUID, m *modelData) error {
	if metricID == uuid.Nil {
		return fmt.Errorf("metricRef is required")
	}
	if m.GetMetricById(metricID) == nil {
		return common.ErrMetricNotFound
	}
	return nil
}

func validateThreshold(t mdlobs.Threshold, m *modelData) error {
	if t == nil {
		return fmt.Errorf("threshold is nil")
	}
	if t.GetThresholdId() == uuid.Nil {
		return common.ErrUUIDNotSet
	}
	return validateMetricRef(t.GetMetricId(), m)
}

func validateMetricValue(v mdlobs.MetricValue, m *modelData) error {
	if v == nil {
		return fmt.Errorf("metric value is nil")
	}
	if v.GetMetricValueId() == uuid.Nil {
		return common.ErrUUIDNotSet
	}
	return validateMetricRef(v.GetMetricId(), m)
}

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
	if err := validateThreshold(threshold, m); err != nil {
		return err
	}
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

// AddMetricValue implements [Model].
func (m *modelData) AddMetricValue(metricValue mdlobs.MetricValue) error {
	if err := validateMetricValue(metricValue, m); err != nil {
		return err
	}
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
