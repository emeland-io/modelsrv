package events

import (
	"context"
	"fmt"

	"go.emeland.io/modelsrv/pkg/events"
)

var _ events.EventManager = (*eventManager)(nil)

type eventManager struct {
	sequenceNumber uint64
	subscribers    []events.Subscriber
	sinkFactory    func() (events.EventSink, error)
}

func NewEventManager() (events.EventManager, error) {
	retval := &eventManager{
		sequenceNumber: 0,
		sinkFactory:    func() (events.EventSink, error) { return events.NewListSink(), nil },
	}
	return retval, nil

}

// GetCurrentSequenceId implements EventManager.
func (e *eventManager) GetCurrentSequenceId(ctx context.Context) (uint64, error) {
	return e.sequenceNumber, nil
}

// IncrementSequenceId implements EventManager.
func (e *eventManager) IncrementSequenceId(ctx context.Context) error {
	e.sequenceNumber++
	return nil
}

// GetSink returns a new EventSink created by the sink factory set with [SetSinkFactory].
// If no factory has been set, a default [ListSink] is returned.
//
// SetSinkFactory implements [EventManager].
func (e *eventManager) SetSinkFactory(factory func() (events.EventSink, error)) {
	e.sinkFactory = factory
}

// GetSink implements [EventManager].
func (e *eventManager) GetSink() (events.EventSink, error) {
	return e.sinkFactory()
}

// AddSubscriber implements [EventManager].
// adding the same subscriber URL again will result in only one entry in the subscriber list.
func (e *eventManager) AddSubscriber(subUrl string) error {
	for _, sub := range e.subscribers {
		if sub.GetURL() == subUrl {
			// already exists
			return nil
		}
	}
	e.subscribers = append(e.subscribers, NewSubscriber(subUrl))
	return nil
}

// GetSubscribers implements [EventManager].
func (e *eventManager) GetSubscribers() []events.Subscriber {
	return e.subscribers
}

// RemoveSubscriber implements [EventManager].
// TODO: this function requires O(n) time. If the subscriber list becomes long, consider using a map for O(1) removal.
func (e *eventManager) RemoveSubscriber(url string) error {
	for i, sub := range e.subscribers {
		if sub.GetURL() == url {
			// remove subscriber
			e.subscribers = append(e.subscribers[:i], e.subscribers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("subscriber %s not found", url)
}
