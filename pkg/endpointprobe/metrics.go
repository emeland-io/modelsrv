package endpointprobe

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics exposes certprobe Prometheus gauges.
// Register on a shared registry (e.g. modelsrv's) — names are prefixed
// with certprobe_ so they sit alongside emeland_* metrics without collision.
type Metrics struct {
	probeSuccess      *prometheus.GaugeVec
	certRemainingSecs *prometheus.GaugeVec
}

// NewMetrics registers certprobe gauges on reg and returns a Metrics recorder.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		probeSuccess: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "certprobe_probe_success",
			Help: "1 when the probe succeeded, 0 otherwise.",
		}, []string{"api_instance_id", "host"}),
		certRemainingSecs: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "certprobe_cert_remaining_seconds",
			Help: "Seconds until TLS certificate expiry; negative when already expired.",
		}, []string{"api_instance_id", "host"}),
	}

	reg.MustRegister(m.probeSuccess, m.certRemainingSecs)
	return m
}

// Record updates Prometheus gauges from a probe result.
func (m *Metrics) Record(result ProbeResult) {
	labels := prometheus.Labels{
		"api_instance_id": result.Target.ApiInstanceID.String(),
		"host":            result.Target.DedupeKey,
	}

	if result.Success {
		m.probeSuccess.With(labels).Set(1)
	} else {
		m.probeSuccess.With(labels).Set(0)
	}

	if result.HasCert {
		m.certRemainingSecs.With(labels).Set(result.CertRemaining.Seconds())
	}
}
