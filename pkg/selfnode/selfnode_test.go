package selfnode_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/selfnode"
)

func newModel(t *testing.T) model.Model {
	t.Helper()
	m, err := model.NewModel(events.NewListSink())
	require.NoError(t, err)
	return m
}

func TestRegister_CreatesSelfNodeAndType(t *testing.T) {
	const selfURL = "http://127.0.0.1:24000/api"
	m := newModel(t)

	require.NoError(t, selfnode.Register(m, selfURL))

	// NodeType present.
	nt := m.GetNodeTypeById(selfnode.NodeTypeID())
	require.NotNil(t, nt)
	assert.Equal(t, selfnode.NodeTypeName, nt.GetDisplayName())

	// Node present, keyed on the deterministic id, advertising selfURL.
	n := m.GetNodeById(selfnode.NodeID(selfURL))
	require.NotNil(t, n)
	assert.Equal(t, selfURL, n.GetUrl())
	assert.Equal(t, selfnode.NodeTypeName, n.GetDisplayName(), "url belongs in url, not displayName")
	assert.Equal(t, selfnode.NodeTypeID(), n.GetNodeTypeId())

	// Exactly one node is reported.
	nodes, err := m.GetNodes()
	require.NoError(t, err)
	assert.Len(t, nodes, 1)
}

func TestRegister_EmitsNodeCreateEvent(t *testing.T) {
	const selfURL = "http://127.0.0.1:24000/api"
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	require.NoError(t, selfnode.Register(m, selfURL))

	var nodeCreates int
	for _, ev := range sink.GetEvents() {
		if ev.ResourceType == events.NodeResource && ev.Operation == events.CreateOperation {
			assert.Equal(t, selfnode.NodeID(selfURL), ev.ResourceId)
			nodeCreates++
		}
	}
	assert.Equal(t, 1, nodeCreates, "self-node must be replicated as exactly one Node Create event")
}

func TestRegister_Idempotent(t *testing.T) {
	const selfURL = "http://127.0.0.1:24000/api"
	m := newModel(t)

	require.NoError(t, selfnode.Register(m, selfURL))
	require.NoError(t, selfnode.Register(m, selfURL))

	nodes, err := m.GetNodes()
	require.NoError(t, err)
	assert.Len(t, nodes, 1, "re-registering the same selfURL must not create a second node")
}

func TestRegister_DeterministicIDPerURL(t *testing.T) {
	a := selfnode.NodeID("http://a:24000/api")
	b := selfnode.NodeID("http://b:24000/api")
	assert.Equal(t, a, selfnode.NodeID("http://a:24000/api"), "same URL must yield same id")
	assert.NotEqual(t, a, b, "different URLs must yield different ids")
}

func TestRegister_EmptyURLRejected(t *testing.T) {
	m := newModel(t)
	assert.Error(t, selfnode.Register(m, ""))
}
