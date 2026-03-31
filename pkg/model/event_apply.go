package model

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	mdlevent "go.emeland.io/modelsrv/pkg/model/event"
)

var _ mdlevent.EventApplier = (*modelData)(nil)

// Apply implements [EventApplier].
func (m *modelData) Apply(ev events.Event) error {
	switch ev.Operation {
	case events.DeleteOperation:
		return m.applyReplicationDelete(ev.ResourceType, ev.ResourceId)
	case events.CreateOperation, events.UpdateOperation:
		if len(ev.Objects) == 0 {
			return fmt.Errorf("missing resource object for %s %s", ev.ResourceType, ev.Operation)
		}
		return m.applyReplicationUpsert(ev.ResourceType, ev.Objects[0])
	default:
		return fmt.Errorf("unsupported operation %v", ev.Operation)
	}
}

func (m *modelData) applyReplicationDelete(rt events.ResourceType, id uuid.UUID) error {
	h, ok := m.handlers[rt]
	if !ok {
		return fmt.Errorf("unsupported resource type for delete: %s", rt)
	}
	err := h.delete(m, id)
	if err != nil && !h.notFound(err) {
		return err
	}
	return nil
}

func (m *modelData) applyReplicationUpsert(rt events.ResourceType, obj any) error {
	h, ok := m.handlers[rt]
	if !ok {
		return fmt.Errorf("unsupported resource type for upsert: %s", rt)
	}
	return h.upsert(m, obj)
}
