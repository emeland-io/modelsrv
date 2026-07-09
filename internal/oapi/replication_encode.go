package oapi

import (
	"encoding/json"
	"fmt"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.emeland.io/modelsrv/pkg/events"
)

// PushWireEventFromDomain builds an OpenAPI [Event] for POST /events/push from a domain [events.Event].
//
// For create/update, the resource body is produced via domain→OpenAPI DTO encoders so annotations,
// scalar refs, and other wire fields match what subscribers decode.
func PushWireEventFromDomain(ev *events.Event) (Event, error) {
	if ev == nil {
		return Event{}, fmt.Errorf("nil event")
	}
	if ev.ResourceType == events.AnnotationsResource {
		return Event{}, fmt.Errorf("annotations are not replicated as standalone events; they are carried on resource payloads")
	}
	kind := ev.ResourceType.WireKind()
	op := ev.Operation.WireOperation()
	if ev.Operation == events.DeleteOperation {
		rid := openapi_types.UUID(ev.ResourceId)
		return Event{
			Kind:       kind,
			Operation:  op,
			ResourceId: &rid,
		}, nil
	}
	obj, ok := firstEventObject(ev)
	if !ok {
		return Event{}, fmt.Errorf("create/update event missing resource object for kind %s", kind)
	}
	m, err := encodeReplicationResourceToWireMap(ev.ResourceType, obj)
	if err != nil {
		return Event{}, err
	}
	return Event{
		Kind:      kind,
		Operation: op,
		Resource:  &m,
	}, nil
}

func firstEventObject(ev *events.Event) (any, bool) {
	if len(ev.Objects) == 0 {
		return nil, false
	}
	return ev.Objects[0], true
}

func jsonMap(v any) (map[string]interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}
