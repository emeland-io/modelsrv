package metrics_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/metrics"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/node"
	"go.emeland.io/modelsrv/pkg/model/system"
)

func TestCollector(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	c := metrics.NewCollector(m)
	reg := prometheus.NewRegistry()
	reg.MustRegister(c)

	// Empty model: all counts should be 0.
	count := testutil.CollectAndCount(c)
	assert.Equal(t, 15, count, "expected one metric per resource type")

	val := getGaugeValue(t, reg, "emeland_resource_count", "System")
	assert.Equal(t, 0.0, val)

	// Add resources and verify counts update.
	sys := system.NewSystem(m.GetSink(), uuid.New())
	sys.SetDisplayName("s1")
	require.NoError(t, m.AddSystem(sys))

	nt := node.NewNodeType(m.GetSink(), uuid.New())
	nt.SetDisplayName("nt1")
	require.NoError(t, m.AddNodeType(nt))

	val = getGaugeValue(t, reg, "emeland_resource_count", "System")
	assert.Equal(t, 1.0, val)

	val = getGaugeValue(t, reg, "emeland_resource_count", "NodeType")
	assert.Equal(t, 1.0, val)

	val = getGaugeValue(t, reg, "emeland_resource_count", "Node")
	assert.Equal(t, 0.0, val)
}

func getGaugeValue(t *testing.T, reg *prometheus.Registry, name, typeLabel string) float64 {
	t.Helper()
	mfs, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "type" && lp.GetValue() == typeLabel {
					return m.GetGauge().GetValue()
				}
			}
		}
	}
	t.Fatalf("metric %s{type=%q} not found", name, typeLabel)
	return 0
}
