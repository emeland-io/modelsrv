package main

type Field struct {
	Name           string
	Type           string
	IsRef          bool
	RefType        string
	MapKey         string
	HasAnnotations bool
	SkipAccessor   bool
}

type ParentLinkSpec struct {
	FieldName         string
	RefTypeName       string
	ResourceTypeName  string
	ResolvedMethod    string
	EffectiveIDMethod string
	EmbedFieldName    string
	RefIDFieldName    string
	ResourceIDGetter  string
	ModelLookupByID   string
	SetParentByID     bool
}

type TypeRefLinkSpec struct {
	FieldName        string
	RefTypeName      string
	ResourceTypeName string
	ResolvedMethod   string
	EmbedFieldName   string
	RefIDFieldName   string
	ResourceIDGetter string
	ModelLookupByID  string

	// EffectiveIDMethod is the method on RefTypeName that returns the effective type id
	// (e.g. "EffectiveContextTypeID"). When set, a Get<RefIDFieldName>() accessor is generated.
	EffectiveIDMethod string
	// SetByID, when true, generates a Set<ResourceTypeName>ById() helper.
	SetByID bool
	// SetByIDParamName is the Go parameter name used in the generated Set<ResourceTypeName>ById()
	// method (e.g. "contextTypeId"). Required when SetByID is true.
	SetByIDParamName string
}

type RefByRefSpec struct {
	MethodName       string
	ParamName        string
	ResourceTypeName string
	ParamGoType      string
	SetterName       string
	RefTypeName      string
	RefTypeGoType    string // e.g. system.SystemRef; empty means RefTypeName
	EmbedFieldName   string
	RefIDFieldName   string
	ResourceIDGetter string
	NilCheck         bool
}

type TypeSpec struct {
	Name          string
	Dir           string // output subdirectory under pkg/model (Go package name)
	EventType     string
	IDField       string
	NameField     string
	Fields        []Field
	CustomMethods []string
	ParentLink    *ParentLinkSpec
	TypeRefLink   *TypeRefLinkSpec
	RefByRefs     []RefByRefSpec
	ExtraImports  []string

	HasHandler          bool   // emit a handlers_<name>_gen.go file in pkg/model
	NotFoundErr         string // e.g. "ErrApiNotFound" from pkg/model/common
	HandlerPkgAlias     string // import alias for this type's package; empty → use Dir
	HandlerMethodSuffix string // override for Add*/Delete* method suffix; empty → use Name
}
