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
