package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// allTypes is the resource metadata that drives all generation. It is built at package init
// from the annotated structs in testdata/model_defs.go (parsed by the loader) plus the
// typeWiring supplement, then ordered and enriched.
var allTypes = buildAllTypes()

// buildAllTypes loads the annotated resource structs, applies wiring, and enriches them.
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
	rank := make(map[string]int, len(order))
	for i, n := range order {
		rank[n] = i
	}
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

var dirDomainMeta = map[string]struct {
	Import string
	Alias  string
}{
	"context":       {Import: "go.emeland.io/modelsrv/pkg/model/context", Alias: "mdlctx"},
	"system":        {Import: "go.emeland.io/modelsrv/pkg/model/system", Alias: "system"},
	"api":           {Import: "go.emeland.io/modelsrv/pkg/model/api", Alias: "mdlapi"},
	"component":     {Import: "go.emeland.io/modelsrv/pkg/model/component", Alias: "component"},
	"node":          {Import: "go.emeland.io/modelsrv/pkg/model/node", Alias: "node"},
	"finding":       {Import: "go.emeland.io/modelsrv/pkg/model/finding", Alias: "finding"},
	"iam":           {Import: "go.emeland.io/modelsrv/pkg/model/iam", Alias: "iam"},
	"artifact":      {Import: "go.emeland.io/modelsrv/pkg/model/artifact", Alias: "artifact"},
	"product":       {Import: "go.emeland.io/modelsrv/pkg/model/product", Alias: "mdlprod"},
	"filterrule":    {Import: "go.emeland.io/modelsrv/pkg/model/filterrule", Alias: "mdlfilterrule"},
	"mergerule":     {Import: "go.emeland.io/modelsrv/pkg/model/mergerule", Alias: "mdlmergerule"},
	"capability":    {Import: "go.emeland.io/modelsrv/pkg/model/capability", Alias: "mdlcapability"},
	"parameter":     {Import: "go.emeland.io/modelsrv/pkg/model/parameter", Alias: "mdlparameter"},
	"capacity":      {Import: "go.emeland.io/modelsrv/pkg/model/capacity", Alias: "mdlcap"},
	"observability": {Import: "go.emeland.io/modelsrv/pkg/model/observability", Alias: "mdlobs"},
}

func enrichDomainMeta(spec *TypeSpec) {
	meta, ok := dirDomainMeta[spec.Dir]
	if !ok {
		return
	}
	spec.DomainPkgImport = meta.Import
	spec.DomainPkgAlias = meta.Alias
	spec.DomainTypeName = spec.Name
	spec.DomainIDGetter = "got.Get" + spec.IDField + "()"
	spec.DomainNameGetter = "got.GetDisplayName()"
}
