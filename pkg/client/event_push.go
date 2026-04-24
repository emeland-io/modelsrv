package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/events"
)

// Register registers this server's API base URL as a callback for upstream event pushes.
func (c *ModelSrvClient) Register(callbackURL string) error {
	resp, err := c.oapi_client.PostEventsRegisterWithResponse(context.TODO(), oapi.PostEventsRegisterJSONRequestBody{
		CallbackUrl: callbackURL,
	})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("expected HTTP 201 but received %d", resp.StatusCode())
	}
	return nil
}

// PostEvent sends a domain event to POST /events/push (used for server-to-server replication).
func (c *ModelSrvClient) PostEvent(ctx context.Context, ev *events.Event) error {
	if ev == nil {
		return fmt.Errorf("nil event")
	}
	body, err := oapi.PushWireEventFromDomain(ev)
	if err != nil {
		return err
	}
	resp, err := c.oapi_client.PostEventsPushWithResponse(ctx, body)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		msg := strings.TrimSpace(string(resp.Body))
		if msg == "" {
			return fmt.Errorf("POST /events/push: expected 200, got %d", resp.StatusCode())
		}
		return fmt.Errorf("POST /events/push: expected 200, got %d: %s", resp.StatusCode(), msg)
	}
	return nil
}
