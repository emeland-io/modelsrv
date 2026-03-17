package model

//go:generate mockgen -destination=../mocks/mock_node_type.go -package=mocks . NodeType

import (
	"fmt"
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

	Register() bool
}

type nodeTypeData struct {
	sink         events.EventSink
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

func NewNodeType(sink events.EventSink, id uuid.UUID) NodeType {
	retval := &nodeTypeData{
		sink:         sink,
		isRegistered: false,
		NodeTypeId:   id,
	}

	retval.Annotations = NewAnnotations(retval)

	return retval
}

func (n *nodeTypeData) Register() bool {
	n.isRegistered = true
	return true
}

// Receive implements [events.EventSink].
func (n *nodeTypeData) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, object ...any) error {
	if resType != events.AnnotationsResource {
		return fmt.Errorf("unsupported resource type %v in NodeType event sink. Only Annotations are supported", resType)
	}
	if n.isRegistered {
		return n.sink.Receive(events.NodeTypeResource, events.UpdateOperation, n.NodeTypeId, n)
	}
	return nil
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
		n.sink.Receive(events.NodeTypeResource, events.UpdateOperation, n.NodeTypeId, n)
	}
}

// SetDisplayName implements [NodeType].
func (n *nodeTypeData) SetDisplayName(s string) {
	n.DisplayName = s

	if n.isRegistered {
		n.sink.Receive(events.NodeTypeResource, events.UpdateOperation, n.NodeTypeId, n)
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
	nodeType.Register()

	return nil
}

// GetNodeTypes implements [Model].
func (m *modelData) GetNodeTypes() ([]NodeType, error) {
	nodeTypeArr := slices.Collect(maps.Values(m.nodeTypesByUUID))
	return nodeTypeArr, nil
}

// GetNodeTypeById implements [Model].
func (m *modelData) GetNodeTypeById(id uuid.UUID) NodeType {
	nodeType, exists := m.nodeTypesByUUID[id]
	if !exists {
		return nil
	}
	return nodeType
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
