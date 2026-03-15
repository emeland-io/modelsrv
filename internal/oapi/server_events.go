package oapi

import (
	"context"
	"strconv"
)

// GetEventsQuerySequenceId implements StrictServerInterface.
func (a *ApiServer) GetEventsQuerySequenceId(ctx context.Context, request GetEventsQuerySequenceIdRequestObject) (GetEventsQuerySequenceIdResponseObject, error) {
	var requestSequenceId uint64

	// parse sequenceId
	requestSequenceId, err := strconv.ParseUint(request.SequenceId, 10, 64)
	if err != nil {
		return nil, err
	}

	currSequenceId, err := a.Events.GetCurrentSequenceId(ctx)
	if err != nil {
		return nil, err
	}

	if requestSequenceId == currSequenceId {
		// no new events
		resp := GetEventsQuerySequenceId200Response{}
		return GetEventsQuerySequenceId200Response(resp), nil
	} else if requestSequenceId < currSequenceId {
		// there are new events
		resp := GetEventsQuerySequenceId308Response{}
		return GetEventsQuerySequenceId308Response(resp), nil
	} else {
		// client is ahead of server?
		resp := ""
		return GetEventsQuerySequenceId404JSONResponse(resp), nil
	}
}

// PostEventsRegister implements StrictServerInterface.
func (a *ApiServer) PostEventsRegister(ctx context.Context, request PostEventsRegisterRequestObject) (PostEventsRegisterResponseObject, error) {
	a.Events.AddSubscriber(request.Body.CallbackUrl)

	return PostEventsRegister201Response{}, nil
}

// PostEventsUnregister implements [StrictServerInterface].
func (a *ApiServer) PostEventsUnregister(ctx context.Context, request PostEventsUnregisterRequestObject) (PostEventsUnregisterResponseObject, error) {
	error := a.Events.RemoveSubscriber(request.Body.CallbackUrl)
	if error != nil {
		return PostEventsUnregister404JSONResponse(error.Error()), nil
	}
	return PostEventsUnregister200Response{}, nil
}

// GetEventsSubscribers implements [StrictServerInterface].
func (a *ApiServer) GetEventsSubscribers(ctx context.Context, request GetEventsSubscribersRequestObject) (GetEventsSubscribersResponseObject, error) {
	panic("unimplemented")
}

// PostEventsPush implements [StrictServerInterface].
func (a *ApiServer) PostEventsPush(ctx context.Context, request PostEventsPushRequestObject) (PostEventsPushResponseObject, error) {
	panic("unimplemented")
}
