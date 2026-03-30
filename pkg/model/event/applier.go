package event

import (
	"go.emeland.io/modelsrv/pkg/events"
)

// EventApplier applies replicated [events.Event] records to local state using the same Add*/Delete* paths
// as normal mutations so the recording sink records once.
type EventApplier interface {
	Apply(ev events.Event) error
}
