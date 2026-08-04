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
