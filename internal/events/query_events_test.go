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
