package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

func (c *ModelSrvClient) GetFindingTypes() ([]finding.FindingType, error) {
	resp, err := c.oapi_client.GetLandscapeFindingTypesWithResponse(context.TODO())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return nil, nil
	}
	out := make([]finding.FindingType, 0, len(*resp.JSON200))
	for i := range *resp.JSON200 {
		ft, err := oapi.FindingTypeFromDto(nil, &(*resp.JSON200)[i])
		if err != nil {
			return nil, err
		}
		out = append(out, ft)
	}
	return out, nil
}

func (c *ModelSrvClient) GetFindingTypeById(id uuid.UUID) (finding.FindingType, error) {
	resp, err := c.oapi_client.GetLandscapeFindingTypesFindingTypeIdWithResponse(context.TODO(), id)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, common.ErrFindingTypeNotFound
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}
	return oapi.FindingTypeFromDto(nil, resp.JSON200)
}
