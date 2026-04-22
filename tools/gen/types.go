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
	// HandlerAddExtraArgs is appended immediately after "v" in the Add call (e.g. ", v.GetSummary()" for AddFinding).
	HandlerAddExtraArgs string
	// HandlerDeleteName, if set, is the full Model delete method name (e.g. "DeleteOrgUnit"); otherwise Delete<Name>ById is used.
	HandlerDeleteName string

	// --- Client integration test fields ---

	// HasClientTest enables generation of a client integration test for this type.
	HasClientTest bool
	// GenClientMethods, when true, generates List and GetById wrapper methods on ModelSrvClient.
	GenClientMethods bool
	// ClientListMethod is the ModelSrvClient method name for listing (e.g. "GetSystems").
	ClientListMethod string
	// ClientGetByIdMethod is the ModelSrvClient method name for get-by-id (e.g. "GetSystemById").
	ClientGetByIdMethod string
	// ClientListOapiMethod is the oapi-codegen generated method prefix for list (e.g. "GetLandscapeSystems").
	ClientListOapiMethod string
	// ClientGetByIdOapiMethod is the oapi-codegen generated method prefix for get-by-id (e.g. "GetLandscapeSystemsSystemId").
	ClientGetByIdOapiMethod string
	// OapiTypeName is the oapi/client type name returned by get-by-id (e.g. "System").
	OapiTypeName string
	// TestDisplayName is the display name set on the test resource.
	TestDisplayName string
	// TestIDAssertExpr is the Go expression to extract the ID from the oapi response (e.g. "*got.SystemId").
	TestIDAssertExpr string
	// TestNameAssertExpr is the Go expression to extract the display name from the oapi response (e.g. "got.DisplayName").
	TestNameAssertExpr string
	// NotFoundSentinel is the sentinel error for 404 (e.g. "common.ErrSystemNotFound").
	NotFoundSentinel string
	// TestSetup is Go code to create and add the resource to the model. Uses testID as the uuid and sink as the event sink.
	TestSetup string
	// TestDeps lists Names of TypeSpecs that must be set up before this one.
	TestDeps []string
	// WireKind is the Event.kind string for this resource type (e.g. "System", "ApiInstance").
	WireKind string
}
