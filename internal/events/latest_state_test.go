package eventmgr

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
)

// recv is a small helper mirroring the old positional Receive signature so the
// tests stay compact after Receive moved to taking a full events.Event.
func recv(s *latestStateStore, rt events.ResourceType, op events.Operation, id uuid.UUID, objects ...any) {
	s.Receive(events.Event{
		ResourceType: rt,
		Operation:    op,
		ResourceId:   id,
		Objects:      objects,
	})
}

func TestLatestStateStore_UpsertReplacesPriorState(t *testing.T) {
	s := newLatestStateStore()
	id := uuid.New()
	recv(s, events.SystemResource, events.CreateOperation, id, "v1")
	recv(s, events.SystemResource, events.UpdateOperation, id, "v2")

	got := s.GetEvents()
	require.Len(t, got, 1)
	assert.Equal(t, events.CreateOperation, got[0].Operation)
	assert.Equal(t, id, got[0].ResourceId)
	assert.Equal(t, []any{"v2"}, got[0].Objects)
}

func TestLatestStateStore_DeleteRemovesEntry(t *testing.T) {
	s := newLatestStateStore()
	id := uuid.New()
	recv(s, events.SystemResource, events.CreateOperation, id, "v1")
	recv(s, events.SystemResource, events.DeleteOperation, id)

	assert.Empty(t, s.GetEvents())
}

func TestLatestStateStore_DeleteUnknownIsNoop(t *testing.T) {
	s := newLatestStateStore()
	recv(s, events.SystemResource, events.DeleteOperation, uuid.New())

	assert.Empty(t, s.GetEvents())
}

func TestLatestStateStore_OrderPreservedAndRecreateAppendsAtEnd(t *testing.T) {
	s := newLatestStateStore()
	idA := uuid.New()
	idB := uuid.New()
	recv(s, events.SystemResource, events.CreateOperation, idA, "a1")
	recv(s, events.SystemResource, events.CreateOperation, idB, "b1")
	recv(s, events.SystemResource, events.DeleteOperation, idA)
	recv(s, events.SystemResource, events.CreateOperation, idA, "a2")

	got := s.GetEvents()
	require.Len(t, got, 2)
	assert.Equal(t, idB, got[0].ResourceId)
	assert.Equal(t, idA, got[1].ResourceId)
	assert.Equal(t, []any{"a2"}, got[1].Objects)
}

func TestLatestStateStore_KeyIncludesResourceType(t *testing.T) {
	s := newLatestStateStore()
	id := uuid.New()
	recv(s, events.SystemResource, events.CreateOperation, id, "sys")
	recv(s, events.NodeResource, events.CreateOperation, id, "node")

	assert.Len(t, s.GetEvents(), 2)
}

// TestLatestStateStore_PreservesOriginAndTimestamp verifies the store keeps the
// event's origin + timestamp so a replay to a new subscriber carries provenance.
func TestLatestStateStore_PreservesOriginAndTimestamp(t *testing.T) {
	s := newLatestStateStore()
	id := uuid.New()
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	s.Receive(events.Event{
		ResourceType: events.NodeResource,
		Operation:    events.CreateOperation,
		ResourceId:   id,
		Objects:      []any{"n"},
		OriginURL:    "http://origin:24000/api",
		Timestamp:    ts,
	})

	got := s.GetEvents()
	require.Len(t, got, 1)
	assert.Equal(t, "http://origin:24000/api", got[0].OriginURL)
	assert.Equal(t, ts, got[0].Timestamp)
}
