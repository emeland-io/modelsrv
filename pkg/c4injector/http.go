package c4injector

import (
	"fmt"
	"net/http"
	"strings"

	"go.emeland.io/modelsrv/pkg/authz"
	"go.emeland.io/modelsrv/pkg/model"
)

// Level identifies a C4 diagram level.
type Level int

const (
	LevelContext Level = iota + 1
	LevelContainer
	LevelDeployment
)

// Reasons for the C4 levels EmELand cannot source a diagram for.
const (
	// ReasonComponentLevel explains why level 3 is unavailable: an EmELand
	// Component is already a C4 Container (it is reachable only over an API),
	// and Components have no sub-parts, so there is nothing to draw inside one.
	ReasonComponentLevel = "C4 Component level has no EmELand source: an EmELand Component is a C4 Container, and Components have no sub-parts"
	// ReasonCodeLevel explains why level 4 is unavailable.
	ReasonCodeLevel = "C4 Code level is out of scope for EmELand System Structure"
)

// HTTP paths for each C4 level / diagram.
const (
	PathContext    = "/documents/c4/context.puml"
	PathContainer  = "/documents/c4/container.puml"
	PathComponent  = "/documents/c4/component.puml"
	PathDeployment = "/documents/c4/deployment.puml"
	PathCode       = "/documents/c4/code.puml"
)

// HandlerOptions configures PlantUML HTTP handlers.
type HandlerOptions struct {
	// Authz, when non-nil, requires an auditor principal. The server only
	// supplies this when --trust-auth-headers is set; nil means open access
	// (dev/test, matching the rest of the API).
	Authz *authz.Evaluator
}

// NewLevelHandler serves on-demand C4-PlantUML text for the given level.
func NewLevelHandler(m model.Model, l Landscape, level Level, opts HandlerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowGetHead(w, r) {
			return
		}
		if !requireAuditor(w, r, opts) {
			return
		}

		body, err := renderLevel(m, l, level)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to build diagram: %v", err), http.StatusInternalServerError)
			return
		}
		writePUML(w, r, body, contentDisposition(level))
	})
}

// NewUnavailableLevelHandler always returns 404 with reason, for C4 levels that
// no EmELand resource can populate.
func NewUnavailableLevelHandler(reason string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowGetHead(w, r) {
			return
		}
		http.Error(w, reason, http.StatusNotFound)
	})
}

func allowGetHead(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

func requireAuditor(w http.ResponseWriter, r *http.Request, opts HandlerOptions) bool {
	if opts.Authz == nil {
		return true
	}
	p := principalFromRequest(r)
	if !opts.Authz.IsAuditor(p) {
		http.Error(w, "forbidden: auditor required for landscape document", http.StatusForbidden)
		return false
	}
	return true
}

func writePUML(w http.ResponseWriter, r *http.Request, body, disposition string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", disposition)
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	_, _ = w.Write([]byte(body))
}

func renderLevel(m model.Model, l Landscape, level Level) (string, error) {
	switch level {
	case LevelContext:
		v, err := BuildContextView(m, l)
		if err != nil {
			return "", err
		}
		return RenderContext(v), nil
	case LevelContainer:
		v, err := BuildContainerView(m)
		if err != nil {
			return "", err
		}
		return RenderContainer(v), nil
	case LevelDeployment:
		v, err := BuildDeploymentView(m)
		if err != nil {
			return "", err
		}
		return RenderDeployment(v), nil
	default:
		return "", fmt.Errorf("unknown C4 level %d", level)
	}
}

func contentDisposition(level Level) string {
	switch level {
	case LevelContext:
		return `inline; filename="context.puml"`
	case LevelContainer:
		return `inline; filename="container.puml"`
	case LevelDeployment:
		return `inline; filename="deployment.puml"`
	default:
		return `inline; filename="diagram.puml"`
	}
}

func principalFromRequest(r *http.Request) authz.Principal {
	return authz.Principal{
		Subject:       r.Header.Get(authz.HeaderAuthSubject),
		Groups:        authz.ParseGroups(r.Header.Get(authz.HeaderAuthGroups)),
		AuditorHeader: strings.EqualFold(r.Header.Get(authz.HeaderAuthAuditor), "true"),
	}
}
