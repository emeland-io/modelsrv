// Package metrics provides Prometheus instrumentation for the modelsrv landscape model.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.emeland.io/modelsrv/pkg/model"
)

// Collector implements [prometheus.Collector] and exposes per-resource-type
// object counts as gauges.
type Collector struct {
	model model.Model
	desc  *prometheus.Desc
}

// NewCollector returns a collector that queries m for resource counts on each scrape.
func NewCollector(m model.Model) *Collector {
	return &Collector{
		model: m,
		desc: prometheus.NewDesc(
			"emeland_resource_count",
			"Number of objects per resource type in the landscape model.",
			[]string{"type"}, nil,
		),
	}
}

// Describe implements [prometheus.Collector].
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// Collect implements [prometheus.Collector].
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	type counter struct {
		label string
		fn    func() (int, error)
	}
	counters := []counter{
		{"Node", lenFunc(c.model.GetNodes)},
		{"NodeType", lenFunc(c.model.GetNodeTypes)},
		{"Context", lenFunc(c.model.GetContexts)},
		{"ContextType", lenFunc(c.model.GetContextTypes)},
		{"System", lenFunc(c.model.GetSystems)},
		{"SystemInstance", lenFunc(c.model.GetSystemInstances)},
		{"API", lenFunc(c.model.GetApis)},
		{"ApiInstance", lenFunc(c.model.GetApiInstances)},
		{"Component", lenFunc(c.model.GetComponents)},
		{"ComponentInstance", lenFunc(c.model.GetComponentInstances)},
		{"Finding", lenFunc(c.model.GetFindings)},
		{"FindingType", lenFunc(c.model.GetFindingTypes)},
		{"OrgUnit", lenFunc(c.model.GetOrgUnits)},
		{"Group", lenFunc(c.model.GetGroups)},
		{"Identity", lenFunc(c.model.GetIdentities)},
	}
	for _, ct := range counters {
		n, err := ct.fn()
		if err != nil {
			continue
		}
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, float64(n), ct.label)
	}
}

func lenFunc[T any](fn func() ([]T, error)) func() (int, error) {
	return func() (int, error) {
		items, err := fn()
		return len(items), err
	}
}
