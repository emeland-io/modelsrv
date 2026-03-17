package model

//go:generate mockgen -destination=../mocks/mock_finding_type.go -package=mocks . FindingType

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

	Register() bool
}

type findingTypeData struct {
	sink         events.EventSink
	isRegistered bool

	FindingTypeId uuid.UUID
	DisplayName   string
	Description   string
	Annotations   Annotations
}

func NewFindingType(sink events.EventSink, id uuid.UUID) FindingType {
	retval := &findingTypeData{
		sink:          sink,
		isRegistered:  false,
		FindingTypeId: id,
	}

	retval.Annotations = NewAnnotations(retval)

	return retval
}

func (f *findingTypeData) Register() bool {
	f.isRegistered = true
	return true
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
		f.sink.Receive(events.FindingTypeResource, events.UpdateOperation, f.FindingTypeId, f)
	}
}

// SetDisplayName implements [FindingType].
func (f *findingTypeData) SetDisplayName(s string) {
	f.DisplayName = s

	if f.isRegistered {
		f.sink.Receive(events.FindingTypeResource, events.UpdateOperation, f.FindingTypeId, f)
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
		f.sink.Receive(events.FindingTypeResource, events.UpdateOperation, f.FindingTypeId, f)
	}

	return nil
}
