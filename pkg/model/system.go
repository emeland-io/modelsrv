package model

import (
	"fmt"

	"github.com/google/uuid"
	"gitlab.com/emeland/modelsrv/pkg/events"
)

// ensure System interface is implemented correctly
var _ System = (*systemData)(nil)

// ensure events.EventSink interface is implemented correctly
var _ events.EventSink = (*systemData)(nil)

type System interface {
	GetSystemId() uuid.UUID

	GetDisplayName() string
	SetDisplayName(name string)

	GetDescription() string
	SetDescription(desc string)

	GetVersion() Version
	SetVersion(ver Version)

	IsAbstract() bool
	SetAbstract(abs bool)

	GetParent() (System, error)
	SetParentByRef(parent System)
	SetParentById(parentId uuid.UUID)

	GetAnnotations() Annotations

	getData() *systemData
}

type systemData struct {
	model        *modelData
	isRegistered bool

	SystemId    uuid.UUID
	DisplayName string
	Description string

	Version  Version
	Abstract bool
	Parent   *SystemRef

	Annotations Annotations
}

type SystemRef struct {
	System    System
	SystemId  uuid.UUID
	SystemRef *EntityVersion
}

func NewSystem(model Model, id uuid.UUID) System {
	retval := &systemData{
		model:        model.getData(),
		isRegistered: false,
		SystemId:     id,
	}

	retval.Annotations = NewAnnotations(model.getData(), retval)

	return retval
}

func (s *systemData) getData() *systemData {
	return s
}

// GetAnnotations implements [System].
func (s *systemData) GetAnnotations() Annotations {
	return s.Annotations
}

// GetDescription implements [System].
func (s *systemData) GetDescription() string {
	return s.Description
}

// GetDisplayName implements [System].
func (s *systemData) GetDisplayName() string {
	return s.DisplayName
}

// GetParent implements [System].
func (s *systemData) GetParent() (System, error) {
	if s.Parent == nil {
		return nil, nil
	}
	if s.Parent.System != nil {
		return s.Parent.System, nil
	}

	parent, ok := s.model.systemsByUUID[s.Parent.SystemId]
	if !ok {
		return nil, SystemNotFoundError
	}
	s.Parent.System = parent

	return parent, nil
}

// GetSystemId implements [System].
func (s *systemData) GetSystemId() uuid.UUID {
	return s.SystemId
}

// GetVersion implements [System].
func (s *systemData) GetVersion() Version {
	return s.Version
}

// IsAbstract implements [System].
func (s *systemData) IsAbstract() bool {
	return s.Abstract
}

// SetAbstract implements [System].
func (s *systemData) SetAbstract(abs bool) {
	s.Abstract = abs

	if s.isRegistered {
		s.model.sink.Receive(events.SystemResource, events.UpdateOperation, s.SystemId, s)
	}
}

// SetDescription implements [System].
func (s *systemData) SetDescription(desc string) {
	s.Description = desc

	if s.isRegistered {
		s.model.sink.Receive(events.SystemResource, events.UpdateOperation, s.SystemId, s)
	}
}

// SetDisplayName implements [System].
func (s *systemData) SetDisplayName(name string) {
	s.DisplayName = name

	if s.isRegistered {
		s.model.sink.Receive(events.SystemResource, events.UpdateOperation, s.SystemId, s)
	}
}

// SetParentById implements [System].
func (s *systemData) SetParentById(parentId uuid.UUID) {
	s.Parent = &SystemRef{
		SystemId: parentId,
	}

	ptr, ok := s.model.systemsByUUID[parentId]
	if ok {
		s.Parent.System = ptr
	}

	if s.isRegistered {
		s.model.sink.Receive(events.SystemResource, events.UpdateOperation, s.SystemId, s)
	}
}

// SetParentByRef implements [System].
func (s *systemData) SetParentByRef(parent System) {
	if parent == nil {
		return
	}

	s.Parent = &SystemRef{
		System:   parent.getData(),
		SystemId: parent.GetSystemId(),
	}
	if s.isRegistered {
		s.model.sink.Receive(events.SystemResource, events.UpdateOperation, s.SystemId, s)
	}
}

// SetVersion implements [System].
func (s *systemData) SetVersion(ver Version) {
	s.Version = ver

	if s.isRegistered {
		s.model.sink.Receive(events.SystemResource, events.UpdateOperation, s.SystemId, s)
	}
}

// Receive implements [events.EventSink].
func (s *systemData) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, object ...any) error {
	if resType != events.AnnotationsResource {
		return fmt.Errorf("unsupported resource type %v in System event sink. Only Annotations are supported", resType)
	}

	// all changes to annotations are automatically reflected in the parent object as updates
	if s.isRegistered {
		s.model.sink.Receive(events.SystemResource, events.UpdateOperation, s.SystemId, s)
	}

	return nil
}

/*
func convertSystemToDTO(ctx Context) systemDTO {
	sysDTO := systemDTO{
		SystemId:    ctx.GetSystemId(),
		DisplayName: ctx.GetDisplayName(),
		Description: ctx.GetDescription(),
		Annotations: convertAnnotationsToDTO(ctx.GetAnnotations()),
	}

	return sysDTO
}

*/

func MakeTestSystem(testModel Model, id uuid.UUID, name string, version Version) System {
	system := NewSystem(testModel, id)
	system.SetDisplayName(name)

	system.SetDescription("a test system")
	system.SetVersion(version)

	return system
}
