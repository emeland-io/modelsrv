package model

import (
	"maps"
	"slices"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

type NodeType interface {
	GetNodeTypeId() uuid.UUID

	GetDisplayName() string
	SetDisplayName(name string)

	GetDescription() string
	SetDescription(desc string)

	GetAnnotations() Annotations

	getData() *nodeTypeData
}

type nodeTypeData struct {
	model        *modelData
	isRegistered bool

	NodeTypeId  uuid.UUID
	DisplayName string
	Description string

	Annotations Annotations
}

type NodeTypeRef struct {
	NodeType   NodeType
	NodeTypeId uuid.UUID
}

// ensure Node interface is implemented correctly
var _ NodeType = (*nodeTypeData)(nil)

// ensure events.EventSink interface is implemented correctly
var _ events.EventSink = (*nodeTypeData)(nil)

func NewNodeType(model Model, id uuid.UUID) NodeType {
	retval := &nodeTypeData{
		model:        model.getData(),
		isRegistered: false,
		NodeTypeId:   id,
	}

	retval.Annotations = NewAnnotations(model.getData(), retval)

	return retval
}

func (n *nodeTypeData) getData() *nodeTypeData {
	return n
}

// Receive implements [events.EventSink].
func (n *nodeTypeData) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, object ...any) error {
	panic("unimplemented")
}

// GetAnnotations implements [NodeType].
func (n *nodeTypeData) GetAnnotations() Annotations {
	return n.Annotations
}

// GetDescription implements [NodeType].
func (n *nodeTypeData) GetDescription() string {
	return n.Description
}

// GetDisplayName implements [NodeType].
func (n *nodeTypeData) GetDisplayName() string {
	return n.DisplayName
}

// GetNodeTypeId implements [NodeType].
func (n *nodeTypeData) GetNodeTypeId() uuid.UUID {
	return n.NodeTypeId
}

// SetDescription implements [NodeType].
func (n *nodeTypeData) SetDescription(s string) {
	n.Description = s

	if n.isRegistered {
		n.model.sink.Receive(events.NodeTypeResource, events.UpdateOperation, n.NodeTypeId, n)
	}
}

// SetDisplayName implements [NodeType].
func (n *nodeTypeData) SetDisplayName(s string) {
	n.DisplayName = s

	if n.isRegistered {
		n.model.sink.Receive(events.NodeTypeResource, events.UpdateOperation, n.NodeTypeId, n)
	}

}

// ############# Model Methods #############

// AddNodeType implements [Model].
func (m *modelData) AddNodeType(nodeType NodeType) error {
	if nodeType.GetNodeTypeId() == uuid.Nil {
		return UUIDNotSetError
	}

	op := events.CreateOperation

	//check if this would overwrite an existing entry -> an update
	if _, ok := m.nodeTypesByUUID[nodeType.GetNodeTypeId()]; ok {
		op = events.UpdateOperation
	}

	m.sink.Receive(events.NodeTypeResource, op, nodeType.GetNodeTypeId(), nodeType)

	m.nodeTypesByUUID[nodeType.GetNodeTypeId()] = nodeType

	// mark NodeType as registered to activate sending events when updating fields
	nodeType.getData().isRegistered = true

	return nil
}

// GetNodeTypes implements [Model].
func (m *modelData) GetNodeTypes() ([]NodeType, error) {
	nodeTypeArr := slices.Collect(maps.Values(m.nodeTypesByUUID))
	return nodeTypeArr, nil

}

// DeleteNodeTypeById implements [Model].
func (m *modelData) DeleteNodeTypeById(id uuid.UUID) error {
	_, exists := m.nodeTypesByUUID[id]
	if !exists {
		return NodeTypeNotFoundError
	}

	delete(m.nodeTypesByUUID, id)

	m.sink.Receive(events.NodeTypeResource, events.DeleteOperation, id)

	return nil
}

// GetNodeTypeById implements [Model].
func (m *modelData) GetNodeTypeById(id uuid.UUID) NodeType {
	instance, exists := m.nodeTypesByUUID[id]
	if !exists {
		return nil
	}

	return instance
}
