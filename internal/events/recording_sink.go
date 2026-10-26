package eventmgr

import (
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
	// Determine origin + timestamp. For a locally-produced change these are
	// this node's own URL and the current time. For a change applied while
	// forwarding a replicated event (see eventManager.ApplyReplicated), the
	// incoming origin + timestamp are preserved unchanged so provenance
	// survives every hop.
	originURL, timestamp := r.mgr.eventStamp()

	ev := events.Event{
		ResourceType: resType,
		Operation:    op,
		ResourceId:   resourceId,
		Objects:      objects,
		OriginURL:    originURL,
		Timestamp:    timestamp,
	}

	r.mgr.mu.Lock()
	r.mgr.latestState.Receive(ev)
	r.mgr.sequenceNumber++
	r.mgr.historyTail.Add(events.NewStoredEvent(
		r.mgr.sequenceNumber, timestamp, originURL, resType, op, resourceId, objects,
	))
	seq := r.mgr.sequenceNumber
	notifiers := make([]*notifier, len(r.mgr.notifiers))
	copy(notifiers, r.mgr.notifiers)
	r.mgr.mu.Unlock()

	r.mgr.logger.Debugw("recorded event; fanning out to subscribers",
		"sequence", seq,
		"kind", resType.WireKind(),
		"operation", op.WireOperation(),
		"resourceId", resourceId.String(),
		"origin", originURL,
		"objects", len(objects),
		"subscribers", len(notifiers),
	)

	for _, n := range notifiers {
		n.enqueue(ev)
	}
	return nil
}
