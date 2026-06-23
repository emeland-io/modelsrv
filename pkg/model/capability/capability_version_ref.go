package capability

import "github.com/google/uuid"

// CapabilityVersionRef is a reference to a CapabilityVersion resource (future phase 3 resource type).
type CapabilityVersionRef struct {
	CapabilityVersionId uuid.UUID `json:"capabilityVersionId"`
}
