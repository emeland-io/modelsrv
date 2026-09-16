package c4injector

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/authz"
)

func TestHTTP_Levels(t *testing.T) {
	m := newTestModel(t)
	l := DefaultLandscape()
	opts := HandlerOptions{} // nil Authz: open, matching --c4-doc without --trust-auth-headers

	for _, tc := range []struct {
		level Level
		want  string
	}{
		{LevelContext, "!include <C4/C4_Context>"},
		{LevelContainer, "!include <C4/C4_Container>"},
		{LevelDeployment, "Deployment Diagram"},
	} {
		h := NewLevelHandler(m, l, tc.level, opts)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code, "level %d", tc.level)
		require.True(t, strings.HasPrefix(rr.Header().Get("Content-Type"), "text/plain"))
		require.Contains(t, rr.Body.String(), tc.want)
		require.True(t, strings.HasPrefix(rr.Body.String(), "@startuml"))
	}
}

func TestHTTP_UnavailableLevels404(t *testing.T) {
	for _, tc := range []struct {
		path, reason, want string
	}{
		{PathComponent, ReasonComponentLevel, "has no EmELand source"},
		{PathCode, ReasonCodeLevel, "out of scope"},
	} {
		h := NewUnavailableLevelHandler(tc.reason)
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusNotFound, rr.Code, tc.path)
		require.Contains(t, rr.Body.String(), tc.want)
	}
}

func TestHTTP_AuditorRequired(t *testing.T) {
	m := newTestModel(t)
	eval := authz.NewEvaluator(authz.Config{AuditorIdentity: "auditor"})
	h := NewLevelHandler(m, DefaultLandscape(), LevelContext, HandlerOptions{Authz: eval})

	req := httptest.NewRequest(http.MethodGet, PathContext, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusForbidden, rr.Code)

	req = httptest.NewRequest(http.MethodGet, PathContext, nil)
	req.Header.Set(authz.HeaderAuthSubject, "auditor")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}
