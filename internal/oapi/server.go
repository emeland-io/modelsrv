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
	"fmt"
	"net/http"
	"strings"
	"time"

	strictnethttp "github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	"github.com/oapi-codegen/runtime/types"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
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
	Events  events.EventManager
	BaseURL string
}

// GetLandscapeContextTypes implements [StrictServerInterface].

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

func NewApiServer(backend model.Model, eventMgr events.EventManager, baseUrl string) *ApiServer {
	return &ApiServer{
		Backend: backend,
		Events:  eventMgr,
		BaseURL: baseUrl,
	}
}

func NewApiHandler(server *ApiServer) ServerInterface {
	handler := NewStrictHandler(server,
		[]strictnethttp.StrictHTTPMiddlewareFunc{ProcessAuthHeader, ProcessContentTypeRequest})

	return handler
}

// GetLandscapeApiInstances implements StrictServerInterface.
func (a *ApiServer) GetLandscapeApiInstances(ctx context.Context, request GetLandscapeApiInstancesRequestObject) (GetLandscapeApiInstancesResponseObject, error) {
	instanceArr, err := a.Backend.GetApiInstances()
	if err != nil {
		return nil, err
	}
	return GetLandscapeApiInstances200JSONResponse(buildInstanceList(a.BaseURL, "/landscape/api-instances", instanceArr)), nil
}

// GetLandscapeApiInstancesApiInstanceId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeApiInstancesApiInstanceId(ctx context.Context, request GetLandscapeApiInstancesApiInstanceIdRequestObject) (GetLandscapeApiInstancesApiInstanceIdResponseObject, error) {
	apiInstance := a.Backend.GetApiInstanceById(request.ApiInstanceId)
	if apiInstance == nil {
		errorstr := fmt.Sprintf("api instance %s not found", request.ApiInstanceId.String())
		return GetLandscapeApiInstancesApiInstanceId404JSONResponse(errorstr), nil
	}

	respBody := ApiInstance{
		ApiInstanceId: apiInstance.GetInstanceId(),
		DisplayName:   apiInstance.GetDisplayName(),
		Annotations:   cloneAnnotations(apiInstance.GetAnnotations()),
	}

	if apiInstance.GetApiRef() != nil {
		respBody.Api = &(apiInstance.GetApiRef().ApiID)
	}

	if apiInstance.GetSystemInstance() != nil {
		respBody.SystemInstance = &(apiInstance.GetSystemInstance().InstanceId)
	}

	return GetLandscapeApiInstancesApiInstanceId200JSONResponse(respBody), nil
}

// GetLandscapeApis implements StrictServerInterface.
func (a *ApiServer) GetLandscapeApis(ctx context.Context, request GetLandscapeApisRequestObject) (GetLandscapeApisResponseObject, error) {
	apiArr, err := a.Backend.GetApis()
	if err != nil {
		return nil, err
	}
	return GetLandscapeApis200JSONResponse(buildInstanceList(a.BaseURL, "/landscape/apis", apiArr)), nil
}

// GetLandscapeApisApiId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeApisApiId(ctx context.Context, request GetLandscapeApisApiIdRequestObject) (GetLandscapeApisApiIdResponseObject, error) {
	api := a.Backend.GetApiById(request.ApiId)
	if api == nil {
		errorstr := fmt.Sprintf("api %s not found", request.ApiId.String())
		return GetLandscapeApisApiId404JSONResponse(errorstr), nil
	}

	apiId := api.GetApiId()
	respBody := API{
		ApiId:       &apiId,
		DisplayName: api.GetDisplayName(),
		Annotations: cloneAnnotations(api.GetAnnotations()),
	}

	if api.GetSystem() != nil {
		respBody.System = &api.GetSystem().SystemId
	}

	return GetLandscapeApisApiId200JSONResponse(respBody), nil
}

// GetLandscapeComponentInstances implements StrictServerInterface.
func (a *ApiServer) GetLandscapeComponentInstances(ctx context.Context, request GetLandscapeComponentInstancesRequestObject) (GetLandscapeComponentInstancesResponseObject, error) {
	instanceArr, err := a.Backend.GetComponentInstances()
	if err != nil {
		return nil, err
	}
	return GetLandscapeComponentInstances200JSONResponse(buildInstanceList(a.BaseURL, "/landscape/component-instances", instanceArr)), nil
}

// GetLandscapeComponentInstancesComponentInstanceId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeComponentInstancesComponentInstanceId(ctx context.Context, request GetLandscapeComponentInstancesComponentInstanceIdRequestObject) (GetLandscapeComponentInstancesComponentInstanceIdResponseObject, error) {
	componentInstance := a.Backend.GetComponentInstanceById(request.ComponentInstanceId)
	if componentInstance == nil {
		errorstr := fmt.Sprintf("componentInstance %s not found", request.ComponentInstanceId.String())
		return GetLandscapeComponentInstancesComponentInstanceId404JSONResponse(errorstr), nil
	}

	respBody := ComponentInstance{
		ComponentInstanceId: componentInstance.GetInstanceId(),
		DisplayName:         componentInstance.GetDisplayName(),
		Annotations:         cloneAnnotations(componentInstance.GetAnnotations()),
	}

	if componentInstance.GetComponentRef() != nil {
		respBody.Component = componentInstance.GetComponentRef().ComponentId
	}

	if componentInstance.GetSystemInstance() != nil {
		respBody.SystemInstance = componentInstance.GetSystemInstance().InstanceId
	}

	return GetLandscapeComponentInstancesComponentInstanceId200JSONResponse(respBody), nil
}

// GetLandscapeComponents implements StrictServerInterface.
func (a *ApiServer) GetLandscapeComponents(ctx context.Context, request GetLandscapeComponentsRequestObject) (GetLandscapeComponentsResponseObject, error) {
	componentArr, err := a.Backend.GetComponents()
	if err != nil {
		return nil, err
	}
	return GetLandscapeComponents200JSONResponse(buildInstanceList(a.BaseURL, "/landscape/components", componentArr)), nil
}

// GetLandscapeComponentsComponentId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeComponentsComponentId(ctx context.Context, request GetLandscapeComponentsComponentIdRequestObject) (GetLandscapeComponentsComponentIdResponseObject, error) {
	component := a.Backend.GetComponentById(request.ComponentId)
	if component == nil {
		errorstr := fmt.Sprintf("component %s not found", request.ComponentId.String())
		return GetLandscapeComponentsComponentId404JSONResponse(errorstr), nil
	}

	componentId := component.GetComponentId()
	description := component.GetDescription()
	respBody := Component{
		ComponentId: &componentId,
		DisplayName: component.GetDisplayName(),
		Description: &description,
		Annotations: cloneAnnotations(component.GetAnnotations()),
	}

	if component.GetSystem() != nil {
		respBody.System = component.GetSystem().SystemId
	}

	return GetLandscapeComponentsComponentId200JSONResponse(respBody), nil
}

// GetLandscapeFindings implements StrictServerInterface.
func (a *ApiServer) GetLandscapeFindings(ctx context.Context, request GetLandscapeFindingsRequestObject) (GetLandscapeFindingsResponseObject, error) {
	findingsArr, err := a.Backend.GetFindings()
	if err != nil {
		return nil, err
	}
	return GetLandscapeFindings200JSONResponse(buildInstanceList(a.BaseURL, "/landscape/findings", findingsArr)), nil
}

// GetLandscapeFindingsFindingId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeFindingsFindingId(ctx context.Context, request GetLandscapeFindingsFindingIdRequestObject) (GetLandscapeFindingsFindingIdResponseObject, error) {
	finding := a.Backend.GetFindingById(request.FindingId)
	if finding == nil {
		return nil, fmt.Errorf("finding %s not found", request.FindingId.String())
	}

	description := finding.GetDescription()
	respBody := Finding{
		FindingId:   finding.GetFindingId(),
		Summary:     finding.GetSummary(),
		Description: &description,
		Resources:   cloneResourceRefs(finding.GetResources()),
		Annotations: cloneAnnotations(finding.GetAnnotations()),
	}
	return GetLandscapeFindingsFindingId200JSONResponse(respBody), nil
}

// GetLandscapeSystemInstances implements StrictServerInterface.
func (a *ApiServer) GetLandscapeSystemInstances(ctx context.Context, request GetLandscapeSystemInstancesRequestObject) (GetLandscapeSystemInstancesResponseObject, error) {
	instanceArr, err := a.Backend.GetSystemInstances()
	if err != nil {
		return nil, err
	}
	return GetLandscapeSystemInstances200JSONResponse(buildInstanceList(a.BaseURL, "/landscape/system-instances", instanceArr)), nil
}

// GetLandscapeSystemInstancesSystemInstanceId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeSystemInstancesSystemInstanceId(ctx context.Context, request GetLandscapeSystemInstancesSystemInstanceIdRequestObject) (GetLandscapeSystemInstancesSystemInstanceIdResponseObject, error) {
	systemInstance := a.Backend.GetSystemInstanceById(request.SystemInstanceId)
	if systemInstance == nil {
		errorstr := fmt.Sprintf("system instance %s not found", request.SystemInstanceId.String())
		return GetLandscapeSystemInstancesSystemInstanceId404JSONResponse(errorstr), nil
	}

	respBody := SystemInstance{
		SystemInstanceId: systemInstance.GetInstanceId(),
		DisplayName:      systemInstance.GetDisplayName(),
		Annotations:      cloneAnnotations(systemInstance.GetAnnotations()),
	}

	if systemInstance.GetSystemRef() != nil {
		respBody.System = systemInstance.GetSystemRef().SystemId
	}

	if systemInstance.GetContextRef() != nil {
		respBody.Context = &systemInstance.GetContextRef().ContextId
	}
	return GetLandscapeSystemInstancesSystemInstanceId200JSONResponse(respBody), nil
}

// GetLandscapeSystems implements StrictServerInterface.
func (a *ApiServer) GetLandscapeSystems(ctx context.Context, request GetLandscapeSystemsRequestObject) (GetLandscapeSystemsResponseObject, error) {
	systemArr, err := a.Backend.GetSystems()

	if err != nil {
		return nil, err
	}

	respBody := []InstanceListItem{}

	for _, system := range systemArr {
		systemId := system.GetSystemId()
		displayName := system.GetDisplayName()
		reference := fmt.Sprintf("%s/landscape/systems/%s", a.BaseURL, systemId.String())

		item := InstanceListItem{
			InstanceId:  &systemId,
			DisplayName: &displayName,
			Reference:   &reference,
		}
		respBody = append(respBody, item)
	}

	return GetLandscapeSystems200JSONResponse(respBody), nil
}

// GetLandscapeSystemsSystemId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeSystemsSystemId(ctx context.Context, request GetLandscapeSystemsSystemIdRequestObject) (GetLandscapeSystemsSystemIdResponseObject, error) {
	system := a.Backend.GetSystemById(request.SystemId)
	if system == nil {
		errorstr := fmt.Sprintf("system %s not found", request.SystemId.String())
		return GetLandscapeSystemsSystemId404JSONResponse(errorstr), nil
	}

	systemId := system.GetSystemId()
	displayName := system.GetDisplayName()
	description := system.GetDescription()

	respBody := System{
		SystemId:    &systemId,
		DisplayName: displayName,
		Description: &description,
		Annotations: cloneAnnotations(system.GetAnnotations()),
	}

	return GetLandscapeSystemsSystemId200JSONResponse(respBody), nil
}

// GetTest implements StrictServerInterface.
func (a *ApiServer) GetTest(ctx context.Context, request GetTestRequestObject) (GetTestResponseObject, error) {
	return GetTest200Response{}, nil
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

// GetLandscapeFindingTypes implements [StrictServerInterface].
func (a *ApiServer) GetLandscapeFindingTypes(ctx context.Context, request GetLandscapeFindingTypesRequestObject) (GetLandscapeFindingTypesResponseObject, error) {
	findingTypesArr, err := a.Backend.GetFindingTypes()

	if err != nil {
		return nil, err
	}

	respBody := []InstanceListItem{}

	for _, findingType := range findingTypesArr {
		findingTypeId := findingType.GetFindingTypeId()
		displayName := findingType.GetDisplayName()
		reference := fmt.Sprintf("%s/landscape/findingTypes/%s", a.BaseURL, findingTypeId.String())

		item := InstanceListItem{
			InstanceId:  &findingTypeId,
			DisplayName: &displayName,
			Reference:   &reference,
		}
		respBody = append(respBody, item)
	}

	return GetLandscapeFindingTypes200JSONResponse(respBody), nil
}

// GetLandscapeFindingTypesFindingTypeId implements [StrictServerInterface].
func (a *ApiServer) GetLandscapeFindingTypesFindingTypeId(ctx context.Context, request GetLandscapeFindingTypesFindingTypeIdRequestObject) (GetLandscapeFindingTypesFindingTypeIdResponseObject, error) {
	findingType := a.Backend.GetFindingTypeById(request.FindingTypeId)
	if findingType == nil {
		errorstr := fmt.Sprintf("finding type %s not found", request.FindingTypeId.String())
		return GetLandscapeFindingTypesFindingTypeId404JSONResponse(errorstr), nil
	}

	displayName := findingType.GetDisplayName()
	findingTypeId := findingType.GetFindingTypeId()

	respBody := FindingType{
		FindingTypeId: &findingTypeId,
		DisplayName:   &displayName,
		Annotations:   cloneAnnotations(findingType.GetAnnotations()),
	}

	return GetLandscapeFindingTypesFindingTypeId200JSONResponse(respBody), nil

}
