package oapi

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

// GetEventsQuerySequenceId implements StrictServerInterface.
func (a *ApiServer) GetEventsQuerySequenceId(ctx context.Context, request GetEventsQuerySequenceIdRequestObject) (GetEventsQuerySequenceIdResponseObject, error) {
	requestSequenceId, err := strconv.ParseUint(request.SequenceId, 10, 64)
	if err != nil {
		return nil, err
	}

	currSequenceId, err := a.Events.GetCurrentSequenceId(ctx)
	if err != nil {
		return nil, err
	}

	status := 404
	if requestSequenceId == currSequenceId {
		status = 200
	} else if requestSequenceId < currSequenceId {
		status = 308
	}
	a.logger().Debugw("GET /events/query",
		"requested", request.SequenceId,
		"current", currSequenceId,
		"status", status,
	)
	if status == 200 {
		return GetEventsQuerySequenceId200Response{}, nil
	}
	if status == 308 {
		return GetEventsQuerySequenceId308Response{}, nil
	}
	return GetEventsQuerySequenceId404JSONResponse(""), nil
}

// PostEventsRegister implements StrictServerInterface.
func (a *ApiServer) PostEventsRegister(ctx context.Context, request PostEventsRegisterRequestObject) (PostEventsRegisterResponseObject, error) {
	callback := ""
	if request.Body != nil {
		callback = request.Body.CallbackUrl
	}
	a.logger().Debugw("POST /events/register", "callbackUrl", callback)
	if err := a.Events.AddSubscriber(callback); err != nil {
		a.logger().Errorw("POST /events/register failed", "callbackUrl", callback, "error", err)
		return nil, err
	}
	a.logger().Debugw("POST /events/register accepted", "callbackUrl", callback, "status", 201)
	return PostEventsRegister201Response{}, nil
}

// PostEventsUnregister implements StrictServerInterface.
func (a *ApiServer) PostEventsUnregister(ctx context.Context, request PostEventsUnregisterRequestObject) (PostEventsUnregisterResponseObject, error) {
	callback := ""
	if request.Body != nil {
		callback = request.Body.CallbackUrl
	}
	a.logger().Debugw("POST /events/unregister", "callbackUrl", callback)
	if err := a.Events.RemoveSubscriber(callback); err != nil {
		a.logger().Warnw("POST /events/unregister unknown subscriber", "callbackUrl", callback, "status", 404, "error", err)
		return PostEventsUnregister404JSONResponse(err.Error()), nil
	}
	a.logger().Debugw("POST /events/unregister accepted", "callbackUrl", callback, "status", 200)
	return PostEventsUnregister200Response{}, nil
}

// GetEventsSubscribers implements StrictServerInterface.
func (a *ApiServer) GetEventsSubscribers(ctx context.Context, request GetEventsSubscribersRequestObject) (GetEventsSubscribersResponseObject, error) {
	_ = ctx
	_ = request
	subs := a.Events.GetSubscribers()
	out := make([]string, 0, len(subs))
	for _, s := range subs {
		out = append(out, s.GetURL())
	}
	a.logger().Debugw("GET /events/subscribers", "count", len(out), "urls", out)
	return GetEventsSubscribers200JSONResponse(out), nil
}

// PostEventsPush receives replicated events from an upstream server and applies them to the local model.
// The recording sink forwards applied changes to any registered downstream subscribers.
func (a *ApiServer) PostEventsPush(ctx context.Context, request PostEventsPushRequestObject) (PostEventsPushResponseObject, error) {
	_ = ctx
	log := a.logger()
	if request.Body == nil {
		log.Warnw("POST /events/push missing body")
		return nil, fmt.Errorf("missing event body")
	}
	log.Debugw("POST /events/push",
		"kind", request.Body.Kind,
		"operation", request.Body.Operation,
		"resourceId", request.Body.ResourceId,
	)
	ev, err := ReplicationEventFromWire(a.Backend, request.Body)
	if err != nil {
		if errors.Is(err, ErrSkipReplication) {
			log.Warnw("skipping unreplicable event", "kind", request.Body.Kind, "error", err)
			return PostEventsPush200Response{}, nil
		}
		log.Errorw("POST /events/push decode failed", "kind", request.Body.Kind, "error", err)
		return nil, fmt.Errorf("replication decode: %w", err)
	}
	log.Debugw("POST /events/push decoded",
		"kind", ev.ResourceType.WireKind(),
		"operation", ev.Operation.WireOperation(),
		"resourceId", ev.ResourceId.String(),
		"objects", len(ev.Objects),
	)
	if err := a.Backend.Apply(ev); err != nil {
		log.Errorw("POST /events/push apply failed",
			"kind", ev.ResourceType.WireKind(),
			"resourceId", ev.ResourceId.String(),
			"error", err,
		)
		return nil, fmt.Errorf("replication apply: %w", err)
	}
	log.Debugw("POST /events/push applied",
		"kind", ev.ResourceType.WireKind(),
		"operation", ev.Operation.WireOperation(),
		"resourceId", ev.ResourceId.String(),
	)
	return PostEventsPush200Response{}, nil
}
