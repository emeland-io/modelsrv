package eventmgr

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
	"go.uber.org/zap"
)

type countSub struct {
	mu       sync.Mutex
	url      string
	id       uuid.UUID
	remote   map[string]int
	countErr error
	got      []events.Event
}

func (s *countSub) GetURL() string    { return s.url }
func (s *countSub) GetId() uuid.UUID  { return s.id }
func (s *countSub) GetStatus() string { return "active" }
func (s *countSub) Notify(_ context.Context, ev *events.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ev != nil {
		s.got = append(s.got, *ev)
	}
	return nil
}
func (s *countSub) CountOrigin(_ context.Context, originURL string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.countErr != nil {
		return 0, s.countErr
	}
	return s.remote[originURL], nil
}

func TestReconcile_ResendsWhenCountsDiffer(t *testing.T) {
	originA := "http://a/api"
	originB := "http://b/api"
	id1, id2 := uuid.New(), uuid.New()
	snapshot := []events.Event{
		{ResourceType: events.SystemResource, Operation: events.CreateOperation, ResourceId: id1, OriginURL: originA},
		{ResourceType: events.NodeResource, Operation: events.CreateOperation, ResourceId: id2, OriginURL: originB},
	}
	sub := &countSub{
		url:    "http://sub/api",
		id:     uuid.New(),
		remote: map[string]int{originA: 0, originB: 1}, // A mismatched, B matched
	}
	n := newNotifier(sub, func() []events.Event { return snapshot }, zap.NewNop().Sugar())

	require.True(t, n.reconcile())

	sub.mu.Lock()
	defer sub.mu.Unlock()
	require.Len(t, sub.got, 1)
	assert.Equal(t, id1, sub.got[0].ResourceId)
	assert.Equal(t, originA, sub.got[0].OriginURL)
}

func TestReconcile_ResendsWhenCountGETFails(t *testing.T) {
	origin := "http://a/api"
	id := uuid.New()
	snapshot := []events.Event{
		{ResourceType: events.SystemResource, Operation: events.CreateOperation, ResourceId: id, OriginURL: origin},
	}
	sub := &countSub{
		url:      "http://sub/api",
		id:       uuid.New(),
		countErr: errors.New("unreachable"),
	}
	n := newNotifier(sub, func() []events.Event { return snapshot }, zap.NewNop().Sugar())

	require.True(t, n.reconcile())

	sub.mu.Lock()
	defer sub.mu.Unlock()
	require.Len(t, sub.got, 1)
	assert.Equal(t, id, sub.got[0].ResourceId)
}

func TestReconcile_SkipsWhenCountsMatch(t *testing.T) {
	origin := "http://a/api"
	id := uuid.New()
	snapshot := []events.Event{
		{ResourceType: events.SystemResource, Operation: events.CreateOperation, ResourceId: id, OriginURL: origin},
	}
	sub := &countSub{
		url:    "http://sub/api",
		id:     uuid.New(),
		remote: map[string]int{origin: 1},
	}
	n := newNotifier(sub, func() []events.Event { return snapshot }, zap.NewNop().Sugar())

	require.True(t, n.reconcile())

	sub.mu.Lock()
	defer sub.mu.Unlock()
	assert.Empty(t, sub.got)
}

func TestManager_CountByOrigin(t *testing.T) {
	em, err := NewEventManager(WithSelfURL("http://self/api"))
	require.NoError(t, err)
	sink, err := em.GetSink()
	require.NoError(t, err)

	id := uuid.New()
	require.NoError(t, sink.Receive(events.SystemResource, events.CreateOperation, id, "sys"))
	assert.Equal(t, 1, em.CountByOrigin("http://self/api"))
	assert.Equal(t, 0, em.CountByOrigin("http://other/api"))
}

func TestSetReconcileIntervalForTest(t *testing.T) {
	restore := SetReconcileIntervalForTest(50 * time.Millisecond)
	defer restore()
	assert.Equal(t, 50*time.Millisecond, time.Duration(notifyReconcileInterval.Load()))
}
