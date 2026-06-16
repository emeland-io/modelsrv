package eventfilter

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

// FilterFunc is the function type every filter step must satisfy.
// It receives the current model state and one incoming event,
// and returns 0 or more outgoing events.
//
// A FilterFunc may:
//   - pass the event through unchanged: return []events.Event{ev}
//   - suppress the event:              return nil (or an empty slice)
//   - expand into multiple events:     return []events.Event{ev, finding, ...}
type FilterFunc func(m model.Model, ev events.Event) []events.Event

// FilterID uniquely identifies a registered [FilterFunc] in a [Chain].
type FilterID uuid.UUID

// Filter bundles a filter's identity with its implementation. Filter packages
// should expose constructors that return Filter so DisplayName and Description
// stay colocated with the filter logic.
type Filter struct {
	DisplayName string
	Description string
	Fn          FilterFunc
}

// Chain manages an ordered list of [FilterFunc]s and executes them
// against incoming events in registration order.
type Chain interface {
	// RegisterFilter appends f to the chain and returns a FilterID that can
	// be passed to Unregister to remove it later. When the chain is associated
	// with a model, a corresponding FilterRule resource is created.
	RegisterFilter(f Filter) FilterID

	// Register appends fn to the chain without metadata. Prefer [RegisterFilter]
	// so filter rules are discoverable in the landscape model.
	Register(fn FilterFunc) FilterID

	// Unregister removes the filter identified by id from the chain.
	// It is a no-op if id was never registered or was already removed.
	Unregister(id FilterID)

	// Apply runs ev through every registered FilterFunc in order,
	// flat-mapping the outputs so that one incoming event may produce
	// 0, 1, or many outgoing events.
	Apply(ev events.Event) []events.Event

	// SetModel replaces the model reference passed to every FilterFunc.
	// It is safe to call concurrently with Apply and is intended for
	// breaking the construction cycle: create the Chain with nil, wrap the
	// sink, create the Model, then call SetModel.
	SetModel(m model.Model)
}
