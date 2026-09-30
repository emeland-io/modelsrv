package main

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// allTypes is the resource metadata that drives all generation. It is built at package init
// from the annotated structs in tools/gen/modeldefs (parsed by the loader) plus the
// typeWiring supplement, then ordered and enriched.
var allTypes = mustBuildAllTypes()

// mustBuildAllTypes builds allTypes or panics with the underlying error. Used only by the
// package-level initializer; tests call buildAllTypes directly to assert on errors.
func mustBuildAllTypes() []TypeSpec {
	specs, err := buildAllTypes()
	if err != nil {
		panic(fmt.Sprintf("building resource types: %v", err))
	}
	return specs
}

// modelDefsPath returns the absolute path to the model-definitions directory, resolved
// relative to this source file (so it works under `go run`/`go generate`).
func modelDefsPath() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(thisFile), modelDefsDir), nil
}

// buildAllTypes loads the annotated resource structs, applies wiring, orders them, and
// enriches them (domain + wire metadata). It returns an error rather than exiting so the
// load/enrich path is testable.
func buildAllTypes() ([]TypeSpec, error) {
	defsDir, err := modelDefsPath()
	if err != nil {
		return nil, err
	}
	specs, err := loadTypeSpecs(defsDir)
	if err != nil {
		return nil, fmt.Errorf("loading resource structs from %s: %w", defsDir, err)
	}
	// Order generated output by an explicit, author-controlled sequence (independent of the
	// loader's declaration/dir order) so the generated files have a stable layout.
	specs = orderTypes(specs, canonicalTypeOrder)
	for i := range specs {
		if err := enrichDomainMeta(&specs[i]); err != nil {
			return nil, err
		}
		if err := enrichWireMeta(&specs[i]); err != nil {
			return nil, err
		}
	}
	return specs, nil
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
