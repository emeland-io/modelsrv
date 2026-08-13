package model

import (
	"log"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

type upsertHandler func(m Model, obj any) error
type deleteHandler func(m Model, id uuid.UUID) error
type notFoundCheck func(err error) bool
type existsHandler func(m Model, id uuid.UUID) bool
type displayNameHandler func(m Model, id uuid.UUID) string

type resourceHandler struct {
	upsert      upsertHandler
	delete      deleteHandler
	notFound    notFoundCheck
	exists      existsHandler
	displayName displayNameHandler
}

var handlerRegistry = map[events.ResourceType]resourceHandler{}

func registerHandler(rt events.ResourceType, h resourceHandler) {
	if _, exists := handlerRegistry[rt]; exists {
		log.Printf("WARNING: handler already registered for resource type %s", rt)
		return
	}
	handlerRegistry[rt] = h
}

func lookupHandler(rt events.ResourceType) (resourceHandler, bool) {
	h, ok := handlerRegistry[rt]
	return h, ok
}
