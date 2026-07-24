package endpointprobe

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/model/api"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func TestRenderHTTPCheckConfig_KnownAnnotations(t *testing.T) {
	t.Parallel()

	ai := api.NewApiInstance(uuid.MustParse("88888888-0000-4000-8000-000000000001"))
	ai.GetAnnotations().Add(annProtocol, "https")
	ai.GetAnnotations().Add(annHost, "payments.prod.eu.example.com")
	ai.GetAnnotations().Add(annPort, "443")
	ai.GetAnnotations().Add(annPath, "/api/v1/health")

	target, ok, err := TargetFromApiInstance(ai)
	require.NoError(t, err)
	require.True(t, ok)

	raw, err := RenderHTTPCheckConfig([]ProbeTarget{target}, 5*time.Minute)
	require.NoError(t, err)

	wantFragment := `
receivers:
  http_check:
    collection_interval: 5m
    metrics:
      httpcheck.tls.cert_remaining:
        enabled: true
    targets:
      - method: GET
        endpoint: https://payments.prod.eu.example.com:443/api/v1/health
`
	assert.Equal(t, strings.TrimSpace(wantFragment)+"\n", string(raw))

	var parsed httpCheckConfig
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	recv, ok := parsed.Receivers["http_check"]
	require.True(t, ok)
	assert.Equal(t, "5m", recv.CollectionInterval)
	assert.True(t, recv.Metrics["httpcheck.tls.cert_remaining"].Enabled)
	require.Len(t, recv.Targets, 1)
	assert.Equal(t, "GET", recv.Targets[0].Method)
	assert.Equal(t, "https://payments.prod.eu.example.com:443/api/v1/health", recv.Targets[0].Endpoint)
}

func TestRenderHTTPCheckConfig_EmptyTargets(t *testing.T) {
	t.Parallel()

	raw, err := RenderHTTPCheckConfig(nil, 0)
	require.NoError(t, err)

	var parsed httpCheckConfig
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	recv := parsed.Receivers["http_check"]
	assert.Equal(t, "5m", recv.CollectionInterval)
	assert.True(t, recv.Metrics["httpcheck.tls.cert_remaining"].Enabled)
	assert.Empty(t, recv.Targets)
	assert.Contains(t, string(raw), "targets:")
}

func TestRenderHTTPCheckConfig_SortsByURL(t *testing.T) {
	t.Parallel()

	raw, err := RenderHTTPCheckConfig([]ProbeTarget{
		{URL: "https://z.example.com:443/"},
		{URL: "https://a.example.com:443/"},
	}, time.Minute)
	require.NoError(t, err)

	var parsed httpCheckConfig
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	require.Len(t, parsed.Receivers["http_check"].Targets, 2)
	assert.Equal(t, "https://a.example.com:443/", parsed.Receivers["http_check"].Targets[0].Endpoint)
	assert.Equal(t, "https://z.example.com:443/", parsed.Receivers["http_check"].Targets[1].Endpoint)
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

	raw, err := RenderHTTPCheckConfig(targets, 5*time.Minute)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "endpoint: https://example.com:443/health")
	assert.NotContains(t, string(raw), "/metrics")
	assert.Equal(t, 1, strings.Count(string(raw), "endpoint:"))
}
