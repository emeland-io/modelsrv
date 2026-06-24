package oapi

import (
	"context"
	"fmt"

	"go.emeland.io/modelsrv/pkg/authz"
	"go.emeland.io/modelsrv/pkg/events"
)

// GetLandscapeFindings implements [StrictServerInterface].
func (a *ApiServer) GetLandscapeFindings(ctx context.Context, request GetLandscapeFindingsRequestObject) (GetLandscapeFindingsResponseObject, error) {
	items, err := a.Backend.GetFindings()
	if err != nil {
		return nil, err
	}
	if a.Authz != nil {
		principal := authz.PrincipalFromCtx(ctx)
		items = authz.FilterVisible(a.Authz, principal, events.FindingResource, items)
	}
	out := make([]FindingView, 0, len(items))
	for _, item := range items {
		out = append(out, findingToView(a.Backend, a.BaseURL, item))
	}
	return GetLandscapeFindings200JSONResponse(out), nil
}

// GetLandscapeFindingsFindingId implements [StrictServerInterface].
func (a *ApiServer) GetLandscapeFindingsFindingId(ctx context.Context, request GetLandscapeFindingsFindingIdRequestObject) (GetLandscapeFindingsFindingIdResponseObject, error) {
	item := a.Backend.GetFindingById(request.FindingId)
	if item == nil {
		msg := fmt.Sprintf("finding %s not found", request.FindingId.String())
		return GetLandscapeFindingsFindingId404JSONResponse(ErrorString(msg)), nil
	}
	if a.Authz != nil && !a.Authz.CanSee(authz.PrincipalFromCtx(ctx), events.FindingResource, item) {
		msg := fmt.Sprintf("finding %s not found", request.FindingId.String())
		return GetLandscapeFindingsFindingId404JSONResponse(ErrorString(msg)), nil
	}
	return GetLandscapeFindingsFindingId200JSONResponse(findingToView(a.Backend, a.BaseURL, item)), nil
}
