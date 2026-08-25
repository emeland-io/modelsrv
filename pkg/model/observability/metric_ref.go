package observability

import "github.com/google/uuid"

// MetricRef references a [Metric] by resolved object and/or id.
type MetricRef struct {
	Metric   Metric
	MetricId uuid.UUID
}

// ResolvedMetric returns the embedded [Metric] when present, or nil.
func (r *MetricRef) ResolvedMetric() Metric {
	if r == nil {
		return nil
	}
	return r.Metric
}

// EffectiveMetricID returns the metric id from the embedded object or from
// [MetricRef.MetricId].
func (r *MetricRef) EffectiveMetricID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.Metric != nil {
		return r.Metric.GetMetricId()
	}
	return r.MetricId
}
