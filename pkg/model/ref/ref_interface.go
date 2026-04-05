package ref

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model/common"
)

type RefTarget interface {
	GetResourceId() uuid.UUID
	GetResourceName() string
}

type Ref[T RefTarget] struct {
	Resource    T
	ResourceID  uuid.UUID
	ResourceRef *common.EntityVersion
}
