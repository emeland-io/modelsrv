package oapi

import (
	"context"
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

	if requestSequenceId == currSequenceId {
		return GetEventsQuerySequenceId200Response{}, nil
	}
	if requestSequenceId < currSequenceId {
		return GetEventsQuerySequenceId308Response{}, nil
	}
	return GetEventsQuerySequenceId404JSONResponse(""), nil
}

// PostEventsRegister implements StrictServerInterface.
func (a *ApiServer) PostEventsRegister(ctx context.Context, request PostEventsRegisterRequestObject) (PostEventsRegisterResponseObject, error) {
	if err := a.Events.AddSubscriber(request.Body.CallbackUrl); err != nil {
		return nil, err
	}
	return PostEventsRegister201Response{}, nil
}

// PostEventsUnregister implements StrictServerInterface.
func (a *ApiServer) PostEventsUnregister(ctx context.Context, request PostEventsUnregisterRequestObject) (PostEventsUnregisterResponseObject, error) {
	if err := a.Events.RemoveSubscriber(request.Body.CallbackUrl); err != nil {
		return PostEventsUnregister404JSONResponse(err.Error()), nil
	}
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
	return GetEventsSubscribers200JSONResponse(out), nil
}

// PostEventsPush receives replicated events from an upstream server and applies them to the local model.
// The recording sink forwards applied changes to any registered downstream subscribers.
func (a *ApiServer) PostEventsPush(ctx context.Context, request PostEventsPushRequestObject) (PostEventsPushResponseObject, error) {
	_ = ctx
	if request.Body == nil {
		return nil, fmt.Errorf("missing event body")
	}
	ev, err := ReplicationEventFromWire(a.Backend, request.Body)
	if err != nil {
		return nil, fmt.Errorf("replication decode: %w", err)
	}
	if err := a.Backend.Apply(ev); err != nil {
		return nil, fmt.Errorf("replication apply: %w", err)
	}
	return PostEventsPush200Response{}, nil
}
