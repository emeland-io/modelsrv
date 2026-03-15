package model

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

type Finding interface {
	GetFindingId() uuid.UUID

	GetDisplayName() string
	SetDisplayName(string)

	GetDescription() string
	SetDescription(s string)

	GetFindingType() FindingType
	SetFindingTypeById(id uuid.UUID)
	SetFindingTypeByRef(t FindingType)

	GetResources() []ResourceRef
	AddResource(events.ResourceType, uuid.UUID)
	RemoveResourceById(uuid.UUID) error

	GetAnnotations() Annotations

	getData() *findingData
}

type findingTypeRef struct {
	ref           FindingType
	findingTypeId uuid.UUID
}

type ResourceRef struct {
	ResourceId   uuid.UUID
	ResourceType events.ResourceType

	// TODO: extract the common interface for all resources: GetIt(), GetDisplayName(), GetDescription()
	// Use it here instead of the generic any type.
	ResourceRef any
}

type findingData struct {
	model        *modelData
	isRegistered bool

	findingId   uuid.UUID
	displayName string
	description string
	findingType *findingTypeRef
	resources   []ResourceRef
	annotations Annotations
}

var _ Finding = (*findingData)(nil)

func NewFinding(model Model, id uuid.UUID) Finding {
	retval := &findingData{
		model:        model.getData(),
		isRegistered: false,
		findingId:    id,
		findingType:  nil,
	}

	retval.annotations = NewAnnotations(model.getData(), retval)

	return retval
}

func (f *findingData) getData() *findingData {
	return f
}

// Receive implements [EventSink].
func (f *findingData) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, object ...any) error {
	if resType != events.AnnotationsResource {
		return fmt.Errorf("unsupported resource type %v in Finding event sink. Only Annotations are supported", resType)
	}

	// all changes to Annotations result in UpdateOperation on Finding
	if f.isRegistered {
		f.model.sink.Receive(events.FindingResource, events.UpdateOperation, f.findingId, f)
	}

	return nil
}

// GetDescription implements [Finding].
func (f *findingData) GetDescription() string {
	return f.description
}

// GetDisplayName implements [Finding].
func (f *findingData) GetDisplayName() string {
	return f.displayName
}

// GetFindingId implements [Finding].
func (f *findingData) GetFindingId() uuid.UUID {
	return f.findingId
}

// GetFindingType implements [Finding].
func (f *findingData) GetFindingType() FindingType {
	return f.findingType.ref
}

// GetResources implements [Finding].
func (f *findingData) GetResources() []ResourceRef {
	return f.resources
}

// SetDescription implements [Finding].
func (f *findingData) SetDescription(s string) {
	f.description = s

	if f.isRegistered {
		f.model.sink.Receive(events.FindingResource, events.UpdateOperation, f.findingId, f)
	}
}

// SetDisplayName implements [Finding].
func (f *findingData) SetDisplayName(s string) {
	f.displayName = s

	if f.isRegistered {
		f.model.sink.Receive(events.FindingResource, events.UpdateOperation, f.findingId, f)
	}
}

func (f *findingData) GetAnnotations() Annotations {
	return f.annotations
}

func (f *findingData) SetFindingTypeById(id uuid.UUID) {
	f.findingType = &findingTypeRef{
		findingTypeId: id,
	}

	if f.isRegistered {
		f.model.sink.Receive(events.FindingResource, events.UpdateOperation, f.findingId, f)
	}
}

func (f *findingData) SetFindingTypeByRef(t FindingType) {
	f.findingType = &findingTypeRef{
		ref:           t,
		findingTypeId: t.GetFindingTypeId(),
	}

	if f.isRegistered {
		f.model.sink.Receive(events.FindingResource, events.UpdateOperation, f.findingId, f)
	}
}

// ###### Model methods ######

// GetFindings implements Model.
func (m modelData) GetFindings() ([]Finding, error) {
	findingArr := slices.Collect(maps.Values(m.findingsByUUID))
	return findingArr, nil
}

// AddFinding implements Model.
func (m *modelData) AddFinding(finding Finding) error {
	if finding.GetFindingId() == uuid.Nil {
		return UUIDNotSetError
	}

	op := events.CreateOperation

	// check if this would overwrite an existing finding -> an update
	if _, ok := m.findingsByUUID[finding.GetFindingId()]; ok {
		op = events.UpdateOperation
	}
	// create event for finding creation
	m.sink.Receive(events.FindingResource, op, finding.GetFindingId(), finding)

	m.findingsByUUID[finding.GetFindingId()] = finding

	// mark as registered
	finding.getData().isRegistered = true

	return nil
}

// DeleteFindingById implements [Model].
func (m *modelData) DeleteFindingById(id uuid.UUID) error {
	_, exists := m.findingsByUUID[id]
	if !exists {
		return FindingNotFoundError
	}

	delete(m.findingsByUUID, id)

	m.sink.Receive(events.FindingResource, events.DeleteOperation, id)

	return nil
}

// GetFindingById implements Model.
func (m *modelData) GetFindingById(id uuid.UUID) Finding {
	finding, exists := m.findingsByUUID[id]
	if !exists {
		return nil
	}
	return finding
}

// AddResource implements [Finding].
func (f *findingData) AddResource(resType events.ResourceType, resId uuid.UUID) {
	f.resources = append(f.resources, ResourceRef{
		ResourceType: resType,
		ResourceId:   resId,
	})

	f.model.sink.Receive(events.FindingResource, events.UpdateOperation, f.findingId, f)
}

// RemoveResourceById implements [Finding].
func (f *findingData) RemoveResourceById(resId uuid.UUID) error {
	for i, r := range f.resources {
		if r.ResourceId == resId {
			f.resources = append(f.resources[:i], f.resources[i+1:]...)

			f.model.sink.Receive(events.FindingResource, events.UpdateOperation, f.findingId, f)
			return nil
		}
	}
	return FindingNotFoundError
}
