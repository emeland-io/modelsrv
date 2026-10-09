/*
Copyright © 2025 Lutz Behnke <lutz.behnke@gmx.de>
*/
package endpoint

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/authz"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/metrics"
	"go.emeland.io/modelsrv/pkg/model"
	"go.uber.org/zap"
)

// WebListenerOptions configures the web API listener.
type WebListenerOptions struct {
	TrustAuthHeaders bool
	AuthzConfig      authz.Config
	// Logger is used for endpoint lifecycle messages and HTTP request logging.
	// When nil, a no-op logger is used (no output).
	Logger *zap.SugaredLogger
	// ExtraHandlers are mounted on the mux before the OpenAPI handler.
	// Use for derived documents and other non-landscape routes.
	ExtraHandlers []ExtraHandler
}

// ExtraHandler mounts an additional HTTP route on the web listener.
type ExtraHandler struct {
	Path    string
	Handler http.Handler
	// Methods limits the handler to these HTTP methods. Empty means all methods.
	Methods []string
}

var (
	webServer      *http.Server
	webListener    net.Listener
	metricsServer  *http.Server
	metricsHandler http.Handler
	metricsReg     *prometheus.Registry
	endpointLog    *zap.SugaredLogger // set by StartWebListener; used by Stop/Metrics helpers
)

func ensureLogger(log *zap.SugaredLogger) *zap.SugaredLogger {
	if log != nil {
		return log
	}
	return zap.NewNop().Sugar()
}

func mountExtraHandlers(r *mux.Router, handlers []ExtraHandler) {
	for _, eh := range handlers {
		if eh.Path == "" || eh.Handler == nil {
			continue
		}
		if len(eh.Methods) == 0 {
			r.Handle(eh.Path, eh.Handler)
			continue
		}
		r.Handle(eh.Path, eh.Handler).Methods(eh.Methods...)
	}
}

// NewHandler builds the modelsrv HTTP handler (API + swagger + metrics) without
// starting a listener. The caller is responsible for serving it on their own
// http.Server. Use this when embedding modelsrv behind additional middleware
// (e.g. an auth layer).
//
// Note: StartMetricsListener is not compatible with NewHandler; it only works
// with StartWebListener which manages its own server lifecycle.
func NewHandler(backend model.Model, eventMgr events.EventManager, baseURL string, opts WebListenerOptions) http.Handler {
	log := ensureLogger(opts.Logger)

	var authzEval *authz.Evaluator
	if opts.TrustAuthHeaders {
		authzEval = authz.NewEvaluator(opts.AuthzConfig)
	}
	server := oapi.NewApiServer(backend, eventMgr, baseURL, authzEval)
	server.Logger = log
	strict := oapi.NewApiHandler(server, oapi.ApiHandlerOptions{TrustAuthHeaders: opts.TrustAuthHeaders})

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(metrics.NewCollector(backend))
	httpMetrics := metrics.NewHTTPMetrics(reg)

	r := mux.NewRouter()
	r.Use(httpMetrics.Middleware)
	r.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	registerSwaggerDocs(r, log)
	r.HandleFunc("/api/events/history", server.HandleGetEventsHistory).Methods("GET")
	mountExtraHandlers(r, opts.ExtraHandlers)

	return requestLoggingMiddleware(log)(oapi.HandlerFromMuxWithBaseURL(strict, r, "/api"))
}

// StartWebListener starts the web endpoint serving the Swagger-UI and API
// on its own goroutine. Use StopWebListener to shut it down.
//
// addr is the address and port to bind to, e.g. "localhost:24000"
func StartWebListener(backend model.Model, eventMgr events.EventManager, addr string, opts WebListenerOptions) error {
	log := ensureLogger(opts.Logger)
	endpointLog = log

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	webListener = ln

	baseURL := fmt.Sprintf("http://%s/api", ln.Addr().String())
	var authzEval *authz.Evaluator
	if opts.TrustAuthHeaders {
		authzEval = authz.NewEvaluator(opts.AuthzConfig)
	}
	server := oapi.NewApiServer(backend, eventMgr, baseURL, authzEval)
	server.Logger = log
	strict := oapi.NewApiHandler(server, oapi.ApiHandlerOptions{TrustAuthHeaders: opts.TrustAuthHeaders})

	metricsReg = prometheus.NewRegistry()
	metricsReg.MustRegister(collectors.NewGoCollector())
	metricsReg.MustRegister(metrics.NewCollector(backend))
	httpMetrics := metrics.NewHTTPMetrics(metricsReg)

	r := mux.NewRouter()
	r.Use(httpMetrics.Middleware)
	// Indirection: metricsHandler can be swapped to a redirect by StartMetricsListener.
	metricsHandler = promhttp.HandlerFor(metricsReg, promhttp.HandlerOpts{})
	r.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		metricsHandler.ServeHTTP(w, req)
	}))

	registerSwaggerDocs(r, log)
	r.HandleFunc("/api/events/history", server.HandleGetEventsHistory).Methods("GET")
	mountExtraHandlers(r, opts.ExtraHandlers)

	h := oapi.HandlerFromMuxWithBaseURL(strict, r, "/api")

	log.Infow("starting web endpoint", "address", ln.Addr().String())

	webServer = &http.Server{
		Handler: requestLoggingMiddleware(log)(h),
	}

	srv := webServer
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Errorw("web server ended with error", "error", err)
		}
	}()

	return nil
}

func StopWebListener() {
	log := ensureLogger(endpointLog)
	if metricsServer != nil {
		if err := metricsServer.Shutdown(context.Background()); err != nil {
			log.Errorw("error shutting down metrics server", "error", err)
		}
	}
	if webServer == nil {
		return
	}
	if err := webServer.Shutdown(context.Background()); err != nil {
		log.Errorw("error shutting down web server", "error", err)
	}
	webServer = nil
	webListener = nil
}

// MetricsRegistry returns the Prometheus registry used by the web listener.
// Callers may register additional collectors after StartWebListener.
// Returns nil if the web listener has not been started.
func MetricsRegistry() *prometheus.Registry {
	return metricsReg
}

// StartMetricsListener starts a dedicated HTTP server for /metrics on the given address.
// When called, the main port's /metrics is replaced with a redirect to the dedicated endpoint.
func StartMetricsListener(addr string) error {
	if metricsReg == nil {
		return fmt.Errorf("metrics registry not initialized; call StartWebListener first")
	}
	log := ensureLogger(endpointLog)
	metricsURL := fmt.Sprintf("http://%s/metrics", addr)
	metricsHandler = http.RedirectHandler(metricsURL, http.StatusTemporaryRedirect)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(metricsReg, promhttp.HandlerOpts{}))
	metricsServer = &http.Server{Handler: mux, Addr: addr}
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorw("metrics server error", "error", err)
		}
	}()
	log.Infow("metrics endpoint started", "url", metricsURL)
	return nil
}

// WebListenerAddr returns the address the web server is listening on.
// Useful when started with ":0" to discover the actual port.
func WebListenerAddr() net.Addr {
	if webListener == nil {
		return nil
	}
	return webListener.Addr()
}

// requestLoggingMiddleware logs each HTTP request with method, path, status
// code, and duration. API requests log at INFO (5xx at WARN). Infrastructure
// paths (/metrics, /swagger) log at DEBUG to avoid noise.
func requestLoggingMiddleware(log *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			path := r.URL.Path
			fields := []any{
				"method", r.Method,
				"path", path,
				"status", sw.status,
				"duration", time.Since(start).String(),
			}
			switch {
			case strings.HasPrefix(path, "/metrics") || strings.HasPrefix(path, "/swagger"):
				log.Debugw("http request", fields...)
			case sw.status >= 500:
				log.Warnw("http request", fields...)
			default:
				log.Infow("http request", fields...)
			}
		})
	}
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
