/*
Copyright © 2025 Lutz Behnke <lutz.behnke@gmx.de>
*/
package endpoint

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/metrics"
	"go.emeland.io/modelsrv/pkg/model"
	"go.uber.org/zap"
)

var (
	webServer      *http.Server
	metricsServer  *http.Server
	metricsHandler http.Handler
	setupLog       zap.SugaredLogger
	metricsReg     *prometheus.Registry
)

// StarWebListener starts the web endpoint serving the Swagger-UI and API
//
// addr is the address and port to bind to, e.g. "localhost:24000"
func StarWebListener(backend model.Model, eventMgr events.EventManager, addr string) error {
	baseUrl := fmt.Sprintf("http://%s/api", addr)
	server := oapi.NewApiServer(backend, eventMgr, baseUrl)
	strict := oapi.NewApiHandler(server)
	setupLog = *zap.NewExample().Sugar()

	metricsReg = prometheus.NewRegistry()
	metricsReg.MustRegister(collectors.NewGoCollector())
	metricsReg.MustRegister(metrics.NewCollector(backend))

	r := mux.NewRouter()
	metricsHandler = promhttp.HandlerFor(metricsReg, promhttp.HandlerOpts{})
	r.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		metricsHandler.ServeHTTP(w, req)
	}))

	// TODO: turn staticPath in configurable value, especially for non-container setups
	spa := spaHandler{staticPath: "/", indexPath: "/swagger/index.html"}
	r.PathPrefix("/swagger").Handler(spa)

	// get an `http.Handler` that we can use
	h := oapi.HandlerFromMuxWithBaseURL(strict, r, "/api")

	setupLog.Info("Starting Web-Endpoint: ", "address: ", addr)

	webServer = &http.Server{
		Handler: h,
		Addr:    addr,
	}

	go runListener(webServer)

	return nil
}

func StopWebListener() {
	if metricsServer != nil {
		if err := metricsServer.Shutdown(context.Background()); err != nil {
			setupLog.Error("Error shutting down metrics server: ", err)
		}
	}
	if webServer == nil {
		return
	}
	if err := webServer.Shutdown(context.Background()); err != nil {
		setupLog.Error("Error shutting down web server: ", err)
	}
}

// StartMetricsListener starts a dedicated HTTP server for /metrics on the given address.
// When called, the main port's /metrics is replaced with a redirect to the dedicated endpoint.
func StartMetricsListener(addr string) error {
	if metricsReg == nil {
		return fmt.Errorf("metrics registry not initialized; call StarWebListener first")
	}
	metricsURL := fmt.Sprintf("http://%s/metrics", addr)
	metricsHandler = http.RedirectHandler(metricsURL, http.StatusTemporaryRedirect)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(metricsReg, promhttp.HandlerOpts{}))
	metricsServer = &http.Server{Handler: mux, Addr: addr}
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			setupLog.Error("metrics server: ", err)
		}
	}()
	setupLog.Info("Metrics endpoint: ", metricsURL)
	return nil
}

func runListener(server *http.Server) {

	err := server.ListenAndServe()
	if err != nil {
		setupLog.Error(err, ". Ended server with error")
	} else {
		setupLog.Info("Ended server.")
	}
}

// spaHandler implements the http.Handler interface, so we can use it
// to respond to HTTP requests. The path to the static directory and
// path to the index file within that static directory are used to
// serve the SPA in the given static directory.
type spaHandler struct {
	staticPath string
	indexPath  string
}

// ServeHTTP inspects the URL path to locate a file within the static dir
// on the SPA handler. If a file is found, it will be served. If not, the
// file located at the index path on the SPA handler will be served. This
// is suitable behavior for serving an SPA (single page application).
func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Join internally call path.Clean to prevent directory traversal
	path := filepath.Join(h.staticPath, r.URL.Path)

	setupLog.Info("SPA Handler called to service file", "path", path, "staticPath", h.staticPath, "URL path", r.URL.Path)

	// check whether a file exists or is a directory at the given path
	fi, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			setupLog.Error("SPA Handler stat error", "path", path, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	// file does not exist or path is a directory: serve index file
	// fi is only non-nil when err == nil, so IsDir() is safe here.
	if err != nil || fi.IsDir() {
		path = filepath.Join(h.staticPath, h.indexPath)
		setupLog.Info("SPA Handler will serve index file", "path", path)
		http.ServeFile(w, r, path)
		return
	}

	// otherwise, use http.FileServer to serve the static file
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}
