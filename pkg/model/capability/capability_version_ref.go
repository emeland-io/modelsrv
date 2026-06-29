package capability

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model/common"
)

// CapabilityVersionRef is a reference to a CapabilityVersion resource (future phase 3 resource type).
type CapabilityVersionRef struct {
	CapabilityVersionId uuid.UUID      `json:"capabilityVersionId"`
	Version             common.Version `json:"version"`
}
