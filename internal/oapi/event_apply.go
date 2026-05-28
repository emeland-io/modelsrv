package oapi

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

// ReplicationEventFromWire converts an OpenAPI push body into a domain [events.Event] for [model.EventApplier.Apply].
func ReplicationEventFromWire(m model.Model, ev *Event) (events.Event, error) {
	if m == nil {
		return events.Event{}, fmt.Errorf("nil model")
	}
	if ev == nil {
		return events.Event{}, fmt.Errorf("nil event")
	}

	kind := strings.TrimSpace(ev.Kind)
	op := wireString(ev.Operation)
	rt := events.ParseWireKind(kind)
	if rt == events.UnknownResourceType {
		return events.Event{}, fmt.Errorf("unknown event kind %q", kind)
	}
	wop := events.ParseWireOperation(op)
	if wop == events.UnknownOperation {
		return events.Event{}, fmt.Errorf("unknown event operation %q", op)
	}

	switch wop {
	case events.DeleteOperation:
		if ev.ResourceId == nil {
			return events.Event{}, fmt.Errorf("delete event missing resourceId")
		}
		id := uuid.UUID(*ev.ResourceId)
		return events.Event{
			ResourceType: rt,
			Operation:    wop,
			ResourceId:   id,
		}, nil
	case events.CreateOperation, events.UpdateOperation:
		if ev.Resource == nil {
			return events.Event{}, fmt.Errorf("missing resource payload for %s %s", kind, op)
		}
		id, obj, err := decodeReplicationResourceFromMap(m, rt, ev.Resource)
		if err != nil {
			return events.Event{}, err
		}
		return events.Event{
			ResourceType: rt,
			Operation:    wop,
			ResourceId:   id,
			Objects:      []any{obj},
		}, nil
	default:
		return events.Event{}, fmt.Errorf("unsupported operation %q", op)
	}
}

func wireString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	default:
		return fmt.Sprint(v)
	}
}
