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
	"strconv"
	"strings"
	"time"

	strictnethttp "github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	"github.com/oapi-codegen/runtime/types"
	"gitlab.com/emeland/modelsrv/pkg/events"
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
	Events  events.EventManager
	BaseURL string
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

// GetEventsQuerySequenceId implements StrictServerInterface.
func (a *ApiServer) GetEventsQuerySequenceId(ctx context.Context, request GetEventsQuerySequenceIdRequestObject) (GetEventsQuerySequenceIdResponseObject, error) {
	var requestSequenceId uint64

	// parse sequenceId
	requestSequenceId, err := strconv.ParseUint(request.SequenceId, 10, 64)
	if err != nil {
		return nil, err
	}

	currSequenceId, err := a.Events.GetCurrentSequenceId(ctx)
	if err != nil {
		return nil, err
	}

	if requestSequenceId == currSequenceId {
		// no new events
		resp := GetEventsQuerySequenceId200Response{}
		return GetEventsQuerySequenceId200Response(resp), nil
	} else if requestSequenceId < currSequenceId {
		// there are new events
		resp := GetEventsQuerySequenceId308Response{}
		return GetEventsQuerySequenceId308Response(resp), nil
	} else {
		// client is ahead of server?
		resp := ""
		return GetEventsQuerySequenceId404JSONResponse(resp), nil
	}
}

// GetLandscapeApiInstances implements StrictServerInterface.
func (a *ApiServer) GetLandscapeApiInstances(ctx context.Context, request GetLandscapeApiInstancesRequestObject) (GetLandscapeApiInstancesResponseObject, error) {
	instanceArr, err := a.Backend.GetApiInstances()

	if err != nil {
		return nil, err
	}

	respBody := []InstanceListItem{}

	for _, instance := range instanceArr {
		reference := fmt.Sprintf("%s/landscape/api-instances/%s", a.BaseURL, instance.InstanceId.String())
		item := InstanceListItem{
			InstanceId:  &instance.InstanceId,
			DisplayName: &instance.DisplayName,
			Reference:   &reference,
		}
		respBody = append(respBody, item)
	}

	return GetLandscapeApiInstances200JSONResponse(respBody), nil
}

// GetLandscapeApiInstancesApiInstanceId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeApiInstancesApiInstanceId(ctx context.Context, request GetLandscapeApiInstancesApiInstanceIdRequestObject) (GetLandscapeApiInstancesApiInstanceIdResponseObject, error) {
	apiInstance := a.Backend.GetApiInstanceById(request.ApiInstanceId)
	if apiInstance == nil {
		errorstr := fmt.Sprintf("api instance %s not found", request.ApiInstanceId.String())
		return GetLandscapeApiInstancesApiInstanceId404JSONResponse(errorstr), nil
	}

	respBody := ApiInstance{
		ApiInstanceId: apiInstance.InstanceId,
		DisplayName:   apiInstance.DisplayName,
		Annotations:   cloneAnnotations(apiInstance.Annotations),
	}

	if apiInstance.ApiRef != nil {
		respBody.Api = &(apiInstance.ApiRef.ApiID)
	}

	if apiInstance.SystemInstance != nil {
		respBody.SystemInstance = &(apiInstance.SystemInstance.InstanceId)
	}

	return GetLandscapeApiInstancesApiInstanceId200JSONResponse(respBody), nil
}

// GetLandscapeApis implements StrictServerInterface.
func (a *ApiServer) GetLandscapeApis(ctx context.Context, request GetLandscapeApisRequestObject) (GetLandscapeApisResponseObject, error) {
	apiArr, err := a.Backend.GetApis()

	if err != nil {
		return nil, err
	}

	respBody := []InstanceListItem{}

	for _, api := range apiArr {
		reference := fmt.Sprintf("%s/landscape/apis/%s", a.BaseURL, api.ApiId.String())
		item := InstanceListItem{
			InstanceId:  &api.ApiId,
			DisplayName: &api.DisplayName,
			Reference:   &reference,
		}
		respBody = append(respBody, item)
	}

	return GetLandscapeApis200JSONResponse(respBody), nil
}

// GetLandscapeApisApiId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeApisApiId(ctx context.Context, request GetLandscapeApisApiIdRequestObject) (GetLandscapeApisApiIdResponseObject, error) {
	api := a.Backend.GetApiById(request.ApiId)
	if api == nil {
		errorstr := fmt.Sprintf("api %s not found", request.ApiId.String())
		return GetLandscapeApisApiId404JSONResponse(errorstr), nil
	}

	respBody := API{
		ApiId:       &api.ApiId,
		DisplayName: api.DisplayName,
		Annotations: cloneAnnotations(api.Annotations),
	}

	if api.System != nil {
		respBody.System = &api.System.SystemId
	}

	return GetLandscapeApisApiId200JSONResponse(respBody), nil
}

// GetLandscapeComponentInstances implements StrictServerInterface.
func (a *ApiServer) GetLandscapeComponentInstances(ctx context.Context, request GetLandscapeComponentInstancesRequestObject) (GetLandscapeComponentInstancesResponseObject, error) {
	instanceArr, err := a.Backend.GetComponentInstances()

	if err != nil {
		return nil, err
	}

	respBody := []InstanceListItem{}

	for _, instance := range instanceArr {
		reference := fmt.Sprintf("%s/landscape/component-instances/%s", a.BaseURL, instance.InstanceId.String())
		item := InstanceListItem{
			InstanceId:  &instance.InstanceId,
			DisplayName: &instance.DisplayName,
			Reference:   &reference,
		}
		respBody = append(respBody, item)
	}

	return GetLandscapeComponentInstances200JSONResponse(respBody), nil
}

// GetLandscapeComponentInstancesComponentInstanceId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeComponentInstancesComponentInstanceId(ctx context.Context, request GetLandscapeComponentInstancesComponentInstanceIdRequestObject) (GetLandscapeComponentInstancesComponentInstanceIdResponseObject, error) {
	componentInstance := a.Backend.GetComponentInstanceById(request.ComponentInstanceId)
	if componentInstance == nil {
		errorstr := fmt.Sprintf("componentInstance %s not found", request.ComponentInstanceId.String())
		return GetLandscapeComponentInstancesComponentInstanceId404JSONResponse(errorstr), nil
	}

	respBody := ComponentInstance{
		ComponentInstanceId: componentInstance.InstanceId,
		DisplayName:         componentInstance.DisplayName,
		Annotations:         cloneAnnotations(componentInstance.Annotations),
	}

	if componentInstance.ComponentRef != nil {
		respBody.Component = componentInstance.ComponentRef.ComponentId
	}

	if componentInstance.SystemInstance != nil {
		respBody.SystemInstance = componentInstance.SystemInstance.InstanceId
	}

	return GetLandscapeComponentInstancesComponentInstanceId200JSONResponse(respBody), nil
}

// GetLandscapeComponents implements StrictServerInterface.
func (a *ApiServer) GetLandscapeComponents(ctx context.Context, request GetLandscapeComponentsRequestObject) (GetLandscapeComponentsResponseObject, error) {
	componentArr, err := a.Backend.GetComponents()

	if err != nil {
		return nil, err
	}

	respBody := []InstanceListItem{}

	for _, component := range componentArr {
		reference := fmt.Sprintf("%s/landscape/components/%s", a.BaseURL, component.ComponentId.String())
		item := InstanceListItem{
			InstanceId:  &component.ComponentId,
			DisplayName: &component.DisplayName,
			Reference:   &reference,
		}
		respBody = append(respBody, item)
	}

	return GetLandscapeComponents200JSONResponse(respBody), nil
}

// GetLandscapeComponentsComponentId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeComponentsComponentId(ctx context.Context, request GetLandscapeComponentsComponentIdRequestObject) (GetLandscapeComponentsComponentIdResponseObject, error) {
	component := a.Backend.GetComponentById(request.ComponentId)
	if component == nil {
		errorstr := fmt.Sprintf("component %s not found", request.ComponentId.String())
		return GetLandscapeComponentsComponentId404JSONResponse(errorstr), nil
	}

	respBody := Component{
		ComponentId: &component.ComponentId,
		DisplayName: component.DisplayName,
		Description: &component.Description,
		Annotations: cloneAnnotations(component.Annotations),
	}

	if component.System != nil {
		respBody.System = component.System.SystemId
	}

	return GetLandscapeComponentsComponentId200JSONResponse(respBody), nil
}

// GetLandscapeFindings implements StrictServerInterface.
func (a *ApiServer) GetLandscapeFindings(ctx context.Context, request GetLandscapeFindingsRequestObject) (GetLandscapeFindingsResponseObject, error) {
	findingsArr, err := a.Backend.GetFindings()

	if err != nil {
		return nil, err
	}

	respBody := []InstanceListItem{}

	for _, finding := range findingsArr {
		reference := fmt.Sprintf("%s/landscape/findings/%s", a.BaseURL, finding.FindingId.String())
		item := InstanceListItem{
			InstanceId:  &finding.FindingId,
			DisplayName: &finding.Summary,
			Reference:   &reference,
		}
		respBody = append(respBody, item)
	}

	return GetLandscapeFindings200JSONResponse(respBody), nil
}

// GetLandscapeFindingsFindingId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeFindingsFindingId(ctx context.Context, request GetLandscapeFindingsFindingIdRequestObject) (GetLandscapeFindingsFindingIdResponseObject, error) {
	finding := a.Backend.GetFindingById(request.FindingId)
	if finding == nil {
		return nil, fmt.Errorf("finding %s not found", request.FindingId.String())
	}

	respBody := Finding{
		FindingId:   finding.FindingId,
		Summary:     finding.Summary,
		Description: &finding.Description,
		Resources:   cloneResourceRefs(finding.Resources),
		Annotations: cloneAnnotations(finding.Annotations),
	}
	return GetLandscapeFindingsFindingId200JSONResponse(respBody), nil
}

// GetLandscapeSystemInstances implements StrictServerInterface.
func (a *ApiServer) GetLandscapeSystemInstances(ctx context.Context, request GetLandscapeSystemInstancesRequestObject) (GetLandscapeSystemInstancesResponseObject, error) {
	instanceArr, err := a.Backend.GetSystemInstances()

	if err != nil {
		return nil, err
	}

	respBody := []InstanceListItem{}

	for _, instance := range instanceArr {
		reference := fmt.Sprintf("%s/landscape/system-instances/%s", a.BaseURL, instance.InstanceId.String())
		item := InstanceListItem{
			InstanceId:  &instance.InstanceId,
			DisplayName: &instance.DisplayName,
			Reference:   &reference,
		}
		respBody = append(respBody, item)
	}

	return GetLandscapeSystemInstances200JSONResponse(respBody), nil
}

// GetLandscapeSystemInstancesSystemInstanceId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeSystemInstancesSystemInstanceId(ctx context.Context, request GetLandscapeSystemInstancesSystemInstanceIdRequestObject) (GetLandscapeSystemInstancesSystemInstanceIdResponseObject, error) {
	systemInstance := a.Backend.GetSystemInstanceById(request.SystemInstanceId)
	if systemInstance == nil {
		errorstr := fmt.Sprintf("system instance %s not found", request.SystemInstanceId.String())
		return GetLandscapeSystemInstancesSystemInstanceId404JSONResponse(errorstr), nil
	}

	respBody := SystemInstance{
		SystemInstanceId: systemInstance.InstanceId,
		DisplayName:      systemInstance.DisplayName,
		Annotations:      cloneAnnotations(systemInstance.Annotations),
	}

	if systemInstance.SystemRef != nil {
		respBody.System = systemInstance.SystemRef.SystemId
	}

	if systemInstance.ContextRef != nil {
		respBody.Context = &systemInstance.ContextRef.ContextId
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
		reference := fmt.Sprintf("%s/landscape/systems/%s", a.BaseURL, system.SystemId.String())
		item := InstanceListItem{
			InstanceId:  &system.SystemId,
			DisplayName: &system.DisplayName,
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

	respBody := System{
		SystemId:    &system.SystemId,
		DisplayName: system.DisplayName,
		Description: &system.Description,
		Annotations: cloneAnnotations(system.Annotations),
	}

	return GetLandscapeSystemsSystemId200JSONResponse(respBody), nil
}

// GetTest implements StrictServerInterface.
func (a *ApiServer) GetTest(ctx context.Context, request GetTestRequestObject) (GetTestResponseObject, error) {
	return GetTest200Response{}, nil
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

// GetLandscapeContexts implements StrictServerInterface.
func (a *ApiServer) GetLandscapeContexts(ctx context.Context, request GetLandscapeContextsRequestObject) (GetLandscapeContextsResponseObject, error) {
	contextArr, err := a.Backend.GetContexts()

	if err != nil {
		return nil, err
	}

	respBody := []InstanceListItem{}

	for _, context := range contextArr {
		reference := fmt.Sprintf("%s/landscape/contexts/%s", a.BaseURL, context.GetContextId().String())
		displayName := context.GetDisplayName()
		contextId := context.GetContextId()
		item := InstanceListItem{
			InstanceId:  &contextId,
			DisplayName: &displayName,
			Reference:   &reference,
		}
		respBody = append(respBody, item)
	}

	return GetLandscapeContexts200JSONResponse(respBody), nil
}

// GetLandscapeContextsContextId implements StrictServerInterface.
func (a *ApiServer) GetLandscapeContextsContextId(ctx context.Context, request GetLandscapeContextsContextIdRequestObject) (GetLandscapeContextsContextIdResponseObject, error) {
	context := a.Backend.GetContextById(request.ContextId)
	if context == nil {
		errorstr := fmt.Sprintf("context %s not found", request.ContextId.String())
		return GetLandscapeContextsContextId404JSONResponse(errorstr), nil
	}

	displayName := context.GetDisplayName()
	contextId := context.GetContextId()

	respBody := Context{
		ContextId:   contextId,
		DisplayName: displayName,
		Annotations: cloneAnnotations2(context),
	}

	return GetLandscapeContextsContextId200JSONResponse(respBody), nil
}
