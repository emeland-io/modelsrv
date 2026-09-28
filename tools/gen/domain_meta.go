package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// allTypes is the resource metadata that drives all generation. It is built at package init
// from the annotated structs in testdata/model_defs.go (parsed by the loader) plus the
// typeWiring supplement, then enriched. TestLoaderMatchesHandWritten asserts this equals the
// retained handwrittenTypes oracle.
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
	// Preserve the hand-written declaration order (loader order is map/dir dependent).
	specs = sortTypesByHandwrittenOrder(specs)
	for i := range specs {
		enrichDomainMeta(&specs[i])
		enrichWireMeta(&specs[i])
	}
	return specs
}

// sortTypesByHandwrittenOrder orders loaded specs to match the handwrittenTypes order so
// generated output (which iterates allTypes) is byte-identical to the pre-refactor output.
func sortTypesByHandwrittenOrder(specs []TypeSpec) []TypeSpec {
	order := map[string]int{}
	for i, s := range handwrittenTypes {
		order[s.Name] = i
	}
	byName := map[string]TypeSpec{}
	for _, s := range specs {
		byName[s.Name] = s
	}
	out := make([]TypeSpec, 0, len(specs))
	// First, emit in hand-written order.
	for _, s := range handwrittenTypes {
		if v, ok := byName[s.Name]; ok {
			out = append(out, v)
			delete(byName, s.Name)
		}
	}
	// Append any loaded types not present in the oracle (should be none during migration).
	for _, s := range specs {
		if _, ok := byName[s.Name]; ok {
			out = append(out, s)
			delete(byName, s.Name)
		}
	}
	_ = order
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
