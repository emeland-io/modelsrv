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
