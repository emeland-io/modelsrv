package oapi

import "go.emeland.io/modelsrv/pkg/events"

// normalizeReplicationWireMap mutates wire so JSON matches OpenAPI shapes after encoding/json was
// applied to domain structs (which emit nested ref objects instead of plain UUID fields).
func normalizeReplicationWireMap(rt events.ResourceType, wire map[string]interface{}) {
	stripInvalidAnnotationsForOpenAPI(wire)
	switch rt {
	case events.SystemResource:
		coalesceObjectToUUIDScalar(wire, "Parent", "parent", "SystemId", "systemId")

	case events.SystemInstanceResource:
		coalesceObjectToUUIDScalar(wire, "System", "system", "SystemId", "systemId")
		coalesceObjectToUUIDScalar(wire, "Context", "context", "ContextId", "contextId")

	case events.APIResource:
		coalesceObjectToUUIDScalar(wire, "System", "system", "SystemId", "systemId")

	case events.APIInstanceResource:
		coalesceObjectToUUIDScalar(wire, "Api", "api", "ApiID", "ApiId", "apiId")
		coalesceObjectToUUIDScalar(wire, "SystemInstance", "systemInstance", "InstanceId", "instanceId")

	case events.ComponentResource:
		coalesceObjectToUUIDScalar(wire, "System", "system", "SystemId", "systemId")

	case events.ComponentInstanceResource:
		coalesceObjectToUUIDScalar(wire, "Component", "component", "ComponentId", "componentId")
		coalesceObjectToUUIDScalar(wire, "SystemInstance", "systemInstance", "InstanceId", "instanceId")

	case events.ContextResource:
		coalesceRefSourcesToOpenAPIField(wire, []string{"TypeRef", "Type", "type"}, "type", "ContextTypeId", "contextTypeId")
		coalesceRefSourcesToOpenAPIField(wire, []string{"Parent", "parent"}, "parent", "ContextId", "contextId")

	case events.NodeResource:
		coalesceRefSourcesToOpenAPIField(wire, []string{"TypeRef", "NodeType", "nodeType"}, "nodeType", "NodeTypeId", "nodeTypeId")

	case events.ProductResource:
		coalesceRefSourcesToOpenAPIField(wire, []string{"Vendor", "vendor"}, "vendor", "OrgUnitId", "orgUnitId")

	case events.FindingResource:
		coalesceObjectToUUIDScalar(wire, "Type", "type", "FindingTypeId", "findingTypeId")
	}
}

// coalesceRefSourcesToOpenAPIField sets wire[openAPIJSONKey] from the first non-nil source key.
// Values may be bare UUID scalars (from OpenAPI-shaped payloads) or nested ref objects whose id
// appears under one of idKeys (domain TypeRef / Parent shapes from encoding/json).
func coalesceRefSourcesToOpenAPIField(wire map[string]interface{}, sourceKeys []string, openAPIJSONKey string, idKeys ...string) {
	for _, sk := range sourceKeys {
		if sk == "" {
			continue
		}
		v, ok := wire[sk]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			wire[openAPIJSONKey] = t
			deleteConflictingRefSourceKeys(wire, sourceKeys, openAPIJSONKey)
			return
		case float64:
			wire[openAPIJSONKey] = t
			deleteConflictingRefSourceKeys(wire, sourceKeys, openAPIJSONKey)
			return
		case map[string]interface{}:
			for _, ik := range idKeys {
				if x, ok := t[ik]; ok && x != nil {
					wire[openAPIJSONKey] = x
					deleteConflictingRefSourceKeys(wire, sourceKeys, openAPIJSONKey)
					return
				}
			}
		default:
			// Unexpected shape; try next source key.
		}
	}
}

// deleteConflictingRefSourceKeys removes domain / alternate wire keys so json.Unmarshal into OpenAPI DTOs
// does not see both "Parent" and "parent" (Go's decoder may bind case-variant keys to one field).
func deleteConflictingRefSourceKeys(wire map[string]interface{}, sourceKeys []string, openAPIJSONKey string) {
	for _, sk := range sourceKeys {
		if sk != "" && sk != openAPIJSONKey {
			delete(wire, sk)
		}
	}
}

// stripInvalidAnnotationsForOpenAPI removes "annotations" values that cannot unmarshal into
// *[]Annotation (OpenAPI). Domain [annotations.Annotations] marshals as an empty JSON object
// {} because its state is unexported, but the wire schema expects a JSON array.
func stripInvalidAnnotationsForOpenAPI(wire map[string]interface{}) {
	for _, key := range []string{"Annotations", "annotations"} {
		v, ok := wire[key]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case map[string]interface{}:
			if len(t) == 0 {
				delete(wire, key)
			}
		default:
			// Already an array from a real OpenAPI-shaped payload; leave as-is.
		}
	}
}

// coalesceObjectToUUIDScalar replaces wire[key] when it is a JSON object that only carries an id
// (e.g. SystemRef) with the bare UUID scalar expected by OpenAPI DTOs.
func coalesceObjectToUUIDScalar(wire map[string]interface{}, objectKeyPrimary, objectKeyAlt string, idKeys ...string) {
	var v interface{}
	var key string
	for _, k := range []string{objectKeyPrimary, objectKeyAlt} {
		if x, ok := wire[k]; ok && x != nil {
			v, key = x, k
			break
		}
	}
	if key == "" {
		return
	}
	if _, ok := v.(string); ok {
		return
	}
	if _, ok := v.(float64); ok {
		return
	}
	sub, ok := v.(map[string]interface{})
	if !ok {
		return
	}
	for _, ik := range idKeys {
		if x, ok := sub[ik]; ok && x != nil {
			wire[key] = x
			return
		}
	}
}
