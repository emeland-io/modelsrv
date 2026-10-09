/*
Copyright © 2025 Lutz Behnke <lutz.behnke@gmx.de>
*/
package endpoint

import (
	"net/http"

	"github.com/gorilla/mux"
	"go.emeland.io/modelsrv/internal/oapi"
	"go.uber.org/zap"
)

// redocPage is a self-contained API documentation page. It renders the embedded
// OpenAPI spec (served at ./openapi.json, relative to /swagger/) using Redoc
// loaded from a CDN. Unlike the previous filesystem-backed Swagger UI, this
// depends on no on-disk assets, so it works in any deployment (including the
// embedded web-ui-server) without bundling static files.
const redocPage = `<!DOCTYPE html>
<html>
  <head>
    <title>EmELand API</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>body { margin: 0; padding: 0; }</style>
  </head>
  <body>
    <redoc spec-url="openapi.json"></redoc>
    <script src="https://cdn.redocly.com/redoc/latest/bundles/redoc.standalone.js"></script>
  </body>
</html>
`

// registerSwaggerDocs mounts a self-contained API documentation site under
// /swagger/:
//
//	GET /swagger/              -> an HTML docs page (Redoc)
//	GET /swagger/openapi.json  -> the embedded OpenAPI spec as JSON
//
// The spec comes from the generated oapi.GetSpec(), so it always matches the
// built binary and needs no files on disk.
func registerSwaggerDocs(r *mux.Router, log *zap.SugaredLogger) {
	r.HandleFunc("/swagger/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		spec, err := oapi.GetSpec()
		if err != nil {
			log.Errorw("failed to load embedded openapi spec", "error", err)
			http.Error(w, "failed to load OpenAPI spec", http.StatusInternalServerError)
			return
		}
		b, err := spec.MarshalJSON()
		if err != nil {
			log.Errorw("failed to marshal openapi spec", "error", err)
			http.Error(w, "failed to encode OpenAPI spec", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(b); err != nil {
			log.Debugw("write openapi spec response failed", "error", err)
		}
	}).Methods(http.MethodGet)

	docsHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write([]byte(redocPage)); err != nil {
			log.Debugw("write docs page response failed", "error", err)
		}
	}
	// Serve the docs page at both /swagger and /swagger/ so either works.
	r.HandleFunc("/swagger/", docsHandler).Methods(http.MethodGet)
	r.HandleFunc("/swagger", docsHandler).Methods(http.MethodGet)
}
