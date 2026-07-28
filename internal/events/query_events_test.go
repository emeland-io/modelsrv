package eventmgr

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
)

func TestQueryEvents(t *testing.T) {
	mgr, err := NewEventManager()
	require.NoError(t, err)

	sink, err := mgr.GetSink()
	require.NoError(t, err)

	id1 := uuid.New()
	id2 := uuid.New()
	require.NoError(t, sink.Receive(events.SystemResource, events.CreateOperation, id1))
	require.NoError(t, sink.Receive(events.NodeResource, events.CreateOperation, id2))
	require.NoError(t, sink.Receive(events.SystemResource, events.DeleteOperation, id1))

	ctx := context.Background()

	t.Run("no filters returns all", func(t *testing.T) {
		results, err := mgr.QueryEvents(ctx, events.EventQuery{})
		require.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, "System", results[0].ResourceType)
		assert.Equal(t, "Create", results[0].Operation)
	})

	t.Run("filter by operation", func(t *testing.T) {
		op := events.DeleteOperation
		results, err := mgr.QueryEvents(ctx, events.EventQuery{Operation: &op})
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "Delete", results[0].Operation)
	})

	t.Run("filter by resourceType", func(t *testing.T) {
		rt := events.NodeResource
		results, err := mgr.QueryEvents(ctx, events.EventQuery{ResourceType: &rt})
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, id2, results[0].ResourceId)
	})

	t.Run("filter by resourceId", func(t *testing.T) {
		results, err := mgr.QueryEvents(ctx, events.EventQuery{ResourceId: &id1})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("pagination with sinceSeq", func(t *testing.T) {
		results, err := mgr.QueryEvents(ctx, events.EventQuery{SinceSeq: 2})
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, uint64(3), results[0].SequenceId)
	})

	t.Run("limit", func(t *testing.T) {
		results, err := mgr.QueryEvents(ctx, events.EventQuery{Limit: 1})
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("payload excluded by default", func(t *testing.T) {
		results, err := mgr.QueryEvents(ctx, events.EventQuery{})
		require.NoError(t, err)
		assert.Nil(t, results[0].Objects)
	})

	t.Run("payload included when requested", func(t *testing.T) {
		require.NoError(t, sink.Receive(events.SystemResource, events.CreateOperation, uuid.New(), "payload"))
		results, err := mgr.QueryEvents(ctx, events.EventQuery{IncludePayload: true, SinceSeq: 3})
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.NotNil(t, results[0].Objects)
	})
}

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
