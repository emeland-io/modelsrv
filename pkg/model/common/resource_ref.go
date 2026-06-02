package common

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

type ResourceRef struct {
	ResourceId   uuid.UUID
	ResourceType events.ResourceType
}

// InstanceListItem is a lightweight id/name/reference triple returned by client list methods.
type InstanceListItem struct {
	Id        uuid.UUID
	Name      string
	Reference string
}
