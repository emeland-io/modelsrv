package filesensor

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/component"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.emeland.io/modelsrv/pkg/model/node"
	"go.emeland.io/modelsrv/pkg/model/system"
)

func firstUUIDField(spec map[string]any, keys ...string) (uuid.UUID, error) {
	for _, k := range keys {
		id, err := parseUUIDField(spec, k)
		if err == nil {
			return id, nil
		}
	}
	return uuid.Nil, fmt.Errorf("spec must include a valid UUID in one of: %s", strings.Join(keys, ", "))
}

func parseResourceTypeForRef(s string) (events.ResourceType, error) {
	s = strings.TrimSpace(s)
	t := events.ParseResourceType(s)
	if t != events.UnknownResourceType {
		return t, nil
	}
	switch s {
	case "Finding":
		return events.FindingResource, nil
	case "FindingType":
		return events.FindingTypeResource, nil
	default:
		return 0, fmt.Errorf("unknown resource type %q", s)
	}
}

func parseResourceRefs(spec map[string]any) ([]*common.ResourceRef, error) {
	raw, ok := spec["resources"]
	if !ok || raw == nil {
		return nil, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("resources must be an array")
	}
	var out []*common.ResourceRef
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("resources[%d] must be an object", i)
		}
		id, err := parseUUIDField(m, "resourceId")
		if err != nil {
			return nil, fmt.Errorf("resources[%d]: %w", i, err)
		}
		rtStr, ok := stringField(m, "resourceType")
		if !ok {
			return nil, fmt.Errorf("resources[%d]: resourceType is required", i)
		}
		rt, err := parseResourceTypeForRef(rtStr)
		if err != nil {
			return nil, fmt.Errorf("resources[%d]: %w", i, err)
		}
		out = append(out, &common.ResourceRef{ResourceId: id, ResourceType: rt})
	}
	return out, nil
}

func parseApiRefSlice(spec map[string]any, key string) ([]mdlapi.ApiRef, error) {
	raw, ok := spec[key]
	if !ok || raw == nil {
		return nil, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an array", key)
	}
	var out []mdlapi.ApiRef
	for i, item := range arr {
		switch t := item.(type) {
		case string:
			id, err := uuid.Parse(strings.TrimSpace(t))
			if err != nil {
				return nil, fmt.Errorf("%s[%d]: %w", key, i, err)
			}
			if id == uuid.Nil {
				return nil, fmt.Errorf("%s[%d]: UUID must not be nil", key, i)
			}
			out = append(out, mdlapi.ApiRef{ApiID: id})
		case map[string]any:
			id, err := parseUUIDField(t, "apiId")
			if err != nil {
				return nil, fmt.Errorf("%s[%d]: %w", key, i, err)
			}
			out = append(out, mdlapi.ApiRef{ApiID: id})
		default:
			return nil, fmt.Errorf("%s[%d]: must be a UUID string or object with apiId", key, i)
		}
	}
	return out, nil
}

func findingTitle(spec map[string]any) (string, error) {
	if s, ok := stringField(spec, "summary"); ok && strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s), nil
	}
	return displayName(spec)
}

func applyContextType(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "contextTypeId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	ct := mdlctx.NewContextType(m.GetSink(), id)
	ct.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		ct.SetDescription(desc)
	}
	if err := applyAnnotations(ct.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddContextType(ct)
}

func applyNodeType(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "nodeTypeId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	nt := node.NewNodeType(m.GetSink(), id)
	nt.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		nt.SetDescription(desc)
	}
	if err := applyAnnotations(nt.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddNodeType(nt)
}

func applyNode(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "nodeId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	n := node.NewNode(m.GetSink(), id)
	n.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		n.SetDescription(desc)
	}
	if typeID, has, err := optionalUUIDRef(spec, "nodeTypeId"); err != nil {
		return err
	} else if has {
		n.SetTypeRef(&node.NodeTypeRef{NodeTypeId: typeID})
	}
	if err := applyAnnotations(n.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddNode(n)
}

func applySystemInstance(spec map[string]any, m model.Model) error {
	id, err := firstUUIDField(spec, "instanceId", "systemInstanceId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	sid, err := firstUUIDField(spec, "system", "systemId")
	if err != nil {
		return err
	}
	si := system.NewSystemInstance(m.GetSink(), id)
	si.SetDisplayName(name)
	si.SetSystemRef(&system.SystemRef{SystemId: sid})
	if cid, has, err := optionalFirstUUIDRef(spec, "context", "contextId"); err != nil {
		return err
	} else if has {
		si.SetContextRef(&mdlctx.ContextRef{ContextId: cid})
	}
	if err := applyAnnotations(si.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddSystemInstance(si)
}

func applyAPIInstance(spec map[string]any, m model.Model) error {
	id, err := firstUUIDField(spec, "instanceId", "apiInstanceId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	apiID, err := firstUUIDField(spec, "api", "apiId")
	if err != nil {
		return err
	}
	ai := mdlapi.NewApiInstance(m.GetSink(), id)
	ai.SetDisplayName(name)
	ai.SetApiRef(&mdlapi.ApiRef{ApiID: apiID})
	if sid, has, err := optionalFirstUUIDRef(spec, "systemInstance", "systemInstanceId"); err != nil {
		return err
	} else if has {
		ai.SetSystemInstance(&system.SystemInstanceRef{InstanceId: sid})
	}
	if err := applyAnnotations(ai.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddApiInstance(ai)
}

func applyComponent(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "componentId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	sysID, err := firstUUIDField(spec, "system", "systemId")
	if err != nil {
		return err
	}
	c := component.NewComponent(m.GetSink(), id)
	c.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		c.SetDescription(desc)
	}
	if ver, err := parseVersionSpec(spec["version"]); err != nil {
		return err
	} else {
		c.SetVersion(ver)
	}
	c.SetSystem(&system.SystemRef{SystemId: sysID})
	if cons, err := parseApiRefSlice(spec, "consumes"); err != nil {
		return err
	} else if len(cons) > 0 {
		c.SetConsumes(cons)
	}
	if prov, err := parseApiRefSlice(spec, "provides"); err != nil {
		return err
	} else if len(prov) > 0 {
		c.SetProvides(prov)
	}
	if err := applyAnnotations(c.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddComponent(c)
}

func applyComponentInstance(spec map[string]any, m model.Model) error {
	id, err := firstUUIDField(spec, "instanceId", "componentInstanceId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	compID, err := firstUUIDField(spec, "component", "componentId")
	if err != nil {
		return err
	}
	ci := component.NewComponentInstance(m.GetSink(), id)
	ci.SetDisplayName(name)
	ci.SetComponentRef(&component.ComponentRef{ComponentId: compID})
	if sid, has, err := optionalFirstUUIDRef(spec, "systemInstance", "systemInstanceId"); err != nil {
		return err
	} else if has {
		ci.SetSystemInstance(&system.SystemInstanceRef{InstanceId: sid})
	}
	if err := applyAnnotations(ci.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddComponentInstance(ci)
}

func applyFinding(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "findingId")
	if err != nil {
		return err
	}
	title, err := findingTitle(spec)
	if err != nil {
		return err
	}
	f := finding.NewFinding(m.GetSink(), id)
	f.SetSummary(title)
	if desc, ok := stringField(spec, "description"); ok {
		f.SetDescription(desc)
	}
	refs, err := parseResourceRefs(spec)
	if err != nil {
		return err
	}
	if len(refs) > 0 {
		f.SetResources(refs)
	}
	if err := applyAnnotations(f.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddFinding(f, title)
}

func applyFindingType(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "findingTypeId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	ft := finding.NewFindingType(m.GetSink(), id)
	ft.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		ft.SetDescription(desc)
	}
	if err := applyAnnotations(ft.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddFindingType(ft)
}
