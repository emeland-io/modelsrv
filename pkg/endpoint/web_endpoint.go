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
	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.uber.org/zap"
)

var (
	webServer *http.Server
	setupLog  zap.SugaredLogger
)

// StarWebListener starts the web endpoint serving the Swagger-UI and API
//
// addr is the address and port to bind to, e.g. "localhost:24000"
func StarWebListener(backend model.Model, eventMgr events.EventManager, addr string) error {
	baseUrl := fmt.Sprintf("http://%s/api", addr)
	server := oapi.NewApiServer(backend, eventMgr, baseUrl)
	strict := oapi.NewApiHandler(server)
	setupLog = *zap.NewExample().Sugar()

	r := mux.NewRouter()

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
	webServer.Shutdown(context.Background())
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
	if os.IsNotExist(err) || fi.IsDir() {
		// file does not exist or path is a directory, serve index file
		path = filepath.Join(h.staticPath, h.indexPath)
		setupLog.Info("SPA Handler will serve index file", "path", path)
		http.ServeFile(w, r, path)
		return
	}

	if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// otherwise, use http.FileServer to serve the static file
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}
