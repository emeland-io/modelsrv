package main

// Wiring holds the freeform, code-like parts of a resource's TypeSpec that do not belong
// in struct tags: relational ref links, custom method signatures, per-type test setup, and
// domain-field overrides (the domain Fields differ from the wire schema for ref types).
//
// This is the ADR step-6 "Path B" supplement: annotated structs own the data and schema;
// this map, keyed by resource type name, owns the wiring. Only types that need it appear
// here; simple vocabulary types (NodeType, ContextType, OrgUnit, ...) have no entry.
type Wiring struct {
	// Fields, when non-nil, replaces the struct-derived domain Fields entirely (needed
	// when the domain representation differs from the wire schema, e.g. a *NodeTypeRef
	// field instead of a nodeType uuid property).
	Fields        []Field
	CustomMethods []string
	ParentLink    *ParentLinkSpec
	TypeRefLink   *TypeRefLinkSpec
	RefByRefs     []RefByRefSpec
	ExtraImports  []string
	TestDeps      []string
	TestSetup     string
	// IDField overrides the loader-derived IDField when non-empty (e.g. the *Instance
	// types share the domain id name "InstanceId" rather than the wire-derived spelling).
	IDField string
	// NotFoundErr / NotFoundSentinel override the loader-derived (name-based) values when
	// non-empty (e.g. API -> "ErrApiNotFound" instead of "ErrAPINotFound").
	NotFoundErr      string
	NotFoundSentinel string
	// HandlerMethodSuffix overrides the Add*/Delete* method suffix when non-empty
	// (e.g. API -> "Api").
	HandlerMethodSuffix string
	// TestDisplayName / TestIDAssertExpr / TestNameAssertExpr override the loader defaults
	// when non-empty.
	TestDisplayName    string
	TestIDAssertExpr   string
	TestNameAssertExpr string
}

// applyWiring overlays typeWiring[spec.Name] onto a loader-built TypeSpec.
func applyWiring(spec *TypeSpec) {
	w, ok := typeWiring[spec.Name]
	if !ok {
		return
	}
	if w.Fields != nil {
		spec.Fields = w.Fields
	}
	if w.CustomMethods != nil {
		spec.CustomMethods = w.CustomMethods
	}
	if w.ParentLink != nil {
		spec.ParentLink = w.ParentLink
	}
	if w.TypeRefLink != nil {
		spec.TypeRefLink = w.TypeRefLink
	}
	if w.RefByRefs != nil {
		spec.RefByRefs = w.RefByRefs
	}
	if w.ExtraImports != nil {
		spec.ExtraImports = w.ExtraImports
	}
	if w.TestDeps != nil {
		spec.TestDeps = w.TestDeps
	}
	if w.IDField != "" {
		spec.IDField = w.IDField
	}
	if w.NotFoundErr != "" {
		spec.NotFoundErr = w.NotFoundErr
	}
	if w.NotFoundSentinel != "" {
		spec.NotFoundSentinel = w.NotFoundSentinel
	}
	if w.HandlerMethodSuffix != "" {
		spec.HandlerMethodSuffix = w.HandlerMethodSuffix
	}
	if w.TestSetup != "" {
		spec.TestSetup = w.TestSetup
	}
	if w.TestDisplayName != "" {
		spec.TestDisplayName = w.TestDisplayName
	}
	if w.TestIDAssertExpr != "" {
		spec.TestIDAssertExpr = w.TestIDAssertExpr
	}
	if w.TestNameAssertExpr != "" {
		spec.TestNameAssertExpr = w.TestNameAssertExpr
	}
}

// typeWiring is the per-type freeform supplement. Populated incrementally during the
// step-6 migration; entries mirror the corresponding hand-written allTypes fields.
var typeWiring = map[string]Wiring{
	"NodeType": {
		TestSetup: `nt := node.NewNodeType(testIDs["NodeType"])
			nt.SetDisplayName("Test NodeType")
			require.NoError(t, m.AddNodeType(nt))`,
	},
	"ContextType": {
		TestSetup: `ct := mdlctx.NewContextType(testIDs["ContextType"])
			ct.SetDisplayName("Test ContextType")
			require.NoError(t, m.AddContextType(ct))`,
	},
	"Artifact": {
		TestSetup: `a := artifact.NewArtifact(testIDs["Artifact"])
			a.SetDisplayName("Test Artifact")
			require.NoError(t, m.AddArtifact(a))`,
	},
	"FilterRule": {
		TestSetup: `fr := mdlfilterrule.NewFilterRule(testIDs["FilterRule"])
			fr.SetDisplayName("Test FilterRule")
			require.NoError(t, m.AddFilterRule(fr))`,
	},
	"MergeRule": {
		TestSetup: `mr := mdlmergerule.NewMergeRule(testIDs["MergeRule"])
			mr.SetDisplayName("Test MergeRule")
			require.NoError(t, m.AddMergeRule(mr))`,
	},
	"Parameter": {
		TestSetup: `param := mdlparameter.NewParameter(testIDs["Parameter"])
			param.SetDisplayName("Test Parameter")
			require.NoError(t, m.AddParameter(param))`,
	},
	"ValidValue": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "ParameterRef", Type: "*ParameterRef", SkipAccessor: true},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/annotations",
		},
		TestDeps: []string{"Parameter"},
		TestSetup: `vv := mdlparameter.NewValidValue(testIDs["ValidValue"])
			vv.SetDisplayName("Test ValidValue")
			vv.SetParameterById(testIDs["Parameter"])
			require.NoError(t, m.AddValidValue(vv))`,
		CustomMethods: []string{
			"GetParameter() (Parameter, error)",
			"GetParameterId() uuid.UUID",
			"SetParameterRef(*ParameterRef)",
			"SetParameterByRef(parameter Parameter)",
			"SetParameterById(parameterId uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "ParameterRef",
			RefTypeName:       "ParameterRef",
			ResourceTypeName:  "Parameter",
			ResolvedMethod:    "ResolvedParameter",
			EmbedFieldName:    "Parameter",
			RefIDFieldName:    "ParameterId",
			ResourceIDGetter:  "GetParameterId",
			ModelLookupByID:   "GetParameterById",
			EffectiveIDMethod: "EffectiveParameterID",
			SetByID:           true,
			SetByIDParamName:  "parameterId",
		},
	},
	"PermissionSpec": {
		TestSetup: `ps := iam.NewPermissionSpec(testIDs["PermissionSpec"])
			ps.SetDisplayName("Test PermissionSpec")
			require.NoError(t, m.AddPermissionSpec(ps))`,
	},
	"Metric": {
		TestDisplayName: "p99 API latency",
		TestSetup: `metric := mdlobs.NewMetric(testIDs["Metric"])
			metric.SetDisplayName("p99 API latency")
			require.NoError(t, m.AddMetric(metric))`,
	},
	"CapacityResourceType": {
		TestDisplayName: "CPU cores",
		TestSetup: `crt := mdlcap.NewCapacityResourceType(testIDs["CapacityResourceType"])
			crt.SetDisplayName("CPU cores")
			crt.SetUnit("cores")
			require.NoError(t, m.AddCapacityResourceType(crt))`,
	},
	"FindingType": {
		TestIDAssertExpr:   "uuid.UUID(*got.FindingTypeId)",
		TestNameAssertExpr: "*got.DisplayName",
		TestSetup: `ft := finding.NewFindingType(testIDs["FindingType"])
			ft.SetDisplayName("Test FindingType")
			require.NoError(t, m.AddFindingType(ft))`,
	},
	"Capability": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Offers", Type: "[]uuid.UUID"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		TestSetup: `cap := mdlcapability.NewCapability(testIDs["Capability"])
			cap.SetDisplayName("Test Capability")
			require.NoError(t, m.AddCapability(cap))`,
	},
	"CapabilityVersion": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "CapabilityRef", Type: "*CapabilityRef", SkipAccessor: true},
			{Name: "Version", Type: "common.Version"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/annotations",
			"go.emeland.io/modelsrv/pkg/model/common",
		},
		TestDeps: []string{"Capability"},
		TestSetup: `cv := mdlcapability.NewCapabilityVersion(testIDs["CapabilityVersion"])
			cv.SetDisplayName("Test CapabilityVersion")
			cv.SetCapabilityById(testIDs["Capability"])
			cv.SetVersion(common.Version{Version: "1.0.0"})
			require.NoError(t, m.AddCapabilityVersion(cv))`,
		CustomMethods: []string{
			"GetCapability() (Capability, error)",
			"GetCapabilityId() uuid.UUID",
			"SetCapabilityRef(*CapabilityRef)",
			"SetCapabilityByRef(capability Capability)",
			"SetCapabilityById(capabilityId uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "CapabilityRef",
			RefTypeName:       "CapabilityRef",
			ResourceTypeName:  "Capability",
			ResolvedMethod:    "ResolvedCapability",
			EmbedFieldName:    "Capability",
			RefIDFieldName:    "CapabilityId",
			ResourceIDGetter:  "GetCapabilityId",
			ModelLookupByID:   "GetCapabilityById",
			EffectiveIDMethod: "EffectiveCapabilityID",
			SetByID:           true,
			SetByIDParamName:  "capabilityId",
		},
	},
	"Variant": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "CapabilityVersionRef", Type: "*CapabilityVersionRef", SkipAccessor: true},
			{Name: "Requires", Type: "[]uuid.UUID"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/annotations",
		},
		TestDeps: []string{"Capability", "CapabilityVersion"},
		TestSetup: `v := mdlcapability.NewVariant(testIDs["Variant"])
			v.SetDisplayName("Test Variant")
			v.SetCapabilityVersionById(testIDs["CapabilityVersion"])
			require.NoError(t, m.AddVariant(v))`,
		CustomMethods: []string{
			"GetCapabilityVersion() (CapabilityVersion, error)",
			"GetCapabilityVersionId() uuid.UUID",
			"SetCapabilityVersionRef(*CapabilityVersionRef)",
			"SetCapabilityVersionByRef(capabilityVersion CapabilityVersion)",
			"SetCapabilityVersionById(capabilityVersionId uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "CapabilityVersionRef",
			RefTypeName:       "CapabilityVersionRef",
			ResourceTypeName:  "CapabilityVersion",
			ResolvedMethod:    "ResolvedCapabilityVersion",
			EmbedFieldName:    "CapabilityVersion",
			RefIDFieldName:    "CapabilityVersionId",
			ResourceIDGetter:  "GetCapabilityVersionId",
			ModelLookupByID:   "GetCapabilityVersionById",
			EffectiveIDMethod: "EffectiveCapabilityVersionID",
			SetByID:           true,
			SetByIDParamName:  "capabilityVersionId",
		},
	},
	"Dependency": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "VariantRef", Type: "*VariantRef", SkipAccessor: true},
			{Name: "CapabilityRef", Type: "*CapabilityRef", SkipAccessor: true},
			{Name: "Mappings", Type: "[]ValueMapping"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/annotations",
		},
		TestDeps: []string{"Capability", "CapabilityVersion", "Variant"},
		TestSetup: `d := mdlcapability.NewDependency(testIDs["Dependency"])
			d.SetDisplayName("Test Dependency")
			d.SetVariantById(testIDs["Variant"])
			d.SetCapabilityById(testIDs["Capability"])
			require.NoError(t, m.AddDependency(d))`,
		CustomMethods: []string{
			"GetVariant() (Variant, error)",
			"GetVariantId() uuid.UUID",
			"SetVariantRef(*VariantRef)",
			"SetVariantByRef(variant Variant)",
			"SetVariantById(variantId uuid.UUID)",
			"GetCapability() (Capability, error)",
			"GetCapabilityId() uuid.UUID",
			"SetCapabilityRef(*CapabilityRef)",
			"SetCapabilityByRef(capability Capability)",
			"SetCapabilityById(capabilityId uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "VariantRef",
			RefTypeName:       "VariantRef",
			ResourceTypeName:  "Variant",
			ResolvedMethod:    "ResolvedVariant",
			EmbedFieldName:    "Variant",
			RefIDFieldName:    "VariantId",
			ResourceIDGetter:  "GetVariantId",
			ModelLookupByID:   "GetVariantById",
			EffectiveIDMethod: "EffectiveVariantID",
			SetByID:           true,
			SetByIDParamName:  "variantId",
		},
		RefByRefs: []RefByRefSpec{
			{
				MethodName:       "SetCapabilityByRef",
				ParamName:        "capability",
				ResourceTypeName: "Capability",
				ParamGoType:      "Capability",
				SetterName:       "SetCapabilityRef",
				RefTypeName:      "CapabilityRef",
				RefTypeGoType:    "CapabilityRef",
				EmbedFieldName:   "Capability",
				RefIDFieldName:   "CapabilityId",
				ResourceIDGetter: "GetCapabilityId",
				NilCheck:         true,
			},
		},
	},
	"Order": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "OrgUnitRef", Type: "*iam.OrgUnitRef", SkipAccessor: true},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/annotations",
			"go.emeland.io/modelsrv/pkg/model/iam",
		},
		TestDeps: []string{"OrgUnit"},
		TestSetup: `o := mdlorder.NewOrder(testIDs["Order"])
			o.SetDisplayName("Test Order")
			o.SetOrgUnitById(testIDs["OrgUnit"])
			require.NoError(t, m.AddOrder(o))`,
		CustomMethods: []string{
			"GetOrgUnit() (iam.OrgUnit, error)",
			"GetOrgUnitId() uuid.UUID",
			"SetOrgUnitRef(*iam.OrgUnitRef)",
			"SetOrgUnitByRef(orgUnit iam.OrgUnit)",
			"SetOrgUnitById(orgUnitId uuid.UUID)",
		},
		RefByRefs: []RefByRefSpec{
			{
				MethodName:       "SetOrgUnitByRef",
				ParamName:        "orgUnit",
				ResourceTypeName: "OrgUnit",
				ParamGoType:      "iam.OrgUnit",
				SetterName:       "SetOrgUnitRef",
				RefTypeName:      "OrgUnitRef",
				RefTypeGoType:    "iam.OrgUnitRef",
				EmbedFieldName:   "OrgUnit",
				RefIDFieldName:   "OrgUnitId",
				ResourceIDGetter: "GetOrgUnitId",
				NilCheck:         true,
			},
		},
	},
	"OrderItem": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "OrderRef", Type: "*OrderRef", SkipAccessor: true},
			{Name: "CapabilityRef", Type: "*capability.CapabilityRef"},
			{Name: "CapabilityVersionRef", Type: "*capability.CapabilityVersionRef"},
			{Name: "VariantRef", Type: "*capability.VariantRef"},
			{Name: "Contexts", Type: "[]uuid.UUID"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/annotations",
			"go.emeland.io/modelsrv/pkg/model/capability",
		},
		TestDeps: []string{"Order", "Capability", "CapabilityVersion", "Variant"},
		TestSetup: `oi := mdlorder.NewOrderItem(testIDs["OrderItem"])
			oi.SetDisplayName("Test OrderItem")
			oi.SetOrderById(testIDs["Order"])
			oi.SetCapabilityRef(&mdlcapability.CapabilityRef{CapabilityId: testIDs["Capability"]})
			oi.SetCapabilityVersionRef(&mdlcapability.CapabilityVersionRef{CapabilityVersionId: testIDs["CapabilityVersion"]})
			oi.SetVariantRef(&mdlcapability.VariantRef{VariantId: testIDs["Variant"]})
			require.NoError(t, m.AddOrderItem(oi))`,
		CustomMethods: []string{
			"GetOrder() (Order, error)",
			"GetOrderId() uuid.UUID",
			"SetOrderRef(*OrderRef)",
			"SetOrderByRef(order Order)",
			"SetOrderById(orderId uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "OrderRef",
			RefTypeName:       "OrderRef",
			ResourceTypeName:  "Order",
			ResolvedMethod:    "ResolvedOrder",
			EmbedFieldName:    "Order",
			RefIDFieldName:    "OrderId",
			ResourceIDGetter:  "GetOrderId",
			ModelLookupByID:   "GetOrderById",
			EffectiveIDMethod: "EffectiveOrderID",
			SetByID:           true,
			SetByIDParamName:  "orderId",
		},
	},
	"BoundValue": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "OrderItemRef", Type: "*OrderItemRef", SkipAccessor: true},
			{Name: "ParameterRef", Type: "*parameter.ParameterRef"},
			{Name: "ValidValueRef", Type: "*parameter.ValidValueRef"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/annotations",
			"go.emeland.io/modelsrv/pkg/model/parameter",
		},
		TestDeps: []string{"OrderItem", "Parameter", "ValidValue"},
		TestSetup: `bv := mdlorder.NewBoundValue(testIDs["BoundValue"])
			bv.SetDisplayName("Test BoundValue")
			bv.SetOrderItemById(testIDs["OrderItem"])
			bv.SetParameterRef(&mdlparameter.ParameterRef{ParameterId: testIDs["Parameter"]})
			bv.SetValidValueRef(&mdlparameter.ValidValueRef{ValidValueId: testIDs["ValidValue"]})
			require.NoError(t, m.AddBoundValue(bv))`,
		CustomMethods: []string{
			"GetOrderItem() (OrderItem, error)",
			"GetOrderItemId() uuid.UUID",
			"SetOrderItemRef(*OrderItemRef)",
			"SetOrderItemByRef(orderItem OrderItem)",
			"SetOrderItemById(orderItemId uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "OrderItemRef",
			RefTypeName:       "OrderItemRef",
			ResourceTypeName:  "OrderItem",
			ResolvedMethod:    "ResolvedOrderItem",
			EmbedFieldName:    "OrderItem",
			RefIDFieldName:    "OrderItemId",
			ResourceIDGetter:  "GetOrderItemId",
			ModelLookupByID:   "GetOrderItemById",
			EffectiveIDMethod: "EffectiveOrderItemID",
			SetByID:           true,
			SetByIDParamName:  "orderItemId",
		},
	},
	"System": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "Version", Type: "common.Version"},
			{Name: "Abstract", Type: "bool"},
			{Name: "Parent", Type: "*SystemRef", SkipAccessor: true},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/common",
			"go.emeland.io/modelsrv/pkg/model/annotations",
		},
		TestIDAssertExpr: "uuid.UUID(*got.SystemId)",
		TestSetup: `sys := system.NewSystem(testIDs["System"])
			sys.SetDisplayName("Test System")
			require.NoError(t, m.AddSystem(sys))`,
		CustomMethods: []string{
			"GetParent() (System, error)",
			"SetParent(*SystemRef)",
			"SetParentByRef(parent System)",
		},
		ParentLink: &ParentLinkSpec{
			FieldName:         "Parent",
			RefTypeName:       "SystemRef",
			ResourceTypeName:  "System",
			ResolvedMethod:    "ResolvedSystem",
			EffectiveIDMethod: "",
			EmbedFieldName:    "System",
			RefIDFieldName:    "SystemId",
			ResourceIDGetter:  "GetSystemId",
			ModelLookupByID:   "GetSystemById",
			SetParentByID:     false,
		},
	},
	"API": {
		NotFoundErr:         "ErrApiNotFound",
		NotFoundSentinel:    "common.ErrApiNotFound",
		HandlerMethodSuffix: "Api",
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "Version", Type: "common.Version"},
			{Name: "Type", Type: "ApiType"},
			{Name: "System", Type: "*system.SystemRef"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/common",
			"go.emeland.io/modelsrv/pkg/model/annotations",
			"go.emeland.io/modelsrv/pkg/model/system",
		},
		TestIDAssertExpr: "uuid.UUID(*got.ApiId)",
		TestDeps:         []string{"System"},
		TestSetup: `a := mdlapi.NewAPI(testIDs["API"])
			a.SetDisplayName("Test API")
			a.SetSystem(&system.SystemRef{SystemId: testIDs["System"]})
			require.NoError(t, m.AddApi(a))`,
		CustomMethods: []string{
			"SetSystemByRef(sys system.System)",
		},
		RefByRefs: []RefByRefSpec{
			{
				MethodName:       "SetSystemByRef",
				ParamName:        "sys",
				ResourceTypeName: "System",
				ParamGoType:      "system.System",
				SetterName:       "SetSystem",
				RefTypeName:      "SystemRef",
				RefTypeGoType:    "system.SystemRef",
				EmbedFieldName:   "System",
				RefIDFieldName:   "SystemId",
				ResourceIDGetter: "GetSystemId",
				NilCheck:         true,
			},
		},
	},
	"ApiInstance": {
		IDField: "InstanceId",
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "ApiRef", Type: "*ApiRef"},
			{Name: "SystemInstance", Type: "*system.SystemInstanceRef"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/annotations",
			"go.emeland.io/modelsrv/pkg/model/system",
		},
		TestDeps: []string{"API"},
		TestSetup: `ai := mdlapi.NewApiInstance(testIDs["ApiInstance"])
			ai.SetDisplayName("Test ApiInstance")
			ai.SetApiRef(&mdlapi.ApiRef{ApiID: testIDs["API"]})
			require.NoError(t, m.AddApiInstance(ai))`,
		CustomMethods: []string{
			"SetApiRefByRef(api API)",
			"SetSystemInstanceByRef(instance system.SystemInstance)",
		},
		RefByRefs: []RefByRefSpec{
			{
				MethodName:       "SetApiRefByRef",
				ParamName:        "api",
				ResourceTypeName: "API",
				SetterName:       "SetApiRef",
				RefTypeName:      "ApiRef",
				EmbedFieldName:   "API",
				RefIDFieldName:   "ApiID",
				ResourceIDGetter: "GetApiId",
			},
			{
				MethodName:       "SetSystemInstanceByRef",
				ParamName:        "instance",
				ResourceTypeName: "SystemInstance",
				ParamGoType:      "system.SystemInstance",
				SetterName:       "SetSystemInstance",
				RefTypeName:      "SystemInstanceRef",
				RefTypeGoType:    "system.SystemInstanceRef",
				EmbedFieldName:   "SystemInstance",
				RefIDFieldName:   "InstanceId",
				ResourceIDGetter: "GetInstanceId",
			},
		},
	},
	"Component": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "Version", Type: "common.Version"},
			{Name: "System", Type: "*system.SystemRef"},
			{Name: "Consumes", Type: "[]api.ApiRef"},
			{Name: "Provides", Type: "[]api.ApiRef"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/common",
			"go.emeland.io/modelsrv/pkg/model/annotations",
			"go.emeland.io/modelsrv/pkg/model/api",
			"go.emeland.io/modelsrv/pkg/model/system",
		},
		TestIDAssertExpr: "uuid.UUID(*got.ComponentId)",
		TestDeps:         []string{"System"},
		TestSetup: `comp := component.NewComponent(testIDs["Component"])
			comp.SetDisplayName("Test Component")
			comp.SetSystem(&system.SystemRef{SystemId: testIDs["System"]})
			require.NoError(t, m.AddComponent(comp))`,
	},
	"ComponentInstance": {
		IDField: "InstanceId",
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "ComponentRef", Type: "*ComponentRef"},
			{Name: "SystemInstance", Type: "*system.SystemInstanceRef"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/annotations",
			"go.emeland.io/modelsrv/pkg/model/system",
		},
		TestDeps: []string{"Component"},
		TestSetup: `ci := component.NewComponentInstance(testIDs["ComponentInstance"])
			ci.SetDisplayName("Test ComponentInstance")
			ci.SetComponentRef(&component.ComponentRef{ComponentId: testIDs["Component"]})
			require.NoError(t, m.AddComponentInstance(ci))`,
	},
	"SystemInstance": {
		IDField: "InstanceId",
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "SystemRef", Type: "*SystemRef"},
			{Name: "ContextRef", Type: "*context.ContextRef"},
			{Name: "OrderItemRef", Type: "*order.OrderItemRef"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/annotations",
			"go.emeland.io/modelsrv/pkg/model/context",
			"go.emeland.io/modelsrv/pkg/model/order",
		},
		TestDeps: []string{"System"},
		TestSetup: `si := system.NewSystemInstance(testIDs["SystemInstance"])
			si.SetDisplayName("Test SystemInstance")
			si.SetSystemRef(&system.SystemRef{SystemId: testIDs["System"]})
			require.NoError(t, m.AddSystemInstance(si))`,
	},
	"Node": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "TypeRef", Type: "*NodeTypeRef", SkipAccessor: true},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		TestDeps: []string{"NodeType"},
		TestSetup: `n := node.NewNode(testIDs["Node"])
			n.SetDisplayName("Test Node")
			n.SetTypeRef(&node.NodeTypeRef{NodeTypeId: testIDs["NodeType"]})
			require.NoError(t, m.AddNode(n))`,
		CustomMethods: []string{
			"GetNodeType() (NodeType, error)",
			"GetNodeTypeId() uuid.UUID",
			"SetTypeRef(*NodeTypeRef)",
			"SetNodeTypeByRef(nodeType NodeType)",
			"SetNodeTypeById(nodeTypeId uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "TypeRef",
			RefTypeName:       "NodeTypeRef",
			ResourceTypeName:  "NodeType",
			ResolvedMethod:    "ResolvedNodeType",
			EmbedFieldName:    "NodeType",
			RefIDFieldName:    "NodeTypeId",
			ResourceIDGetter:  "GetNodeTypeId",
			ModelLookupByID:   "GetNodeTypeById",
			EffectiveIDMethod: "EffectiveNodeTypeID",
			SetByID:           true,
			SetByIDParamName:  "nodeTypeId",
		},
	},
	"Context": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "TypeRef", Type: "*ContextTypeRef", SkipAccessor: true},
			{Name: "Parent", Type: "*ContextRef", SkipAccessor: true},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		TestDeps: []string{"ContextType"},
		TestSetup: `c := mdlctx.NewContext(testIDs["Context"])
			c.SetDisplayName("Test Context")
			c.SetContextTypeById(testIDs["ContextType"])
			require.NoError(t, m.AddContext(c))`,
		CustomMethods: []string{
			"GetContextType() (ContextType, error)",
			"GetContextTypeId() uuid.UUID",
			"SetTypeRef(*ContextTypeRef)",
			"SetContextTypeByRef(contextType ContextType)",
			"SetContextTypeById(contextTypeId uuid.UUID)",
			"GetParent() (Context, error)",
			"GetParentId() uuid.UUID",
			"SetParent(*ContextRef)",
			"SetParentByRef(parent Context)",
			"SetParentById(parentId uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "TypeRef",
			RefTypeName:       "ContextTypeRef",
			ResourceTypeName:  "ContextType",
			ResolvedMethod:    "ResolvedContextType",
			EmbedFieldName:    "ContextType",
			RefIDFieldName:    "ContextTypeId",
			ResourceIDGetter:  "GetContextTypeId",
			ModelLookupByID:   "GetContextTypeById",
			EffectiveIDMethod: "EffectiveContextTypeID",
			SetByID:           true,
			SetByIDParamName:  "contextTypeId",
		},
		ParentLink: &ParentLinkSpec{
			FieldName:         "Parent",
			RefTypeName:       "ContextRef",
			ResourceTypeName:  "Context",
			ResolvedMethod:    "ResolvedContext",
			EffectiveIDMethod: "EffectiveParentContextID",
			EmbedFieldName:    "Context",
			RefIDFieldName:    "ContextId",
			ResourceIDGetter:  "GetContextId",
			ModelLookupByID:   "GetContextById",
			SetParentByID:     true,
		},
	},
	"Finding": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "TypeRef", Type: "*FindingTypeRef", SkipAccessor: true},
			{Name: "Resources", Type: "[]*common.ResourceRef"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/common",
			"go.emeland.io/modelsrv/pkg/model/annotations",
		},
		TestDeps: []string{"FindingType"},
		TestSetup: `f := finding.NewFinding(testIDs["Finding"])
			f.SetDisplayName("Test Finding")
			f.SetFindingTypeById(testIDs["FindingType"])
			require.NoError(t, m.AddFinding(f))`,
		CustomMethods: []string{
			"GetFindingType() (FindingType, error)",
			"GetFindingTypeId() uuid.UUID",
			"SetTypeRef(*FindingTypeRef)",
			"SetFindingTypeByRef(findingType FindingType)",
			"SetFindingTypeById(findingTypeId uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "TypeRef",
			RefTypeName:       "FindingTypeRef",
			ResourceTypeName:  "FindingType",
			ResolvedMethod:    "ResolvedFindingType",
			EmbedFieldName:    "FindingType",
			RefIDFieldName:    "FindingTypeId",
			ResourceIDGetter:  "GetFindingTypeId",
			ModelLookupByID:   "GetFindingTypeById",
			EffectiveIDMethod: "EffectiveFindingTypeID",
			SetByID:           true,
			SetByIDParamName:  "findingTypeId",
		},
	},
	"OrgUnit": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "Parent", Type: "*OrgUnitRef", SkipAccessor: true},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		TestSetup: `ou := iam.NewOrgUnit(testIDs["OrgUnit"])
			ou.SetDisplayName("Test OrgUnit")
			require.NoError(t, m.AddOrgUnit(ou))`,
		ParentLink: &ParentLinkSpec{
			FieldName:         "Parent",
			RefTypeName:       "OrgUnitRef",
			ResourceTypeName:  "OrgUnit",
			ResolvedMethod:    "ResolvedOrgUnit",
			EffectiveIDMethod: "EffectiveParentOrgUnitID",
			EmbedFieldName:    "OrgUnit",
			RefIDFieldName:    "OrgUnitId",
			ResourceIDGetter:  "GetOrgUnitId",
			ModelLookupByID:   "GetOrgUnitById",
			SetParentByID:     true,
		},
	},
	"Group": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "Members", Type: "[]*IdentityRef"},
			{Name: "OrgUnit", Type: "*OrgUnitRef"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		TestSetup: `g := iam.NewGroup(testIDs["Group"])
			g.SetDisplayName("Test Group")
			require.NoError(t, m.AddGroup(g))`,
		RefByRefs: []RefByRefSpec{
			{
				MethodName:       "SetOrgUnitByRef",
				ParamName:        "orgUnit",
				ResourceTypeName: "OrgUnit",
				ParamGoType:      "OrgUnit",
				SetterName:       "SetOrgUnit",
				RefTypeName:      "OrgUnitRef",
				RefTypeGoType:    "OrgUnitRef",
				EmbedFieldName:   "OrgUnit",
				RefIDFieldName:   "OrgUnitId",
				ResourceIDGetter: "GetOrgUnitId",
				NilCheck:         true,
			},
		},
	},
	"Identity": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "OrgUnit", Type: "*OrgUnitRef"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		TestSetup: `id := iam.NewIdentity(testIDs["Identity"])
			id.SetDisplayName("Test Identity")
			require.NoError(t, m.AddIdentity(id))`,
		RefByRefs: []RefByRefSpec{
			{
				MethodName:       "SetOrgUnitByRef",
				ParamName:        "orgUnit",
				ResourceTypeName: "OrgUnit",
				ParamGoType:      "OrgUnit",
				SetterName:       "SetOrgUnit",
				RefTypeName:      "OrgUnitRef",
				RefTypeGoType:    "OrgUnitRef",
				EmbedFieldName:   "OrgUnit",
				RefIDFieldName:   "OrgUnitId",
				ResourceIDGetter: "GetOrgUnitId",
				NilCheck:         true,
			},
		},
	},
	"RoleSpec": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "Permissions", Type: "[]*PermissionSpecRef"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		TestDeps: []string{"PermissionSpec"},
		TestSetup: `rs := iam.NewRoleSpec(testIDs["RoleSpec"])
			rs.SetDisplayName("Test RoleSpec")
			rs.SetPermissions([]*iam.PermissionSpecRef{&iam.PermissionSpecRef{PermissionSpecId: testIDs["PermissionSpec"]}})
			require.NoError(t, m.AddRoleSpec(rs))`,
	},
	"Permission": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "Spec", Type: "*PermissionSpecRef", SkipAccessor: true},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		TestDeps: []string{"PermissionSpec"},
		TestSetup: `p := iam.NewPermission(testIDs["Permission"])
			p.SetDisplayName("Test Permission")
			p.SetPermissionSpecById(testIDs["PermissionSpec"])
			require.NoError(t, m.AddPermission(p))`,
		CustomMethods: []string{
			"GetPermissionSpec() (PermissionSpec, error)",
			"GetPermissionSpecId() uuid.UUID",
			"SetSpec(*PermissionSpecRef)",
			"SetPermissionSpecByRef(permissionSpec PermissionSpec)",
			"SetPermissionSpecById(permissionSpecId uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "Spec",
			RefTypeName:       "PermissionSpecRef",
			ResourceTypeName:  "PermissionSpec",
			ResolvedMethod:    "ResolvedPermissionSpec",
			EmbedFieldName:    "PermissionSpec",
			RefIDFieldName:    "PermissionSpecId",
			ResourceIDGetter:  "GetPermissionSpecId",
			ModelLookupByID:   "GetPermissionSpecById",
			EffectiveIDMethod: "EffectivePermissionSpecID",
			SetByID:           true,
			SetByIDParamName:  "permissionSpecId",
		},
	},
	"Role": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "Spec", Type: "*RoleSpecRef", SkipAccessor: true},
			{Name: "Permissions", Type: "[]*PermissionRef"},
			{Name: "Resources", Type: "[]*common.ResourceRef"},
			{Name: "ContextRef", Type: "*context.ContextRef"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/annotations",
			"go.emeland.io/modelsrv/pkg/model/common",
			"go.emeland.io/modelsrv/pkg/model/context",
		},
		TestDeps: []string{"RoleSpec", "Permission", "Context"},
		TestSetup: `r := iam.NewRole(testIDs["Role"])
			r.SetDisplayName("Test Role")
			r.SetRoleSpecById(testIDs["RoleSpec"])
			r.SetContextRef(&mdlctx.ContextRef{ContextId: testIDs["Context"]})
			r.SetPermissions([]*iam.PermissionRef{&iam.PermissionRef{PermissionId: testIDs["Permission"]}})
			require.NoError(t, m.AddRole(r))`,
		CustomMethods: []string{
			"GetRoleSpec() (RoleSpec, error)",
			"GetRoleSpecId() uuid.UUID",
			"SetSpec(*RoleSpecRef)",
			"SetRoleSpecByRef(roleSpec RoleSpec)",
			"SetRoleSpecById(roleSpecId uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "Spec",
			RefTypeName:       "RoleSpecRef",
			ResourceTypeName:  "RoleSpec",
			ResolvedMethod:    "ResolvedRoleSpec",
			EmbedFieldName:    "RoleSpec",
			RefIDFieldName:    "RoleSpecId",
			ResourceIDGetter:  "GetRoleSpecId",
			ModelLookupByID:   "GetRoleSpecById",
			EffectiveIDMethod: "EffectiveRoleSpecID",
			SetByID:           true,
			SetByIDParamName:  "roleSpecId",
		},
		RefByRefs: []RefByRefSpec{
			{
				MethodName:       "SetContextByRef",
				ParamName:        "ctx",
				ResourceTypeName: "Context",
				ParamGoType:      "context.Context",
				SetterName:       "SetContextRef",
				RefTypeName:      "ContextRef",
				RefTypeGoType:    "context.ContextRef",
				EmbedFieldName:   "Context",
				RefIDFieldName:   "ContextId",
				ResourceIDGetter: "GetContextId",
				NilCheck:         true,
			},
		},
	},
	"Binding": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "Role", Type: "*RoleRef"},
			{Name: "Subject", Type: "*SubjectRef"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		TestDeps: []string{"Role", "Group"},
		TestSetup: `b := iam.NewBinding(testIDs["Binding"])
			b.SetDisplayName("Test Binding")
			b.SetRole(&iam.RoleRef{RoleId: testIDs["Role"]})
			b.SetSubject(&iam.SubjectRef{Group: &iam.GroupRef{GroupId: testIDs["Group"]}})
			require.NoError(t, m.AddBinding(b))`,
		RefByRefs: []RefByRefSpec{
			{
				MethodName:       "SetRoleByRef",
				ParamName:        "role",
				ResourceTypeName: "Role",
				ParamGoType:      "Role",
				SetterName:       "SetRole",
				RefTypeName:      "RoleRef",
				RefTypeGoType:    "RoleRef",
				EmbedFieldName:   "Role",
				RefIDFieldName:   "RoleId",
				ResourceIDGetter: "GetRoleId",
				NilCheck:         true,
			},
		},
	},
	"ArtifactInstance": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "ArtifactRef", Type: "*ArtifactRef"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		TestDeps: []string{"Artifact"},
		TestSetup: `ai := artifact.NewArtifactInstance(testIDs["ArtifactInstance"])
			ai.SetDisplayName("Test ArtifactInstance")
			require.NoError(t, m.AddArtifactInstance(ai))`,
		RefByRefs: []RefByRefSpec{
			{
				MethodName:       "SetArtifactByRef",
				ParamName:        "a",
				ResourceTypeName: "Artifact",
				ParamGoType:      "Artifact",
				SetterName:       "SetArtifactRef",
				RefTypeName:      "ArtifactRef",
				EmbedFieldName:   "Artifact",
				RefIDFieldName:   "ArtifactId",
				ResourceIDGetter: "GetArtifactId",
				NilCheck:         true,
			},
		},
	},
	"Product": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "Vendor", Type: "*iam.OrgUnitRef"},
			{Name: "Versions", Type: "[]ProductionVersion"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/annotations",
			"go.emeland.io/modelsrv/pkg/model/iam",
		},
		TestDeps: []string{"OrgUnit"},
		TestSetup: `p := mdlproduct.NewProduct(testIDs["Product"])
			p.SetDisplayName("Test Product")
			p.SetVendor(&iam.OrgUnitRef{OrgUnitId: testIDs["OrgUnit"]})
			require.NoError(t, m.AddProduct(p))`,
		RefByRefs: []RefByRefSpec{
			{
				MethodName:       "SetVendorByRef",
				ParamName:        "orgUnit",
				ResourceTypeName: "OrgUnit",
				ParamGoType:      "iam.OrgUnit",
				SetterName:       "SetVendor",
				RefTypeName:      "OrgUnitRef",
				RefTypeGoType:    "iam.OrgUnitRef",
				EmbedFieldName:   "OrgUnit",
				RefIDFieldName:   "OrgUnitId",
				ResourceIDGetter: "GetOrgUnitId",
				NilCheck:         true,
			},
		},
	},
	"Capacity": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "ResourceTypeRef", Type: "*CapacityResourceTypeRef", SkipAccessor: true},
			{Name: "ContextRef", Type: "*context.ContextRef", SkipAccessor: true},
			{Name: "Category", Type: "Category"},
			{Name: "Amount", Type: "Amount"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/annotations",
			"go.emeland.io/modelsrv/pkg/model/context",
		},
		TestDisplayName: "Production CPU provided",
		TestDeps:        []string{},
		TestSetup: `cap := mdlcap.NewCapacity(testIDs["Capacity"])
			cap.SetDisplayName("Production CPU provided")
			crt := mdlcap.NewCapacityResourceType(uuid.New())
			crt.SetDisplayName("CPU")
			crt.SetUnit("cores")
			ct := mdlctx.NewContextType(uuid.New())
			ct.SetDisplayName("Test ContextType")
			ctx := mdlctx.NewContext(uuid.New())
			ctx.SetDisplayName("Test Context")
			ctx.SetContextTypeById(ct.GetContextTypeId())
			cap.SetCapacityResourceTypeById(crt.GetCapacityResourceTypeId())
			cap.SetContextById(ctx.GetContextId())
			cap.SetCategory(mdlcap.CategoryProvided)
			cap.SetAmount(mdlcap.Amount("64"))
			require.NoError(t, m.AddCapacityResourceType(crt))
			require.NoError(t, m.AddContextType(ct))
			require.NoError(t, m.AddContext(ctx))
			require.NoError(t, m.AddCapacity(cap))`,
		CustomMethods: []string{
			"GetCapacityResourceType() (CapacityResourceType, error)",
			"GetCapacityResourceTypeId() uuid.UUID",
			"SetResourceTypeRef(*CapacityResourceTypeRef)",
			"SetCapacityResourceTypeByRef(capacityResourceType CapacityResourceType)",
			"SetCapacityResourceTypeById(capacityResourceTypeId uuid.UUID)",
			"GetContext() (context.Context, error)",
			"GetContextId() uuid.UUID",
			"SetContextRef(*context.ContextRef)",
			"SetContextByRef(ctx context.Context)",
			"SetContextById(contextId uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "ResourceTypeRef",
			RefTypeName:       "CapacityResourceTypeRef",
			ResourceTypeName:  "CapacityResourceType",
			ResolvedMethod:    "ResolvedCapacityResourceType",
			EmbedFieldName:    "CapacityResourceType",
			RefIDFieldName:    "CapacityResourceTypeId",
			ResourceIDGetter:  "GetCapacityResourceTypeId",
			ModelLookupByID:   "GetCapacityResourceTypeById",
			EffectiveIDMethod: "EffectiveCapacityResourceTypeID",
			SetByID:           true,
			SetByIDParamName:  "capacityResourceTypeId",
		},
		RefByRefs: []RefByRefSpec{
			{
				MethodName:       "SetContextByRef",
				ParamName:        "ctx",
				ResourceTypeName: "Context",
				ParamGoType:      "context.Context",
				SetterName:       "SetContextRef",
				RefTypeName:      "ContextRef",
				RefTypeGoType:    "context.ContextRef",
				EmbedFieldName:   "Context",
				RefIDFieldName:   "ContextId",
				ResourceIDGetter: "GetContextId",
				NilCheck:         true,
			},
		},
	},
	"Threshold": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "MetricInstanceRef", Type: "*MetricInstanceRef", SkipAccessor: true},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		TestDisplayName: "Latency SLO breach",
		TestDeps:        []string{"Metric", "MetricInstance"},
		TestSetup: `th := mdlobs.NewThreshold(testIDs["Threshold"])
			th.SetDisplayName("Latency SLO breach")
			metric := mdlobs.NewMetric(uuid.New())
			metric.SetDisplayName("p99 API latency")
			mi := mdlobs.NewMetricInstance(uuid.New())
			mi.SetDisplayName("p99 API latency for orders-api")
			mi.SetMetricById(metric.GetMetricId())
			th.SetMetricInstanceById(mi.GetMetricInstanceId())
			require.NoError(t, m.AddMetric(metric))
			require.NoError(t, m.AddMetricInstance(mi))
			require.NoError(t, m.AddThreshold(th))`,
		CustomMethods: []string{
			"GetMetricInstance() (MetricInstance, error)",
			"GetMetricInstanceId() uuid.UUID",
			"SetMetricInstanceRef(*MetricInstanceRef)",
			"SetMetricInstanceByRef(mi MetricInstance)",
			"SetMetricInstanceById(id uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "MetricInstanceRef",
			RefTypeName:       "MetricInstanceRef",
			ResourceTypeName:  "MetricInstance",
			ResolvedMethod:    "ResolvedMetricInstance",
			EmbedFieldName:    "MetricInstance",
			RefIDFieldName:    "MetricInstanceId",
			ResourceIDGetter:  "GetMetricInstanceId",
			ModelLookupByID:   "GetMetricInstanceById",
			EffectiveIDMethod: "EffectiveMetricInstanceID",
			SetByID:           true,
			SetByIDParamName:  "id",
		},
	},
	"MetricInstance": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "MetricRef", Type: "*MetricRef", SkipAccessor: true},
			{Name: "Subject", Type: "*common.ResourceRef"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		ExtraImports: []string{
			"go.emeland.io/modelsrv/pkg/model/annotations",
			"go.emeland.io/modelsrv/pkg/model/common",
		},
		TestDisplayName: "p99 API latency for orders-api",
		TestDeps:        []string{"Metric"},
		TestSetup: `mi := mdlobs.NewMetricInstance(testIDs["MetricInstance"])
			mi.SetDisplayName("p99 API latency for orders-api")
			metric := mdlobs.NewMetric(uuid.New())
			metric.SetDisplayName("p99 API latency")
			mi.SetMetricById(metric.GetMetricId())
			require.NoError(t, m.AddMetric(metric))
			require.NoError(t, m.AddMetricInstance(mi))`,
		CustomMethods: []string{
			"GetMetric() (Metric, error)",
			"GetMetricId() uuid.UUID",
			"SetMetricRef(*MetricRef)",
			"SetMetricByRef(metric Metric)",
			"SetMetricById(metricId uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "MetricRef",
			RefTypeName:       "MetricRef",
			ResourceTypeName:  "Metric",
			ResolvedMethod:    "ResolvedMetric",
			EmbedFieldName:    "Metric",
			RefIDFieldName:    "MetricId",
			ResourceIDGetter:  "GetMetricId",
			ModelLookupByID:   "GetMetricById",
			EffectiveIDMethod: "EffectiveMetricID",
			SetByID:           true,
			SetByIDParamName:  "metricId",
		},
	},
	"MetricValue": {
		Fields: []Field{
			{Name: "DisplayName", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "MetricInstanceRef", Type: "*MetricInstanceRef", SkipAccessor: true},
			{Name: "Value", Type: "string"},
			{Name: "Annotations", Type: "annotations.Annotations", HasAnnotations: true},
		},
		TestDisplayName: "Current p99 latency",
		TestDeps:        []string{"Metric", "MetricInstance"},
		TestSetup: `mv := mdlobs.NewMetricValue(testIDs["MetricValue"])
			mv.SetDisplayName("Current p99 latency")
			metric := mdlobs.NewMetric(uuid.New())
			metric.SetDisplayName("p99 API latency")
			mi := mdlobs.NewMetricInstance(uuid.New())
			mi.SetDisplayName("p99 API latency for orders-api")
			mi.SetMetricById(metric.GetMetricId())
			mv.SetMetricInstanceById(mi.GetMetricInstanceId())
			mv.SetValue("412")
			require.NoError(t, m.AddMetric(metric))
			require.NoError(t, m.AddMetricInstance(mi))
			require.NoError(t, m.AddMetricValue(mv))`,
		CustomMethods: []string{
			"GetMetricInstance() (MetricInstance, error)",
			"GetMetricInstanceId() uuid.UUID",
			"SetMetricInstanceRef(*MetricInstanceRef)",
			"SetMetricInstanceByRef(mi MetricInstance)",
			"SetMetricInstanceById(id uuid.UUID)",
		},
		TypeRefLink: &TypeRefLinkSpec{
			FieldName:         "MetricInstanceRef",
			RefTypeName:       "MetricInstanceRef",
			ResourceTypeName:  "MetricInstance",
			ResolvedMethod:    "ResolvedMetricInstance",
			EmbedFieldName:    "MetricInstance",
			RefIDFieldName:    "MetricInstanceId",
			ResourceIDGetter:  "GetMetricInstanceId",
			ModelLookupByID:   "GetMetricInstanceById",
			EffectiveIDMethod: "EffectiveMetricInstanceID",
			SetByID:           true,
			SetByIDParamName:  "id",
		},
	},
}
