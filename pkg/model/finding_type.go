package model

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

type FindingType interface {
	GetFindingTypeId() uuid.UUID

	GetDisplayName() string
	SetDisplayName(string)

	GetDescription() string
	SetDescription(s string)

	GetAnnotations() Annotations

	getData() *findingTypeData
}

type findingTypeData struct {
	model        *modelData
	isRegistered bool

	FindingTypeId uuid.UUID
	DisplayName   string
	Description   string
	Annotations   Annotations
}

func NewFindingType(model Model, id uuid.UUID) FindingType {
	retval := &findingTypeData{
		model:         model.getData(),
		isRegistered:  false,
		FindingTypeId: id,
	}

	retval.Annotations = NewAnnotations(model.getData(), retval)

	return retval
}

func (f *findingTypeData) getData() *findingTypeData {
	return f
}

// GetDescription implements [FindingType].
func (f *findingTypeData) GetDescription() string {
	return f.Description
}

// GetDisplayName implements [FindingType].
func (f *findingTypeData) GetDisplayName() string {
	return f.DisplayName
}

// GetFindingTypeId implements [FindingType].
func (f *findingTypeData) GetFindingTypeId() uuid.UUID {
	return f.FindingTypeId
}

// SetDescription implements [FindingType].
func (f *findingTypeData) SetDescription(s string) {
	f.Description = s

	if f.isRegistered {
		f.model.sink.Receive(events.FindingTypeResource, events.UpdateOperation, f.FindingTypeId, f)
	}
}

// SetDisplayName implements [FindingType].
func (f *findingTypeData) SetDisplayName(s string) {
	f.DisplayName = s

	if f.isRegistered {
		f.model.sink.Receive(events.FindingTypeResource, events.UpdateOperation, f.FindingTypeId, f)
	}
}

// GetAnnotations implements [FindingType].
func (f *findingTypeData) GetAnnotations() Annotations {
	return f.Annotations
}

// Receive implements [EventSink].
func (f *findingTypeData) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, object ...any) error {
	if resType != events.AnnotationsResource {
		return fmt.Errorf("unsupported resource type %v in FindingType event sink. Only Annotations are supported", resType)
	}

	// all changes to Annotations result in UpdateOperation on FindingType
	if f.isRegistered {
		f.model.sink.Receive(events.FindingTypeResource, events.UpdateOperation, f.FindingTypeId, f)
	}

	return nil
}
