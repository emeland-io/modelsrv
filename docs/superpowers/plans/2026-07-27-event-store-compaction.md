# Event Store Compaction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bound the heap used by `internal/events.eventManager`'s two in-memory event logs — one now scales with live (undeleted) resource count, the other with a configurable retention limit — instead of both growing forever with total events observed.

**Architecture:** Replace `masterList` (`*events.ListSink`, unbounded) with `latestStateStore`, a compacted map keyed by `(ResourceType, ResourceId)` holding only each live resource's latest event. Replace `storedEvents` (`[]events.StoredEvent`, unbounded) with `historyRing`, a fixed-capacity ring buffer. `QueryEvents` synthesizes missing history (for resources evicted from the ring but still live) directly from `latestStateStore` at query time, so no third data structure or eviction-time bookkeeping is needed beyond a `compactionSeq`/`compactionAt` boundary marker on the ring.

**Tech Stack:** Go, ginkgo/gomega (BDD tests in this package) and stdlib `testing` + testify (table-style tests in this package) — both already used side-by-side in `internal/events`.

## Global Constraints

- `pkg/events.ListSink` (public type) must not change — it's a full-fidelity test double used across ~30 test files and is unrelated to these two logs.
- All existing zero-arg call sites of `eventmgr.NewEventManager()` and `backend.New()` (~15+ across tests and `cmd/modelsrv/server.go`) must keep compiling and behaving identically (default retention: 1000 events).
- No change to `EventQuery`/`StoredEvent` wire shapes, `SequenceId` semantics, or `GetCurrentSequenceId`/`IncrementSequenceId`.
- When total events ever observed is at or below the retention limit, `QueryEvents` output must be byte-for-byte identical to today's unbounded implementation (no synthesized entries at all).
- Spec: `docs/superpowers/specs/2026-07-27-event-store-compaction-design.md`.

---

### Task 1: `latestStateStore` — compacted per-resource state

**Files:**
- Create: `internal/events/latest_state.go`
- Test: `internal/events/latest_state_test.go`

**Interfaces:**
- Produces: `type resourceKey struct { resType events.ResourceType; id uuid.UUID }`, `type latestStateStore struct{...}`, `newLatestStateStore() *latestStateStore`, `(*latestStateStore) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, objects ...any)`, `(*latestStateStore) GetEvents() []events.Event`.

- [ ] **Step 1: Write the failing tests**

Create `internal/events/latest_state_test.go`:

```go
package eventmgr

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
)

func TestLatestStateStore_UpsertReplacesPriorState(t *testing.T) {
	s := newLatestStateStore()
	id := uuid.New()
	s.Receive(events.SystemResource, events.CreateOperation, id, "v1")
	s.Receive(events.SystemResource, events.UpdateOperation, id, "v2")

	got := s.GetEvents()
	require.Len(t, got, 1)
	assert.Equal(t, events.CreateOperation, got[0].Operation)
	assert.Equal(t, id, got[0].ResourceId)
	assert.Equal(t, []any{"v2"}, got[0].Objects)
}

func TestLatestStateStore_DeleteRemovesEntry(t *testing.T) {
	s := newLatestStateStore()
	id := uuid.New()
	s.Receive(events.SystemResource, events.CreateOperation, id, "v1")
	s.Receive(events.SystemResource, events.DeleteOperation, id)

	assert.Empty(t, s.GetEvents())
}

func TestLatestStateStore_DeleteUnknownIsNoop(t *testing.T) {
	s := newLatestStateStore()
	s.Receive(events.SystemResource, events.DeleteOperation, uuid.New())

	assert.Empty(t, s.GetEvents())
}

func TestLatestStateStore_OrderPreservedAndRecreateAppendsAtEnd(t *testing.T) {
	s := newLatestStateStore()
	idA := uuid.New()
	idB := uuid.New()
	s.Receive(events.SystemResource, events.CreateOperation, idA, "a1")
	s.Receive(events.SystemResource, events.CreateOperation, idB, "b1")
	s.Receive(events.SystemResource, events.DeleteOperation, idA)
	s.Receive(events.SystemResource, events.CreateOperation, idA, "a2")

	got := s.GetEvents()
	require.Len(t, got, 2)
	assert.Equal(t, idB, got[0].ResourceId)
	assert.Equal(t, idA, got[1].ResourceId)
	assert.Equal(t, []any{"a2"}, got[1].Objects)
}

func TestLatestStateStore_KeyIncludesResourceType(t *testing.T) {
	s := newLatestStateStore()
	id := uuid.New()
	s.Receive(events.SystemResource, events.CreateOperation, id, "sys")
	s.Receive(events.NodeResource, events.CreateOperation, id, "node")

	assert.Len(t, s.GetEvents(), 2)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/events/... -run TestLatestStateStore -v`
Expected: FAIL with `undefined: newLatestStateStore` (the type doesn't exist yet)

- [ ] **Step 3: Write the implementation**

Create `internal/events/latest_state.go`:

```go
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

func (s *latestStateStore) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, objects ...any) {
	key := resourceKey{resType: resType, id: resourceId}

	if op == events.DeleteOperation {
		if _, ok := s.state[key]; ok {
			delete(s.state, key)
			s.removeFromOrder(key)
		}
		return
	}

	if _, exists := s.state[key]; !exists {
		s.order = append(s.order, key)
	}
	s.state[key] = events.Event{
		ResourceType: resType,
		Operation:    op,
		ResourceId:   resourceId,
		Objects:      objects,
	}
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
		ev.Operation = events.CreateOperation
		out = append(out, ev)
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/events/... -run TestLatestStateStore -v`
Expected: PASS (all 5 subtests)

- [ ] **Step 5: Commit**

```bash
git add internal/events/latest_state.go internal/events/latest_state_test.go
git commit -m "$(cat <<'EOF'
Add latestStateStore: compacted per-resource event storage

Keeps only the most recent event per live resource, keyed by
(ResourceType, ResourceId), so heap usage scales with live resource
count rather than total events observed. Not yet wired into eventManager.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: `historyRing` — fixed-capacity event ring buffer

**Files:**
- Create: `internal/events/history_ring.go`
- Test: `internal/events/history_ring_test.go`

**Interfaces:**
- Consumes: `events.StoredEvent` (from `pkg/events/history.go`, fields `SequenceId uint64`, `Timestamp time.Time`, `ResourceType string`, `Operation string`, `ResourceId uuid.UUID`, `Objects []any`).
- Produces: `const DefaultHistoryLimit = 1000`, `type historyRing struct{...}`, `newHistoryRing(limit int) *historyRing`, `(*historyRing) Add(ev events.StoredEvent)`, `(*historyRing) Snapshot() []events.StoredEvent`, `(*historyRing) CompactionBoundary() (uint64, time.Time)`.

- [ ] **Step 1: Write the failing tests**

Create `internal/events/history_ring_test.go`:

```go
package eventmgr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
)

func storedEvent(seq uint64) events.StoredEvent {
	return events.StoredEvent{
		SequenceId:   seq,
		Timestamp:    time.Unix(int64(seq), 0),
		ResourceType: "System",
		Operation:    "Create",
	}
}

func TestHistoryRing_UnderCapacityKeepsCompactionSeqZero(t *testing.T) {
	r := newHistoryRing(3)
	r.Add(storedEvent(1))
	r.Add(storedEvent(2))

	seq, _ := r.CompactionBoundary()
	assert.Equal(t, uint64(0), seq)
	assert.Equal(t, []events.StoredEvent{storedEvent(1), storedEvent(2)}, r.Snapshot())
}

func TestHistoryRing_WraparoundEvictsOldestAndTracksBoundary(t *testing.T) {
	r := newHistoryRing(2)
	r.Add(storedEvent(1))
	r.Add(storedEvent(2))
	r.Add(storedEvent(3)) // evicts seq 1

	seq, at := r.CompactionBoundary()
	assert.Equal(t, uint64(1), seq)
	assert.Equal(t, storedEvent(1).Timestamp, at)
	assert.Equal(t, []events.StoredEvent{storedEvent(2), storedEvent(3)}, r.Snapshot())
}

func TestHistoryRing_ContinuedWraparoundKeepsOrder(t *testing.T) {
	r := newHistoryRing(2)
	r.Add(storedEvent(1))
	r.Add(storedEvent(2))
	r.Add(storedEvent(3))
	r.Add(storedEvent(4))

	seq, _ := r.CompactionBoundary()
	assert.Equal(t, uint64(2), seq)
	assert.Equal(t, []events.StoredEvent{storedEvent(3), storedEvent(4)}, r.Snapshot())
}

func TestHistoryRing_DefaultsInvalidLimit(t *testing.T) {
	r := newHistoryRing(0)
	require.NotPanics(t, func() { r.Add(storedEvent(1)) })
	assert.Len(t, r.Snapshot(), 1)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/events/... -run TestHistoryRing -v`
Expected: FAIL with `undefined: newHistoryRing`

- [ ] **Step 3: Write the implementation**

Create `internal/events/history_ring.go`:

```go
package eventmgr

import (
	"time"

	"go.emeland.io/modelsrv/pkg/events"
)

// DefaultHistoryLimit is the number of recent events QueryEvents can serve
// exactly when no WithHistoryLimit option is given (see manager.go).
// History reaching further back than this is synthesized from latestState
// instead of stored verbatim.
const DefaultHistoryLimit = 1000

// historyRing is a fixed-capacity ring buffer of the most recent
// StoredEvents. Its backing array is preallocated once at construction, so
// heap usage is a fixed function of limit, independent of event volume.
// Every call site holds eventManager.mu, so this type has no locking of its
// own.
type historyRing struct {
	buf   []events.StoredEvent
	limit int
	next  int
	size  int

	compactionSeq uint64
	compactionAt  time.Time
}

func newHistoryRing(limit int) *historyRing {
	if limit <= 0 {
		limit = DefaultHistoryLimit
	}
	return &historyRing{
		buf:   make([]events.StoredEvent, limit),
		limit: limit,
	}
}

// Add appends ev, overwriting the oldest retained entry once the buffer is
// full. The overwritten entry's SequenceId/Timestamp become the new
// compaction boundary, so callers can tell how far back real events are
// still available.
func (r *historyRing) Add(ev events.StoredEvent) {
	if r.size == r.limit {
		evicted := r.buf[r.next]
		r.compactionSeq = evicted.SequenceId
		r.compactionAt = evicted.Timestamp
	} else {
		r.size++
	}
	r.buf[r.next] = ev
	r.next = (r.next + 1) % r.limit
}

// Snapshot returns a copy of the retained entries, oldest to newest.
func (r *historyRing) Snapshot() []events.StoredEvent {
	out := make([]events.StoredEvent, r.size)
	if r.size < r.limit {
		copy(out, r.buf[:r.size])
		return out
	}
	n := copy(out, r.buf[r.next:])
	copy(out[n:], r.buf[:r.next])
	return out
}

// CompactionBoundary returns the SequenceId/Timestamp of the most recently
// evicted entry (zero value if nothing has been evicted yet).
func (r *historyRing) CompactionBoundary() (uint64, time.Time) {
	return r.compactionSeq, r.compactionAt
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/events/... -run TestHistoryRing -v`
Expected: PASS (all 4 subtests)

- [ ] **Step 5: Commit**

```bash
git add internal/events/history_ring.go internal/events/history_ring_test.go
git commit -m "$(cat <<'EOF'
Add historyRing: fixed-capacity ring buffer for recent event history

Preallocated once at construction, so heap usage is a fixed function of
the configured limit rather than growing with total events observed. Not
yet wired into eventManager.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Wire `eventManager` to the bounded stores, with query-time compaction

**Files:**
- Modify: `internal/events/manager.go`
- Modify: `internal/events/recording_sink.go`
- Modify: `pkg/events/history.go:32-45` (doc comments only)
- Test: `internal/events/query_events_test.go` (append new tests)

**Interfaces:**
- Consumes: `newLatestStateStore()`, `(*latestStateStore).Receive/.GetEvents()` (Task 1); `DefaultHistoryLimit`, `newHistoryRing(limit int)`, `(*historyRing).Add/.Snapshot/.CompactionBoundary()` (Task 2).
- Produces: `type Option func(*eventManager)`, `func WithHistoryLimit(n int) Option`, `func NewEventManager(opts ...Option) (events.EventManager, error)` (signature change: now variadic, existing zero-arg calls keep compiling).

- [ ] **Step 1: Write the failing test for compaction behavior**

Append to `internal/events/query_events_test.go` (same file, same `package eventmgr`, add after the existing `TestQueryEvents` function):

```go
func TestQueryEvents_CompactionSynthesizesEvictedResourceState(t *testing.T) {
	mgr, err := NewEventManager(WithHistoryLimit(2))
	require.NoError(t, err)

	sink, err := mgr.GetSink()
	require.NoError(t, err)

	idA := uuid.New()
	idB := uuid.New()
	idC := uuid.New()

	require.NoError(t, sink.Receive(events.SystemResource, events.CreateOperation, idA, "a1")) // seq 1
	require.NoError(t, sink.Receive(events.SystemResource, events.CreateOperation, idB, "b1")) // seq 2
	require.NoError(t, sink.Receive(events.SystemResource, events.CreateOperation, idC, "c1")) // seq 3, evicts seq 1 (idA)

	ctx := context.Background()

	t.Run("query reaching past the retention window synthesizes a Create for the evicted resource", func(t *testing.T) {
		results, err := mgr.QueryEvents(ctx, events.EventQuery{IncludePayload: true})
		require.NoError(t, err)
		require.Len(t, results, 3)

		assert.Equal(t, idA, results[0].ResourceId)
		assert.Equal(t, "Create", results[0].Operation)
		assert.Equal(t, []any{"a1"}, results[0].Objects)

		assert.Equal(t, idB, results[1].ResourceId)
		assert.Equal(t, uint64(2), results[1].SequenceId)

		assert.Equal(t, idC, results[2].ResourceId)
		assert.Equal(t, uint64(3), results[2].SequenceId)
	})

	t.Run("query inside the retained tail is unaffected by compaction", func(t *testing.T) {
		results, err := mgr.QueryEvents(ctx, events.EventQuery{SinceSeq: 2})
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, idC, results[0].ResourceId)
	})

	t.Run("a resource still present in the tail is not duplicated by synthesis", func(t *testing.T) {
		results, err := mgr.QueryEvents(ctx, events.EventQuery{ResourceId: &idC})
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, uint64(3), results[0].SequenceId)
	})

	t.Run("an evicted resource updated again reflects the newer value as its last entry", func(t *testing.T) {
		require.NoError(t, sink.Receive(events.SystemResource, events.UpdateOperation, idA, "a2")) // seq 4, evicts seq 2 (idB)

		results, err := mgr.QueryEvents(ctx, events.EventQuery{ResourceId: &idA, IncludePayload: true})
		require.NoError(t, err)
		require.NotEmpty(t, results)
		assert.Equal(t, []any{"a2"}, results[len(results)-1].Objects)
	})

	t.Run("under the cap, behavior is unaffected by compaction entirely", func(t *testing.T) {
		mgr2, err := NewEventManager(WithHistoryLimit(1000))
		require.NoError(t, err)
		sink2, err := mgr2.GetSink()
		require.NoError(t, err)
		id := uuid.New()
		require.NoError(t, sink2.Receive(events.SystemResource, events.CreateOperation, id, "only"))

		results, err := mgr2.QueryEvents(ctx, events.EventQuery{IncludePayload: true})
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, uint64(1), results[0].SequenceId)
		assert.Equal(t, []any{"only"}, results[0].Objects)
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/events/... -run TestQueryEvents_CompactionSynthesizesEvictedResourceState -v`
Expected: FAIL to compile — `WithHistoryLimit` and the new `NewEventManager` signature don't exist yet.

- [ ] **Step 3: Rewrite `internal/events/manager.go`**

Replace the whole file with:

```go
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

	var all []events.StoredEvent
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
			all = append(all, events.NewStoredEvent(
				compactionSeq, compactionAt, ev.ResourceType, events.CreateOperation, ev.ResourceId, ev.Objects,
			))
		}
	}
	all = append(all, tail...)
	e.mu.RUnlock()

	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}

	var results []events.StoredEvent
	for _, ev := range all {
		if ev.SequenceId <= q.SinceSeq {
			continue
		}
		if q.Operation != nil && ev.Operation != q.Operation.WireOperation() {
			continue
		}
		if q.ResourceType != nil && ev.ResourceType != q.ResourceType.WireKind() {
			continue
		}
		if q.ResourceId != nil && ev.ResourceId != *q.ResourceId {
			continue
		}
		entry := ev
		if !q.IncludePayload {
			entry.Objects = nil
		}
		results = append(results, entry)
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}
```

- [ ] **Step 4: Update `internal/events/recording_sink.go`**

Replace the body of `Receive`:

```go
func (r *recordingSink) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, objects ...any) error {
	ev := events.Event{
		ResourceType: resType,
		Operation:    op,
		ResourceId:   resourceId,
		Objects:      objects,
	}

	r.mgr.mu.Lock()
	r.mgr.latestState.Receive(resType, op, resourceId, objects...)
	r.mgr.sequenceNumber++
	r.mgr.historyTail.Add(events.NewStoredEvent(
		r.mgr.sequenceNumber, time.Now(), resType, op, resourceId, objects,
	))
	subs := make([]events.Subscriber, len(r.mgr.subscribers))
	copy(subs, r.mgr.subscribers)
	r.mgr.mu.Unlock()

	for _, sub := range subs {
		s := sub
		evCopy := ev
		go func() {
			err := s.Notify(context.Background(), &evCopy)
			if err != nil {
				fmt.Printf("failed to notify subscriber %s: %v\n", s.GetURL(), err)
			}
		}()
	}
	return nil
}
```

(Only the two lines inside the lock change: `_ = r.mgr.masterList.Receive(...)` → `r.mgr.latestState.Receive(...)` (no longer discarding a return value — `latestStateStore.Receive` returns nothing), and `r.mgr.storedEvents = append(...)` → `r.mgr.historyTail.Add(...)`. The rest of the file, including its imports, is unchanged.)

- [ ] **Step 5: Update doc comments in `pkg/events/history.go`**

Read the current `EventQuery`/`EventQuerier` doc comments (lines 32-45) and replace with:

```go
// EventQuery defines filters and pagination for querying event history.
//
// SinceSeq queries reaching further back than the EventManager's configured
// retention window (see eventmgr.WithHistoryLimit) still succeed: history
// for resources that aged out of the window but still exist is synthesized
// as a Create carrying their current state, so replaying the full result in
// order reconstructs the same state a new subscriber would receive.
type EventQuery struct {
	Operation      *Operation
	ResourceType   *ResourceType
	ResourceId     *uuid.UUID
	SinceSeq       uint64 // return events with SequenceId > SinceSeq
	Limit          int    // max results; 0 means default (100)
	IncludePayload bool
}

// EventQuerier can query stored event history.
type EventQuerier interface {
	QueryEvents(ctx context.Context, q EventQuery) ([]StoredEvent, error)
}
```

- [ ] **Step 6: Run the full `internal/events` package test suite**

Run: `go test ./internal/events/... -v`
Expected: PASS — every existing test (`TestQueryEvents`, `manager_test.go`'s ginkgo specs, `TestLatestStateStore_*`, `TestHistoryRing_*`) plus the new `TestQueryEvents_CompactionSynthesizesEvictedResourceState` subtests.

- [ ] **Step 7: Run `go vet` and `go build` across the module**

Run: `go build ./... && go vet ./...`
Expected: no errors (confirms no other package references the removed `masterList`/`storedEvents` field names — there shouldn't be any, since they were unexported).

- [ ] **Step 8: Commit**

```bash
git add internal/events/manager.go internal/events/recording_sink.go internal/events/query_events_test.go pkg/events/history.go
git commit -m "$(cat <<'EOF'
Wire eventManager to bounded latestStateStore and historyRing

masterList (unbounded, subscriber replay) and storedEvents (unbounded,
history API) are replaced by latestStateStore and historyRing. Heap for
subscriber replay now scales with live resource count; heap for history
queries is bounded by a configurable retention limit (default 1000).
Queries reaching past the retention window synthesize missing resource
state from latestStateStore at query time, so no per-eviction bookkeeping
beyond a compaction boundary marker is needed.

NewEventManager gains a variadic ...Option parameter (WithHistoryLimit);
all existing zero-arg call sites are unaffected.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Configurable retention limit through `pkg/backend`

**Files:**
- Modify: `pkg/backend/backend.go`
- Test: `pkg/backend/backend_test.go` (append)

**Interfaces:**
- Consumes: `eventmgr.WithHistoryLimit(n int) eventmgr.Option`, `eventmgr.NewEventManager(opts ...eventmgr.Option)` (Task 3).
- Produces: `type Option func(*config)`, `func WithEventHistoryLimit(n int) Option`, `func New(opts ...Option) (Backend, error)` (signature change: now variadic; existing zero-arg calls keep compiling).

- [ ] **Step 1: Write the failing test**

Append to `pkg/backend/backend_test.go`, inside the existing `Describe("New", func() { ... })` block (add as a new `It` alongside the others):

```go
		It("propagates WithEventHistoryLimit down to the event manager's history retention", func() {
			b, err := backend.New(backend.WithEventHistoryLimit(2))
			Expect(err).NotTo(HaveOccurred())

			sink, err := b.GetEventManager().GetSink()
			Expect(err).NotTo(HaveOccurred())

			idA := uuid.New()
			idB := uuid.New()
			idC := uuid.New()
			Expect(sink.Receive(events.SystemResource, events.CreateOperation, idA, "a1")).To(Succeed())
			Expect(sink.Receive(events.SystemResource, events.CreateOperation, idB, "b1")).To(Succeed())
			Expect(sink.Receive(events.SystemResource, events.CreateOperation, idC, "c1")).To(Succeed())

			results, err := b.GetEventManager().QueryEvents(context.Background(), events.EventQuery{IncludePayload: true})
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(3))
			Expect(results[0].ResourceId).To(Equal(idA))
			Expect(results[0].Objects).To(Equal([]any{"a1"}))
		})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/backend/... -run TestBackend -v`
Expected: FAIL to compile — `backend.WithEventHistoryLimit` doesn't exist yet.

- [ ] **Step 3: Implement the option in `pkg/backend/backend.go`**

Add near the top of the file, after the imports, and change `New`'s signature:

```go
// config holds construction-time settings for New.
type config struct {
	eventHistoryLimit int
}

// Option configures a Backend at construction time.
type Option func(*config)

// WithEventHistoryLimit sets how many recent events the event manager's
// history-query API can serve exactly; see eventmgr.WithHistoryLimit for
// what happens to queries reaching further back. n must be positive or it
// is ignored (the event manager's own default applies).
func WithEventHistoryLimit(n int) Option {
	return func(c *config) { c.eventHistoryLimit = n }
}
```

Change the start of `New`:

```go
func New(opts ...Option) (Backend, error) {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}

	var mgrOpts []eventmgr.Option
	if cfg.eventHistoryLimit > 0 {
		mgrOpts = append(mgrOpts, eventmgr.WithHistoryLimit(cfg.eventHistoryLimit))
	}
	eventMgr, err := eventmgr.NewEventManager(mgrOpts...)
	if err != nil {
		return nil, err
	}
	// ... rest of the function body is unchanged from here (GetSink, chain, model, etc.)
```

(Only the first four lines of the function body change: `eventMgr, err := eventmgr.NewEventManager()` becomes the block above; everything after it is untouched.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/backend/... -v`
Expected: PASS — all existing `Describe("Backend", ...)` specs plus the new one.

- [ ] **Step 5: Commit**

```bash
git add pkg/backend/backend.go pkg/backend/backend_test.go
git commit -m "$(cat <<'EOF'
Add backend.WithEventHistoryLimit to configure event history retention

Threads a configurable retention limit through to eventmgr.WithHistoryLimit.
backend.New gains a variadic ...Option parameter; all existing zero-arg
call sites are unaffected.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: `--event-history-limit` CLI flag / `EVENT_HISTORY_LIMIT` env var

**Files:**
- Modify: `cmd/modelsrv/server.go`

**Interfaces:**
- Consumes: `backend.WithEventHistoryLimit(n int) backend.Option`, `backend.New(opts ...backend.Option)` (Task 4).

- [ ] **Step 1: Add the `strconv` import and `envIntOrDefault` helper**

Add `"strconv"` to the import block (alphabetically, after `"path/filepath"` and before `"runtime"`).

Add this function next to the existing `envOrDefault`/`envDurationOrDefault` helpers (after `envDurationOrDefault`, around line 205):

```go
func envIntOrDefault(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
```

- [ ] **Step 2: Add the flag variable and registration**

Add alongside the other flag variables (after `var maxConcurrentProbes int`):

```go
var eventHistoryLimit int
```

Add alongside the other `serverCmd.Flags()...` registrations (after the `max-concurrent-probes` line):

```go
	serverCmd.Flags().IntVar(&eventHistoryLimit, "event-history-limit", envIntOrDefault("EVENT_HISTORY_LIMIT", eventmgr.DefaultHistoryLimit), "Number of recent events the /events history API can serve exactly; older queries return synthesized current-state entries instead of an error")
```

This needs the `eventmgr` package imported for its `DefaultHistoryLimit` constant — add `eventmgr "go.emeland.io/modelsrv/internal/events"` to the import block (grouped with the other `go.emeland.io/modelsrv/...` imports).

- [ ] **Step 3: Pass it through to `backend.New`**

Change the call site:

```go
			b, err := backend.New(backend.WithEventHistoryLimit(eventHistoryLimit))
```

- [ ] **Step 4: Verify it builds and the flag is wired**

Run: `go build ./... && go run . server --help 2>&1 | grep -A1 event-history-limit`
Expected: the build succeeds and the help output shows the new flag with its default value (1000).

- [ ] **Step 5: Commit**

```bash
git add cmd/modelsrv/server.go
git commit -m "$(cat <<'EOF'
Add --event-history-limit flag / EVENT_HISTORY_LIMIT env var

Makes the event manager's history retention window configurable at the
process level, following the existing envOrDefault flag pattern.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: End-to-end verification (no code changes)

**Files:** none modified — this task only runs existing and newly-added test suites plus the load/scale harness to confirm the heap-scaling goal from the spec is actually met.

- [ ] **Step 1: Run the full test suite**

Run: `go test $(go list ./... | grep -v /e2e) -v 2>&1 | tail -100`
Expected: all packages pass, including `internal/events`, `pkg/backend`, `pkg/events`, and every `pkg/model/...` package that constructs an `events.ListSink` directly (unaffected, since `pkg/events.ListSink` itself was not changed).

- [ ] **Step 2: Run the load/scale harness before-vs-after comparison**

This was already possible before this change (heap scaling with total events); now confirm it scales with resource count instead:

Run: `make test-load-scale LOAD_SCALE_INSTANCE_COUNTS=10,100,1000 LOAD_SCALE_CHANGE_COUNTS=10,100,1000`

Expected: the generated `test/loadscale/load_scale_report.md` shows `Heap Alloc (after)` growing primarily with the `Instances/Resource` column and staying roughly flat across the `Changes/Instance` column for a fixed instance count — the opposite of pre-change behavior, where heap grew with `Total Events` (instances × changes) regardless of how the two factors were split.

- [ ] **Step 3: Report the before/after comparison**

Read `test/loadscale/load_scale_report.md` and summarize, for a fixed instance count (e.g. 100), how heap alloc changes across change counts (10/100/1000) — it should now be close to flat, confirming the fix. No commit needed for this task (verification only); if the report reveals a regression, stop and fix it in the relevant earlier task before proceeding.
