package model

import (
	"iter"
	"maps"

	"github.com/google/uuid"
	"gitlab.com/emeland/modelsrv/pkg/events"
)

// ensure Context interface is implemented correctly
var _ Context = (*contextData)(nil)

type Context interface {
	GetContextId() uuid.UUID

	GetDisplayName() string
	SetDisplayName(string)

	SetDescription(s string)
	GetDescription() string

	GetParent() (Context, error)
	SetParentByRef(parent *Context)
	SetParentById(parentId uuid.UUID)

	AddAnnotation(key string, value string)
	DeleteAnnotation(key string)
	GetAnnotationValue(key string) string
	GetAnnotationKeys() iter.Seq[string]
	getData() *contextData
}

type contextData struct {
	model        *modelData
	isRegistered bool

	ContextId   uuid.UUID
	DisplayName string
	Description string
	Parent      *ContextRef
	Annotations map[string]string
}

type ContextRef struct {
	Context   *contextData
	ContextId uuid.UUID
}

func NewContext(model Model, id uuid.UUID) Context {
	return &contextData{
		model:        model.getData(),
		isRegistered: false,
		ContextId:    id,
	}
}

func (c *contextData) getData() *contextData {
	return c
}

// AddAnnotation implements [Context].
func (c *contextData) AddAnnotation(key string, value string) {
	c.Annotations[key] = value
}

// DeleteAnnotation implements [Context].
func (c *contextData) DeleteAnnotation(key string) {
	delete(c.Annotations, key)
}

// GetAnnotationKeys implements [Context].
func (c *contextData) GetAnnotationKeys() iter.Seq[string] {
	return maps.Keys(c.Annotations)
}

// GetAnnotationValue implements [Context].
func (c *contextData) GetAnnotationValue(key string) string {
	return c.Annotations[key]
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
	panic("unimplemented")
}

// SetParentByRef implements [Context].
func (c *contextData) SetParentByRef(parent *Context) {
	panic("unimplemented")
}
