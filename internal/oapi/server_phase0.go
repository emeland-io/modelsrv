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
	"fmt"

	"github.com/google/uuid"
)

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
	var parentContextId uuid.UUID
	parent, err := context.GetParent()
	if err != nil || parent == nil {
		parentContextId = uuid.Nil
	} else {
		parentContextId = parent.GetContextId()
	}

	respBody := Context{
		ContextId:   contextId,
		DisplayName: displayName,
		Parent:      &parentContextId,
		Annotations: cloneAnnotations2(context.GetAnnotations()),
	}

	return GetLandscapeContextsContextId200JSONResponse(respBody), nil
}

func (a *ApiServer) GetLandscapeContextTypes(ctx context.Context, request GetLandscapeContextTypesRequestObject) (GetLandscapeContextTypesResponseObject, error) {
	contextTypesArr, err := a.Backend.GetContextTypes()

	if err != nil {
		return nil, err
	}

	respBody := []InstanceListItem{}

	for _, contextType := range contextTypesArr {
		contextTypeId := contextType.GetContextTypeId()
		displayName := contextType.GetDisplayName()
		reference := fmt.Sprintf("%s/landscape/contextTypes/%s", a.BaseURL, contextTypeId.String())

		item := InstanceListItem{
			InstanceId:  &contextTypeId,
			DisplayName: &displayName,
			Reference:   &reference,
		}
		respBody = append(respBody, item)
	}

	return GetLandscapeContextTypes200JSONResponse(respBody), nil

}

// GetLandscapeContextTypesContextTypeId implements [StrictServerInterface].
func (a *ApiServer) GetLandscapeContextTypesContextTypeId(ctx context.Context, request GetLandscapeContextTypesContextTypeIdRequestObject) (GetLandscapeContextTypesContextTypeIdResponseObject, error) {
	contextType := a.Backend.GetContextTypeById(request.ContextTypeId)
	if contextType == nil {
		errorstr := fmt.Sprintf("context type %s not found", request.ContextTypeId.String())
		return GetLandscapeContextTypesContextTypeId404JSONResponse(errorstr), nil
	}

	displayName := contextType.GetDisplayName()
	contextTypeId := contextType.GetContextTypeId()

	respBody := ContextType{
		ContextTypeId: contextTypeId,
		DisplayName:   displayName,
		Annotations:   cloneAnnotations2(contextType.GetAnnotations()),
	}

	return GetLandscapeContextTypesContextTypeId200JSONResponse(respBody), nil

}

// GetLandscapeNodeTypes implements [StrictServerInterface].
func (a *ApiServer) GetLandscapeNodeTypes(ctx context.Context, request GetLandscapeNodeTypesRequestObject) (GetLandscapeNodeTypesResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeNodeTypesNodeTypeId implements [StrictServerInterface].
func (a *ApiServer) GetLandscapeNodeTypesNodeTypeId(ctx context.Context, request GetLandscapeNodeTypesNodeTypeIdRequestObject) (GetLandscapeNodeTypesNodeTypeIdResponseObject, error) {
	panic("unimplemented")
}

// GetLandscapeNodes implements [StrictServerInterface].
func (a *ApiServer) GetLandscapeNodes(ctx context.Context, request GetLandscapeNodesRequestObject) (GetLandscapeNodesResponseObject, error) {
	nodesArr, err := a.Backend.GetNodes()

	if err != nil {
		return nil, err
	}

	respBody := []InstanceListItem{}

	for _, node := range nodesArr {
		nodeId := node.GetNodeId()
		displayName := node.GetDisplayName()
		reference := fmt.Sprintf("%s/landscape/nodes/%s", a.BaseURL, nodeId.String())

		item := InstanceListItem{
			InstanceId:  &nodeId,
			DisplayName: &displayName,
			Reference:   &reference,
		}
		respBody = append(respBody, item)
	}

	return GetLandscapeNodes200JSONResponse(respBody), nil
}

// GetLandscapeNodesNodeId implements [StrictServerInterface].
func (a *ApiServer) GetLandscapeNodesNodeId(ctx context.Context, request GetLandscapeNodesNodeIdRequestObject) (GetLandscapeNodesNodeIdResponseObject, error) {
	node := a.Backend.GetNodeById(request.NodeId)
	if node == nil {
		errorstr := fmt.Sprintf("node %s not found", request.NodeId.String())
		return GetLandscapeNodesNodeId404JSONResponse(errorstr), nil
	}

	displayName := node.GetDisplayName()
	nodeId := node.GetNodeId()

	respBody := Node{
		NodeId:      nodeId,
		DisplayName: displayName,
		Annotations: cloneAnnotations2(node.GetAnnotations()),
	}

	return GetLandscapeNodesNodeId200JSONResponse(respBody), nil
}
