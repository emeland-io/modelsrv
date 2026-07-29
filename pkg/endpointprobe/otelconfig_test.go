package endpointprobe

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/model/api"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func TestRenderCollectorConfig_KnownAnnotations(t *testing.T) {
	t.Parallel()

	apiInstanceID := uuid.MustParse("88888888-0000-4000-8000-000000000001")
	ai := api.NewApiInstance(apiInstanceID)
	ai.GetAnnotations().Add(annProtocol, "https")
	ai.GetAnnotations().Add(annHost, "payments.prod.eu.example.com")
	ai.GetAnnotations().Add(annPort, "443")
	ai.GetAnnotations().Add(annPath, "/api/v1/health")

	target, ok, err := TargetFromApiInstance(ai)
	require.NoError(t, err)
	require.True(t, ok)

	raw, err := RenderCollectorConfig([]ProbeTarget{target}, CollectorConfigOptions{
		CollectionInterval: 5 * time.Minute,
		ListenAddr:         "0.0.0.0:24200",
		Subscribers:        []string{"http://modelsrv:8080"},
	})
	require.NoError(t, err)

	out := string(raw)

	// Receiver
	assert.Contains(t, out, "httpcheck:")
	assert.Contains(t, out, "collection_interval: 5m")
	assert.Contains(t, out, "endpoint: https://payments.prod.eu.example.com:443/api/v1/health")

	// Exporter
	assert.Contains(t, out, "emeland:")
	assert.Contains(t, out, "listen_addr: 0.0.0.0:24200")
	assert.Contains(t, out, "expiry_threshold: 720h")
	assert.Contains(t, out, "http://modelsrv:8080")

	// Endpoint mapping
	assert.Contains(t, out, "https://payments.prod.eu.example.com:443/api/v1/health: 88888888-0000-4000-8000-000000000001")

	// Service pipeline
	assert.Contains(t, out, "pipelines:")

	// Verify it's valid YAML
	var parsed collectorConfig
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	recv := parsed.Receivers["httpcheck"]
	assert.Equal(t, "5m", recv.CollectionInterval)
	assert.True(t, recv.Metrics["httpcheck.tls.cert_remaining"].Enabled)
	require.Len(t, recv.Targets, 1)
	assert.Equal(t, "GET", recv.Targets[0].Method)
	assert.Equal(t, "https://payments.prod.eu.example.com:443/api/v1/health", recv.Targets[0].Endpoint)
	assert.Equal(t, apiInstanceID.String(), parsed.Exporters["emeland"].EndpointMapping["https://payments.prod.eu.example.com:443/api/v1/health"])
}

func TestRenderCollectorConfig_EmptyTargets(t *testing.T) {
	t.Parallel()

	raw, err := RenderCollectorConfig(nil, CollectorConfigOptions{})
	require.NoError(t, err)

	var parsed collectorConfig
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	recv := parsed.Receivers["httpcheck"]
	assert.Equal(t, "5m", recv.CollectionInterval)
	assert.True(t, recv.Metrics["httpcheck.tls.cert_remaining"].Enabled)
	assert.Empty(t, recv.Targets)
	assert.Empty(t, parsed.Exporters["emeland"].EndpointMapping)
}

func TestRenderCollectorConfig_SortsByURL(t *testing.T) {
	t.Parallel()

	raw, err := RenderCollectorConfig([]ProbeTarget{
		{ApiInstanceID: uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001"), URL: "https://z.example.com:443/"},
		{ApiInstanceID: uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000002"), URL: "https://a.example.com:443/"},
	}, CollectorConfigOptions{CollectionInterval: time.Minute})
	require.NoError(t, err)

	var parsed collectorConfig
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	require.Len(t, parsed.Receivers["httpcheck"].Targets, 2)
	assert.Equal(t, "https://a.example.com:443/", parsed.Receivers["httpcheck"].Targets[0].Endpoint)
	assert.Equal(t, "https://z.example.com:443/", parsed.Receivers["httpcheck"].Targets[1].Endpoint)
}

func TestCollectTargets_SkipAndDedupe(t *testing.T) {
	t.Parallel()

	instanceWithHost := uuid.MustParse("88888888-0000-4000-8000-000000000001")
	instanceNoHost := uuid.MustParse("88888888-0000-4000-8000-000000000002")
	instanceDupHost := uuid.MustParse("88888888-0000-4000-8000-000000000003")

	ai1 := api.NewApiInstance(instanceWithHost)
	ai1.GetAnnotations().Add(annProtocol, "https")
	ai1.GetAnnotations().Add(annHost, "example.com")
	ai1.GetAnnotations().Add(annPath, "/health")

	ai2 := api.NewApiInstance(instanceNoHost)
	ai2.GetAnnotations().Add(annProtocol, "https")

	ai3 := api.NewApiInstance(instanceDupHost)
	ai3.GetAnnotations().Add(annProtocol, "https")
	ai3.GetAnnotations().Add(annHost, "example.com")
	ai3.GetAnnotations().Add(annPath, "/metrics")

	client := &fakeApiInstanceClient{instances: []api.ApiInstance{ai1, ai2, ai3}}
	targets, err := CollectTargets(context.Background(), client, zap.NewNop().Sugar())
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, "https://example.com:443/health", targets[0].URL)
	assert.Equal(t, instanceWithHost, targets[0].ApiInstanceID)
}
