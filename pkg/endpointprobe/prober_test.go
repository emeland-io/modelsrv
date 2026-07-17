package endpointprobe

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.uber.org/zap"
)

func TestProber_HTTPS_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	prober := NewProber(5 * time.Second)
	target := ProbeTarget{
		ApiInstanceID: uuid.New(),
		URL:           server.URL,
		DedupeKey:     "127.0.0.1:443",
	}

	result := prober.Probe(t.Context(), target)

	require.True(t, result.Success)
	require.NoError(t, result.Err)
	require.True(t, result.HasCert)
	assert.Greater(t, result.CertRemaining, time.Duration(0))
}

func TestProber_Unreachable_Failure(t *testing.T) {
	t.Parallel()

	prober := NewProber(500 * time.Millisecond)
	target := ProbeTarget{
		ApiInstanceID: uuid.New(),
		URL:           "http://127.0.0.1:1/",
		DedupeKey:     "127.0.0.1:1",
	}

	result := prober.Probe(t.Context(), target)

	assert.False(t, result.Success)
	assert.Error(t, result.Err)
	assert.False(t, result.HasCert)
}

func TestProber_ExpiredCert_NegativeRemaining(t *testing.T) {
	t.Parallel()

	probedAt := time.Now()
	notAfter := probedAt.Add(-24 * time.Hour)
	remaining := notAfter.Sub(probedAt)

	assert.Less(t, remaining.Seconds(), 0.0)
}

func TestMetrics_Record(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	metrics := NewMetrics(reg)

	instanceID := uuid.MustParse("88888888-0000-4000-8000-000000000001")
	target := ProbeTarget{
		ApiInstanceID: instanceID,
		DedupeKey:     "example.com:443",
	}

	metrics.Record(ProbeResult{
		Target:        target,
		Success:       true,
		HasCert:       true,
		CertRemaining: 30 * 24 * time.Hour,
	})

	metrics.Record(ProbeResult{
		Target:  target,
		Success: false,
	})

	families, err := reg.Gather()
	require.NoError(t, err)
	require.NotEmpty(t, families)
}

func TestProberWithClient_CustomTransport(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}
	prober := NewProberWithClient(client)

	target := ProbeTarget{
		ApiInstanceID: uuid.New(),
		URL:           server.URL,
		DedupeKey:     "127.0.0.1:443",
	}

	result := prober.Probe(t.Context(), target)
	require.True(t, result.Success)
	require.True(t, result.HasCert)
}

type fakeApiInstanceClient struct {
	instances []api.ApiInstance
	listErr   error
}

func (f *fakeApiInstanceClient) GetApiInstances() ([]common.InstanceListItem, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	items := make([]common.InstanceListItem, 0, len(f.instances))
	for _, ai := range f.instances {
		items = append(items, common.InstanceListItem{
			Id:   ai.GetInstanceId(),
			Name: ai.GetDisplayName(),
		})
	}
	return items, nil
}

func (f *fakeApiInstanceClient) GetApiInstanceById(id uuid.UUID) (api.ApiInstance, error) {
	for _, ai := range f.instances {
		if ai.GetInstanceId() == id {
			return ai, nil
		}
	}
	return nil, common.ErrApiInstanceNotFound
}

func TestScheduler_Scan_SkipsAndDedupes(t *testing.T) {
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

	logger := zap.NewNop().Sugar()
	sched := &Scheduler{
		Client: &fakeApiInstanceClient{instances: []api.ApiInstance{ai1, ai2, ai3}},
		Logger: logger,
	}

	targets, err := sched.scan(t.Context())
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, instanceWithHost, targets[0].ApiInstanceID)
	assert.Equal(t, "example.com:443", targets[0].DedupeKey)
}

func TestScheduler_Scan_ModelsrvUnreachable(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop().Sugar()
	sched := &Scheduler{
		Client: &fakeApiInstanceClient{listErr: assert.AnError},
		Logger: logger,
	}

	_, err := sched.scan(t.Context())
	require.Error(t, err)
}
