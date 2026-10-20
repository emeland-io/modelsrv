package eventmgr

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

type resourceKey struct {
	resType events.ResourceType
	id      uuid.UUID
}

// latestStateStore keeps only the most recent event of each live resource,
// so heap usage scales with the number of resources not yet deleted rather
// than the total number of events ever observed. Every call site holds
// eventManager.mu, so this type has no locking of its own.
type latestStateStore struct {
	order []resourceKey
	state map[resourceKey]events.Event
}

func newLatestStateStore() *latestStateStore {
	return &latestStateStore{state: make(map[resourceKey]events.Event)}
}

func (s *latestStateStore) Receive(ev events.Event) {
	key := resourceKey{resType: ev.ResourceType, id: ev.ResourceId}

	if ev.Operation == events.DeleteOperation {
		if _, ok := s.state[key]; ok {
			delete(s.state, key)
			s.removeFromOrder(key)
		}
		return
	}

	if _, exists := s.state[key]; !exists {
		s.order = append(s.order, key)
	}
	s.state[key] = ev
}

func (s *latestStateStore) removeFromOrder(key resourceKey) {
	for i, k := range s.order {
		if k == key {
			s.order = append(s.order[:i], s.order[i+1:]...)
			return
		}
	}
}

// GetEvents returns one event per live resource, oldest-creation-first,
// always reported as a Create: to a party seeing this snapshot as their
// first knowledge of the resource (a new subscriber, or a history query
// reaching past the retention window), it is first appearing now.
func (s *latestStateStore) GetEvents() []events.Event {
	out := make([]events.Event, 0, len(s.order))
	for _, key := range s.order {
		ev := s.state[key]
		// Report as Create for a party seeing this as first knowledge of the
		// resource, but keep the stored OriginURL + Timestamp: a replayed
		// resource must still name the node that originally produced it and
		// when, not the replaying node or replay time.
		ev.Operation = events.CreateOperation
		out = append(out, ev)
	}
	return out
}
