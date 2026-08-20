package eventmgr

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.uber.org/zap"
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

// WithLogger sets the logger used for subscriber notify failures. A nil
// logger is ignored (tests keep the no-op default).
func WithLogger(log *zap.SugaredLogger) Option {
	return func(e *eventManager) {
		if log != nil {
			e.logger = log
		}
	}
}

type eventManager struct {
	mu             sync.RWMutex
	sequenceNumber uint64
	notifiers      []*notifier
	sinkFactory    func() (events.EventSink, error)

	latestState  *latestStateStore
	historyTail  *historyRing
	historyLimit int
	modelSink    events.EventSink

	logger *zap.SugaredLogger
}

func NewEventManager(opts ...Option) (events.EventManager, error) {
	e := &eventManager{
		sequenceNumber: 0,
		notifiers:      make([]*notifier, 0),
		latestState:    newLatestStateStore(),
		historyLimit:   DefaultHistoryLimit,
		logger:         zap.NewNop().Sugar(),
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
	for _, n := range e.notifiers {
		if n.sub.GetURL() == subURL {
			e.mu.Unlock()
			return nil
		}
	}
	newSub, err := NewSubscriber(subURL)
	if err != nil {
		e.mu.Unlock()
		return err
	}
	n := newNotifier(newSub, e.stateSnapshot, e.logger)
	e.notifiers = append(e.notifiers, n)
	past := e.latestState.GetEvents()
	e.mu.Unlock()

	// Replay synchronously, before the delivery goroutine starts: events
	// recorded in the meantime wait in the queue and go out afterwards, so
	// the subscriber never sees a live event ahead of the state it builds on.
	for i := range past {
		if !n.deliver(past[i]) {
			n.resync.Store(true)
			break
		}
	}
	n.start()
	return nil
}

func (e *eventManager) GetSubscribers() []events.Subscriber {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]events.Subscriber, 0, len(e.notifiers))
	for _, n := range e.notifiers {
		out = append(out, n.sub)
	}
	return out
}

func (e *eventManager) RemoveSubscriber(url string) error {
	e.mu.Lock()
	for i, n := range e.notifiers {
		if n.sub.GetURL() == url {
			e.notifiers = append(e.notifiers[:i], e.notifiers[i+1:]...)
			e.mu.Unlock()
			n.stopDelivery()
			return nil
		}
	}
	e.mu.Unlock()
	return fmt.Errorf("subscriber %s not found", url)
}

// stateSnapshot returns one event per live resource, for a notifier that
// needs to rebuild a subscriber that fell behind.
func (e *eventManager) stateSnapshot() []events.Event {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.latestState.GetEvents()
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
