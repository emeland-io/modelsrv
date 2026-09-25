package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/internal/oapi"
	"go.uber.org/zap"
)

type ModelSrvClient struct {
	oapi_client *oapi.ClientWithResponses
	hc          *http.Client
	log         *zap.SugaredLogger
}

// SetLogger sets the logger used for replication HTTP calls. A nil logger
// is ignored; the client keeps a no-op logger so debug lines stay quiet
// unless the process log level includes them.
func (c *ModelSrvClient) SetLogger(log *zap.SugaredLogger) {
	if c == nil || log == nil {
		return
	}
	c.log = log
}

func (c *ModelSrvClient) logger() *zap.SugaredLogger {
	if c == nil || c.log == nil {
		return zap.NewNop().Sugar()
	}
	return c.log
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

// Ensure uuid import is used when client_gen.go is empty during development.
var _ = uuid.Nil
