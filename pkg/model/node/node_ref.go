package node

import "github.com/google/uuid"

// NodeTypeRef references a [NodeType] by resolved object and/or id.
type NodeTypeRef struct {
	NodeType   NodeType
	NodeTypeId uuid.UUID
}

// ResolvedNodeType returns the embedded [NodeType] when present, or nil.
func (r *NodeTypeRef) ResolvedNodeType() NodeType {
	if r == nil {
		return nil
	}
	return r.NodeType
}
