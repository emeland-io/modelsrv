# Event store compaction: bounded memory for subscriber replay and history queries

Date: 2026-07-27
Status: approved, pending implementation

## Problem

`internal/events.eventManager` (`internal/events/manager.go`) keeps two
in-memory logs that both grow without bound, one entry per event the model
ever receives:

- `masterList` (`*events.ListSink`, from `pkg/events/events.go`) — records
  every event ever seen, used only to replay past state to a newly-added
  subscriber (`AddSubscriber`).
- `storedEvents` (`[]events.StoredEvent`) — records every event ever seen,
  used to serve the `/events` history-query API (`QueryEvents`,
  `SinceSeq`-based pagination). Already flagged in code with
  `// TODO: add a cap/eviction strategy for production use`.

`test/loadscale/load_scale_test.go` (added in 956679d) demonstrates the
resulting heap growth: for a fixed number of resources, heap usage scales
with the number of *changes* applied, not the number of *resources*, because
both logs retain full history forever.

## Goals

1. `masterList`: heap usage must scale with the number of **live** (not yet
   deleted) resources, not the total number of events ever observed. From
   outside the system, subscriber replay must remain observably equivalent —
   a new subscriber ends up with the same model state as before.
2. `storedEvents`: heap usage must be bounded by a **configurable retention
   limit** (default 1000 events), not the total number of events ever
   observed. Once the limit is exceeded, evicted history must not simply
   vanish: querying further back than the retention window must synthesize
   the missing prefix as a "Create" event carrying the current state of
   each live resource that has no remaining representation in the retained
   tail (resources already present in the tail are skipped — a synthetic
   Create would only duplicate them). Applying the full response
   (synthetic prefix + retained tail) still reconstructs the same overall
   state that a new subscriber would receive via `masterList`'s replay.
3. No other externally observable behavior changes (API shapes, semantics of
   in-window queries, subscriber replay content) except as described in (2).

## Non-goals

- `pkg/events.ListSink` (the public type) is untouched. It is a full-fidelity
  test double used across ~30 test files to assert exact event sequences
  (`GetEvents()[0]`, `[1]`, ...) and is unrelated to the two logs above.
- No change to `EventQuery`/wire API shapes, sequence ID semantics, or
  `GetCurrentSequenceId`/`IncrementSequenceId`.
- No attempt to synthesize entries for resources still represented in the
  retained tail, nor to make any synthetic prefix historically exact (see
  "Correctness argument" below) — only final-state equivalence is
  guaranteed.

## Architecture

### Shared compacted store: `latestState`

Replaces `masterList`. A single per-resource compacted structure, private to
`internal/events` (package `eventmgr`), living in a new file
`internal/events/latest_state.go`:

```go
type resourceKey struct {
    resType events.ResourceType
    id      uuid.UUID
}

type latestStateStore struct {
    order []resourceKey
    state map[resourceKey]events.Event
}
```

- `Receive(resType, op, resourceId, objects...)`: on `Create`/`Update`, upsert
  `state[key]`; if `key` is new, append it to `order`. On `Delete`, remove
  `key` from both `state` and `order` entirely.
- `GetEvents() []events.Event`: walks `order`, returns one `Event` per live
  resource, **always with `Operation` rewritten to `CreateOperation`**
  regardless of the resource's actual last operation. Rationale: to any
  observer receiving this snapshot as their first knowledge of the resource
  (a new subscriber, or a history query starting before the retention
  window), the resource is first appearing now — presenting it as anything
  but a Create would be misleading, and the model's replication apply logic
  (`pkg/model/event_apply.go`) already treats Create and Update identically
  (both go through `applyReplicationUpsert`), so this is a safe, uniform
  rule, not a special case.
- No internal mutex: every call site already holds `eventManager.mu`
  (`ListSink`'s own mutex was redundant here and is dropped).
- `order` removal on delete is a linear scan (`O(n)` in live resource count).
  This is a deliberate simplification — the goal is bounded *heap*, not
  optimal delete latency, and there is no stated performance requirement
  beyond memory. If delete-heavy workloads make this a bottleneck later, an
  intrusive doubly-linked-list (LRU-style) index can replace the slice scan
  without changing the public shape of this type.

Used by:
- `AddSubscriber` (unchanged call site shape: `e.latestState.GetEvents()`).
- `QueryEvents`, as the source of synthesized prefix entries (see below).

### Bounded tail: `historyTail`

Replaces `storedEvents`. A fixed-capacity ring buffer of raw
`events.StoredEvent`, in a new file `internal/events/history_ring.go`:

```go
type historyRing struct {
    buf   []events.StoredEvent // len == limit, preallocated once
    limit int
    next  int    // write cursor
    size  int    // valid entries, <= limit

    compactionSeq uint64    // SequenceId of the most recently evicted entry
    compactionAt  time.Time // Timestamp of the most recently evicted entry
}
```

- `newHistoryRing(limit int) *historyRing`: preallocates `buf` at length
  `limit` once; the array is never grown, so heap usage for this structure is
  a fixed function of `limit`, independent of event volume.
- `Add(ev events.StoredEvent)`: if the buffer is full, the slot about to be
  overwritten (`buf[next]`) holds the current oldest entry — before
  overwriting, its `SequenceId`/`Timestamp` become the new
  `compactionSeq`/`compactionAt`. Then writes `ev` at `next` and advances
  `next = (next + 1) % limit`.
- `Snapshot() []events.StoredEvent`: returns a copy of retained entries,
  oldest to newest (handles the wraparound case).
- `CompactionBoundary() (uint64, time.Time)`: returns `compactionSeq`,
  `compactionAt`.

No internal mutex, same rationale as `latestStateStore` — always accessed
under `eventManager.mu`.

### `QueryEvents`

```go
func (e *eventManager) QueryEvents(ctx context.Context, q events.EventQuery) ([]events.StoredEvent, error) {
    e.mu.RLock()
    tail := e.historyTail.Snapshot()
    compactionSeq, compactionAt := e.historyTail.CompactionBoundary()
    var synthetic []events.StoredEvent
    if q.SinceSeq < compactionSeq {
        // Skip resources already present in the tail — synthesizing them
        // would only produce redundant Creates that the tail already covers.
        inTail := /* set of (ResourceType, ResourceId) in tail */
        for _, ev := range e.latestState.GetEvents() {
            if _, present := inTail[key]; present {
                continue
            }
            synthetic = append(synthetic, events.NewStoredEvent(
                compactionSeq, compactionAt, ev.ResourceType, events.CreateOperation, ev.ResourceId, ev.Objects,
            ))
        }
    }
    e.mu.RUnlock()

    // Filters (SinceSeq/Operation/ResourceType/ResourceId/IncludePayload)
    // apply to both synthetic and tail. Limit caps the retained tail only:
    // the synthetic prefix is always delivered in full (see below).
    ...
}
```

- When nothing has ever been evicted (`compactionSeq == 0`, since real
  `SequenceId`s start at 1), `q.SinceSeq < compactionSeq` is always false —
  behavior is byte-for-byte identical to the current unbounded
  implementation. Divergence is only observable once the retention limit has
  actually been exceeded.
- Synthesis is partial: only live resources with **no** remaining entry in
  the retained tail get a synthetic Create. Resources already represented
  in the tail are omitted from the prefix (response shape is shorter than
  "one synthetic entry per live resource"; final-state LWW still holds
  because the tail alone reconstructs those resources).
- All synthesized entries share `SequenceId = compactionSeq`, so they sort
  correctly before the real tail (whose oldest entry has
  `SequenceId = compactionSeq + 1`). Existing `SinceSeq`/`Operation`/
  `ResourceType`/`ResourceId`/`IncludePayload` filters apply to them;
  **`Limit` does not** — the synthetic prefix is always returned in full
  within a single response. Capping it would let a paginating client
  (`SinceSeq` = last delivered `SequenceId`, which equals `compactionSeq`
  for every synthetic entry) permanently skip whatever didn't fit on the
  first page. Once `SinceSeq` reaches the retention boundary, subsequent
  calls page the retained tail normally and honor `Limit`.
- Ordering of synthesized entries follows `latestState`'s insertion order
  (creation order), matching today's general "older resources first"
  character of the log.

### Correctness argument for goal (2)

Claim: applying the full response (synthetic prefix, in order, followed by
the retained tail, in order) always reconstructs the same final state as
`latestState.GetEvents()` (i.e. what a new subscriber gets).

- Resources with no representation in the tail: the synthetic entry *is*
  their current state. Final state after applying it matches `latestState`
  trivially.
- Resources already present in the tail: no synthetic entry is emitted
  (would be redundant). The tail's real events alone reconstruct them;
  because `applyReplicationUpsert` is last-write-wins per resource, the
  last real tail event determines the final state, which equals
  `latestState`. (The older "prefix+tail for every live resource" algorithm
  also reached the same final state via LWW, but produced a longer
  response and a different intermediate apply path; this design prefers
  the reduced shape.)
- Deleted resources: absent from `latestState`, so no synthetic entry is
  produced. If a real Delete event for that resource is still in the tail,
  it's a harmless no-op delete-of-nothing. If the Delete event itself has
  already been evicted, the resource is correctly absent from the whole
  response.

This holds without needing per-eviction bookkeeping beyond
`compactionSeq`/`compactionAt` — the synthesis is computed lazily at query
time directly from the always-current `latestState`.

## Configuration

The retention limit becomes configurable via the functional-options pattern,
keeping all existing zero-arg call sites (`backend.New()`,
`eventmgr.NewEventManager()`, ~15 call sites across tests and
`cmd/modelsrv/server.go`) source-compatible and behaviorally unchanged
(default 1000):

```go
// internal/events
const DefaultHistoryLimit = 1000

type Option func(*eventManager)

func WithHistoryLimit(n int) Option {
    return func(e *eventManager) {
        if n > 0 {
            e.historyLimit = n
        }
    }
}

func NewEventManager(opts ...Option) (events.EventManager, error)
```

```go
// pkg/backend
type Option func(*config)

func WithEventHistoryLimit(n int) Option {
    return func(c *config) { c.eventHistoryLimit = n }
}

func New(opts ...Option) (Backend, error)
```

`cmd/modelsrv/server.go` gains a new flag, following the existing
`envOrDefault` pattern used for the other flags in that file:

```go
var eventHistoryLimit int
...
serverCmd.Flags().IntVar(&eventHistoryLimit, "event-history-limit",
    envIntOrDefault("EVENT_HISTORY_LIMIT", eventmgr.DefaultHistoryLimit),
    "Number of recent events to retain for exact history queries; older queries return synthesized current-state entries")
```

wired through as `backend.New(backend.WithEventHistoryLimit(eventHistoryLimit))`.
A new `envIntOrDefault` helper is added alongside the existing
`envOrDefault`/`envDurationOrDefault` helpers.

## Files touched

- `internal/events/manager.go` — field renames (`masterList` → `latestState`,
  `storedEvents` → `historyTail`), `NewEventManager` options,
  `AddSubscriber`, `QueryEvents`.
- `internal/events/recording_sink.go` — update field references and the
  `storedEvents` append to `historyTail.Add(...)`.
- `internal/events/latest_state.go` — new.
- `internal/events/history_ring.go` — new.
- `pkg/backend/backend.go` — `Option`/`WithEventHistoryLimit`, `New(opts
  ...Option)`.
- `cmd/modelsrv/server.go` — new flag/env var, `envIntOrDefault` helper.

Not touched: `pkg/events/events.go` (`ListSink` stays as-is),
`pkg/events/history.go` (`EventQuery`/`StoredEvent` shapes unchanged, only
doc comments updated to describe compaction).

## Testing plan

- `internal/events/latest_state_test.go` (new): upsert/overwrite semantics,
  delete removes from both map and order, re-create after delete re-appends
  at the end of order, `GetEvents()` always reports `CreateOperation`.
- `internal/events/history_ring_test.go` (new): fills under capacity
  (`compactionSeq` stays 0), wraparound overwrite, `Snapshot()` ordering
  correctness, `CompactionBoundary()` tracks the right evicted entry.
- `internal/events/query_events_test.go` (existing, 4 events well under the
  default 1000 cap): unaffected, add a new sub-test that constructs a
  manager with `WithHistoryLimit(2)` (or similar small number), pushes past
  the cap, and asserts: (a) a query with `SinceSeq` before the eviction
  boundary returns a synthesized `Create` for the evicted resource only
  (resources still in the tail are not duplicated), (b) a query with
  `SinceSeq` inside the retained tail is unaffected, (c) the synthetic
  prefix is returned in full even when `Limit` is smaller than the
  prefix, (d) applying the full synthesized+tail response ends up matching
  `latestState`.
- `internal/events/manager_test.go` (existing): verify `AddSubscriber`
  replay is unaffected for the common case (events well under the cap) and
  correctly reflects compacted state once resources have been deleted.
- `make test-load-scale`: run before/after to confirm heap now scales with
  resource count (via `LOAD_SCALE_INSTANCE_COUNTS`) rather than change count
  (via `LOAD_SCALE_CHANGE_COUNTS`), using the existing harness added in
  956679d.

## Open questions / risks

None outstanding — scope and correctness invariants were confirmed with the
user during design review.
