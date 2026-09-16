package c4injector

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/node"
)

// NodeTypeName is the EmELand NodeType display name for this Injector.
const NodeTypeName = "c4-doc-injector"

// nodeTypeNamespace derives a stable NodeType UUID for c4-doc-injector.
var nodeTypeNamespace = uuid.MustParse("b1c2d3e4-f5a6-7890-abcd-ef1234567890")

// NodeTypeID returns the deterministic NodeType UUID for c4-doc-injector.
func NodeTypeID() uuid.UUID {
	return uuid.NewSHA1(nodeTypeNamespace, []byte(NodeTypeName))
}

// RegisterNode ensures the c4-doc-injector NodeType and a process Node exist
// in the landscape. nodeID should be unique per process (e.g. uuid.New()).
func RegisterNode(m model.Model, nodeID uuid.UUID, displayName string) error {
	ntID := NodeTypeID()
	if m.GetNodeTypeById(ntID) == nil {
		nt := node.NewNodeType(ntID)
		nt.SetDisplayName(NodeTypeName)
		nt.SetDescription("Injector that emits C4-PlantUML diagrams from landscape resources and serves them over HTTP.")
		if err := m.AddNodeType(nt); err != nil {
			return err
		}
	}

	if m.GetNodeById(nodeID) != nil {
		return nil
	}
	n := node.NewNode(nodeID)
	if displayName == "" {
		displayName = NodeTypeName
	}
	n.SetDisplayName(displayName)
	n.SetDescription("In-process C4-PlantUML document Injector.")
	n.SetNodeTypeById(ntID)
	return m.AddNode(n)
}
