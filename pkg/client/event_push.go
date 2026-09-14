package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/events"
)

// PushError is a non-200 response from POST /events/push.
type PushError struct {
	StatusCode int
	Body       string
}

func (e *PushError) Error() string {
	if e == nil {
		return "POST /events/push: empty error"
	}
	if strings.TrimSpace(e.Body) == "" {
		return fmt.Sprintf("POST /events/push: expected 200, got %d", e.StatusCode)
	}
	return fmt.Sprintf("POST /events/push: expected 200, got %d: %s", e.StatusCode, e.Body)
}

// Permanent reports whether retrying this push cannot succeed: a replication
// decode or apply failure on the subscriber is a property of this event, not
// of transient connectivity.
func (e *PushError) Permanent() bool {
	if e == nil {
		return false
	}
	if e.StatusCode != http.StatusInternalServerError {
		return false
	}
	return strings.Contains(e.Body, "replication decode:") || strings.Contains(e.Body, "replication apply:")
}

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
		if errors.Is(err, oapi.ErrSkipReplication) {
			log.Printf("WARNING: skipping unreplicable event: %v", err)
			return nil
		}
		return err
	}
	resp, err := c.oapi_client.PostEventsPushWithResponse(ctx, body)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return &PushError{
			StatusCode: resp.StatusCode(),
			Body:       strings.TrimSpace(string(resp.Body)),
		}
	}
	return nil
}
