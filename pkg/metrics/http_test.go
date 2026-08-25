package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.emeland.io/modelsrv/pkg/metrics"
)

func TestHTTPMetrics_RecordsRequestsAndDuration(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewHTTPMetrics(reg)

	r := mux.NewRouter()
	r.Use(m.Middleware)
	r.HandleFunc("/api/systems", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/systems", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify counter was incremented.
	val := getCounterValue(t, reg, "http_requests_total", map[string]string{
		"method": "GET",
		"path":   "/api/systems",
		"status": "200",
	})
	assert.Equal(t, 1.0, val)

	// Verify histogram has one observation.
	count := getHistogramCount(t, reg, "http_request_duration_seconds", map[string]string{
		"method": "GET",
		"path":   "/api/systems",
	})
	assert.Equal(t, uint64(1), count)

	// Verify response size histogram has one observation.
	count = getHistogramCount(t, reg, "http_response_size_bytes", map[string]string{
		"method": "GET",
		"path":   "/api/systems",
	})
	assert.Equal(t, uint64(1), count)
}

func TestHTTPMetrics_NormalizesPathParameters(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewHTTPMetrics(reg)

	r := mux.NewRouter()
	r.Use(m.Middleware)
	r.HandleFunc("/api/systems/{id}", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Call with a specific UUID.
	req := httptest.NewRequest("GET", "/api/systems/550e8400-e29b-41d4-a716-446655440000", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// The label should use the route template, not the actual UUID.
	val := getCounterValue(t, reg, "http_requests_total", map[string]string{
		"method": "GET",
		"path":   "/api/systems/{id}",
		"status": "200",
	})
	assert.Equal(t, 1.0, val)

	// A second call with a different UUID should increment the same series.
	req = httptest.NewRequest("GET", "/api/systems/deadbeef-0000-1111-2222-333344445555", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	val = getCounterValue(t, reg, "http_requests_total", map[string]string{
		"method": "GET",
		"path":   "/api/systems/{id}",
		"status": "200",
	})
	assert.Equal(t, 2.0, val)
}

func TestHTTPMetrics_RecordsNon200Status(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewHTTPMetrics(reg)

	r := mux.NewRouter()
	r.Use(m.Middleware)
	r.HandleFunc("/api/components", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	req := httptest.NewRequest("POST", "/api/components", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	val := getCounterValue(t, reg, "http_requests_total", map[string]string{
		"method": "POST",
		"path":   "/api/components",
		"status": "404",
	})
	assert.Equal(t, 1.0, val)
}

func TestHTTPMetrics_ExcludesMetricsPath(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewHTTPMetrics(reg)

	r := mux.NewRouter()
	r.Use(m.Middleware)
	r.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assertNoMetricsForPath(t, reg, "/metrics")
}

func TestHTTPMetrics_ExcludesSwaggerPath(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewHTTPMetrics(reg)

	r := mux.NewRouter()
	r.Use(m.Middleware)
	r.PathPrefix("/swagger").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/swagger/index.html", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assertNoMetricsForPath(t, reg, "/swagger/index.html")
}

func TestHTTPMetrics_InFlightGauge(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewHTTPMetrics(reg)

	// Verify in-flight is 1 during request handling.
	var inFlightDuringRequest float64
	r := mux.NewRouter()
	r.Use(m.Middleware)
	r.HandleFunc("/api/slow", func(w http.ResponseWriter, req *http.Request) {
		inFlightDuringRequest = getGaugeValueByName(t, reg, "http_requests_in_flight")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/slow", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, 1.0, inFlightDuringRequest)

	// After request completes, in-flight should be 0.
	val := getGaugeValueByName(t, reg, "http_requests_in_flight")
	assert.Equal(t, 0.0, val)
}

func TestHTTPMetrics_ResponseSizeTracked(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewHTTPMetrics(reg)

	r := mux.NewRouter()
	r.Use(m.Middleware)
	r.HandleFunc("/api/data", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("hello world")) // 11 bytes
	})

	req := httptest.NewRequest("GET", "/api/data", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// Check the histogram sum equals 11 bytes.
	sum := getHistogramSum(t, reg, "http_response_size_bytes", map[string]string{
		"method": "GET",
		"path":   "/api/data",
	})
	assert.Equal(t, 11.0, sum)
}

func TestHTTPMetrics_DefaultStatusOnImplicitOK(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewHTTPMetrics(reg)

	r := mux.NewRouter()
	r.Use(m.Middleware)
	r.HandleFunc("/api/health", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest("GET", "/api/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	val := getCounterValue(t, reg, "http_requests_total", map[string]string{
		"method": "GET",
		"path":   "/api/health",
		"status": "200",
	})
	assert.Equal(t, 1.0, val)
}

func TestHTTPMetrics_UnmatchedPathUseSentinel(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewHTTPMetrics(reg)

	r := mux.NewRouter()
	r.Use(m.Middleware)
	r.HandleFunc("/api/known", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Hit a path that doesn't match any registered route.
	req := httptest.NewRequest("GET", "/wp-admin/login.php", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// Mux returns 405 or falls through; regardless, the path label should
	// be the sentinel value, not the raw probe path.
	mfs, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() == "http_requests_total" {
			for _, metric := range mf.GetMetric() {
				for _, lp := range metric.GetLabel() {
					if lp.GetName() == "path" {
						assert.NotEqual(t, "/wp-admin/login.php", lp.GetValue(),
							"raw probe path should not appear as label")
					}
				}
			}
		}
	}
}

// --- helpers ---

func assertNoMetricsForPath(t *testing.T, reg *prometheus.Registry, path string) {
	t.Helper()
	mfs, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() == "http_requests_total" {
			for _, metric := range mf.GetMetric() {
				for _, lp := range metric.GetLabel() {
					if lp.GetName() == "path" && lp.GetValue() == path {
						t.Fatalf("path %q should not be instrumented", path)
					}
				}
			}
		}
	}
}

func getCounterValue(t *testing.T, reg *prometheus.Registry, name string, labels map[string]string) float64 {
	t.Helper()
	mfs, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			if matchLabels(m, labels) {
				return m.GetCounter().GetValue()
			}
		}
	}
	t.Fatalf("metric %s with labels %v not found", name, labels)
	return 0
}

func getHistogramCount(t *testing.T, reg *prometheus.Registry, name string, labels map[string]string) uint64 {
	t.Helper()
	mfs, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			if matchLabels(m, labels) {
				return m.GetHistogram().GetSampleCount()
			}
		}
	}
	t.Fatalf("metric %s with labels %v not found", name, labels)
	return 0
}

func getHistogramSum(t *testing.T, reg *prometheus.Registry, name string, labels map[string]string) float64 {
	t.Helper()
	mfs, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			if matchLabels(m, labels) {
				return m.GetHistogram().GetSampleSum()
			}
		}
	}
	t.Fatalf("metric %s with labels %v not found", name, labels)
	return 0
}

func getGaugeValueByName(t *testing.T, reg *prometheus.Registry, name string) float64 {
	t.Helper()
	mfs, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() == name {
			metrics := mf.GetMetric()
			if len(metrics) > 0 {
				return metrics[0].GetGauge().GetValue()
			}
		}
	}
	t.Fatalf("metric %s not found", name)
	return 0
}

func matchLabels(m interface{ GetLabel() []*dto.LabelPair }, want map[string]string) bool {
	labels := m.GetLabel()
	if len(labels) < len(want) {
		return false
	}
	for k, v := range want {
		found := false
		for _, lp := range labels {
			if lp.GetName() == k && lp.GetValue() == v {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
