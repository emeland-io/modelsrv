package endpointprobe

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func TestNewConfigWriter_Validation(t *testing.T) {
	t.Parallel()

	_, err := NewConfigWriter(ConfigWriterConfig{Debounce: time.Second})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")

	_, err = NewConfigWriter(ConfigWriterConfig{Path: "/tmp/out.yaml", Debounce: 0})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "debounce must be positive")
}

func TestConfigWriter_UpsertRemoveFlush(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "collector.yaml")

	w, err := NewConfigWriter(ConfigWriterConfig{
		Path:     path,
		Debounce: time.Second,
		Logger:   zap.NewNop().Sugar(),
		Opts: CollectorConfigOptions{
			CollectionInterval: 5 * time.Minute,
			ListenAddr:         "0.0.0.0:24200",
			Subscribers:        []string{"http://modelsrv:8080"},
		},
	})
	require.NoError(t, err)

	id := uuid.MustParse("88888888-0000-4000-8000-000000000001")
	w.Upsert(ProbeTarget{
		ApiInstanceID: id,
		URL:           "https://payments.example.com:443/health",
		DedupeKey:     "payments.example.com:443",
	})
	require.NoError(t, w.Flush())

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NoFileExists(t, path+".tmp")

	var parsed collectorConfig
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	recv := parsed.Receivers["http_check"]
	require.Len(t, recv.Targets, 1)
	assert.Equal(t, "https://payments.example.com:443/health", recv.Targets[0].Endpoint)
	assert.Equal(t, id.String(), parsed.Exporters["emeland"].EndpointMapping["https://payments.example.com:443/health"])

	w.Remove(id)
	require.NoError(t, w.Flush())

	raw, err = os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	assert.Empty(t, parsed.Receivers["http_check"].Targets)
	assert.Empty(t, parsed.Exporters["emeland"].EndpointMapping)
}

func TestConfigWriter_SkipUnchangedContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "collector.yaml")

	w, err := NewConfigWriter(ConfigWriterConfig{
		Path:     path,
		Debounce: time.Second,
		Logger:   zap.NewNop().Sugar(),
	})
	require.NoError(t, err)

	id := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	w.Upsert(ProbeTarget{
		ApiInstanceID: id,
		URL:           "https://a.example.com:443/",
		DedupeKey:     "a.example.com:443",
	})
	require.NoError(t, w.Flush())

	info1, err := os.Stat(path)
	require.NoError(t, err)

	// Force another flush of the same content.
	w.mu.Lock()
	w.dirty = true
	w.mu.Unlock()
	require.NoError(t, w.Flush())

	info2, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, info1.ModTime(), info2.ModTime())
}

func TestConfigWriter_DedupeDeterministic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "collector.yaml")

	w, err := NewConfigWriter(ConfigWriterConfig{
		Path:     path,
		Debounce: time.Second,
		Logger:   zap.NewNop().Sugar(),
	})
	require.NoError(t, err)

	low := uuid.MustParse("11111111-0000-0000-0000-000000000001")
	high := uuid.MustParse("99999999-0000-0000-0000-000000000002")

	// Insert higher ID first; lowest ApiInstanceID must still win.
	w.Upsert(ProbeTarget{
		ApiInstanceID: high,
		URL:           "https://shared.example.com:443/high",
		DedupeKey:     "shared.example.com:443",
	})
	w.Upsert(ProbeTarget{
		ApiInstanceID: low,
		URL:           "https://shared.example.com:443/low",
		DedupeKey:     "shared.example.com:443",
	})
	require.NoError(t, w.Flush())

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var parsed collectorConfig
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	require.Len(t, parsed.Receivers["http_check"].Targets, 1)
	assert.Equal(t, "https://shared.example.com:443/low", parsed.Receivers["http_check"].Targets[0].Endpoint)
	assert.Equal(t, low.String(), parsed.Exporters["emeland"].EndpointMapping["https://shared.example.com:443/low"])
}

func TestDedupeTargets(t *testing.T) {
	t.Parallel()

	low := uuid.MustParse("11111111-0000-0000-0000-000000000001")
	high := uuid.MustParse("99999999-0000-0000-0000-000000000002")
	other := uuid.MustParse("22222222-0000-0000-0000-000000000003")

	out := dedupeTargets([]ProbeTarget{
		{ApiInstanceID: high, URL: "https://a.example.com:443/high", DedupeKey: "a.example.com:443"},
		{ApiInstanceID: other, URL: "https://b.example.com:443/", DedupeKey: "b.example.com:443"},
		{ApiInstanceID: low, URL: "https://a.example.com:443/low", DedupeKey: "a.example.com:443"},
	})
	require.Len(t, out, 2)
	assert.Equal(t, low, out[0].ApiInstanceID)
	assert.Equal(t, other, out[1].ApiInstanceID)
}

func TestConfigWriter_EmptyFlushCreatesFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "collector.yaml")

	w, err := NewConfigWriter(ConfigWriterConfig{
		Path:     path,
		Debounce: time.Second,
		Logger:   zap.NewNop().Sugar(),
	})
	require.NoError(t, err)
	require.NoError(t, w.Flush())

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NoFileExists(t, path+".tmp")

	var parsed collectorConfig
	require.NoError(t, yaml.Unmarshal(raw, &parsed))
	assert.Empty(t, parsed.Receivers["http_check"].Targets)
}
