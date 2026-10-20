package oapi

import (
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
)

// TestNodeWireRoundTripSubscriptions verifies that url and subscriptions survive the
// domain<->wire conversion used for replication between components.
func TestNodeWireRoundTripSubscriptions(t *testing.T) {
	nodeID := uuid.New()
	url := "http://prometheus-sensor.emeland-demo.svc:24200/api/"
	filter := "http://emeland-demo-filter:80/api/"
	server := "http://emeland-demo-server:80/api/"

	dto := &Node{
		NodeId:        openapi_types.UUID(nodeID),
		DisplayName:   "prometheus-sensor",
		NodeType:      openapi_types.UUID(uuid.New()),
		Url:           url,
		Subscriptions: []string{filter, server},
	}

	dom, err := NodeFromDto(nil, dto)
	require.NoError(t, err)
	require.Equal(t, url, dom.GetUrl())
	require.Equal(t, []string{filter, server}, dom.GetSubscriptions())

	out := NodeToDto(dom)
	require.Equal(t, url, out.Url)
	require.Equal(t, []string{filter, server}, out.Subscriptions)
}
