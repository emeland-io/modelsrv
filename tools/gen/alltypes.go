package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// allTypes is the resource metadata that drives all generation. It is built at package init
// from the annotated structs in tools/gen/modeldefs (parsed by the loader) plus the
// typeWiring supplement, then ordered and enriched.
var allTypes = buildAllTypes()

// buildAllTypes loads the annotated resource structs, applies wiring, orders them, and
// enriches them (domain + wire metadata).
func buildAllTypes() []TypeSpec {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "buildAllTypes: runtime.Caller failed")
		os.Exit(1)
	}
	defsDir := filepath.Join(filepath.Dir(thisFile), modelDefsDir)
	specs, err := loadTypeSpecs(defsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buildAllTypes: loading resource structs from %s: %v\n", defsDir, err)
		os.Exit(1)
	}
	// Order generated output by an explicit, author-controlled sequence (independent of the
	// loader's declaration/dir order) so the generated files have a stable layout.
	specs = orderTypes(specs, canonicalTypeOrder)
	for i := range specs {
		enrichDomainMeta(&specs[i])
		enrichWireMeta(&specs[i])
	}
	return specs
}

// canonicalTypeOrder is the order in which resource types appear in generated output.
// It is the historical order; new types can be appended.
var canonicalTypeOrder = []string{
	"ContextType", "Context", "System", "NodeType", "FindingType", "Node",
	"ApiInstance", "API", "Component", "SystemInstance", "ComponentInstance",
	"Finding", "OrgUnit", "Group", "Identity", "PermissionSpec", "RoleSpec",
	"Permission", "Role", "Binding", "Artifact", "ArtifactInstance", "Product",
	"FilterRule", "MergeRule", "Capability", "Parameter", "CapacityResourceType",
	"Capacity", "Metric", "Threshold", "MetricInstance", "MetricValue",
}

// orderTypes returns specs sorted by their index in order; any type not listed is appended
// in its original (loader) order after the listed ones.
func orderTypes(specs []TypeSpec, order []string) []TypeSpec {
	byName := make(map[string]TypeSpec, len(specs))
	for _, s := range specs {
		byName[s.Name] = s
	}
	out := make([]TypeSpec, 0, len(specs))
	for _, n := range order {
		if v, ok := byName[n]; ok {
			out = append(out, v)
			delete(byName, n)
		}
	}
	for _, s := range specs {
		if _, ok := byName[s.Name]; ok {
			out = append(out, s)
			delete(byName, s.Name)
		}
	}
	return out
}
