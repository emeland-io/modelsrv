package eventmgr

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

var _ events.EventManager = (*eventManager)(nil)

// Option configures an eventManager at construction time.
type Option func(*eventManager)

// WithHistoryLimit sets how many recent events QueryEvents can serve
// exactly. n must be positive or it is ignored (the default applies).
func WithHistoryLimit(n int) Option {
	return func(e *eventManager) {
		if n > 0 {
			e.historyLimit = n
		}
	}
}

type eventManager struct {
	mu             sync.RWMutex
	sequenceNumber uint64
	subscribers    []events.Subscriber
	sinkFactory    func() (events.EventSink, error)

	latestState  *latestStateStore
	historyTail  *historyRing
	historyLimit int
	modelSink    events.EventSink
}

func NewEventManager(opts ...Option) (events.EventManager, error) {
	e := &eventManager{
		sequenceNumber: 0,
		subscribers:    make([]events.Subscriber, 0),
		latestState:    newLatestStateStore(),
		historyLimit:   DefaultHistoryLimit,
	}
	for _, opt := range opts {
		opt(e)
	}
	e.historyTail = newHistoryRing(e.historyLimit)
	e.sinkFactory = func() (events.EventSink, error) {
		return e.getOrCreateModelSink(), nil
	}
	return e, nil
}

func (e *eventManager) getOrCreateModelSink() events.EventSink {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.modelSink == nil {
		e.modelSink = newRecordingSink(e)
	}
	return e.modelSink
}

func (e *eventManager) GetCurrentSequenceId(ctx context.Context) (uint64, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.sequenceNumber, nil
}

func (e *eventManager) IncrementSequenceId(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sequenceNumber++
	return nil
}

func (e *eventManager) SetSinkFactory(factory func() (events.EventSink, error)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sinkFactory = factory
}

func (e *eventManager) GetSink() (events.EventSink, error) {
	return e.sinkFactory()
}

func (e *eventManager) AddSubscriber(subURL string) error {
	e.mu.Lock()
	for _, sub := range e.subscribers {
		if sub.GetURL() == subURL {
			e.mu.Unlock()
			return nil
		}
	}
	newSub, err := NewSubscriber(subURL)
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.subscribers = append(e.subscribers, newSub)
	past := e.latestState.GetEvents()
	e.mu.Unlock()

	for i := range past {
		ev := past[i]
		evCopy := ev
		if err := newSub.Notify(context.Background(), &evCopy); err != nil {
			fmt.Printf("failed to notify subscriber %s during replay: %v\n", newSub.GetURL(), err) // TODO: handle errors in the middle of the replay
		}
	}
	return nil
}

func (e *eventManager) GetSubscribers() []events.Subscriber {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]events.Subscriber, len(e.subscribers))
	copy(out, e.subscribers)
	return out
}

func (e *eventManager) RemoveSubscriber(url string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, sub := range e.subscribers {
		if sub.GetURL() == url {
			e.subscribers = append(e.subscribers[:i], e.subscribers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("subscriber %s not found", url)
}

// tailResourceKey identifies a resource within a historyRing snapshot, using
// the same wire-format ResourceType string StoredEvent already carries (no
// parsing round-trip needed).
type tailResourceKey struct {
	resourceType string
	resourceId   uuid.UUID
}

func (e *eventManager) QueryEvents(ctx context.Context, q events.EventQuery) ([]events.StoredEvent, error) {
	e.mu.RLock()
	tail := e.historyTail.Snapshot()
	compactionSeq, compactionAt := e.historyTail.CompactionBoundary()

	var synthetic []events.StoredEvent
	if q.SinceSeq < compactionSeq {
		// Some events have aged out of the tail. Synthesize a Create for
		// every live resource that has no representation left in the tail
		// at all; resources with at least one real tail event don't need
		// one (upsert semantics mean the tail alone reconstructs them
		// correctly, so a synthetic entry would only be a redundant
		// duplicate).
		inTail := make(map[tailResourceKey]struct{}, len(tail))
		for _, ev := range tail {
			inTail[tailResourceKey{ev.ResourceType, ev.ResourceId}] = struct{}{}
		}
		for _, ev := range e.latestState.GetEvents() {
			key := tailResourceKey{ev.ResourceType.WireKind(), ev.ResourceId}
			if _, present := inTail[key]; present {
				continue
			}
			synthetic = append(synthetic, events.NewStoredEvent(
				compactionSeq, compactionAt, ev.ResourceType, events.CreateOperation, ev.ResourceId, ev.Objects,
			))
		}
	}
	e.mu.RUnlock()

	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}

	matches := func(ev events.StoredEvent) (events.StoredEvent, bool) {
		if ev.SequenceId <= q.SinceSeq {
			return events.StoredEvent{}, false
		}
		if q.Operation != nil && ev.Operation != q.Operation.WireOperation() {
			return events.StoredEvent{}, false
		}
		if q.ResourceType != nil && ev.ResourceType != q.ResourceType.WireKind() {
			return events.StoredEvent{}, false
		}
		if q.ResourceId != nil && ev.ResourceId != *q.ResourceId {
			return events.StoredEvent{}, false
		}
		entry := ev
		if !q.IncludePayload {
			entry.Objects = nil
		}
		return entry, true
	}

	var results []events.StoredEvent
	// The synthesized prefix is delivered in full, uncapped by limit: it
	// represents live resources with no remaining representation in the
	// tail at all, and letting a page boundary land inside it would let a
	// paginating client (SinceSeq = last delivered SequenceId, which for
	// every synthetic entry equals compactionSeq) skip the rest forever,
	// since the next call's SinceSeq would no longer be < compactionSeq.
	for _, ev := range synthetic {
		if entry, ok := matches(ev); ok {
			results = append(results, entry)
		}
	}
	for _, ev := range tail {
		if entry, ok := matches(ev); ok {
			results = append(results, entry)
			if len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}
