package model

import (
	"log"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

type upsertHandler func(m Model, obj any) error
type deleteHandler func(m Model, id uuid.UUID) error
type notFoundCheck func(err error) bool

type resourceHandler struct {
	upsert   upsertHandler
	delete   deleteHandler
	notFound notFoundCheck
}

var handlerRegistry = map[events.ResourceType]resourceHandler{}

func registerHandler(rt events.ResourceType, h resourceHandler) {
	if _, exists := handlerRegistry[rt]; exists {
		log.Printf("WARNING: handler already registered for resource type %s", rt)
		return
	}
	handlerRegistry[rt] = h
}
