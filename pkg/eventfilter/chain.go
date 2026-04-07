package eventfilter

import (
	"sync"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

type entry struct {
	id FilterID
	fn FilterFunc
}

type chainData struct {
	model   model.Model
	mu      sync.RWMutex
	filters []entry
}

var _ Chain = (*chainData)(nil)

// NewChain creates a new filter Chain associated with m.
// The model is passed to every FilterFunc on each Apply call so that
// filter functions can inspect the current landscape state.
func NewChain(m model.Model) Chain {
	return &chainData{
		model:   m,
		filters: make([]entry, 0),
	}
}

// Register implements [Chain].
func (c *chainData) Register(fn FilterFunc) FilterID {
	id := FilterID(uuid.New())
	c.mu.Lock()
	c.filters = append(c.filters, entry{id: id, fn: fn})
	c.mu.Unlock()
	return id
}

// Unregister implements [Chain].
func (c *chainData) Unregister(id FilterID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, e := range c.filters {
		if e.id == id {
			c.filters = append(c.filters[:i], c.filters[i+1:]...)
			return
		}
	}
}

// SetModel implements [Chain].
func (c *chainData) SetModel(m model.Model) {
	c.mu.Lock()
	c.model = m
	c.mu.Unlock()
}

// Apply implements [Chain].
//
// It snapshots the current filter slice and model reference, then flat-maps
// each FilterFunc over the event batch: every filter receives each event in
// the current batch and its outputs replace the batch for the next filter.
func (c *chainData) Apply(ev events.Event) []events.Event {
	c.mu.RLock()
	snapshot := make([]entry, len(c.filters))
	copy(snapshot, c.filters)
	m := c.model
	c.mu.RUnlock()

	if len(snapshot) == 0 {
		return []events.Event{ev}
	}

	current := []events.Event{ev}
	for _, e := range snapshot {
		var next []events.Event
		for _, inEv := range current {
			next = append(next, e.fn(m, inEv)...)
		}
		current = next
	}
	return current
}
