package endpointprobe

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/backend"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/api"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func newTestConfigWriter(t *testing.T) (*ConfigWriter, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "collector.yaml")
	w, err := NewConfigWriter(ConfigWriterConfig{
		Path:     path,
		Debounce: time.Hour, // large debounce; tests call Flush explicitly
		Logger:   zap.NewNop().Sugar(),
		Opts: CollectorConfigOptions{
			CollectionInterval: 5 * time.Minute,
			ListenAddr:         "0.0.0.0:24200",
		},
	})
	require.NoError(t, err)
	return w, path
}

func annotatedInstance(id uuid.UUID, host, protocol string) api.ApiInstance {
	ai := api.NewApiInstance(id)
	ai.SetDisplayName("test")
	if protocol != "" {
		ai.GetAnnotations().Add(annProtocol, protocol)
	}
	if host != "" {
		ai.GetAnnotations().Add(annHost, host)
	}
	return ai
}

func TestConfigSyncFilter_PassthroughNonApiInstance(t *testing.T) {
	t.Parallel()

	w, _ := newTestConfigWriter(t)
	filter := NewConfigSyncFilter(w)

	ev := events.Event{
		ResourceType: events.SystemResource,
		Operation:    events.CreateOperation,
		ResourceId:   uuid.New(),
	}
	out := filter.Fn(nil, ev)
	require.Len(t, out, 1)
	assert.Equal(t, ev, out[0])

	require.NoError(t, w.Flush())
	w.mu.Lock()
	assert.Empty(t, w.targets)
	w.mu.Unlock()
}

func TestConfigSyncFilter_CreateUpdateDelete(t *testing.T) {
	t.Parallel()

	w, path := newTestConfigWriter(t)
	filter := NewConfigSyncFilter(w)
	id := uuid.MustParse("88888888-0000-4000-8000-000000000001")
	ai := annotatedInstance(id, "payments.example.com", "https")

	createEv := events.Event{
		ResourceType: events.APIInstanceResource,
		Operation:    events.CreateOperation,
		ResourceId:   id,
		Objects:      []any{ai},
	}
	out := filter.Fn(nil, createEv)
	require.Len(t, out, 1)
	assert.Equal(t, createEv, out[0])
	require.NoError(t, w.Flush())

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var parsed collectorConfig
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	require.Len(t, parsed.Receivers["httpcheck"].Targets, 1)
	assert.Equal(t, "https://payments.example.com:443/", parsed.Receivers["httpcheck"].Targets[0].Endpoint)

	// Update host
	ai.GetAnnotations().Add(annHost, "orders.example.com")
	updateEv := events.Event{
		ResourceType: events.APIInstanceResource,
		Operation:    events.UpdateOperation,
		ResourceId:   id,
		Objects:      []any{ai},
	}
	out = filter.Fn(nil, updateEv)
	require.Len(t, out, 1)
	assert.Equal(t, updateEv, out[0])
	require.NoError(t, w.Flush())

	raw, err = os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	require.Len(t, parsed.Receivers["httpcheck"].Targets, 1)
	assert.Equal(t, "https://orders.example.com:443/", parsed.Receivers["httpcheck"].Targets[0].Endpoint)

	deleteEv := events.Event{
		ResourceType: events.APIInstanceResource,
		Operation:    events.DeleteOperation,
		ResourceId:   id,
	}
	out = filter.Fn(nil, deleteEv)
	require.Len(t, out, 1)
	assert.Equal(t, deleteEv, out[0])
	require.NoError(t, w.Flush())

	raw, err = os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	assert.Empty(t, parsed.Receivers["httpcheck"].Targets)
}

func TestConfigSyncFilter_AnnotationRemoval(t *testing.T) {
	t.Parallel()

	w, path := newTestConfigWriter(t)
	filter := NewConfigSyncFilter(w)
	id := uuid.MustParse("88888888-0000-4000-8000-000000000002")
	ai := annotatedInstance(id, "payments.example.com", "https")

	filter.Fn(nil, events.Event{
		ResourceType: events.APIInstanceResource,
		Operation:    events.CreateOperation,
		ResourceId:   id,
		Objects:      []any{ai},
	})
	require.NoError(t, w.Flush())

	// Remove host annotation — target must be dropped.
	ai.GetAnnotations().Delete(annHost)
	filter.Fn(nil, events.Event{
		ResourceType: events.APIInstanceResource,
		Operation:    events.UpdateOperation,
		ResourceId:   id,
		Objects:      []any{ai},
	})
	require.NoError(t, w.Flush())

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var parsed collectorConfig
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	assert.Empty(t, parsed.Receivers["httpcheck"].Targets)
}

func TestConfigSyncFilter_InvalidProtocolRemovesTarget(t *testing.T) {
	t.Parallel()

	w, path := newTestConfigWriter(t)
	filter := NewConfigSyncFilter(w)
	id := uuid.MustParse("88888888-0000-4000-8000-000000000003")
	ai := annotatedInstance(id, "payments.example.com", "https")

	filter.Fn(nil, events.Event{
		ResourceType: events.APIInstanceResource,
		Operation:    events.CreateOperation,
		ResourceId:   id,
		Objects:      []any{ai},
	})
	require.NoError(t, w.Flush())

	ai.GetAnnotations().Add(annProtocol, "ftp")
	filter.Fn(nil, events.Event{
		ResourceType: events.APIInstanceResource,
		Operation:    events.UpdateOperation,
		ResourceId:   id,
		Objects:      []any{ai},
	})
	require.NoError(t, w.Flush())

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var parsed collectorConfig
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	assert.Empty(t, parsed.Receivers["httpcheck"].Targets)
}

func TestConfigSyncFilter_BackendIntegration(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "collector.yaml")
	w, err := NewConfigWriter(ConfigWriterConfig{
		Path:     path,
		Debounce: time.Hour,
		Logger:   zap.NewNop().Sugar(),
		Opts: CollectorConfigOptions{
			CollectionInterval: 5 * time.Minute,
			ListenAddr:         "0.0.0.0:24200",
			Subscribers:        []string{"http://modelsrv:8080"},
		},
	})
	require.NoError(t, err)

	b, err := backend.New()
	require.NoError(t, err)
	filterID := b.GetChain().RegisterFilter(NewConfigSyncFilter(w))
	t.Cleanup(func() { b.GetChain().Unregister(filterID) })

	id := uuid.MustParse("88888888-0000-4000-8000-000000000010")
	ai := annotatedInstance(id, "payments.example.com", "https")
	ai.GetAnnotations().Add(annPath, "/api/v1/health")

	require.NoError(t, b.GetModel().AddApiInstance(ai))
	require.NoError(t, w.Flush())

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var parsed collectorConfig
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	require.Len(t, parsed.Receivers["httpcheck"].Targets, 1)
	assert.Equal(t, "https://payments.example.com:443/api/v1/health", parsed.Receivers["httpcheck"].Targets[0].Endpoint)
	assert.Equal(t, id.String(), parsed.Exporters["emeland"].EndpointMapping["https://payments.example.com:443/api/v1/health"])

	// Annotation update via model
	stored := b.GetModel().GetApiInstanceById(id)
	require.NotNil(t, stored)
	stored.GetAnnotations().Add(annHost, "orders.example.com")
	require.NoError(t, w.Flush())

	raw, err = os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	require.Len(t, parsed.Receivers["httpcheck"].Targets, 1)
	assert.Equal(t, "https://orders.example.com:443/api/v1/health", parsed.Receivers["httpcheck"].Targets[0].Endpoint)

	require.NoError(t, b.GetModel().DeleteApiInstanceById(id))
	require.NoError(t, w.Flush())

	raw, err = os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	assert.Empty(t, parsed.Receivers["httpcheck"].Targets)
}
