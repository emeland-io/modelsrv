package endpointprobe

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultCollectionInterval = 5 * time.Minute

// httpCheckTarget is one OpenTelemetry Collector http_check target entry.
type httpCheckTarget struct {
	Method   string `yaml:"method"`
	Endpoint string `yaml:"endpoint"`
}

type httpCheckMetric struct {
	Enabled bool `yaml:"enabled"`
}

type httpCheckReceiver struct {
	CollectionInterval string                     `yaml:"collection_interval"`
	Metrics            map[string]httpCheckMetric `yaml:"metrics"`
	Targets            []httpCheckTarget          `yaml:"targets"`
}

type httpCheckConfig struct {
	Receivers map[string]httpCheckReceiver `yaml:"receivers"`
}

// RenderHTTPCheckConfig builds an OpenTelemetry Collector receivers fragment for
// the http_check receiver, with httpcheck.tls.cert_remaining enabled and one GET
// target per ProbeTarget.URL. Targets are sorted by URL for stable output.
// collectionInterval defaults to 5m when zero or negative.
func RenderHTTPCheckConfig(targets []ProbeTarget, collectionInterval time.Duration) ([]byte, error) {
	if collectionInterval <= 0 {
		collectionInterval = defaultCollectionInterval
	}

	sorted := make([]ProbeTarget, len(targets))
	copy(sorted, targets)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].URL < sorted[j].URL
	})

	outTargets := make([]httpCheckTarget, 0, len(sorted))
	for _, t := range sorted {
		outTargets = append(outTargets, httpCheckTarget{
			Method:   "GET",
			Endpoint: t.URL,
		})
	}

	cfg := httpCheckConfig{
		Receivers: map[string]httpCheckReceiver{
			"http_check": {
				CollectionInterval: formatCollectionInterval(collectionInterval),
				Metrics: map[string]httpCheckMetric{
					"httpcheck.tls.cert_remaining": {Enabled: true},
				},
				Targets: outTargets,
			},
		},
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&cfg); err != nil {
		return nil, fmt.Errorf("marshal http_check config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("marshal http_check config: %w", err)
	}
	return buf.Bytes(), nil
}

// formatCollectionInterval prefers compact units (5m, 1h) over Go's String() (5m0s).
func formatCollectionInterval(d time.Duration) string {
	if d <= 0 {
		return formatCollectionInterval(defaultCollectionInterval)
	}
	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", d/time.Hour)
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", d/time.Minute)
	}
	if d%time.Second == 0 {
		return fmt.Sprintf("%ds", d/time.Second)
	}
	return d.String()
}
