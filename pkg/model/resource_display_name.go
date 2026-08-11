package model

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model/common"
)

// ResourceExists reports whether the referenced resource is registered in the model.
// Returns false when m or ref is nil, the id is unknown, or the type has no handler.
func ResourceExists(m Model, ref *common.ResourceRef) bool {
	if m == nil || ref == nil || ref.ResourceId == uuid.Nil {
		return false
	}
	h, ok := lookupHandler(ref.ResourceType)
	if !ok || h.exists == nil {
		return false
	}
	return h.exists(m, ref.ResourceId)
}

// ResourceDisplayName resolves the human-readable name of a referenced resource
// when it is registered in the model. Returns empty string when the resource is
// missing or the type has no handler.
func ResourceDisplayName(m Model, ref *common.ResourceRef) string {
	if m == nil || ref == nil {
		return ""
	}
	h, ok := lookupHandler(ref.ResourceType)
	if !ok || h.displayName == nil {
		return ""
	}
	return h.displayName(m, ref.ResourceId)
}
