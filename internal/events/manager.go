package eventmgr

import (
	"context"
	"fmt"
	"sync"
	"time"

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

// WithSelfURL sets this node's own base API URL. It is stamped as the origin
// of every locally-produced event, so receivers can tell which node produced
// a change. An empty value is ignored.
func WithSelfURL(url string) Option {
	return func(e *eventManager) {
		if url != "" {
			e.selfURL = url
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

	// selfURL is this node's own base API URL, stamped as the origin of
	// locally-produced events.
	selfURL string
	// replicationStamps holds the preserved origin + timestamp of an event
	// currently being forwarded, keyed by the goroutine running ApplyReplicated.
	// Keying by goroutine is what makes stamping correct under concurrency: the
	// whole replicated apply (model mutation -> sink.Receive -> eventStamp) runs
	// synchronously on one goroutine, so only that goroutine's records read the
	// preserved stamp. A concurrent local mutation on another goroutine finds no
	// entry and is correctly stamped with selfURL + now. (The EventSink.Receive
	// signature carries no context, so the stamp cannot be threaded as a
	// parameter without changing every sink implementation; goroutine scoping is
	// the contained alternative.)
	replicationMu     sync.Mutex
	replicationStamps map[uint64]replicationStamp

	logger *zap.SugaredLogger
}

// replicationStamp is the origin + timestamp preserved while forwarding a
// replicated event.
type replicationStamp struct {
	originURL string
	timestamp time.Time
}

func NewEventManager(opts ...Option) (events.EventManager, error) {
	e := &eventManager{
		sequenceNumber:    0,
		notifiers:         make([]*notifier, 0),
		latestState:       newLatestStateStore(),
		historyLimit:      DefaultHistoryLimit,
		replicationStamps: make(map[uint64]replicationStamp),
		logger:            zap.NewNop().Sugar(),
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

// eventStamp returns the origin URL and timestamp to record for an event
// about to flow through the recording sink. If the calling goroutine is inside
// an ApplyReplicated (forwarding an upstream event), it returns the preserved
// upstream origin + timestamp; otherwise it returns this node's own URL and the
// current time. Keying on the goroutine ensures a concurrent local mutation on
// another goroutine is never stamped with a remote origin.
func (e *eventManager) eventStamp() (string, time.Time) {
	gid := goroutineID()
	e.replicationMu.Lock()
	rep, ok := e.replicationStamps[gid]
	e.replicationMu.Unlock()
	if ok {
		return rep.originURL, rep.timestamp
	}
	return e.selfURL, time.Now()
}

// ApplyReplicated runs fn (which applies an upstream event to the local model)
// with origin + timestamp preserved, so the resulting recorded event keeps the
// originating node's identity and time rather than being re-stamped as locally
// produced.
//
// The stamp is scoped to the current goroutine. POST /events/push is served on
// a per-request goroutine and the whole apply (model mutation -> recording sink)
// runs synchronously on it, so concurrent pushes and concurrent local mutations
// each read their own correct stamp without interfering. Nested calls on the
// same goroutine restore the previous stamp on return.
func (e *eventManager) ApplyReplicated(originURL string, timestamp time.Time, fn func() error) error {
	gid := goroutineID()

	e.replicationMu.Lock()
	prev, hadPrev := e.replicationStamps[gid]
	e.replicationStamps[gid] = replicationStamp{originURL: originURL, timestamp: timestamp}
	e.replicationMu.Unlock()

	defer func() {
		e.replicationMu.Lock()
		if hadPrev {
			e.replicationStamps[gid] = prev
		} else {
			delete(e.replicationStamps, gid)
		}
		e.replicationMu.Unlock()
	}()

	return fn()
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
			e.logger.Debugw("subscriber already registered",
				"url", subURL,
				"subscriberId", n.sub.GetId().String(),
			)
			return nil
		}
	}
	e.logger.Debugw("registering subscriber", "url", subURL)
	newSub, err := NewSubscriber(subURL, e.logger)
	if err != nil {
		e.mu.Unlock()
		e.logger.Errorw("failed to create subscriber client", "url", subURL, "error", err)
		return err
	}
	n := newNotifier(newSub, e.stateSnapshot, e.logger)
	e.notifiers = append(e.notifiers, n)
	past := e.latestState.GetEvents()
	e.mu.Unlock()

	e.logger.Infow("registered subscriber; replaying current state",
		"url", subURL,
		"subscriberId", newSub.GetId().String(),
		"resources", len(past),
		"subscribers", e.subscriberCount(),
	)

	// Replay synchronously, before the delivery goroutine starts: events
	// recorded in the meantime wait in the queue and go out afterwards, so
	// the subscriber never sees a live event ahead of the state it builds on.
	for i := range past {
		e.logger.Debugw("replaying resource to new subscriber",
			"url", subURL,
			"index", i,
			"of", len(past),
			"kind", past[i].ResourceType.WireKind(),
			"operation", past[i].Operation.WireOperation(),
			"resourceId", past[i].ResourceId.String(),
		)
		if !n.deliver(past[i]) {
			e.logger.Warnw("initial replay failed; scheduling resync",
				"url", subURL,
				"delivered", i,
				"resources", len(past),
			)
			n.resync.Store(true)
			break
		}
	}
	n.start()
	e.logger.Debugw("subscriber delivery loop started",
		"url", subURL,
		"subscriberId", newSub.GetId().String(),
	)
	return nil
}

func (e *eventManager) subscriberCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.notifiers)
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
			remaining := len(e.notifiers)
			e.mu.Unlock()
			e.logger.Infow("unregistered subscriber",
				"url", url,
				"subscriberId", n.sub.GetId().String(),
				"subscribers", remaining,
			)
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
				compactionSeq, compactionAt, ev.OriginURL, ev.ResourceType, events.CreateOperation, ev.ResourceId, ev.Objects,
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
