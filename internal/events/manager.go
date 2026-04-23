package eventmgr

import (
	"context"
	"fmt"
	"sync"

	"go.emeland.io/modelsrv/pkg/events"
)

var _ events.EventManager = (*eventManager)(nil)

type eventManager struct {
	mu             sync.RWMutex
	sequenceNumber uint64
	subscribers    []events.Subscriber
	sinkFactory    func() (events.EventSink, error)

	masterList *events.ListSink
	modelSink  events.EventSink
}

func NewEventManager() (events.EventManager, error) {
	e := &eventManager{
		sequenceNumber: 0,
		subscribers:    make([]events.Subscriber, 0),
		masterList:     events.NewListSink(),
	}
	e.sinkFactory = func() (events.EventSink, error) {
		return e.getOrCreateModelSink(), nil
	}
	return e, nil
}

func (e *eventManager) getOrCreateModelSink() events.EventSink {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.modelSink == nil {
		e.modelSink = newRecordingSink(e)
	}
	return e.modelSink
}

func (e *eventManager) GetCurrentSequenceId(ctx context.Context) (uint64, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.sequenceNumber, nil
}

func (e *eventManager) IncrementSequenceId(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sequenceNumber++
	return nil
}

func (e *eventManager) SetSinkFactory(factory func() (events.EventSink, error)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sinkFactory = factory
}

func (e *eventManager) GetSink() (events.EventSink, error) {
	return e.sinkFactory()
}

func (e *eventManager) AddSubscriber(subURL string) error {
	e.mu.Lock()
	for _, sub := range e.subscribers {
		if sub.GetURL() == subURL {
			e.mu.Unlock()
			return nil
		}
	}
	newSub, err := NewSubscriber(subURL)
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.subscribers = append(e.subscribers, newSub)
	past := e.masterList.GetEvents()
	e.mu.Unlock()

	for i := range past {
		ev := past[i]
		evCopy := ev
		if err := newSub.Notify(context.Background(), &evCopy); err != nil {
			fmt.Printf("failed to notify subscriber %s during replay: %v\n", newSub.GetURL(), err) // TODO: handle errors in the middle of the replay
		}
	}
	return nil
}

func (e *eventManager) GetSubscribers() []events.Subscriber {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]events.Subscriber, len(e.subscribers))
	copy(out, e.subscribers)
	return out
}

func (e *eventManager) RemoveSubscriber(url string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, sub := range e.subscribers {
		if sub.GetURL() == url {
			e.subscribers = append(e.subscribers[:i], e.subscribers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("subscriber %s not found", url)
}
