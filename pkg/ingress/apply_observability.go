package ingress

import (
	"fmt"

	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlobs "go.emeland.io/modelsrv/pkg/model/observability"
)

func applyMetric(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "metricId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	metric := mdlobs.NewMetric(id)
	metric.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		metric.SetDescription(desc)
	}
	if err := applyAnnotations(metric.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddMetric(metric)
}

func applyThreshold(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "thresholdId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	metricInstanceID, err := uuidRefFromMap(spec, "metricInstanceRef", "metricInstanceId")
	if err != nil {
		return err
	}

	th := mdlobs.NewThreshold(id)
	th.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		th.SetDescription(desc)
	}
	th.SetMetricInstanceById(metricInstanceID)
	if err := applyAnnotations(th.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddThreshold(th)
}

func applyMetricInstance(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "metricInstanceId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	metricID, err := uuidRefFromMap(spec, "metricRef", "metricId")
	if err != nil {
		return err
	}

	mi := mdlobs.NewMetricInstance(id)
	mi.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		mi.SetDescription(desc)
	}
	mi.SetMetricById(metricID)
	subject, err := parseMetricInstanceSubject(spec)
	if err != nil {
		return err
	}
	if subject != nil {
		mi.SetSubject(subject)
	}
	if err := applyAnnotations(mi.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddMetricInstance(mi)
}

// parseMetricInstanceSubject parses the optional MetricInstance.subject
// ResourceRef ({resourceId, resourceType}). Returns nil when absent.
func parseMetricInstanceSubject(spec map[string]any) (*common.ResourceRef, error) {
	raw, ok := spec["subject"]
	if !ok || raw == nil {
		return nil, nil
	}
	sm, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("subject must be an object with resourceId and resourceType")
	}
	id, err := parseUUIDField(sm, "resourceId")
	if err != nil {
		return nil, fmt.Errorf("subject: %w", err)
	}
	rtStr, ok := stringField(sm, "resourceType")
	if !ok {
		return nil, fmt.Errorf("subject: resourceType is required")
	}
	rt, err := parseResourceTypeForRef(rtStr)
	if err != nil {
		return nil, fmt.Errorf("subject: %w", err)
	}
	return &common.ResourceRef{ResourceId: id, ResourceType: rt}, nil
}

func applyMetricValue(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "metricValueId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	metricInstanceID, err := uuidRefFromMap(spec, "metricInstanceRef", "metricInstanceId")
	if err != nil {
		return err
	}

	valueRaw, ok := stringField(spec, "value")
	if !ok {
		if n, ok := spec["value"]; ok {
			valueRaw = fmt.Sprint(n)
		}
	}
	if valueRaw == "" {
		return fmt.Errorf("value is required")
	}

	mv := mdlobs.NewMetricValue(id)
	mv.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		mv.SetDescription(desc)
	}
	mv.SetMetricInstanceById(metricInstanceID)
	mv.SetValue(valueRaw)
	if err := applyAnnotations(mv.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddMetricValue(mv)
}
