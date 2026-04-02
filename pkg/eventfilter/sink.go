package eventfilter

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

type filteringSink struct {
	chain      Chain
	downstream events.EventSink
}

var _ events.EventSink = (*filteringSink)(nil)

// NewFilteringSink wraps downstream with a filtering layer.
//
// On each Receive call the event is passed through chain.Apply; every
// event returned by the chain is forwarded to downstream in order.
//
// With an empty chain (no filters registered), Apply returns the
// original event unchanged, making this a pure passthrough.
func NewFilteringSink(chain Chain, downstream events.EventSink) events.EventSink {
	return &filteringSink{
		chain:      chain,
		downstream: downstream,
	}
}

// Receive implements [events.EventSink].
func (s *filteringSink) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, object ...any) error {
	ev := events.Event{
		ResourceType: resType,
		Operation:    op,
		ResourceId:   resourceId,
		Objects:      object,
	}
	for _, out := range s.chain.Apply(ev) {
		if err := s.downstream.Receive(out.ResourceType, out.Operation, out.ResourceId, out.Objects...); err != nil {
			return err
		}
	}
	return nil
}
