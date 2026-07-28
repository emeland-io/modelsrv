package eventmgr

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

// recordingSink records to the manager's latest-state store and history
// ring, bumps the sequence number, and notifies subscribers.
type recordingSink struct {
	mgr *eventManager
}

var _ events.EventSink = (*recordingSink)(nil)

func newRecordingSink(mgr *eventManager) *recordingSink {
	return &recordingSink{mgr: mgr}
}

func (r *recordingSink) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, objects ...any) error {
	ev := events.Event{
		ResourceType: resType,
		Operation:    op,
		ResourceId:   resourceId,
		Objects:      objects,
	}

	r.mgr.mu.Lock()
	r.mgr.latestState.Receive(resType, op, resourceId, objects...)
	r.mgr.sequenceNumber++
	r.mgr.historyTail.Add(events.NewStoredEvent(
		r.mgr.sequenceNumber, time.Now(), resType, op, resourceId, objects,
	))
	subs := make([]events.Subscriber, len(r.mgr.subscribers))
	copy(subs, r.mgr.subscribers)
	r.mgr.mu.Unlock()

	for _, sub := range subs {
		s := sub
		evCopy := ev
		go func() {
			err := s.Notify(context.Background(), &evCopy)
			if err != nil {
				fmt.Printf("failed to notify subscriber %s: %v\n", s.GetURL(), err)
			}
		}()
	}
	return nil
}
