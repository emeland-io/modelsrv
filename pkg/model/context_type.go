package model

//go:generate mockgen -destination=../mocks/mock_context_type.go -package=mocks . ContextType

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

type ContextType interface {
	GetContextTypeId() uuid.UUID

	GetDisplayName() string
	SetDisplayName(string)

	GetDescription() string
	SetDescription(s string)

	GetAnnotations() Annotations

	Register() bool
}

type contextTypeData struct {
	sink         events.EventSink
	isRegistered bool

	ContextTypeId uuid.UUID
	DisplayName   string
	Description   string
	Annotations   Annotations
}

func NewContextType(sink events.EventSink, id uuid.UUID) ContextType {
	retval := &contextTypeData{
		sink:          sink,
		isRegistered:  false,
		ContextTypeId: id,
	}

	retval.Annotations = NewAnnotations(retval)

	return retval
}

func (c *contextTypeData) Register() bool {
	c.isRegistered = true
	return true
}

// GetAnnotations implements [ContextType].
func (c *contextTypeData) GetAnnotations() Annotations {
	return c.Annotations
}

// GetContextTypeId implements [ContextType].
func (c *contextTypeData) GetContextTypeId() uuid.UUID {
	return c.ContextTypeId
}

// GetDisplayName implements [ContextType].
func (c *contextTypeData) GetDisplayName() string {
	return c.DisplayName
}

// SetDisplayName implements [ContextType].
func (c *contextTypeData) SetDisplayName(name string) {
	c.DisplayName = name

	if c.isRegistered {
		c.sink.Receive(events.ContextTypeResource, events.UpdateOperation, c.ContextTypeId, c)
	}
}

// GetDescription implements [ContextType].
func (c *contextTypeData) GetDescription() string {
	return c.Description
}

// SetDescription implements [ContextType]
func (c *contextTypeData) SetDescription(s string) {
	c.Description = s

	if c.isRegistered {
		c.sink.Receive(events.ContextTypeResource, events.UpdateOperation, c.ContextTypeId, c)
	}
}

func (c *contextTypeData) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, object ...any) error {
	if resType != events.AnnotationsResource {
		return fmt.Errorf("unsupported resource type %v in ContextType event sink. Only Annotations are supported", resType)
	}

	if c.isRegistered {
		return c.sink.Receive(events.ContextTypeResource, events.UpdateOperation, c.ContextTypeId, c)
	}

	return nil
}
