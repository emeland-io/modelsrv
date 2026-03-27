package common

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

type ResourceRef struct {
	ResourceId   uuid.UUID
	ResourceType events.ResourceType
}
