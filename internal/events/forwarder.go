package eventmgr

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

// EnumeratedListSink is a simple in-memory sink used in tests.
type EnumeratedListSink struct {
	events []events.Event
}

var _ events.EventSink = (*EnumeratedListSink)(nil)

func NewEnumeratedListSink() *EnumeratedListSink {
	return &EnumeratedListSink{events: make([]events.Event, 0)}
}

func (l *EnumeratedListSink) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, objects ...any) error {
	l.events = append(l.events, events.Event{
		ResourceType: resType,
		Operation:    op,
		ResourceId:   resourceId,
		Objects:      objects,
	})
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
	return l.events[index], nil
}

type eventForwarder struct {
	subscriberURL string
	sink          *EnumeratedListSink
}

func NewEventForwarder(subscriberURL string, sink *EnumeratedListSink) *eventForwarder {
	return &eventForwarder{subscriberURL: subscriberURL, sink: sink}
}
