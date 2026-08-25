package ingress

import (
	"fmt"

	"go.emeland.io/modelsrv/pkg/model"
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
	metricID, err := uuidRefFromMap(spec, "metricRef", "metricId")
	if err != nil {
		return err
	}

	th := mdlobs.NewThreshold(id)
	th.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		th.SetDescription(desc)
	}
	th.SetMetricById(metricID)
	if err := applyAnnotations(th.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddThreshold(th)
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
	metricID, err := uuidRefFromMap(spec, "metricRef", "metricId")
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
	mv.SetMetricById(metricID)
	mv.SetValue(valueRaw)
	if err := applyAnnotations(mv.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddMetricValue(mv)
}
