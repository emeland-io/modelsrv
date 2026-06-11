//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=config-server.yaml ../../api/openapi/EmergingEnterpriseLandscape-0.1.0-oapi-3.0.3.yaml
/*
Copyright 2025 Lutz Behnke <lutz.behnke@gmx.de>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package oapi

import (
	"context"
	_ "embed"
	"errors"
	"net/http"
	"strings"
	"time"

	strictnethttp "github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	"github.com/oapi-codegen/runtime/types"
	"go.emeland.io/modelsrv/pkg/authz"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

type HeaderLabel string

const (
	HEADER_ACCEPT     = "Accept"
	CONTENT_TYPE_JSON = HeaderLabel("application/json")
	CONTENT_TYPE_HTML = HeaderLabel("text/html")
)

// ctxKey is the type for context.WithValue keys in this package (SA1029).
type ctxKey int

const (
	ctxKeyNegotiatedContentType ctxKey = iota
)

type ApiServer struct {
	Backend model.Model
	Events  events.EventManager
	BaseURL string
	Authz   *authz.Evaluator
}

var _ StrictServerInterface = (*ApiServer)(nil)

/*
// TODO: enable templates when HTML rendering is implemented

/ /go:embed html_templates/service_list.tmpl
var serviceListTemplateStr string
var serviceListTemplate = template.Must(template.New("serviceList").Parse(serviceListTemplateStr))

/ /go:embed html_templates/service.tmpl
var serviceTemplateStr string
var serviceTemplate = template.Must(template.New("service").Parse(serviceTemplateStr))

/ /go:embed html_templates/blueprint.tmpl
var blueprintTemplateStr string
var blueprintTemplate = template.Must(template.New("service").Parse(blueprintTemplateStr))
*/

// ProcessAuthHeaders reads trusted identity headers from the BFF and stores a Principal in context.
func ProcessAuthHeaders(f StrictHandlerFunc, _ string) StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (response any, err error) {
		subject := strings.TrimSpace(r.Header.Get(authz.HeaderAuthSubject))
		groups := authz.ParseGroups(r.Header.Get(authz.HeaderAuthGroups))
		auditor := strings.EqualFold(strings.TrimSpace(r.Header.Get(authz.HeaderAuthAuditor)), "true")

		p := authz.Principal{
			Subject:       subject,
			Groups:        groups,
			AuditorHeader: auditor,
		}
		return f(authz.WithPrincipal(ctx, p), w, r, request)
	}
}

/*
ProcessAcceptHeader is a middleware to process the header indicating the content type(s) accepted by the client
and place them into a parameter of the context for the call to the Strict Server Interface.

As the server currently only supports HTML and JSON, this middleware will default to the HTML content type if
no valid header is set.
*/
func ProcessContentTypeRequest(f StrictHandlerFunc, _ string) StrictHandlerFunc {

	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (response any, err error) {
		var newCtx context.Context
		var contentType string

		// check if header is set at all.
		_, ok := r.Header[HEADER_ACCEPT]
		if ok {
			// this has more compliant processing for edge cases like multiple values and
			// case insensitive matches
			accepted := r.Header.Get(HEADER_ACCEPT)

			contentType = negotiateContent(accepted, []string{"application/json", "text/html"})
		} else {
			contentType = "application/json" // default to HTML if no header is set
		}

		newCtx = context.WithValue(ctx, ctxKeyNegotiatedContentType, contentType)

		return f(newCtx, w, r, request)
	}
}

func negotiateContent(acceptedStr string, supported []string) string {
	accepted := strings.Split(acceptedStr, ",")

	for _, acc := range supported {
		for _, supp := range accepted {
			if strings.EqualFold(acc, supp) {
				return supp
			}
		}
	}

	// if nothing matched, we return the first supported content type
	return supported[0]
}

func NewApiServer(backend model.Model, eventMgr events.EventManager, baseUrl string, authzEval *authz.Evaluator) *ApiServer {
	return &ApiServer{
		Backend: backend,
		Events:  eventMgr,
		BaseURL: baseUrl,
		Authz:   authzEval,
	}
}

// ApiHandlerOptions configures strict-handler middleware for the API.
type ApiHandlerOptions struct {
	TrustAuthHeaders bool
}

func NewApiHandler(server *ApiServer, opts ApiHandlerOptions) ServerInterface {
	middlewares := []strictnethttp.StrictHTTPMiddlewareFunc{ProcessContentTypeRequest}
	if opts.TrustAuthHeaders {
		middlewares = append([]strictnethttp.StrictHTTPMiddlewareFunc{ProcessAuthHeaders}, middlewares...)
	}
	handler := NewStrictHandler(server, middlewares)
	return handler
}

// GetTest implements StrictServerInterface.
func (a *ApiServer) GetTest(ctx context.Context, request GetTestRequestObject) (GetTestResponseObject, error) {
	return GetTest200Response{}, nil
}

// parseISO8601 is more tolerant when parsing the input string, than the rfc3339 compliant parsing implemented by the golang default
//
//nolint:unused
func parseISO8601(input string) (*types.Date, error) {
	parseError := &time.ParseError{}

	t, err := time.Parse("2006-01-02T15:04:05Z07:00", input)
	if errors.As(err, &parseError) {
		// It may be a date stamp only.
		t, err = time.Parse("2006-01-02", input)
	}
	if err != nil {
		return nil, err
	}

	// It may be a date stamp only.

	return &types.Date{
		Time: t,
	}, nil
}
