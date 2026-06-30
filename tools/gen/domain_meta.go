package main

func init() {
	for i := range allTypes {
		enrichDomainMeta(&allTypes[i])
		enrichWireMeta(&allTypes[i])
	}
}

var dirDomainMeta = map[string]struct {
	Import string
	Alias  string
}{
	"context":    {Import: "go.emeland.io/modelsrv/pkg/model/context", Alias: "mdlctx"},
	"system":     {Import: "go.emeland.io/modelsrv/pkg/model/system", Alias: "system"},
	"api":        {Import: "go.emeland.io/modelsrv/pkg/model/api", Alias: "mdlapi"},
	"component":  {Import: "go.emeland.io/modelsrv/pkg/model/component", Alias: "component"},
	"node":       {Import: "go.emeland.io/modelsrv/pkg/model/node", Alias: "node"},
	"finding":    {Import: "go.emeland.io/modelsrv/pkg/model/finding", Alias: "finding"},
	"iam":        {Import: "go.emeland.io/modelsrv/pkg/model/iam", Alias: "iam"},
	"artifact":   {Import: "go.emeland.io/modelsrv/pkg/model/artifact", Alias: "artifact"},
	"product":    {Import: "go.emeland.io/modelsrv/pkg/model/product", Alias: "mdlprod"},
	"filterrule": {Import: "go.emeland.io/modelsrv/pkg/model/filterrule", Alias: "mdlfilterrule"},
	"mergerule":  {Import: "go.emeland.io/modelsrv/pkg/model/mergerule", Alias: "mdlmergerule"},
	"capability": {Import: "go.emeland.io/modelsrv/pkg/model/capability", Alias: "mdlcapability"},
	"parameter":  {Import: "go.emeland.io/modelsrv/pkg/model/parameter", Alias: "mdlparameter"},
	"capacity":   {Import: "go.emeland.io/modelsrv/pkg/model/capacity", Alias: "mdlcap"},
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
