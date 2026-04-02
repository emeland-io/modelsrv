// Package backend provides a [Backend] that wires together a [model.Model],
// an [eventfilter.Chain], and an [events.EventManager] into a single cohesive
// unit.
//
// Construction order:
//  1. Create the EventManager and obtain its recording sink.
//  2. Create a Chain with a nil model placeholder.
//  3. Wrap the recording sink in a [eventfilter.FilteringSink] backed by the chain.
//  4. Create the Model using the FilteringSink — every model mutation now flows
//     through the chain before reaching the recording sink.
//  5. Back-fill the chain with the now-constructed model via SetModel.
//
// This resolves the construction cycle: Chain needs Model, Model needs sink,
// sink needs Chain.
package backend

import (
	eventmgr "go.emeland.io/modelsrv/internal/events"
	"go.emeland.io/modelsrv/pkg/eventfilter"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

// Backend bundles the model, the filter chain, and the event manager.
// External consumers (sensors, the web endpoint) retrieve only the part they need.
type Backend interface {
	// GetModel returns the landscape model.
	// Sensors call Add*/Delete* on it to mutate state; those mutations flow
	// through the FilteringSink and the Chain before reaching the EventManager.
	GetModel() model.Model

	// GetChain returns the filter chain.
	// Sensors register their FilterFuncs here to intercept and enrich events.
	GetChain() eventfilter.Chain

	// GetEventManager returns the event manager.
	// Used by the web endpoint to serve subscribers and manage sequence IDs.
	GetEventManager() events.EventManager
}

type backendData struct {
	model    model.Model
	chain    eventfilter.Chain
	eventMgr events.EventManager
}

// New constructs a fully wired Backend.
//
// The returned Backend has an empty filter chain; callers register their
// [eventfilter.FilterFunc]s via [Backend.GetChain].Register before feeding
// events into the model.
func New() (Backend, error) {
	eventMgr, err := eventmgr.NewEventManager()
	if err != nil {
		return nil, err
	}

	recordingSink, err := eventMgr.GetSink()
	if err != nil {
		return nil, err
	}

	// Create the chain with a nil model; the model is back-filled below after
	// the Model is constructed (SetModel breaks the construction cycle).
	chain := eventfilter.NewChain(nil)
	filteredSink := eventfilter.NewFilteringSink(chain, recordingSink)

	m, err := model.NewModel(filteredSink)
	if err != nil {
		return nil, err
	}

	chain.SetModel(m)

	return &backendData{
		model:    m,
		chain:    chain,
		eventMgr: eventMgr,
	}, nil
}

// GetModel implements [Backend].
func (b *backendData) GetModel() model.Model {
	return b.model
}

// GetChain implements [Backend].
func (b *backendData) GetChain() eventfilter.Chain {
	return b.chain
}

// GetEventManager implements [Backend].
func (b *backendData) GetEventManager() events.EventManager {
	return b.eventMgr
}
