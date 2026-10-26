package node_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"go.emeland.io/modelsrv/pkg/model/node"
)

func TestNodeURLAndSubscriptions(t *testing.T) {
	n := node.NewNode(uuid.New())
	n.SetUrl("http://sensor:24200/api/")
	require.Equal(t, "http://sensor:24200/api/", n.GetUrl())

	filter := "http://filter:80/api/"
	server := "http://server:80/api/"

	require.True(t, node.AddSubscription(n, filter))
	require.True(t, node.AddSubscription(n, server))
	// Adding a duplicate is a no-op.
	require.False(t, node.AddSubscription(n, filter))
	// Empty is ignored.
	require.False(t, node.AddSubscription(n, ""))

	require.ElementsMatch(t, []string{filter, server}, n.GetSubscriptions())
	require.True(t, node.HasSubscription(n, filter))

	require.True(t, node.RemoveSubscription(n, filter))
	require.False(t, node.RemoveSubscription(n, filter))
	require.Equal(t, []string{server}, n.GetSubscriptions())
}
