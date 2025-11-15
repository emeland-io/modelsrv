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
	"gitlab.com/emeland/modelsrv/pkg/model"
)

type ContextLabel string
type HeaderLabel string
type RequestHeaderKey string

const (
	HEADER_KEY_AUTH_USER              = "X-Snackmgr-Authenticated-User"
	HEADER_ACCEPT                     = "Accept"
	OWNER_KEY            ContextLabel = "owner"
	CONTENT_TYPE_JSON                 = HeaderLabel("application/json")
	CONTENT_TYPE_HTML                 = HeaderLabel("text/html")
)

type ApiServer struct {
	Backend model.Model
}

var _ StrictServerInterface = (*ApiServer)(nil)

/*
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

/*
ProcessAuthHeader is a middleware to transfer the authentication header "X-Shmits-Authenticated-User" into the context for
the call to the Strict Server Interface.

	Since the requirement for the existence of a valid user depends on the actual method an path being accessed, validation
	is handled in the individual methods of the Strict Service Interface implementation.
*/
func ProcessAuthHeader(f StrictHandlerFunc, _ string) StrictHandlerFunc {

	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (response any, err error) {
		var newCtx context.Context

		// check if header is set at all.
		_, ok := r.Header[HEADER_KEY_AUTH_USER]
		if ok {
			// this has more compliant processing for edge cases like multiple values and
			// case insensitive matches
			owner := r.Header.Get(HEADER_KEY_AUTH_USER)

			newCtx = context.WithValue(ctx, OWNER_KEY, owner)

		} else {
			newCtx = ctx
		}

		return f(newCtx, w, r, request)
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

		newCtx = context.WithValue(ctx, HEADER_ACCEPT, contentType)

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

func NewApiServer(backend model.Model) *ApiServer {
	return &ApiServer{
		Backend: backend,
	}
}

func NewApiHandler(backend model.Model) ServerInterface {
	server := NewApiServer(backend)
	handler := NewStrictHandler(server,
		[]strictnethttp.StrictHTTPMiddlewareFunc{ProcessAuthHeader, ProcessContentTypeRequest})

	return handler
}

// GetEventsQuerySequenceId implements StrictServerInterface.
func (a *ApiServer) GetEventsQuerySequenceId(ctx context.Context, request GetEventsQuerySequenceIdRequestObject) (GetEventsQuerySequenceIdResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeApiInstances implements StrictServerInterface.
func (a *ApiServer) GetLandscapeApiInstances(ctx context.Context, request GetLandscapeApiInstancesRequestObject) (GetLandscapeApiInstancesResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeApiInstancesApiInstanceId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeApiInstancesApiInstanceId(ctx context.Context, request GetLandscapeApiInstancesApiInstanceIdRequestObject) (GetLandscapeApiInstancesApiInstanceIdResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeApis implements StrictServerInterface.
func (a *ApiServer) GetLandscapeApis(ctx context.Context, request GetLandscapeApisRequestObject) (GetLandscapeApisResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeApisApiId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeApisApiId(ctx context.Context, request GetLandscapeApisApiIdRequestObject) (GetLandscapeApisApiIdResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeComponentInstances implements StrictServerInterface.
func (a *ApiServer) GetLandscapeComponentInstances(ctx context.Context, request GetLandscapeComponentInstancesRequestObject) (GetLandscapeComponentInstancesResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeComponentInstancesComponentInstanceId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeComponentInstancesComponentInstanceId(ctx context.Context, request GetLandscapeComponentInstancesComponentInstanceIdRequestObject) (GetLandscapeComponentInstancesComponentInstanceIdResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeComponents implements StrictServerInterface.
func (a *ApiServer) GetLandscapeComponents(ctx context.Context, request GetLandscapeComponentsRequestObject) (GetLandscapeComponentsResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeComponentsComponentId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeComponentsComponentId(ctx context.Context, request GetLandscapeComponentsComponentIdRequestObject) (GetLandscapeComponentsComponentIdResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeFindings implements StrictServerInterface.
func (a *ApiServer) GetLandscapeFindings(ctx context.Context, request GetLandscapeFindingsRequestObject) (GetLandscapeFindingsResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeFindingsFindingId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeFindingsFindingId(ctx context.Context, request GetLandscapeFindingsFindingIdRequestObject) (GetLandscapeFindingsFindingIdResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeSystemInstances implements StrictServerInterface.
func (a *ApiServer) GetLandscapeSystemInstances(ctx context.Context, request GetLandscapeSystemInstancesRequestObject) (GetLandscapeSystemInstancesResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeSystemInstancesSystemInstanceId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeSystemInstancesSystemInstanceId(ctx context.Context, request GetLandscapeSystemInstancesSystemInstanceIdRequestObject) (GetLandscapeSystemInstancesSystemInstanceIdResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeSystems implements StrictServerInterface.
func (a *ApiServer) GetLandscapeSystems(ctx context.Context, request GetLandscapeSystemsRequestObject) (GetLandscapeSystemsResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeSystemsSystemId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeSystemsSystemId(ctx context.Context, request GetLandscapeSystemsSystemIdRequestObject) (GetLandscapeSystemsSystemIdResponseObject, error) {
	panic("unimplemented")
}

// GetTest implements StrictServerInterface.
func (a *ApiServer) GetTest(ctx context.Context, request GetTestRequestObject) (GetTestResponseObject, error) {
	panic("unimplemented")
}

// PostEventsRegister implements StrictServerInterface.
func (a *ApiServer) PostEventsRegister(ctx context.Context, request PostEventsRegisterRequestObject) (PostEventsRegisterResponseObject, error) {
	panic("unimplemented")
}

// parseISO8601 is more tolerant when parsing the input string, than the rfc3339 compliant parsing implemented by the golang default
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
