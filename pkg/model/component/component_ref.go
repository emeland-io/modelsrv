package component

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model/common"
)

// ComponentRef links to a [Component] by resolved object and/or id.
type ComponentRef struct {
	Component    Component
	ComponentId  uuid.UUID
	ComponentRef *common.EntityVersion
}
