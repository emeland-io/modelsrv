package oapi

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

// ApplyPushEvent applies a replicated wire event to the local model. It uses the same Model add/delete
// paths as normal mutations so the recording sink records once and forwards to downstream subscribers.
func ApplyPushEvent(m model.Model, ev *Event) error {
	if m == nil {
		return fmt.Errorf("nil model")
	}
	if ev == nil {
		return fmt.Errorf("nil event")
	}

	kind := wireString(ev.Kind)
	op := wireString(ev.Operation)
	rt := events.ParseWireKind(kind)
	if rt == events.UnknownResourceType {
		return fmt.Errorf("unknown event kind %q", kind)
	}
	wop := events.ParseWireOperation(op)
	if wop == events.UnknownOperation {
		return fmt.Errorf("unknown event operation %q", op)
	}

	switch wop {
	case events.DeleteOperation:
		return applyDelete(m, rt, ev.ResourceId)
	case events.CreateOperation, events.UpdateOperation:
		if ev.Resource == nil {
			return fmt.Errorf("missing resource payload for %s %s", kind, op)
		}
		return applyUpsert(m, rt, ev.Resource)
	default:
		return fmt.Errorf("unsupported operation %q", op)
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

func applyDelete(m model.Model, rt events.ResourceType, rid *openapi_types.UUID) error {
	if rid == nil {
		return fmt.Errorf("delete event missing resourceId")
	}
	id := uuid.UUID(*rid)
	var err error
	switch rt {
	case events.SystemResource:
		err = m.DeleteSystemById(id)
	case events.SystemInstanceResource:
		err = m.DeleteSystemInstanceById(id)
	case events.APIResource:
		err = m.DeleteApiById(id)
	case events.APIInstanceResource:
		err = m.DeleteApiInstanceById(id)
	case events.ComponentResource:
		err = m.DeleteComponentById(id)
	case events.ComponentInstanceResource:
		err = m.DeleteComponentInstanceById(id)
	default:
		return fmt.Errorf("unsupported resource type for delete: %s", rt)
	}
	if err != nil && !isNotFound(err) {
		return err
	}
	return nil
}

func isNotFound(err error) bool {
	return errors.Is(err, model.ErrSystemNotFound) ||
		errors.Is(err, model.ErrSystemInstanceNotFound) ||
		errors.Is(err, model.ErrApiNotFound) ||
		errors.Is(err, model.ErrApiInstanceNotFound) ||
		errors.Is(err, model.ErrComponentNotFound) ||
		errors.Is(err, model.ErrComponentInstanceNotFound)
}

func applyUpsert(m model.Model, rt events.ResourceType, res *Event_Resource) error {
	switch rt {
	case events.SystemResource:
		os, err := res.AsSystem()
		if err != nil {
			return err
		}
		return applySystem(m, os)
	case events.SystemInstanceResource:
		os, err := res.AsSystemInstance()
		if err != nil {
			return err
		}
		return applySystemInstance(m, os)
	case events.APIResource:
		oa, err := res.AsAPI()
		if err != nil {
			return err
		}
		return applyAPI(m, oa)
	case events.APIInstanceResource:
		oa, err := res.AsApiInstance()
		if err != nil {
			return err
		}
		return applyAPIInstance(m, oa)
	case events.ComponentResource:
		oc, err := res.AsComponent()
		if err != nil {
			return err
		}
		return applyComponent(m, oc)
	case events.ComponentInstanceResource:
		oc, err := res.AsComponentInstance()
		if err != nil {
			return err
		}
		return applyComponentInstance(m, oc)
	default:
		return fmt.Errorf("unsupported resource type for upsert: %s", rt)
	}
}

func applySystem(m model.Model, os System) error {
	if os.SystemId == nil {
		return fmt.Errorf("system event missing systemId")
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
	return m.AddSystem(sys)
}

func applySystemInstance(m model.Model, os SystemInstance) error {
	id := uuid.UUID(os.SystemInstanceId)
	si := model.NewSystemInstance(m, id)
	si.SetDisplayName(os.DisplayName)
	si.SetSystemRef(refSystem(m, uuid.UUID(os.System)))
	if os.Context != nil {
		si.SetContextRef(&model.ContextRef{ContextId: uuid.UUID(*os.Context)})
	}
	mergeOapiAnnotations(si.GetAnnotations(), os.Annotations)
	return m.AddSystemInstance(si)
}

func applyAPI(m model.Model, oa API) error {
	if oa.ApiId == nil {
		return fmt.Errorf("API event missing apiId")
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
	return m.AddApi(api)
}

func applyAPIInstance(m model.Model, oa ApiInstance) error {
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
	return m.AddApiInstance(ai)
}

func applyComponent(m model.Model, oc Component) error {
	if oc.ComponentId == nil {
		return fmt.Errorf("component event missing componentId")
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
	return m.AddComponent(c)
}

func applyComponentInstance(m model.Model, oc ComponentInstance) error {
	id := uuid.UUID(oc.ComponentInstanceId)
	ci := model.NewComponentInstance(m, id)
	ci.SetDisplayName(oc.DisplayName)
	ci.SetComponentRef(refComponent(m, uuid.UUID(oc.Component)))
	ci.SetSystemInstance(refSystemInstance(m, uuid.UUID(oc.SystemInstance)))
	mergeOapiAnnotations(ci.GetAnnotations(), oc.Annotations)
	return m.AddComponentInstance(ci)
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
	switch s {
	case "OpenAPI":
		return model.OpenAPI
	case "GraphQL":
		return model.GraphQL
	case "GRPC", "gRPC":
		return model.GRPC
	case "Other":
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
