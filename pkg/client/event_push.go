package client

import (
	"context"
	"errors"
	"fmt"
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
	c.logger().Debugw("POST /events/register", "callbackUrl", callbackURL)
	resp, err := c.oapi_client.PostEventsRegisterWithResponse(context.TODO(), oapi.PostEventsRegisterJSONRequestBody{
		CallbackUrl: callbackURL,
	})
	if err != nil {
		c.logger().Errorw("POST /events/register failed", "callbackUrl", callbackURL, "error", err)
		return err
	}
	c.logger().Debugw("POST /events/register response", "callbackUrl", callbackURL, "status", resp.StatusCode())
	if resp.StatusCode() != http.StatusCreated {
		c.logger().Warnw("POST /events/register rejected", "callbackUrl", callbackURL, "status", resp.StatusCode())
		return fmt.Errorf("expected HTTP 201 but received %d", resp.StatusCode())
	}
	return nil
}

// OriginCount asks GET /events/origin-count for how many live resources the
// peer holds with the given OriginURL.
func (c *ModelSrvClient) OriginCount(ctx context.Context, originURL string) (int, error) {
	c.logger().Debugw("GET /events/origin-count", "originUrl", originURL)
	resp, err := c.oapi_client.GetEventsOriginCountWithResponse(ctx, &oapi.GetEventsOriginCountParams{
		OriginUrl: originURL,
	})
	if err != nil {
		c.logger().Errorw("GET /events/origin-count transport error", "originUrl", originURL, "error", err)
		return 0, err
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		c.logger().Warnw("GET /events/origin-count rejected",
			"originUrl", originURL,
			"status", resp.StatusCode(),
			"body", strings.TrimSpace(string(resp.Body)),
		)
		return 0, fmt.Errorf("GET /events/origin-count: expected 200, got %d", resp.StatusCode())
	}
	c.logger().Debugw("GET /events/origin-count ok",
		"originUrl", originURL,
		"count", resp.JSON200.Count,
	)
	return resp.JSON200.Count, nil
}

// PostEvent sends a domain event to POST /events/push (used for server-to-server replication).
func (c *ModelSrvClient) PostEvent(ctx context.Context, ev *events.Event) error {
	if ev == nil {
		return fmt.Errorf("nil event")
	}
	log := c.logger()
	log.Debugw("POST /events/push",
		"kind", ev.ResourceType.WireKind(),
		"operation", ev.Operation.WireOperation(),
		"resourceId", ev.ResourceId.String(),
	)
	body, err := oapi.PushWireEventFromDomain(ev)
	if err != nil {
		if errors.Is(err, oapi.ErrSkipReplication) {
			log.Warnw("skipping unreplicable event",
				"kind", ev.ResourceType.WireKind(),
				"operation", ev.Operation.WireOperation(),
				"resourceId", ev.ResourceId.String(),
				"error", err,
			)
			return nil
		}
		log.Errorw("POST /events/push encode failed",
			"kind", ev.ResourceType.WireKind(),
			"operation", ev.Operation.WireOperation(),
			"resourceId", ev.ResourceId.String(),
			"error", err,
		)
		return err
	}
	log.Debugw("POST /events/push wire",
		"kind", body.Kind,
		"operation", body.Operation,
		"resourceId", body.ResourceId,
	)
	resp, err := c.oapi_client.PostEventsPushWithResponse(ctx, body)
	if err != nil {
		log.Errorw("POST /events/push transport error",
			"kind", body.Kind,
			"resourceId", ev.ResourceId.String(),
			"error", err,
		)
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		log.Warnw("POST /events/push rejected",
			"kind", body.Kind,
			"resourceId", ev.ResourceId.String(),
			"status", resp.StatusCode(),
			"body", strings.TrimSpace(string(resp.Body)),
		)
		return &PushError{
			StatusCode: resp.StatusCode(),
			Body:       strings.TrimSpace(string(resp.Body)),
		}
	}
	log.Debugw("POST /events/push accepted",
		"kind", body.Kind,
		"resourceId", ev.ResourceId.String(),
		"status", resp.StatusCode(),
	)
	return nil
}
