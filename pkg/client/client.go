//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=config-model.yaml ../../api/openapi/EmergingEnterpriseLandscape-0.1.0-oapi-3.0.3.yaml

package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"gitlab.com/emeland/modelsrv/internal/oapi"
	"gitlab.com/emeland/modelsrv/pkg/model"
)

type ModelSrvClient struct {
	oapi_client *oapi.ClientWithResponses
	hc          *http.Client
}

func NewModelSrvClient(url string) (*ModelSrvClient, error) {
	ret := &ModelSrvClient{
		hc: &http.Client{},
	}

	oapiClient, err := oapi.NewClientWithResponses(url, oapi.WithHTTPClient(ret.hc))
	if err != nil {
		return nil, err
	}

	ret.oapi_client = oapiClient

	return ret, nil
}

func (c *ModelSrvClient) GetTest() error {
	resp, err := c.oapi_client.GetTestWithResponse(context.TODO())
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return nil
}

func (c *ModelSrvClient) GetContexts() (*oapi.InstanceList, error) {
	resp, err := c.oapi_client.GetLandscapeContextsWithResponse(context.TODO())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return (*oapi.InstanceList)(resp.JSON200), nil
}

func (c *ModelSrvClient) GetContextById(contextId uuid.UUID) (*oapi.Context, error) {
	resp, err := c.oapi_client.GetLandscapeContextsContextIdWithResponse(context.TODO(), contextId)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, model.ContextNotFoundError
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return (*oapi.Context)(resp.JSON200), nil
}

func (c *ModelSrvClient) GetSystems() (*oapi.InstanceList, error) {
	resp, err := c.oapi_client.GetLandscapeSystemsWithResponse(context.TODO())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return (*oapi.InstanceList)(resp.JSON200), nil
}

func (c *ModelSrvClient) GetSystemById(systemId uuid.UUID) (*oapi.System, error) {
	resp, err := c.oapi_client.GetLandscapeSystemsSystemIdWithResponse(context.TODO(), systemId)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, model.SystemNotFoundError
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return (*oapi.System)(resp.JSON200), nil
}

func (c *ModelSrvClient) GetSystemInstances() (*oapi.InstanceList, error) {
	resp, err := c.oapi_client.GetLandscapeSystemsWithResponse(context.TODO())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return (*oapi.InstanceList)(resp.JSON200), nil
}

func (c *ModelSrvClient) GetSystemInstanceById(systemInstanceId uuid.UUID) (*oapi.SystemInstance, error) {
	resp, err := c.oapi_client.GetLandscapeSystemInstancesSystemInstanceIdWithResponse(context.TODO(), systemInstanceId)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, model.SystemInstanceNotFoundError
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return (*oapi.SystemInstance)(resp.JSON200), nil
}

func (c *ModelSrvClient) GetAPIs() (*oapi.InstanceList, error) {
	resp, err := c.oapi_client.GetLandscapeApisWithResponse(context.TODO())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return (*oapi.InstanceList)(resp.JSON200), nil
}

func (c *ModelSrvClient) GetAPIById(apiId uuid.UUID) (*oapi.API, error) {
	resp, err := c.oapi_client.GetLandscapeApisApiIdWithResponse(context.TODO(), apiId)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, model.ApiNotFoundError
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return (*oapi.API)(resp.JSON200), nil
}

func (c *ModelSrvClient) GetApiInstances() (*oapi.InstanceList, error) {
	resp, err := c.oapi_client.GetLandscapeApiInstancesWithResponse(context.TODO())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return (*oapi.InstanceList)(resp.JSON200), nil
}

func (c *ModelSrvClient) GetApiInstanceById(apiInstanceId uuid.UUID) (*oapi.ApiInstance, error) {
	resp, err := c.oapi_client.GetLandscapeApiInstancesApiInstanceIdWithResponse(context.TODO(), apiInstanceId)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, model.ApiInstanceNotFoundError
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return (*oapi.ApiInstance)(resp.JSON200), nil
}

func (c *ModelSrvClient) GetComponents() (*oapi.InstanceList, error) {
	resp, err := c.oapi_client.GetLandscapeComponentsWithResponse(context.TODO())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return (*oapi.InstanceList)(resp.JSON200), nil
}

func (c *ModelSrvClient) GetComponentById(componentId uuid.UUID) (*oapi.Component, error) {
	resp, err := c.oapi_client.GetLandscapeComponentsComponentIdWithResponse(context.TODO(), componentId)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, model.ComponentNotFoundError
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return (*oapi.Component)(resp.JSON200), nil
}

func (c *ModelSrvClient) GetComponentInstances() (*oapi.InstanceList, error) {
	resp, err := c.oapi_client.GetLandscapeComponentInstancesWithResponse(context.TODO())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return (*oapi.InstanceList)(resp.JSON200), nil
}

func (c *ModelSrvClient) GetComponentInstanceById(componentId uuid.UUID) (*oapi.ComponentInstance, error) {
	resp, err := c.oapi_client.GetLandscapeComponentInstancesComponentInstanceIdWithResponse(context.TODO(), componentId)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, model.ComponentInstanceNotFoundError
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}

	return (*oapi.ComponentInstance)(resp.JSON200), nil
}
