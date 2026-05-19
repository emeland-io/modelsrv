// Package metrics provides Prometheus instrumentation for the modelsrv landscape model.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.emeland.io/modelsrv/pkg/model"
)

// Collector implements [prometheus.Collector] and exposes per-resource-type
// object counts as gauges.
type Collector struct {
	types []model.ResourceTypeInfo
	desc  *prometheus.Desc
}

// NewCollector returns a collector that queries m for resource counts on each scrape.
func NewCollector(m model.Model) *Collector {
	return &Collector{
		types: model.ResourceTypes(m),
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
	for _, rt := range c.types {
		n, err := rt.Count()
		if err != nil {
			continue
		}
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, float64(n), rt.Name)
	}
}
