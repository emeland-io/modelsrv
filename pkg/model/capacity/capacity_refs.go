package capacity

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/context"
)

// GetContext returns the embedded context when present.
func (o *capacityData) GetContext() (context.Context, error) {
	if o.ContextRef == nil {
		return nil, nil
	}
	return o.ContextRef.ResolvedContext(), nil
}

// GetContextId returns the context id when set.
func (o *capacityData) GetContextId() uuid.UUID {
	if o.ContextRef == nil {
		return uuid.Nil
	}
	return o.ContextRef.EffectiveParentContextID()
}

// SetContextRef sets the low-level context reference and emits when registered.
func (o *capacityData) SetContextRef(val *context.ContextRef) {
	o.ContextRef = val

	if o.isRegistered {
		o.sink.Receive(events.CapacityResource, events.UpdateOperation, o.CapacityId, o)
	}
}

// SetContextById records only the context id (resolved object may be nil).
func (o *capacityData) SetContextById(contextId uuid.UUID) {
	if contextId == uuid.Nil {
		o.SetContextRef(nil)
		return
	}
	o.SetContextRef(&context.ContextRef{ContextId: contextId})
}
