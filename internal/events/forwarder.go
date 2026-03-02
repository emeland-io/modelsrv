package events

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/client"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

type eventDTO struct {
	resourceType events.ResourceType
	operation    events.Operation
	resourceId   uuid.UUID
	objectJson   []string
}

type EnumeratedEventSink interface {
	events.EventSink
	GetEventCount() int
	GetEventByIndex(index int) (events.Event, error)
}

type eventForwarder struct {
	front         int
	sink          EnumeratedEventSink
	subscriberUrl string
}

func NewEventForwarder(subscriberUrl string, sink EnumeratedEventSink) *eventForwarder {
	return &eventForwarder{
		subscriberUrl: subscriberUrl,
		front:         0, // index of the next event to be forwarded in the sink's event list
		sink:          sink,
	}
}

type EnumeratedListSink struct {
	events []events.Event
}

var _ EnumeratedEventSink = (*EnumeratedListSink)(nil)

func NewEnumeratedListSink() *EnumeratedListSink {
	return &EnumeratedListSink{

		events: make([]events.Event, 0),
	}
}

// Receive implements [EventSink].
func (l *EnumeratedListSink) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, objects ...any) error {
	currEvent := events.Event{
		ResourceType: resType,
		Operation:    op,
		ResourceId:   resourceId,
		Objects:      objects,
	}
	l.events = append(l.events, currEvent)

	return nil
}

func (l *EnumeratedListSink) GetEvents() []events.Event {
	return l.events
}

func (l *EnumeratedListSink) GetEventCount() int {
	return len(l.events)
}

func (l *EnumeratedListSink) GetEventByIndex(index int) (events.Event, error) {
	if index < 0 || index >= len(l.events) {
		return events.Event{}, fmt.Errorf("index %d out of bounds", index)
	}
	e := l.events[index]
	return e, nil
}

func (e *eventForwarder) ForwardEvent(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, objects ...any) error {
	objJsons := make([]string, 0, len(objects))
	for _, obj := range objects {
		switch o := obj.(type) {
		case string:
			objJsons = append(objJsons, o)
		case model.Context:
			jsonStr, err := json.Marshal(convertContextToDTO(obj.(model.Context)))
			if err != nil {
				return fmt.Errorf("failed to marshal context: %w", err)
			}
			objJsons = append(objJsons, string(jsonStr))
		default:
			return fmt.Errorf("unknown type %v", obj)
			// objJsons = append(objJsons, "json_string_placeholder")
		}
	}

	/* TODO:  implement actual forwarding logic here, e.g. by sending the eventDTO to the subscriber URL via HTTP POST request or using a message queue.
	event := eventDTO{
		resourceType: resType,
		operation:    op,
		resourceId:   resourceId,
		objectJson:   objJsons,
	}
	*/
	return nil
}

func convertContextToDTO(context model.Context) client.Context {
	description := context.GetDescription()

	retval := client.Context{
		ContextId:   context.GetContextId(),
		DisplayName: context.GetDisplayName(),
		Description: &description,
		Annotations: convertAnnotationsToDTO(context.GetAnnotations()),
	}

	return retval
}

func convertAnnotationsToDTO(modelAnnons model.Annotations) *[]client.Annotation {

	retval := make([]client.Annotation, 0)
	for key := range modelAnnons.GetKeys() {
		retval = append(retval, client.Annotation{Key: key, Value: modelAnnons.GetValue(key)})
	}
	return &retval
}
