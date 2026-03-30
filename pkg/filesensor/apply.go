package filesensor

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/annotations"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/system"
)

// AnnotationContextTypeID is the annotation key used to store a Context's context-type UUID when the domain type has no dedicated field.
const AnnotationContextTypeID = "contextTypeId"

func displayName(spec map[string]any) (string, error) {
	if s, ok := stringField(spec, "displayName"); ok && strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s), nil
	}
	if s, ok := stringField(spec, "name"); ok && strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s), nil
	}
	return "", fmt.Errorf("spec must include non-empty displayName or name")
}

func stringField(spec map[string]any, key string) (string, bool) {
	v, ok := spec[key]
	if !ok || v == nil {
		return "", false
	}
	switch t := v.(type) {
	case string:
		return t, true
	default:
		return fmt.Sprint(t), true
	}
}

func parseUUIDField(spec map[string]any, key string) (uuid.UUID, error) {
	s, ok := stringField(spec, key)
	if !ok || strings.TrimSpace(s) == "" {
		return uuid.Nil, fmt.Errorf("missing or empty %q", key)
	}
	id, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%q: %w", key, err)
	}
	if id == uuid.Nil {
		return uuid.Nil, fmt.Errorf("%q must not be nil UUID", key)
	}
	return id, nil
}

func optionalUUIDRef(spec map[string]any, key string) (uuid.UUID, bool, error) {
	v, ok := spec[key]
	if !ok || v == nil {
		return uuid.Nil, false, nil
	}
	s, ok := v.(string)
	if !ok {
		return uuid.Nil, false, fmt.Errorf("%q must be a string UUID or null", key)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return uuid.Nil, false, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("%q: %w", key, err)
	}
	if id == uuid.Nil {
		return uuid.Nil, true, fmt.Errorf("%q must not be nil UUID", key)
	}
	return id, true, nil
}

func optionalFirstUUIDRef(spec map[string]any, keys ...string) (uuid.UUID, bool, error) {
	for _, k := range keys {
		id, has, err := optionalUUIDRef(spec, k)
		if err != nil {
			return uuid.Nil, false, err
		}
		if has {
			return id, true, nil
		}
	}
	return uuid.Nil, false, nil
}

func applyAnnotations(ann annotations.Annotations, spec map[string]any) error {
	raw, ok := spec["annotations"]
	if !ok || raw == nil {
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("annotations must be a map")
	}
	for k, v := range m {
		if v == nil {
			continue
		}
		val, ok := v.(string)
		if !ok {
			val = fmt.Sprint(v)
		}
		ann.Add(k, val)
	}
	return nil
}

func parseVersionSpec(v any) (common.Version, error) {
	var out common.Version
	if v == nil {
		return out, nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return out, fmt.Errorf("version must be a map")
	}
	if s, ok := stringField(m, "version"); ok {
		out.Version = s
	}
	if t, err := parseOptionalTime(m, "availableFrom"); err != nil {
		return out, err
	} else {
		out.AvailableFrom = t
	}
	if t, err := parseOptionalTime(m, "deprecatedFrom"); err != nil {
		return out, err
	} else {
		out.DeprecatedFrom = t
	}
	if t, err := parseOptionalTime(m, "terminatedFrom"); err != nil {
		return out, err
	} else {
		out.TerminatedFrom = t
	}
	if out.TerminatedFrom == nil {
		if t, err := parseOptionalTime(m, "retiredFrom"); err != nil {
			return out, err
		} else {
			out.TerminatedFrom = t
		}
	}
	return out, nil
}

func parseOptionalTime(m map[string]any, key string) (*time.Time, error) {
	v, ok := m[key]
	if !ok || v == nil {
		return nil, nil
	}
	switch t := v.(type) {
	case string:
		pt, err := time.Parse(time.RFC3339, strings.TrimSpace(t))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		return &pt, nil
	case time.Time:
		return &t, nil
	default:
		return nil, fmt.Errorf("%s must be an RFC3339 string or time value", key)
	}
}

func parseApiType(s string) (mdlapi.ApiType, error) {
	switch strings.TrimSpace(s) {
	case "OpenAPI":
		return mdlapi.OpenAPI, nil
	case "GraphQL":
		return mdlapi.GraphQL, nil
	case "GRPC":
		return mdlapi.GRPC, nil
	case "Other":
		return mdlapi.Other, nil
	case "Unknown":
		return mdlapi.Unknown, nil
	default:
		return mdlapi.Unknown, fmt.Errorf("invalid API type %q (expected OpenAPI, GraphQL, GRPC, Other, or Unknown)", s)
	}
}

// ApplyDocument validates and applies a single decoded [Document] to the model.
func ApplyDocument(doc Document, m model.Model) error {
	if !ValidVersion(doc.Version) {
		return fmt.Errorf("unsupported version %q (expected emeland.io/...)", doc.Version)
	}
	if doc.Spec == nil {
		return fmt.Errorf("spec is required")
	}

	rt := doc.Kind.ResourceType()
	switch rt {
	case events.ContextResource:
		return applyContext(doc.Spec, m)
	case events.ContextTypeResource:
		return applyContextType(doc.Spec, m)
	case events.NodeResource:
		return applyNode(doc.Spec, m)
	case events.NodeTypeResource:
		return applyNodeType(doc.Spec, m)
	case events.SystemResource:
		return applySystem(doc.Spec, m)
	case events.SystemInstanceResource:
		return applySystemInstance(doc.Spec, m)
	case events.APIResource:
		return applyAPI(doc.Spec, m)
	case events.APIInstanceResource:
		return applyAPIInstance(doc.Spec, m)
	case events.ComponentResource:
		return applyComponent(doc.Spec, m)
	case events.ComponentInstanceResource:
		return applyComponentInstance(doc.Spec, m)
	case events.FindingResource:
		return applyFinding(doc.Spec, m)
	case events.FindingTypeResource:
		return applyFindingType(doc.Spec, m)
	case events.UnknownResourceType:
		return fmt.Errorf("kind is required")
	default:
		return fmt.Errorf("unsupported kind %s", rt)
	}
}

func applyContext(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "contextId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}

	ctx := mdlctx.NewContext(m.GetSink(), id)
	ctx.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		ctx.SetDescription(desc)
	}

	if parentID, hasParent, err := optionalUUIDRef(spec, "parent"); err != nil {
		return err
	} else if hasParent {
		ctx.SetParentById(parentID)
	}

	if typeID, hasType, err := optionalUUIDRef(spec, "type"); err != nil {
		return err
	} else if hasType {
		ctx.GetAnnotations().Add(AnnotationContextTypeID, typeID.String())
	}

	if err := applyAnnotations(ctx.GetAnnotations(), spec); err != nil {
		return err
	}

	return m.AddContext(ctx)
}

func applySystem(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "systemId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}

	sys := system.NewSystem(m.GetSink(), id)
	sys.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		sys.SetDescription(desc)
	}
	if v, ok := spec["abstract"]; ok {
		b, ok := v.(bool)
		if !ok {
			return fmt.Errorf("abstract must be a boolean")
		}
		sys.SetAbstract(b)
	}
	if ver, err := parseVersionSpec(spec["version"]); err != nil {
		return err
	} else {
		sys.SetVersion(ver)
	}

	if parentID, hasParent, err := optionalUUIDRef(spec, "parent"); err != nil {
		return err
	} else if hasParent {
		sys.SetParent(&system.SystemRef{SystemId: parentID})
	}

	if err := applyAnnotations(sys.GetAnnotations(), spec); err != nil {
		return err
	}

	return m.AddSystem(sys)
}

func applyAPI(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "apiId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}

	var systemID uuid.UUID
	if s, ok := stringField(spec, "system"); ok && strings.TrimSpace(s) != "" {
		systemID, err = uuid.Parse(strings.TrimSpace(s))
		if err != nil {
			return fmt.Errorf("system: %w", err)
		}
	} else if s, ok := stringField(spec, "systemId"); ok && strings.TrimSpace(s) != "" {
		systemID, err = uuid.Parse(strings.TrimSpace(s))
		if err != nil {
			return fmt.Errorf("systemId: %w", err)
		}
	} else {
		return fmt.Errorf("spec must include system or systemId UUID")
	}
	if systemID == uuid.Nil {
		return fmt.Errorf("system UUID must not be nil")
	}

	typeStr := "OpenAPI"
	if s, ok := stringField(spec, "type"); ok && strings.TrimSpace(s) != "" {
		typeStr = strings.TrimSpace(s)
	}
	apiType, err := parseApiType(typeStr)
	if err != nil {
		return err
	}

	api := mdlapi.NewAPI(m.GetSink(), id)
	api.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		api.SetDescription(desc)
	}
	api.SetType(apiType)
	if ver, err := parseVersionSpec(spec["version"]); err != nil {
		return err
	} else {
		api.SetVersion(ver)
	}
	api.SetSystem(&system.SystemRef{SystemId: systemID})

	if err := applyAnnotations(api.GetAnnotations(), spec); err != nil {
		return err
	}

	return m.AddApi(api)
}
