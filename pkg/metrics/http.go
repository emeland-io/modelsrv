package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics holds the Prometheus metrics for HTTP request instrumentation.
type HTTPMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight prometheus.Gauge
	respSize *prometheus.HistogramVec
}

// NewHTTPMetrics creates and registers HTTP request metrics on the given registry.
func NewHTTPMetrics(reg prometheus.Registerer) *HTTPMetrics {
	m := &HTTPMetrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		}, []string{"method", "path", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being handled.",
		}),
		respSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "Size of HTTP response bodies in bytes.",
			Buckets: prometheus.ExponentialBuckets(100, 10, 7), // 100B .. 100MB
		}, []string{"method", "path"}),
	}
	reg.MustRegister(m.requests, m.duration, m.inFlight, m.respSize)
	return m
}

// Middleware returns a mux.MiddlewareFunc that records request count, duration,
// in-flight concurrency, and response size. Requests to infrastructure paths
// (/metrics, /swagger) are excluded to avoid noise and self-referencing loops.
//
// This should be registered via mux.Router.Use() so that route matching has
// already occurred and path normalization can use the route template.
func (m *HTTPMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawPath := r.URL.Path
		if strings.HasPrefix(rawPath, "/metrics") || strings.HasPrefix(rawPath, "/swagger") {
			next.ServeHTTP(w, r)
			return
		}

		m.inFlight.Inc()
		defer m.inFlight.Dec()

		start := time.Now()
		rc := &responseCapture{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rc, r)

		path := normalizePath(r)
		method := r.Method
		m.requests.WithLabelValues(method, path, strconv.Itoa(rc.status)).Inc()
		m.duration.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
		m.respSize.WithLabelValues(method, path).Observe(float64(rc.written))
	})
}

// normalizePath extracts the matched route template from gorilla/mux so that
// path parameters (UUIDs, IDs) are replaced with their placeholder names,
// keeping metric cardinality bounded. Returns a fixed sentinel for unmatched
// paths to prevent cardinality explosion from bot probes or typos.
func normalizePath(r *http.Request) string {
	if route := mux.CurrentRoute(r); route != nil {
		if tmpl, err := route.GetPathTemplate(); err == nil {
			return tmpl
		}
	}
	return "<unmatched>"
}

// responseCapture wraps http.ResponseWriter to capture the status code and
// number of bytes written.
type responseCapture struct {
	http.ResponseWriter
	status  int
	written int64
}

func (w *responseCapture) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseCapture) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.written += int64(n)
	return n, err
}
