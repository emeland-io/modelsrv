package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/model/common"
)

func (c *ModelSrvClient) GetFindings() ([]oapi.FindingView, error) {
	resp, err := c.oapi_client.GetLandscapeFindingsWithResponse(context.TODO())
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

func (c *ModelSrvClient) GetFindingById(id uuid.UUID) (oapi.FindingView, error) {
	resp, err := c.oapi_client.GetLandscapeFindingsFindingIdWithResponse(context.TODO(), id)
	if err != nil {
		return oapi.FindingView{}, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return oapi.FindingView{}, common.ErrFindingNotFound
	}
	if resp.StatusCode() != http.StatusOK {
		return oapi.FindingView{}, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return oapi.FindingView{}, nil
	}
	return *resp.JSON200, nil
}
