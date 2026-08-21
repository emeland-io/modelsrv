/*
Copyright © 2025 Lutz Behnke <lutz.behnke@gmx.de>
*/
package endpoint

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

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
	strict := oapi.NewApiHandler(server, oapi.ApiHandlerOptions{TrustAuthHeaders: opts.TrustAuthHeaders})

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(metrics.NewCollector(backend))

	r := mux.NewRouter()
	r.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	spa := spaHandler{staticPath: "/", indexPath: "/swagger/index.html", log: log}
	r.PathPrefix("/swagger").Handler(spa)
	r.HandleFunc("/api/events/history", server.HandleGetEventsHistory).Methods("GET")

	return oapi.HandlerFromMuxWithBaseURL(strict, r, "/api")
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
	strict := oapi.NewApiHandler(server, oapi.ApiHandlerOptions{TrustAuthHeaders: opts.TrustAuthHeaders})

	metricsReg = prometheus.NewRegistry()
	metricsReg.MustRegister(collectors.NewGoCollector())
	metricsReg.MustRegister(metrics.NewCollector(backend))

	r := mux.NewRouter()
	// Indirection: metricsHandler can be swapped to a redirect by StartMetricsListener.
	metricsHandler = promhttp.HandlerFor(metricsReg, promhttp.HandlerOpts{})
	r.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		metricsHandler.ServeHTTP(w, req)
	}))

	spa := spaHandler{staticPath: "/", indexPath: "/swagger/index.html", log: log}
	r.PathPrefix("/swagger").Handler(spa)
	r.HandleFunc("/api/events/history", server.HandleGetEventsHistory).Methods("GET")

	h := oapi.HandlerFromMuxWithBaseURL(strict, r, "/api")

	log.Infow("starting web endpoint", "address", ln.Addr().String())

	webServer = &http.Server{
		Handler: h,
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

// spaHandler implements the http.Handler interface, so we can use it
// to respond to HTTP requests. The path to the static directory and
// path to the index file within that static directory are used to
// serve the SPA in the given static directory.
type spaHandler struct {
	staticPath string
	indexPath  string
	log        *zap.SugaredLogger
}

// ServeHTTP inspects the URL path to locate a file within the static dir
// on the SPA handler. If a file is found, it will be served. If not, the
// file located at the index path on the SPA handler will be served. This
// is suitable behavior for serving an SPA (single page application).
func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Join internally call path.Clean to prevent directory traversal
	path := filepath.Join(h.staticPath, r.URL.Path)

	h.log.Debugw("serving static file", "path", path, "url", r.URL.Path)

	// check whether a file exists or is a directory at the given path
	fi, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			h.log.Errorw("static file stat error", "path", path, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	// file does not exist or path is a directory: serve index file
	// fi is only non-nil when err == nil, so IsDir() is safe here.
	if err != nil || fi.IsDir() {
		path = filepath.Join(h.staticPath, h.indexPath)
		http.ServeFile(w, r, path)
		return
	}

	// otherwise, use http.FileServer to serve the static file
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}
