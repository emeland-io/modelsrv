package capability

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

// GetCapability returns the embedded [Capability] when present (otherwise nil).
// Resolve by id via [Model.GetCapabilityById].
func (o *dependencyData) GetCapability() (Capability, error) {
	if o.CapabilityRef == nil {
		return nil, nil
	}
	return o.CapabilityRef.ResolvedCapability(), nil
}

// GetCapabilityId returns the capability id when set.
func (o *dependencyData) GetCapabilityId() uuid.UUID {
	if o.CapabilityRef == nil {
		return uuid.Nil
	}
	return o.CapabilityRef.EffectiveCapabilityID()
}

// SetCapabilityRef sets the low-level capability reference and emits when registered.
func (o *dependencyData) SetCapabilityRef(val *CapabilityRef) {
	o.CapabilityRef = val

	if o.isRegistered {
		_ = o.sink.Receive(events.DependencyResource, events.UpdateOperation, o.DependencyId, o)
	}
}

// SetCapabilityById records only the capability id (resolved object may be nil).
func (o *dependencyData) SetCapabilityById(capabilityId uuid.UUID) {
	if capabilityId == uuid.Nil {
		o.SetCapabilityRef(nil)
		return
	}
	o.SetCapabilityRef(&CapabilityRef{CapabilityId: capabilityId})
}
