package oapi

import (
	"encoding/json"
	"fmt"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.emeland.io/modelsrv/pkg/events"
)

// PushWireEventFromDomain builds an OpenAPI [Event] for POST /events/push from a domain [events.Event].
//
// For create/update, the resource body is produced by encoding/json on the first domain object.
// That matches how subscribers decode: JSON is unmarshalled into OpenAPI structs (tags such as
// "systemId" accept common casings from the encoder).
//
// Callers should avoid deeply cyclic object graphs (for example a [system.System] whose parent
// embeds a fully-resolved upstream system); in those cases prefer refs that carry IDs only, or
// build a dedicated DTO before placing it in [events.Event.Objects].
func PushWireEventFromDomain(ev *events.Event) (Event, error) {
	if ev == nil {
		return Event{}, fmt.Errorf("nil event")
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
	m, err := jsonMap(obj)
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
