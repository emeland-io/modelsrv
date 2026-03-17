package model

//go:generate mockgen -destination=../mocks/mock_context.go -package=mocks . Context

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

// ensure Context interface is implemented correctly
var _ Context = (*contextData)(nil)

// ensure events.EventSink interface is implemented correctly
var _ events.EventSink = (*contextData)(nil)

type Context interface {
	GetContextId() uuid.UUID

	GetDisplayName() string
	SetDisplayName(string)

	GetDescription() string
	SetDescription(s string)

	GetParent() (Context, error)
	GetParentId() uuid.UUID
	SetParentByRef(parent Context)
	SetParentById(parentId uuid.UUID)

	GetAnnotations() Annotations

	Register() bool
}

type contextData struct {
	sink         events.EventSink
	isRegistered bool

	ContextId   uuid.UUID
	DisplayName string
	Description string
	Parent      *ContextRef
	Annotations Annotations
}

type ContextRef struct {
	Context   Context
	ContextId uuid.UUID
}

func NewContext(sink events.EventSink, id uuid.UUID) Context {
	retval := &contextData{
		sink:         sink,
		isRegistered: false,
		ContextId:    id,
	}

	retval.Annotations = NewAnnotations(retval)

	return retval
}

func (c *contextData) Register() bool {
	c.isRegistered = true

	return true
}

// GetAnnotations implements [Context].
func (c *contextData) GetAnnotations() Annotations {
	return c.Annotations
}

// GetContextId implements [Context].
func (c *contextData) GetContextId() uuid.UUID {
	return c.ContextId
}

// GetDescription implements [Context].
func (c *contextData) GetDescription() string {
	return c.Description
}

// SetDescription implements [Context]
func (c *contextData) SetDescription(s string) {
	c.Description = s

	if c.isRegistered {
		c.sink.Receive(events.ContextResource, events.UpdateOperation, c.ContextId, c)
	}
}

// GetDisplayName implements [Context].
func (c *contextData) GetDisplayName() string {
	return c.DisplayName
}

func (c *contextData) SetDisplayName(name string) {
	c.DisplayName = name

	if c.isRegistered {
		c.sink.Receive(events.ContextResource, events.UpdateOperation, c.ContextId, c)
	}
}

// GetParent implements [Context].
func (c *contextData) GetParent() (Context, error) {
	if c.Parent == nil || c.Parent.Context == nil {
		return nil, nil
	}
	return c.Parent.Context, nil
}

// GetParentId implements [Context].
func (c *contextData) GetParentId() uuid.UUID {
	if c.Parent == nil {
		return uuid.Nil
	}
	return c.Parent.ContextId
}

// SetParentById implements [Context].
func (c *contextData) SetParentById(parentId uuid.UUID) {
	c.Parent = &ContextRef{
		ContextId: parentId,
	}

	if c.isRegistered {
		c.sink.Receive(events.ContextResource, events.UpdateOperation, c.ContextId, c)
	}
}

// SetParentByRef implements [Context].
func (c *contextData) SetParentByRef(parent Context) {
	if parent == nil {
		return
	}
	c.Parent = &ContextRef{
		Context:   parent,
		ContextId: parent.GetContextId(),
	}

	if c.isRegistered {
		c.sink.Receive(events.ContextResource, events.UpdateOperation, c.ContextId, c)
	}
}

// Receive implements [events.EventSink].
func (c *contextData) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, object ...any) error {
	if resType != events.AnnotationsResource {
		return fmt.Errorf("unsupported resource type %v in Context event sink. Only Annotations are supported", resType)
	}

	// all changes to annotations are automatically reflected in the parent object as updates
	if c.isRegistered {
		return c.sink.Receive(events.ContextResource, events.UpdateOperation, c.ContextId, c)
	}

	return nil
}
