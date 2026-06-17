package eventfilter

import (
	"log"
	"sync"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	mdlfilterrule "go.emeland.io/modelsrv/pkg/model/filterrule"
)

type entry struct {
	id          FilterID
	displayName string
	description string
	fn          FilterFunc
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

// RegisterFilter implements [Chain].
func (c *chainData) RegisterFilter(f Filter) FilterID {
	id := FilterID(uuid.New())
	c.mu.Lock()
	c.filters = append(c.filters, entry{
		id:          id,
		displayName: f.DisplayName,
		description: f.Description,
		fn:          f.Fn,
	})
	c.mu.Unlock()
	c.syncFilterRuleToModel(id, f.DisplayName, f.Description)
	return id
}

// Register implements [Chain].
func (c *chainData) Register(fn FilterFunc) FilterID {
	return c.RegisterFilter(Filter{Fn: fn})
}

// Unregister implements [Chain].
func (c *chainData) Unregister(id FilterID) {
	if !c.removeFilter(id) {
		return // unknown ID — no model work
	}
	c.removeFilterRuleFromModel(id)
}
func (c *chainData) removeFilter(id FilterID) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, e := range c.filters {
		if e.id == id {
			c.filters = append(c.filters[:i], c.filters[i+1:]...)
			return true
		}
	}
	return false
}

// SetModel implements [Chain].
func (c *chainData) SetModel(m model.Model) {
	snapshot := c.swapModel(m)
	for _, e := range snapshot {
		c.syncFilterRuleToModel(e.id, e.displayName, e.description)
	}
}

// swapModel swaps the model reference and returns a snapshot of the current filter slice.
func (c *chainData) swapModel(m model.Model) []entry {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.model = m

	return append([]entry(nil), c.filters...)
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

func (c *chainData) syncFilterRuleToModel(id FilterID, displayName, description string) {
	c.mu.RLock()
	m := c.model
	c.mu.RUnlock()
	if m == nil {
		return
	}

	ruleID := uuid.UUID(id)
	if m.GetFilterRuleById(ruleID) != nil {
		return
	}

	fr := mdlfilterrule.NewFilterRule(ruleID)
	fr.SetDisplayName(displayName)
	fr.SetDescription(description)
	if err := m.AddFilterRule(fr); err != nil {
		log.Printf("eventfilter: AddFilterRule id=%s: %v", ruleID, err)
	}
}

func (c *chainData) removeFilterRuleFromModel(id FilterID) {
	c.mu.RLock()
	m := c.model
	c.mu.RUnlock()
	if m == nil {
		return
	}

	ruleID := uuid.UUID(id)
	if err := m.DeleteFilterRuleById(ruleID); err != nil {
		log.Printf("eventfilter: DeleteFilterRuleById id=%s: %v", ruleID, err)
	}
}
