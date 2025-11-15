//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=config-client.yaml ../../api/openapi/EmergingEnterpriseLandscape-0.1.0-oapi-3.0.3.yaml

package oapi

import (
	"net/http"
)

type ModelSrvClient struct {
	oapi_client *ClientWithResponses
	hc          *http.Client
}

func NewModelSrvClient(url string) (*ModelSrvClient, error) {
	ret := &ModelSrvClient{
		hc: &http.Client{},
	}

	oapiClient, err := NewClientWithResponses(url, WithHTTPClient(ret.hc))
	if err != nil {
		return nil, err
	}

	ret.oapi_client = oapiClient

	return ret, nil
}
