// Package selfnode registers a Node resource that represents the running
// modelsrv instance itself, so the instance reports itself through the
// EmELand API (GET /landscape/nodes).
//
// A node's UUID is per-process and not stable across restarts; its base API
// URL is the stable identity used as the subscription edge-key between peers
// (see pkg/model/node.node_subscriptions). The self-node therefore derives a
// deterministic UUID from the instance's own base API URL (selfURL), so a
// restart on the same address reports the same node id and re-registration is
// idempotent.
package selfnode

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/node"
)

// NodeTypeName is the EmELand NodeType display name for a modelsrv instance.
const NodeTypeName = "modelsrv"

// nodeNamespace and nodeTypeNamespace are fixed namespaces used to derive
// stable UUIDv5 values for the self-node and its NodeType. They were generated
// once and are intentionally hardcoded: changing them changes the node ids
// that peers have already seen, so they must not be changed after the first
// release.
var (
	nodeNamespace     = uuid.MustParse("6f1a2b3c-4d5e-6f70-8192-a3b4c5d6e7f8")
	nodeTypeNamespace = uuid.MustParse("7a2b3c4d-5e6f-7081-92a3-b4c5d6e7f809")
)

// NodeTypeID returns the deterministic NodeType UUID for the modelsrv self-node.
func NodeTypeID() uuid.UUID {
	return uuid.NewSHA1(nodeTypeNamespace, []byte(NodeTypeName))
}

// NodeID returns the deterministic Node UUID for the modelsrv instance that
// serves the given base API URL.
func NodeID(selfURL string) uuid.UUID {
	return uuid.NewSHA1(nodeNamespace, []byte(selfURL))
}

// Register ensures a NodeType "modelsrv" and a Node representing this instance
// exist in the model. The Node's url is selfURL, matching the event origin.
// Calling it more than once for the same selfURL is a no-op.
//
// Adding the Node emits a Node Create event like any other model change, so
// the self-node is replicated to subscribers and peers learn about this
// instance.
//
// selfURL must be this instance's canonical base API URL
// (endpoint.APIBaseURL). An empty selfURL is rejected: without a stable URL the
// self-node has no stable identity.
func Register(m model.Model, selfURL string) error {
	if selfURL == "" {
		return fmt.Errorf("selfnode: empty selfURL; cannot register self-node without a base API URL")
	}

	ntID := NodeTypeID()
	if m.GetNodeTypeById(ntID) == nil {
		nt := node.NewNodeType(ntID)
		nt.SetDisplayName(NodeTypeName)
		nt.SetDescription("A modelsrv instance serving the EmELand model over the REST API.")
		if err := m.AddNodeType(nt); err != nil {
			return fmt.Errorf("selfnode: add NodeType: %w", err)
		}
	}

	nodeID := NodeID(selfURL)
	if m.GetNodeById(nodeID) != nil {
		return nil
	}

	n := node.NewNode(nodeID)
	n.SetDisplayName(NodeTypeName)
	n.SetDescription("This modelsrv instance, reporting itself as a Node.")
	n.SetNodeTypeById(ntID)
	n.SetUrl(selfURL)
	if err := m.AddNode(n); err != nil {
		return fmt.Errorf("selfnode: add Node: %w", err)
	}
	return nil
}
