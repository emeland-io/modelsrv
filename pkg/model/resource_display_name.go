package model

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/common"
)

type displayNamed interface {
	GetDisplayName() string
}

func lookupDisplayName[T displayNamed](get func(uuid.UUID) T, id uuid.UUID) string {
	if v := get(id); any(v) != nil {
		return v.GetDisplayName()
	}
	return ""
}

// ResourceDisplayName resolves the human-readable name of a referenced resource
// when it is registered in the model. Returns empty string when the resource is
// missing or the type is not supported.
func ResourceDisplayName(m Model, ref *common.ResourceRef) string {
	if m == nil || ref == nil {
		return ""
	}
	id := ref.ResourceId
	switch ref.ResourceType {
	case events.ContextResource:
		return lookupDisplayName(m.GetContextById, id)
	case events.ContextTypeResource:
		return lookupDisplayName(m.GetContextTypeById, id)
	case events.NodeResource:
		return lookupDisplayName(m.GetNodeById, id)
	case events.NodeTypeResource:
		return lookupDisplayName(m.GetNodeTypeById, id)
	case events.SystemResource:
		return lookupDisplayName(m.GetSystemById, id)
	case events.SystemInstanceResource:
		return lookupDisplayName(m.GetSystemInstanceById, id)
	case events.APIResource:
		return lookupDisplayName(m.GetApiById, id)
	case events.APIInstanceResource:
		return lookupDisplayName(m.GetApiInstanceById, id)
	case events.ComponentResource:
		return lookupDisplayName(m.GetComponentById, id)
	case events.ComponentInstanceResource:
		return lookupDisplayName(m.GetComponentInstanceById, id)
	case events.OrgUnitResource:
		return lookupDisplayName(m.GetOrgUnitById, id)
	case events.GroupResource:
		return lookupDisplayName(m.GetGroupById, id)
	case events.IdentityResource:
		return lookupDisplayName(m.GetIdentityById, id)
	case events.PermissionSpecResource:
		return lookupDisplayName(m.GetPermissionSpecById, id)
	case events.RoleSpecResource:
		return lookupDisplayName(m.GetRoleSpecById, id)
	case events.PermissionResource:
		return lookupDisplayName(m.GetPermissionById, id)
	case events.RoleResource:
		return lookupDisplayName(m.GetRoleById, id)
	case events.BindingResource:
		return lookupDisplayName(m.GetBindingById, id)
	case events.ProductResource:
		return lookupDisplayName(m.GetProductById, id)
	case events.FindingResource:
		return lookupDisplayName(m.GetFindingById, id)
	case events.FindingTypeResource:
		return lookupDisplayName(m.GetFindingTypeById, id)
	case events.ArtifactResource:
		return lookupDisplayName(m.GetArtifactById, id)
	case events.ArtifactInstanceResource:
		return lookupDisplayName(m.GetArtifactInstanceById, id)
	case events.FilterRuleResource:
		return lookupDisplayName(m.GetFilterRuleById, id)
	case events.MergeRuleResource:
		return lookupDisplayName(m.GetMergeRuleById, id)
	default:
		return ""
	}
}
