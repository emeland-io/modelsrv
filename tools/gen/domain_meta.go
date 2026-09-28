package main

// dirDomainMeta maps a resource's model sub-package directory to its Go import path and
// alias, used to qualify domain types in generated code. Extend when adding a new model
// sub-package.
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

// enrichDomainMeta fills the domain-package fields of a TypeSpec from its Dir.
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
