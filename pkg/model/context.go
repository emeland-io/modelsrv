package model

import (
	"fmt"

	"github.com/google/uuid"
	"gitlab.com/emeland/modelsrv/pkg/events"
)

// ensure Context interface is implemented correctly
var _ Context = (*contextData)(nil)

// ensure events.EventSink interface is implemented correctly
var _ events.EventSink = (*contextData)(nil)

type Context interface {
	GetContextId() uuid.UUID

	GetDisplayName() string
	SetDisplayName(string)

	SetDescription(s string)
	GetDescription() string

	GetParent() (Context, error)
	SetParentByRef(parent Context)
	SetParentById(parentId uuid.UUID)

	GetAnnotations() Annotations

	getData() *contextData
}

type contextData struct {
	model        *modelData
	isRegistered bool

	ContextId   uuid.UUID
	DisplayName string
	Description string
	Parent      *ContextRef
	Annotations Annotations
}

type ContextRef struct {
	Context   *contextData
	ContextId uuid.UUID
}

func NewContext(model Model, id uuid.UUID) Context {
	retval := &contextData{
		model:        model.getData(),
		isRegistered: false,
		ContextId:    id,
	}

	retval.Annotations = NewAnnotations(model.getData(), retval)

	return retval
}

func (c *contextData) getData() *contextData {
	return c
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
		c.model.sink.Receive(events.ContextResource, events.UpdateOperation, c.ContextId, c)
	}
}

// GetDisplayName implements [Context].
func (c *contextData) GetDisplayName() string {
	return c.DisplayName
}

func (c *contextData) SetDisplayName(name string) {
	c.DisplayName = name

	if c.isRegistered {
		c.model.sink.Receive(events.ContextResource, events.UpdateOperation, c.ContextId, c)
	}

}

// GetParent implements [Context].
func (c *contextData) GetParent() (Context, error) {
	if c.Parent == nil {
		return nil, nil
	}
	if c.Parent.Context != nil {
		return c.Parent.Context, nil
	}

	parent, ok := c.model.contextsByUUID[c.Parent.ContextId]
	if !ok {
		return nil, ContextNotFoundError
	}
	c.Parent.Context = parent

	return parent, nil
}

// SetParentById implements [Context].
func (c *contextData) SetParentById(parentId uuid.UUID) {
	c.Parent = &ContextRef{
		ContextId: parentId,
	}

	ptr, ok := c.model.contextsByUUID[parentId]
	if ok {
		c.Parent.Context = ptr
	}

	if c.isRegistered {
		c.model.sink.Receive(events.ContextResource, events.UpdateOperation, c.ContextId, c)
	}
}

// SetParentByRef implements [Context].
func (c *contextData) SetParentByRef(parent Context) {

	if parent == nil {
		return
	}
	c.Parent = &ContextRef{
		Context:   parent.getData(),
		ContextId: parent.GetContextId(),
	}

	if c.isRegistered {
		c.model.sink.Receive(events.ContextResource, events.UpdateOperation, c.ContextId, c)
	}
}

// Receive implements [events.EventSink].
func (c *contextData) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, object ...any) error {
	if resType != events.AnnotationsResource {
		return fmt.Errorf("unsupported resource type %v in Context event sink. Only Annotations are supported", resType)
	}

	// all changes to annotations are automatically reflected in the parent object as updates
	if c.isRegistered {
		c.model.sink.Receive(events.ContextResource, events.UpdateOperation, c.ContextId, c)
	}

	return nil
}
