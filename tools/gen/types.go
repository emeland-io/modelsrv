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

	// --- OAPI wire / server generation (populated by enrichWireMeta) ---

	// SkipConvert marks types whose FromDto/ToDto stay hand-written in internal/oapi/convert_special.go.
	SkipConvert bool
	// SkipAuthz marks types whose GET handlers do not apply visibility filtering (e.g. infrastructure metadata).
	SkipAuthz bool
	// ConvertDomainIDMethod is the getter used after FromDto in replication decode (e.g. "GetSystemId()").
	ConvertDomainIDMethod string
	// WireDomainIDGetter is ConvertDomainIDMethod without parentheses (e.g. "GetSystemId").
	WireDomainIDGetter string
	// BackendListMethod is model.Model list method (e.g. "GetApis").
	BackendListMethod string
	// BackendGetByIdMethod is model.Model get-by-id method (e.g. "GetApiById").
	BackendGetByIdMethod string
	// RestListPath is the landscape collection path (e.g. "/landscape/apis").
	RestListPath string
	// ServerRequestIDField is the path parameter field on get-by-id request objects (e.g. "ApiId").
	ServerRequestIDField string
	// ServerResourceLabel is used in 404 messages (e.g. "api").
	ServerResourceLabel string
	// Server404UseErrorString wraps 404 bodies in ErrorString when true.
	Server404UseErrorString bool
	// WireIDField is the OpenAPI JSON id field name (e.g. "ApiId").
	WireIDField string
	// WireIDOptional is true when the wire id is a pointer on the DTO.
	WireIDOptional bool
	// GenServerHandlers emits GetLandscape* list/get handlers in server_handlers_gen.go.
	GenServerHandlers bool
	// SkipClientMethods skips generated pkg/client list/get wrappers (hand-written instead).
	SkipClientMethods bool
	// EventsResource is the events.ResourceType const name (e.g. "SystemResource").
	EventsResource string
	// OapiWireTypeName is the oapi DTO struct name (defaults to Name; API stays "API").
	OapiWireTypeName string
	// FromDtoFuncName is the generated/hand-written FromDto name (e.g. "APIFromDto").
	FromDtoFuncName string
	// ToDtoFuncName is the generated/hand-written ToDto name (e.g. "APIToDto").
	ToDtoFuncName string
	// ConvertNilLabel is the noun in nil FromDto errors (e.g. "context type").
	ConvertNilLabel string
	// ConvertMissingIDMsg is the error when a required wire id pointer is nil.
	ConvertMissingIDMsg string

	// --- Client domain type fields (populated by enrichDomainMeta) ---

	DomainPkgImport  string
	DomainPkgAlias   string
	DomainTypeName   string
	DomainIDGetter   string
	DomainNameGetter string
}
