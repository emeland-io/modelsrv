package main

import "strings"

var skipConvertByName = map[string]bool{
	"Context":           true,
	"Node":              true,
	"System":            true,
	"SystemInstance":    true,
	"API":               true,
	"ApiInstance":       true,
	"Component":         true,
	"ComponentInstance": true,
	"Finding":           true,
	"FindingType":       true,
	"Product":           true,
	"ArtifactInstance":  true,
	"PermissionSpec":    true,
	"RoleSpec":          true,
	"Permission":        true,
	"Role":              true,
	"Binding":           true,
}

var wireKindToEventsResource = map[string]string{
	"ContextType":       "ContextTypeResource",
	"Context":           "ContextResource",
	"System":            "SystemResource",
	"SystemInstance":    "SystemInstanceResource",
	"API":               "APIResource",
	"ApiInstance":       "APIInstanceResource",
	"Component":         "ComponentResource",
	"ComponentInstance": "ComponentInstanceResource",
	"NodeType":          "NodeTypeResource",
	"Node":              "NodeResource",
	"FindingType":       "FindingTypeResource",
	"Finding":           "FindingResource",
	"OrgUnit":           "OrgUnitResource",
	"Group":             "GroupResource",
	"Identity":          "IdentityResource",
	"Product":           "ProductResource",
	"Artifact":          "ArtifactResource",
	"ArtifactInstance":  "ArtifactInstanceResource",
	"PermissionSpec":    "PermissionSpecResource",
	"RoleSpec":          "RoleSpecResource",
	"Permission":        "PermissionResource",
	"Role":              "RoleResource",
	"Binding":           "BindingResource",
}

var restListPathByName = map[string]string{
	"ContextType":       "/landscape/contextTypes",
	"Context":           "/landscape/contexts",
	"System":            "/landscape/systems",
	"SystemInstance":    "/landscape/system-instances",
	"API":               "/landscape/apis",
	"ApiInstance":       "/landscape/api-instances",
	"Component":         "/landscape/components",
	"ComponentInstance": "/landscape/component-instances",
	"NodeType":          "/landscape/nodeTypes",
	"Node":              "/landscape/nodes",
	"FindingType":       "/landscape/findingTypes",
	"Finding":           "/landscape/findings",
	"OrgUnit":           "/landscape/orgUnits",
	"Group":             "/landscape/groups",
	"Identity":          "/landscape/identities",
	"Product":           "/landscape/products",
	"Artifact":          "/landscape/artifacts",
	"ArtifactInstance":  "/landscape/artifactInstances",
	"PermissionSpec":    "/landscape/permissionSpecs",
	"RoleSpec":          "/landscape/roleSpecs",
	"Permission":        "/landscape/permissions",
	"Role":              "/landscape/roles",
	"Binding":           "/landscape/bindings",
}

var serverRequestIDByName = map[string]string{
	"SystemInstance":    "SystemInstanceId",
	"ApiInstance":       "ApiInstanceId",
	"ComponentInstance": "ComponentInstanceId",
	"ArtifactInstance":  "ArtifactInstanceId",
}

var serverResourceLabelByName = map[string]string{
	"Context":           "context",
	"ContextType":       "context type",
	"Node":              "node",
	"NodeType":          "node type",
	"System":            "system",
	"SystemInstance":    "system instance",
	"API":               "api",
	"ApiInstance":       "api instance",
	"Component":         "component",
	"ComponentInstance": "componentInstance",
	"Finding":           "finding",
	"FindingType":       "finding type",
	"OrgUnit":           "organizational unit",
	"Group":             "group",
	"Identity":          "identity",
	"Product":           "product",
	"Artifact":          "artifact",
	"ArtifactInstance":  "artifact instance",
	"PermissionSpec":    "permission specification",
	"RoleSpec":          "role specification",
	"Permission":        "permission",
	"Role":              "role",
	"Binding":           "binding",
}

var server404UseErrorString = map[string]bool{
	"Finding":          true,
	"Group":            true,
	"Identity":         true,
	"Product":          true,
	"OrgUnit":          true,
	"Artifact":         true,
	"ArtifactInstance": true,
	"PermissionSpec":   true,
	"RoleSpec":         true,
	"Permission":       true,
	"Role":             true,
	"Binding":          true,
}

var wireIDFieldByName = map[string]string{
	"SystemInstance":    "SystemInstanceId",
	"ApiInstance":       "ApiInstanceId",
	"ComponentInstance": "ComponentInstanceId",
	"ArtifactInstance":  "ArtifactInstanceId",
}

var convertDomainIDByName = map[string]string{
	"SystemInstance":    "GetInstanceId()",
	"ApiInstance":       "GetInstanceId()",
	"ComponentInstance": "GetInstanceId()",
}

var backendListByName = map[string]string{
	"API": "GetApis",
}

var backendGetByIDByName = map[string]string{
	"API": "GetApiById",
}

var convertNilLabelByName = map[string]string{
	"API": "API",
}

var convertMissingIDByName = map[string]string{
	"System":      "system event missing systemId",
	"API":         "API event missing apiId",
	"Component":   "component event missing componentId",
	"FindingType": "finding type event missing findingTypeId",
}

func enrichWireMeta(spec *TypeSpec) {
	spec.SkipConvert = skipConvertByName[spec.Name]

	if spec.OapiTypeName != "" {
		spec.OapiWireTypeName = spec.OapiTypeName
	} else {
		spec.OapiWireTypeName = spec.Name
	}

	spec.FromDtoFuncName = spec.Name + "FromDto"
	spec.ToDtoFuncName = spec.Name + "ToDto"
	if spec.Name == "API" {
		spec.FromDtoFuncName = "APIFromDto"
		spec.ToDtoFuncName = "APIToDto"
	}

	if spec.WireKind != "" {
		spec.EventsResource = wireKindToEventsResource[spec.WireKind]
	}

	if v, ok := wireIDFieldByName[spec.Name]; ok {
		spec.WireIDField = v
	} else {
		spec.WireIDField = spec.IDField
	}

	spec.WireIDOptional = strings.Contains(spec.TestIDAssertExpr, "*got.")

	if v, ok := convertDomainIDByName[spec.Name]; ok {
		spec.ConvertDomainIDMethod = v
	} else {
		spec.ConvertDomainIDMethod = "Get" + spec.IDField + "()"
	}
	spec.WireDomainIDGetter = strings.TrimSuffix(spec.ConvertDomainIDMethod, "()")

	if spec.GenClientMethods {
		spec.GenServerHandlers = true
		if v, ok := backendListByName[spec.Name]; ok {
			spec.BackendListMethod = v
		} else {
			spec.BackendListMethod = spec.ClientListMethod
		}
		if v, ok := backendGetByIDByName[spec.Name]; ok {
			spec.BackendGetByIdMethod = v
		} else {
			spec.BackendGetByIdMethod = spec.ClientGetByIdMethod
		}
	}

	if v, ok := restListPathByName[spec.Name]; ok {
		spec.RestListPath = v
	}
	if v, ok := serverRequestIDByName[spec.Name]; ok {
		spec.ServerRequestIDField = v
	} else {
		spec.ServerRequestIDField = spec.IDField
	}
	if v, ok := serverResourceLabelByName[spec.Name]; ok {
		spec.ServerResourceLabel = v
	}
	spec.Server404UseErrorString = server404UseErrorString[spec.Name]

	if v, ok := convertNilLabelByName[spec.Name]; ok {
		spec.ConvertNilLabel = v
	} else if spec.ServerResourceLabel != "" {
		spec.ConvertNilLabel = spec.ServerResourceLabel
	} else {
		spec.ConvertNilLabel = strings.ToLower(spec.Name)
	}

	if v, ok := convertMissingIDByName[spec.Name]; ok {
		spec.ConvertMissingIDMsg = v
	}
}
