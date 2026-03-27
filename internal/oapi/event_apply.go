package oapi

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

// ReplicationEventFromWire converts an OpenAPI push body into a domain [events.Event] for [model.EventApplier.Apply].
func ReplicationEventFromWire(m model.Model, ev *Event) (events.Event, error) {
	if m == nil {
		return events.Event{}, fmt.Errorf("nil model")
	}
	if ev == nil {
		return events.Event{}, fmt.Errorf("nil event")
	}

	kind := wireString(ev.Kind)
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
		id, obj, err := replicationObjectFromWire(m, rt, ev.Resource)
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

func replicationObjectFromWire(m model.Model, rt events.ResourceType, res *Event_Resource) (uuid.UUID, any, error) {
	switch rt {
	case events.SystemResource:
		os, err := res.AsSystem()
		if err != nil {
			return uuid.Nil, nil, err
		}
		s, err := systemFromWire(m, os)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return s.GetSystemId(), s, nil
	case events.SystemInstanceResource:
		os, err := res.AsSystemInstance()
		if err != nil {
			return uuid.Nil, nil, err
		}
		s, err := systemInstanceFromWire(m, os)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return s.GetInstanceId(), s, nil
	case events.APIResource:
		oa, err := res.AsAPI()
		if err != nil {
			return uuid.Nil, nil, err
		}
		a, err := apiFromWire(m, oa)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return a.GetApiId(), a, nil
	case events.APIInstanceResource:
		oa, err := res.AsApiInstance()
		if err != nil {
			return uuid.Nil, nil, err
		}
		a, err := apiInstanceFromWire(m, oa)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return a.GetInstanceId(), a, nil
	case events.ComponentResource:
		oc, err := res.AsComponent()
		if err != nil {
			return uuid.Nil, nil, err
		}
		c, err := componentFromWire(m, oc)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return c.GetComponentId(), c, nil
	case events.ComponentInstanceResource:
		oc, err := res.AsComponentInstance()
		if err != nil {
			return uuid.Nil, nil, err
		}
		c, err := componentInstanceFromWire(m, oc)
		if err != nil {
			return uuid.Nil, nil, err
		}
		return c.GetInstanceId(), c, nil
	default:
		return uuid.Nil, nil, fmt.Errorf("unsupported resource type for upsert: %s", rt)
	}
}

func systemFromWire(m model.Model, os System) (model.System, error) {
	if os.SystemId == nil {
		return nil, fmt.Errorf("system event missing systemId")
	}
	id := uuid.UUID(*os.SystemId)
	sys := model.NewSystem(m.GetSink(), id)
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

func systemInstanceFromWire(m model.Model, os SystemInstance) (model.SystemInstance, error) {
	id := uuid.UUID(os.SystemInstanceId)
	si := model.NewSystemInstance(m, id)
	si.SetDisplayName(os.DisplayName)
	si.SetSystemRef(refSystem(m, uuid.UUID(os.System)))
	if os.Context != nil {
		si.SetContextRef(&model.ContextRef{ContextId: uuid.UUID(*os.Context)})
	}
	mergeOapiAnnotations(si.GetAnnotations(), os.Annotations)
	return si, nil
}

func apiFromWire(m model.Model, oa API) (model.API, error) {
	if oa.ApiId == nil {
		return nil, fmt.Errorf("API event missing apiId")
	}
	id := uuid.UUID(*oa.ApiId)
	api := model.NewAPI(m, id)
	api.SetDisplayName(oa.DisplayName)
	if oa.Description != nil {
		api.SetDescription(*oa.Description)
	}
	api.SetType(parseAPIType(oa.Type))
	if oa.Version != nil {
		api.SetVersion(oapiVersionToModel(oa.Version))
	}
	if oa.System != nil {
		api.SetSystem(refSystem(m, uuid.UUID(*oa.System)))
	}
	mergeOapiAnnotations(api.GetAnnotations(), oa.Annotations)
	return api, nil
}

func apiInstanceFromWire(m model.Model, oa ApiInstance) (model.ApiInstance, error) {
	id := uuid.UUID(oa.ApiInstanceId)
	ai := model.NewApiInstance(m.GetSink(), id)
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

func componentFromWire(m model.Model, oc Component) (model.Component, error) {
	if oc.ComponentId == nil {
		return nil, fmt.Errorf("component event missing componentId")
	}
	id := uuid.UUID(*oc.ComponentId)
	c := model.NewComponent(m, id)
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

func componentInstanceFromWire(m model.Model, oc ComponentInstance) (model.ComponentInstance, error) {
	id := uuid.UUID(oc.ComponentInstanceId)
	ci := model.NewComponentInstance(m, id)
	ci.SetDisplayName(oc.DisplayName)
	ci.SetComponentRef(refComponent(m, uuid.UUID(oc.Component)))
	ci.SetSystemInstance(refSystemInstance(m, uuid.UUID(oc.SystemInstance)))
	mergeOapiAnnotations(ci.GetAnnotations(), oc.Annotations)
	return ci, nil
}

func refSystem(m model.Model, id uuid.UUID) *model.SystemRef {
	if id == uuid.Nil {
		return nil
	}
	if s := m.GetSystemById(id); s != nil {
		return &model.SystemRef{System: s, SystemId: id}
	}
	return &model.SystemRef{SystemId: id}
}

func refAPI(m model.Model, id uuid.UUID) *model.ApiRef {
	if id == uuid.Nil {
		return nil
	}
	if a := m.GetApiById(id); a != nil {
		return &model.ApiRef{API: a, ApiID: id}
	}
	return &model.ApiRef{ApiID: id}
}

func refSystemInstance(m model.Model, id uuid.UUID) *model.SystemInstanceRef {
	if id == uuid.Nil {
		return nil
	}
	if si := m.GetSystemInstanceById(id); si != nil {
		return &model.SystemInstanceRef{SystemInstance: si, InstanceId: id}
	}
	return &model.SystemInstanceRef{InstanceId: id}
}

func refComponent(m model.Model, id uuid.UUID) *model.ComponentRef {
	if id == uuid.Nil {
		return nil
	}
	if c := m.GetComponentById(id); c != nil {
		return &model.ComponentRef{Component: c, ComponentId: id}
	}
	return &model.ComponentRef{ComponentId: id}
}

func apiRefsFromUUIDs(m model.Model, ids []openapi_types.UUID) []model.ApiRef {
	out := make([]model.ApiRef, 0, len(ids))
	for _, x := range ids {
		id := uuid.UUID(x)
		out = append(out, *refAPI(m, id))
	}
	return out
}

func oapiVersionToModel(v *Version) model.Version {
	if v == nil {
		return model.Version{}
	}
	out := model.Version{Version: v.Version}
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

func parseAPIType(v interface{}) model.ApiType {
	s, _ := v.(string)
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "openapi":
		return model.OpenAPI
	case "graphql":
		return model.GraphQL
	case "grpc":
		return model.GRPC
	case "other":
		return model.Other
	default:
		return model.Unknown
	}
}

func mergeOapiAnnotations(dst model.Annotations, src *[]Annotation) {
	if src == nil {
		return
	}
	for _, a := range *src {
		dst.Add(a.Key, a.Value)
	}
}
