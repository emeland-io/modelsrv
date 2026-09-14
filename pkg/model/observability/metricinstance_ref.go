package observability

import "github.com/google/uuid"

// MetricInstanceRef references a [MetricInstance] by resolved object and/or id.
type MetricInstanceRef struct {
	MetricInstance   MetricInstance
	MetricInstanceId uuid.UUID
}

// ResolvedMetricInstance returns the embedded [MetricInstance] when present, or nil.
func (r *MetricInstanceRef) ResolvedMetricInstance() MetricInstance {
	if r == nil {
		return nil
	}
	return r.MetricInstance
}

// EffectiveMetricInstanceID returns the metric instance id from the embedded
// object or from [MetricInstanceRef.MetricInstanceId].
func (r *MetricInstanceRef) EffectiveMetricInstanceID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.MetricInstance != nil {
		return r.MetricInstance.GetMetricInstanceId()
	}
	return r.MetricInstanceId
}
