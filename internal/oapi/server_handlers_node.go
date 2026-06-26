package oapi

import (
	"context"
	"fmt"

	"go.emeland.io/modelsrv/pkg/authz"
	"go.emeland.io/modelsrv/pkg/events"
)

// GetLandscapeNodes implements [StrictServerInterface].
func (a *ApiServer) GetLandscapeNodes(ctx context.Context, request GetLandscapeNodesRequestObject) (GetLandscapeNodesResponseObject, error) {
	items, err := a.Backend.GetNodes()
	if err != nil {
		return nil, err
	}
	if a.Authz != nil {
		principal := authz.PrincipalFromCtx(ctx)
		items = authz.FilterVisible(a.Authz, principal, events.NodeResource, items)
	}
	out := make([]NodeSummaryView, 0, len(items))
	for _, item := range items {
		out = append(out, nodeSummaryToView(a.Backend, a.BaseURL, item))
	}
	return GetLandscapeNodes200JSONResponse(out), nil
}

// GetLandscapeNodesNodeId implements [StrictServerInterface].
func (a *ApiServer) GetLandscapeNodesNodeId(ctx context.Context, request GetLandscapeNodesNodeIdRequestObject) (GetLandscapeNodesNodeIdResponseObject, error) {
	item := a.Backend.GetNodeById(request.NodeId)
	if item == nil {
		msg := fmt.Sprintf("node %s not found", request.NodeId.String())
		return GetLandscapeNodesNodeId404JSONResponse(msg), nil
	}
	if a.Authz != nil && !a.Authz.CanSee(authz.PrincipalFromCtx(ctx), events.NodeResource, item) {
		msg := fmt.Sprintf("node %s not found", request.NodeId.String())
		return GetLandscapeNodesNodeId404JSONResponse(msg), nil
	}
	return GetLandscapeNodesNodeId200JSONResponse(nodeToView(a.Backend, a.BaseURL, item)), nil
}
