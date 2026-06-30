package model

// ResourceTypeInfo describes a resource type and provides a function to count its instances.
type ResourceTypeInfo struct {
	Name  string
	Count func() (int, error)
}

// ResourceTypes returns all resource types known to the model with their count functions.
// This is the single source of truth — metrics, handlers, and other consumers should
// derive their resource-type lists from here.
func ResourceTypes(m Model) []ResourceTypeInfo {
	return []ResourceTypeInfo{
		{"Node", countFunc(m.GetNodes)},
		{"NodeType", countFunc(m.GetNodeTypes)},
		{"Context", countFunc(m.GetContexts)},
		{"ContextType", countFunc(m.GetContextTypes)},
		{"System", countFunc(m.GetSystems)},
		{"SystemInstance", countFunc(m.GetSystemInstances)},
		{"API", countFunc(m.GetApis)},
		{"ApiInstance", countFunc(m.GetApiInstances)},
		{"Component", countFunc(m.GetComponents)},
		{"ComponentInstance", countFunc(m.GetComponentInstances)},
		{"Finding", countFunc(m.GetFindings)},
		{"FindingType", countFunc(m.GetFindingTypes)},
		{"Artifact", countFunc(m.GetArtifacts)},
		{"ArtifactInstance", countFunc(m.GetArtifactInstances)},
		{"OrgUnit", countFunc(m.GetOrgUnits)},
		{"Group", countFunc(m.GetGroups)},
		{"Identity", countFunc(m.GetIdentities)},
		{"PermissionSpec", countFunc(m.GetPermissionSpecs)},
		{"RoleSpec", countFunc(m.GetRoleSpecs)},
		{"Permission", countFunc(m.GetPermissions)},
		{"Role", countFunc(m.GetRoles)},
		{"Binding", countFunc(m.GetBindings)},
		{"Product", countFunc(m.GetProducts)},
		{"FilterRule", countFunc(m.GetFilterRules)},
		{"MergeRule", countFunc(m.GetMergeRules)},
		{"Capability", countFunc(m.GetCapabilities)},
		{"Parameter", countFunc(m.GetParameters)},
	}
}

func countFunc[T any](fn func() ([]T, error)) func() (int, error) {
	return func() (int, error) {
		items, err := fn()
		return len(items), err
	}
}
