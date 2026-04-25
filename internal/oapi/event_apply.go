package oapi

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/annotations"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/component"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/system"
)

// ReplicationEventFromWire converts an OpenAPI push body into a domain [events.Event] for [model.EventApplier.Apply].
func ReplicationEventFromWire(m model.Model, ev *Event) (events.Event, error) {
	if m == nil {
		return events.Event{}, fmt.Errorf("nil model")
	}
	if ev == nil {
		return events.Event{}, fmt.Errorf("nil event")
	}

	kind := strings.TrimSpace(ev.Kind)
	op := wireString(ev.Operation)
	rt := events.ParseWireKind(kind)
	if rt == events.UnknownResourceType {
		return events.Event{}, fmt.Errorf("unknown event kind %q", kind)
	}
	wop := events.ParseWireOperation(op)
	if wop == events.UnknownOperation {
		return events.Event{}, fmt.Errorf("unknown event operation %q", op)
	}

	switch wop {
	case events.DeleteOperation:
		if ev.ResourceId == nil {
			return events.Event{}, fmt.Errorf("delete event missing resourceId")
		}
		id := uuid.UUID(*ev.ResourceId)
		return events.Event{
			ResourceType: rt,
			Operation:    wop,
			ResourceId:   id,
		}, nil
	case events.CreateOperation, events.UpdateOperation:
		if ev.Resource == nil {
			return events.Event{}, fmt.Errorf("missing resource payload for %s %s", kind, op)
		}
		id, obj, err := decodeReplicationResourceFromMap(m, rt, ev.Resource)
		if err != nil {
			return events.Event{}, err
		}
		return events.Event{
			ResourceType: rt,
			Operation:    wop,
			ResourceId:   id,
			Objects:      []any{obj},
		}, nil
	default:
		return events.Event{}, fmt.Errorf("unsupported operation %q", op)
	}
}

func wireString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	default:
		return fmt.Sprint(v)
	}
}

func systemFromWire(m model.Model, os System) (system.System, error) {
	if os.SystemId == nil {
		return nil, fmt.Errorf("system event missing systemId")
	}
	id := uuid.UUID(*os.SystemId)
	sys := system.NewSystem(m.GetSink(), id)
	sys.SetDisplayName(os.DisplayName)
	if os.Description != nil {
		sys.SetDescription(*os.Description)
	}
	sys.SetAbstract(os.Abstract)
	if os.Version != nil {
		sys.SetVersion(oapiVersionToModel(os.Version))
	}
	if os.Parent != nil {
		sys.SetParent(refSystem(m, uuid.UUID(*os.Parent)))
	}
	mergeOapiAnnotations(sys.GetAnnotations(), os.Annotations)
	return sys, nil
}

func systemInstanceFromWire(m model.Model, os SystemInstance) (system.SystemInstance, error) {
	id := uuid.UUID(os.SystemInstanceId)
	si := system.NewSystemInstance(m.GetSink(), id)
	si.SetDisplayName(os.DisplayName)
	si.SetSystemRef(refSystem(m, uuid.UUID(os.System)))
	if os.Context != nil {
		si.SetContextRef(&mdlctx.ContextRef{ContextId: uuid.UUID(*os.Context)})
	}
	mergeOapiAnnotations(si.GetAnnotations(), os.Annotations)
	return si, nil
}

func apiFromWire(m model.Model, oa API) (mdlapi.API, error) {
	if oa.ApiId == nil {
		return nil, fmt.Errorf("API event missing apiId")
	}
	id := uuid.UUID(*oa.ApiId)
	dom := mdlapi.NewAPI(m.GetSink(), id)
	dom.SetDisplayName(oa.DisplayName)
	if oa.Description != nil {
		dom.SetDescription(*oa.Description)
	}
	dom.SetType(parseAPIType(oa.Type))
	if oa.Version != nil {
		dom.SetVersion(oapiVersionToModel(oa.Version))
	}
	if oa.System != nil {
		dom.SetSystem(refSystem(m, uuid.UUID(*oa.System)))
	}
	mergeOapiAnnotations(dom.GetAnnotations(), oa.Annotations)
	return dom, nil
}

func apiInstanceFromWire(m model.Model, oa ApiInstance) (mdlapi.ApiInstance, error) {
	id := uuid.UUID(oa.ApiInstanceId)
	ai := mdlapi.NewApiInstance(m.GetSink(), id)
	ai.SetDisplayName(oa.DisplayName)
	if oa.Api != nil {
		ai.SetApiRef(refAPI(m, uuid.UUID(*oa.Api)))
	}
	if oa.SystemInstance != nil {
		ai.SetSystemInstance(refSystemInstance(m, uuid.UUID(*oa.SystemInstance)))
	}
	mergeOapiAnnotations(ai.GetAnnotations(), oa.Annotations)
	return ai, nil
}

func componentFromWire(m model.Model, oc Component) (component.Component, error) {
	if oc.ComponentId == nil {
		return nil, fmt.Errorf("component event missing componentId")
	}
	id := uuid.UUID(*oc.ComponentId)
	c := component.NewComponent(m.GetSink(), id)
	c.SetDisplayName(oc.DisplayName)
	if oc.Description != nil {
		c.SetDescription(*oc.Description)
	}
	if oc.Version != nil {
		c.SetVersion(oapiVersionToModel(oc.Version))
	}
	c.SetSystem(refSystem(m, uuid.UUID(oc.System)))
	if oc.Consumes != nil {
		c.SetConsumes(apiRefsFromUUIDs(m, *oc.Consumes))
	}
	if oc.Provides != nil {
		c.SetProvides(apiRefsFromUUIDs(m, *oc.Provides))
	}
	mergeOapiAnnotations(c.GetAnnotations(), oc.Annotations)
	return c, nil
}

func componentInstanceFromWire(m model.Model, oc ComponentInstance) (component.ComponentInstance, error) {
	id := uuid.UUID(oc.ComponentInstanceId)
	ci := component.NewComponentInstance(m.GetSink(), id)
	ci.SetDisplayName(oc.DisplayName)
	ci.SetComponentRef(refComponent(m, uuid.UUID(oc.Component)))
	ci.SetSystemInstance(refSystemInstance(m, uuid.UUID(oc.SystemInstance)))
	mergeOapiAnnotations(ci.GetAnnotations(), oc.Annotations)
	return ci, nil
}

func refSystem(m model.Model, id uuid.UUID) *system.SystemRef {
	if id == uuid.Nil {
		return nil
	}
	if s := m.GetSystemById(id); s != nil {
		return &system.SystemRef{System: s, SystemId: id}
	}
	return &system.SystemRef{SystemId: id}
}

func refAPI(m model.Model, id uuid.UUID) *mdlapi.ApiRef {
	if id == uuid.Nil {
		return nil
	}
	if a := m.GetApiById(id); a != nil {
		return &mdlapi.ApiRef{API: a, ApiID: id}
	}
	return &mdlapi.ApiRef{ApiID: id}
}

func refSystemInstance(m model.Model, id uuid.UUID) *system.SystemInstanceRef {
	if id == uuid.Nil {
		return nil
	}
	if si := m.GetSystemInstanceById(id); si != nil {
		return &system.SystemInstanceRef{SystemInstance: si, InstanceId: id}
	}
	return &system.SystemInstanceRef{InstanceId: id}
}

func refComponent(m model.Model, id uuid.UUID) *component.ComponentRef {
	if id == uuid.Nil {
		return nil
	}
	if c := m.GetComponentById(id); c != nil {
		return &component.ComponentRef{Component: c, ComponentId: id}
	}
	return &component.ComponentRef{ComponentId: id}
}

func apiRefsFromUUIDs(m model.Model, ids []openapi_types.UUID) []mdlapi.ApiRef {
	out := make([]mdlapi.ApiRef, 0, len(ids))
	for _, x := range ids {
		id := uuid.UUID(x)
		if id == uuid.Nil {
			continue
		}
		ref := refAPI(m, id)
		if ref == nil {
			continue
		}
		out = append(out, *ref)
	}
	return out
}

func oapiVersionToModel(v *Version) common.Version {
	if v == nil {
		return common.Version{}
	}
	out := common.Version{Version: v.Version}
	if v.AvailableFrom != nil {
		t := *v.AvailableFrom
		out.AvailableFrom = &t
	}
	if v.DeprecatedFrom != nil {
		t := *v.DeprecatedFrom
		out.DeprecatedFrom = &t
	}
	if v.TerminatedFrom != nil {
		t := *v.TerminatedFrom
		out.TerminatedFrom = &t
	}
	return out
}

func parseAPIType(v interface{}) mdlapi.ApiType {
	s, _ := v.(string)
	r, _ := mdlapi.ParseApiType(strings.TrimSpace(s))
	return r
}

func mergeOapiAnnotations(dst annotations.Annotations, src *[]Annotation) {
	if src == nil {
		return
	}
	for _, a := range *src {
		dst.Add(a.Key, a.Value)
	}
}
