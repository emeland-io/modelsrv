package model

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

type Node interface {
	GetNodeId() uuid.UUID

	GetDisplayName() string
	SetDisplayName(string)

	GetDescription() string
	SetDescription(s string)

	GetNodeType() (NodeType, error)
	SetNodeTypeByRef(nodeType NodeType)
	SetNodeTypeById(nodeTypeId uuid.UUID)

	GetAnnotations() Annotations

	getData() *nodeData
}

type nodeData struct {
	model        *modelData
	isRegistered bool

	nodeId      uuid.UUID
	displayName string
	description string

	typeRef *NodeTypeRef

	Annotations Annotations
}

// ensure Node interface is implemented correctly
var _ Node = (*nodeData)(nil)

// ensure events.EventSink interface is implemented correctly
var _ events.EventSink = (*nodeData)(nil)

func NewNode(model Model, id uuid.UUID) Node {
	retval := &nodeData{
		model:        model.getData(),
		isRegistered: false,
		nodeId:       id,
	}

	retval.Annotations = NewAnnotations(model.getData(), retval)

	return retval
}

func (n *nodeData) getData() *nodeData {
	return n
}

// Receive implements [events.EventSink].
func (n *nodeData) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, object ...any) error {
	if resType != events.AnnotationsResource {
		return fmt.Errorf("unsupported resource type %v in System event sink. Only Annotations are supported", resType)
	}

	// all changes to annotations are automatically reflected in the parent object as updates
	if n.isRegistered {
		n.model.sink.Receive(events.NodeResource, events.UpdateOperation, n.nodeId, n)
	}

	return nil

}

// GetAnnotations implements [Node].
func (n *nodeData) GetAnnotations() Annotations {
	return n.Annotations
}

// GetDescription implements [Node].
func (n *nodeData) GetDescription() string {
	return n.description
}

// GetDisplayName implements [Node].
func (n *nodeData) GetDisplayName() string {
	return n.displayName
}

// GetNodeId implements [Node].
func (n *nodeData) GetNodeId() uuid.UUID {
	return n.nodeId
}

// SetDescription implements [Node].
func (n *nodeData) SetDescription(s string) {
	n.description = s

	if n.isRegistered {
		n.model.sink.Receive(events.NodeResource, events.UpdateOperation, n.nodeId, n)
	}
}

// SetDisplayName implements [Node].
func (n *nodeData) SetDisplayName(name string) {
	n.displayName = name

	if n.isRegistered {
		n.model.sink.Receive(events.NodeResource, events.UpdateOperation, n.nodeId, n)
	}
}

// GetNodeType implements [Node].
func (n *nodeData) GetNodeType() (NodeType, error) {
	if n.typeRef == nil {
		return nil, nil
	}

	if n.typeRef.NodeType != nil {
		return n.typeRef.NodeType, nil
	}

	nodeType, ok := n.model.nodeTypesByUUID[n.typeRef.NodeTypeId]
	if !ok {
		return nil, NodeTypeNotFoundError
	}
	n.typeRef.NodeType = nodeType

	return nodeType, nil
}

// SetNodeTypeById implements [Node].
func (n *nodeData) SetNodeTypeById(nodeTypeId uuid.UUID) {
	n.typeRef = &NodeTypeRef{
		NodeTypeId: nodeTypeId,
	}

	ptr, ok := n.model.nodeTypesByUUID[nodeTypeId]
	if ok {
		n.typeRef.NodeType = ptr
	}
	if n.isRegistered {
		n.model.sink.Receive(events.NodeResource, events.UpdateOperation, n.nodeId, n)
	}
}

// SetNodeTypeByRef implements [Node].
func (n *nodeData) SetNodeTypeByRef(nodeType NodeType) {
	if nodeType == nil {
		return
	}

	n.typeRef = &NodeTypeRef{
		NodeType:   nodeType.getData(),
		NodeTypeId: nodeType.GetNodeTypeId(),
	}
	if n.isRegistered {
		n.model.sink.Receive(events.NodeResource, events.UpdateOperation, n.nodeId, n)
	}
}

// ##### Model Methods #####

// AddNode implements [Model].
func (m *modelData) AddNode(node Node) error {
	if node.GetNodeId() == uuid.Nil {
		return UUIDNotSetError
	}

	op := events.CreateOperation

	//check if this would overwrite an existing entry -> an update
	if _, ok := m.nodesByUUID[node.GetNodeId()]; ok {
		op = events.UpdateOperation
	}

	m.sink.Receive(events.NodeResource, op, node.GetNodeId(), node)

	m.nodesByUUID[node.GetNodeId()] = node

	// mark Node as registered to activate sending events when updating fields
	node.getData().isRegistered = true

	return nil
}

// DeleteNodeById implements [Model].
func (m *modelData) DeleteNodeById(id uuid.UUID) error {
	_, exists := m.nodesByUUID[id]
	if !exists {
		return NodeNotFoundError
	}

	delete(m.nodesByUUID, id)

	m.sink.Receive(events.NodeResource, events.DeleteOperation, id)

	return nil
}

// GetNodeById implements [Model].
func (m *modelData) GetNodeById(id uuid.UUID) Node {
	node, exists := m.nodesByUUID[id]
	if !exists {
		return nil
	}
	return node
}

// GetNodes implements [Model].
func (m *modelData) GetNodes() ([]Node, error) {
	nodeArr := slices.Collect(maps.Values(m.nodesByUUID))
	return nodeArr, nil
}
