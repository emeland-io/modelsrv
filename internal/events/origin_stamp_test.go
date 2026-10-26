package eventmgr

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.emeland.io/modelsrv/pkg/events"
)

// TestLocalMutationsDuringReplicatedApplyKeepSelfOrigin is the regression test
// for the cross-contamination between a replicated apply and concurrent local
// mutations: while ApplyReplicated(remoteOrigin) runs on one goroutine, local
// changes on other goroutines must still be stamped with selfURL, never the
// remote origin.
func TestLocalMutationsDuringReplicatedApplyKeepSelfOrigin(t *testing.T) {
	const selfURL = "http://self:24000/api"
	const remoteURL = "http://remote:24000/api"
	remoteTS := time.Date(2026, 7, 8, 9, 10, 11, 0, time.UTC)
	m := managerForStampTest(t, selfURL)
	sink := m.getOrCreateModelSink()

	const n = 40
	localIDs := make([]uuid.UUID, n)
	remoteIDs := make([]uuid.UUID, n)
	for i := range localIDs {
		localIDs[i] = uuid.New()
		remoteIDs[i] = uuid.New()
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(2)
		// replicated apply on its own goroutine
		go func(id uuid.UUID) {
			defer wg.Done()
			<-start
			require.NoError(t, m.ApplyReplicated(remoteURL, remoteTS, func() error {
				time.Sleep(time.Millisecond)
				return sink.Receive(events.NodeResource, events.CreateOperation, id, "r")
			}))
		}(remoteIDs[i])
		// concurrent local mutation (no ApplyReplicated) on another goroutine
		go func(id uuid.UUID) {
			defer wg.Done()
			<-start
			require.NoError(t, sink.Receive(events.NodeResource, events.CreateOperation, id, "l"))
		}(localIDs[i])
	}
	close(start)
	wg.Wait()

	for _, id := range remoteIDs {
		ev, ok := latestFor(m, id)
		require.True(t, ok)
		assert.Equal(t, remoteURL, ev.OriginURL, "replicated resource %s lost its remote origin", id)
	}
	for _, id := range localIDs {
		ev, ok := latestFor(m, id)
		require.True(t, ok)
		assert.Equal(t, selfURL, ev.OriginURL, "local resource %s was mis-stamped with a remote origin", id)
	}
}

// TestApplyReplicatedConcurrentPushesKeepOwnOrigin is the regression test for
// the race where concurrent POST /events/push handlers (each on its own
// goroutine) could overwrite each other's preserved origin/timestamp. Each
// concurrent apply records its own resource and must end up stamped with the
// origin it was given, never a sibling's.
func TestApplyReplicatedConcurrentPushesKeepOwnOrigin(t *testing.T) {
	m := managerForStampTest(t, "http://relay:24000/api")
	sink := m.getOrCreateModelSink()

	const n = 50
	type want struct {
		id     uuid.UUID
		origin string
		ts     time.Time
	}
	wants := make([]want, n)
	for i := range wants {
		wants[i] = want{
			id:     uuid.New(),
			origin: fmt.Sprintf("http://origin-%d:24000/api", i),
			ts:     time.Unix(int64(1_700_000_000+i), 0).UTC(),
		}
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, w := range wants {
		wg.Add(1)
		go func(w want) {
			defer wg.Done()
			<-start // release all at once to maximise interleaving
			err := m.ApplyReplicated(w.origin, w.ts, func() error {
				// Widen the window between stamping and recording so a racy
				// implementation would observe a sibling's stamp.
				time.Sleep(time.Millisecond)
				return sink.Receive(events.NodeResource, events.CreateOperation, w.id, "n")
			})
			require.NoError(t, err)
		}(w)
	}
	close(start)
	wg.Wait()

	for _, w := range wants {
		ev, ok := latestFor(m, w.id)
		require.True(t, ok, "event for %s missing", w.id)
		assert.Equal(t, w.origin, ev.OriginURL, "resource %s got a sibling's origin (race)", w.id)
		assert.Equal(t, w.ts, ev.Timestamp, "resource %s got a sibling's timestamp (race)", w.id)
	}
}

// managerForStampTest builds a concrete *eventManager with a known self URL so
// the origin-stamping behaviour can be inspected via its latest-state store.
func managerForStampTest(t *testing.T, selfURL string) *eventManager {
	t.Helper()
	em, err := NewEventManager(WithSelfURL(selfURL))
	require.NoError(t, err)
	m, ok := em.(*eventManager)
	require.True(t, ok)
	return m
}

func latestFor(m *eventManager, id uuid.UUID) (events.Event, bool) {
	for _, ev := range m.latestState.GetEvents() {
		if ev.ResourceId == id {
			return ev, true
		}
	}
	return events.Event{}, false
}

// TestLocalEventStampsSelfOriginAndNow verifies a locally-produced event is
// stamped with this node's own URL and a current timestamp.
func TestLocalEventStampsSelfOriginAndNow(t *testing.T) {
	const selfURL = "http://self:24000/api"
	m := managerForStampTest(t, selfURL)
	sink := m.getOrCreateModelSink()

	id := uuid.New()
	before := time.Now()
	require.NoError(t, sink.Receive(events.NodeResource, events.CreateOperation, id, "local-node"))
	after := time.Now()

	ev, ok := latestFor(m, id)
	require.True(t, ok)
	assert.Equal(t, selfURL, ev.OriginURL, "local event must carry this node's URL as origin")
	assert.False(t, ev.Timestamp.Before(before), "timestamp should be >= before")
	assert.False(t, ev.Timestamp.After(after), "timestamp should be <= after")
}

// TestApplyReplicatedPreservesOriginAndTimestamp is the core forward-hop
// guarantee: when a node forwards an event received from upstream, the recorded
// event keeps the originating node's URL and timestamp rather than being
// re-stamped with this node's identity and the current time.
func TestApplyReplicatedPreservesOriginAndTimestamp(t *testing.T) {
	const selfURL = "http://relay:24000/api"
	const upstreamURL = "http://origin:24000/api"
	upstreamTS := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	m := managerForStampTest(t, selfURL)
	sink := m.getOrCreateModelSink()

	id := uuid.New()
	// Simulate POST /events/push: the handler decodes origin+timestamp from the
	// wire and applies the change within ApplyReplicated. The apply here stands
	// in for the model re-mutation that re-enters the recording sink.
	err := m.ApplyReplicated(upstreamURL, upstreamTS, func() error {
		return sink.Receive(events.NodeResource, events.CreateOperation, id, "forwarded-node")
	})
	require.NoError(t, err)

	ev, ok := latestFor(m, id)
	require.True(t, ok)
	assert.Equal(t, upstreamURL, ev.OriginURL, "forwarded event must preserve upstream origin, not relay URL")
	assert.Equal(t, upstreamTS, ev.Timestamp, "forwarded event must preserve upstream timestamp, not now")

	// After the replicated apply returns, a subsequent local event is stamped
	// with this node again: the override is scoped to the forward call only.
	localID := uuid.New()
	require.NoError(t, sink.Receive(events.NodeResource, events.CreateOperation, localID, "local-again"))
	localEv, ok := latestFor(m, localID)
	require.True(t, ok)
	assert.Equal(t, selfURL, localEv.OriginURL)
	assert.NotEqual(t, upstreamTS, localEv.Timestamp)
}
