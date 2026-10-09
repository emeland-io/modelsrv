/*
Copyright © 2025 Lutz Behnke <lutz.behnke@gmx.de>
*/
package endpoint

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func newSwaggerTestServer(t *testing.T) http.Handler {
	t.Helper()
	r := mux.NewRouter()
	registerSwaggerDocs(r, zap.NewNop().Sugar())
	return r
}

func TestSwaggerDocsPage(t *testing.T) {
	srv := newSwaggerTestServer(t)

	for _, path := range []string{"/swagger/", "/swagger"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: status = %d, want 200", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("GET %s: content-type = %q, want text/html", path, ct)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "<redoc") || !strings.Contains(body, `spec-url="openapi.json"`) {
			t.Errorf("GET %s: body does not reference redoc + openapi.json; got:\n%s", path, body)
		}
	}
}

func TestSwaggerOpenAPISpec(t *testing.T) {
	srv := newSwaggerTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/swagger/openapi.json", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content-type = %q, want application/json", ct)
	}

	// The body must be a valid OpenAPI document (has openapi + paths).
	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("openapi.json is not valid JSON: %v", err)
	}
	if _, ok := doc["openapi"]; !ok {
		t.Errorf("openapi.json missing 'openapi' field")
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok || len(paths) == 0 {
		t.Fatalf("openapi.json missing/empty 'paths'")
	}
	// Spot-check a known landscape endpoint is documented.
	if _, ok := paths["/landscape/nodes"]; !ok {
		t.Errorf("openapi.json paths missing /landscape/nodes")
	}
}
