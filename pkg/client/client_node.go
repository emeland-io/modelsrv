package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/model/common"
)

func (c *ModelSrvClient) GetNodes() ([]oapi.NodeSummaryView, error) {
	resp, err := c.oapi_client.GetLandscapeNodesWithResponse(context.TODO())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return nil, nil
	}
	return *resp.JSON200, nil
}

func (c *ModelSrvClient) GetNodeById(id uuid.UUID) (oapi.NodeView, error) {
	resp, err := c.oapi_client.GetLandscapeNodesNodeIdWithResponse(context.TODO(), id)
	if err != nil {
		return oapi.NodeView{}, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return oapi.NodeView{}, common.ErrNodeNotFound
	}
	if resp.StatusCode() != http.StatusOK {
		return oapi.NodeView{}, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return oapi.NodeView{}, nil
	}
	return *resp.JSON200, nil
}
