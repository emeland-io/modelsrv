package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/annotations"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/component"
	"go.emeland.io/modelsrv/pkg/model/system"
)

func phase1Resource(rt events.ResourceType) bool {
	switch rt {
	case events.SystemResource, events.SystemInstanceResource, events.APIResource,
		events.APIInstanceResource, events.ComponentResource, events.ComponentInstanceResource:
		return true
	default:
		return false
	}
}

// Register registers this server's API base URL as a callback for upstream event pushes.
func (c *ModelSrvClient) Register(callbackURL string) error {
	resp, err := c.oapi_client.PostEventsRegisterWithResponse(context.TODO(), oapi.PostEventsRegisterJSONRequestBody{
		CallbackUrl: callbackURL,
	})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("expected HTTP 201 but received %d", resp.StatusCode())
	}
	return nil
}

// PostEvent sends a domain event to POST /events/push (used for server-to-server replication).
func (c *ModelSrvClient) PostEvent(ctx context.Context, ev *events.Event) error {
	if ev == nil {
		return fmt.Errorf("nil event")
	}
	if !phase1Resource(ev.ResourceType) {
		return fmt.Errorf("event replication supports phase-1 resource types only, got %s", ev.ResourceType)
	}
	body, err := buildPushEvent(ev)
	if err != nil {
		return err
	}
	resp, err := c.oapi_client.PostEventsPushWithResponse(ctx, body)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode())
	}
	return nil
}

func buildPushEvent(ev *events.Event) (oapi.Event, error) {
	kind := ev.ResourceType.WireKind()
	op := ev.Operation.WireOperation()

	if ev.Operation == events.DeleteOperation {
		rid := openapi_types.UUID(ev.ResourceId)
		return oapi.Event{
			Kind:       kind,
			Operation:  op,
			ResourceId: &rid,
		}, nil
	}

	var res oapi.Event_Resource
	var err error

	switch ev.ResourceType {
	case events.SystemResource:
		s := minimalSystem(ev)
		err = res.FromSystem(s)
	case events.SystemInstanceResource:
		s := minimalSystemInstance(ev)
		err = res.FromSystemInstance(s)
	case events.APIResource:
		a := minimalAPI(ev)
		err = res.FromAPI(a)
	case events.APIInstanceResource:
		a := minimalAPIInstance(ev)
		err = res.FromApiInstance(a)
	case events.ComponentResource:
		c := minimalComponent(ev)
		err = res.FromComponent(c)
	case events.ComponentInstanceResource:
		c := minimalComponentInstance(ev)
		err = res.FromComponentInstance(c)
	default:
		return oapi.Event{}, fmt.Errorf("unsupported resource type %s", ev.ResourceType)
	}
	if err != nil {
		return oapi.Event{}, err
	}
	return oapi.Event{
		Kind:      kind,
		Operation: op,
		Resource:  &res,
	}, nil
}

func firstObject(ev *events.Event) (any, bool) {
	if len(ev.Objects) == 0 {
		return nil, false
	}
	return ev.Objects[0], true
}

func minimalSystem(ev *events.Event) oapi.System {
	if o, ok := firstObject(ev); ok {
		if s, ok := o.(system.System); ok {
			sid := openapi_types.UUID(s.GetSystemId())
			dn := s.GetDisplayName()
			desc := s.GetDescription()
			ab := s.GetAbstract()
			out := oapi.System{
				SystemId:    &sid,
				DisplayName: dn,
				Description: &desc,
				Abstract:    ab,
				Annotations: cloneAnn(s.GetAnnotations()),
			}
			if p, err := s.GetParent(); err == nil && p != nil {
				pid := openapi_types.UUID(p.GetSystemId())
				out.Parent = &pid
			}
			return out
		}
	}
	dn := "-"
	return oapi.System{DisplayName: dn, Abstract: false}
}

func minimalSystemInstance(ev *events.Event) oapi.SystemInstance {
	if o, ok := firstObject(ev); ok {
		if si, ok := o.(system.SystemInstance); ok {
			iid := openapi_types.UUID(si.GetInstanceId())
			dn := si.GetDisplayName()
			out := oapi.SystemInstance{
				SystemInstanceId: iid,
				DisplayName:      dn,
				Annotations:      cloneAnn(si.GetAnnotations()),
			}
			if sr := si.GetSystemRef(); sr != nil {
				out.System = openapi_types.UUID(sr.SystemId)
			}
			if cr := si.GetContextRef(); cr != nil {
				cid := openapi_types.UUID(cr.ContextId)
				out.Context = &cid
			}
			return out
		}
	}
	return oapi.SystemInstance{
		SystemInstanceId: openapi_types.UUID(uuid.Nil),
		DisplayName:      " ",
		System:           openapi_types.UUID(uuid.Nil),
	}
}

func minimalAPI(ev *events.Event) oapi.API {
	if o, ok := firstObject(ev); ok {
		if a, ok := o.(mdlapi.API); ok {
			id := openapi_types.UUID(a.GetApiId())
			dn := a.GetDisplayName()
			desc := a.GetDescription()
			out := oapi.API{
				ApiId:       &id,
				DisplayName: dn,
				Description: &desc,
				Type:        "Unknown",
				Annotations: cloneAnn(a.GetAnnotations()),
			}
			if sys := a.GetSystem(); sys != nil {
				sid := openapi_types.UUID(sys.SystemId)
				out.System = &sid
			}
			return out
		}
	}
	dn := "-"
	return oapi.API{DisplayName: dn, Type: "Unknown"}
}

func minimalAPIInstance(ev *events.Event) oapi.ApiInstance {
	if o, ok := firstObject(ev); ok {
		if a, ok := o.(mdlapi.ApiInstance); ok {
			id := openapi_types.UUID(a.GetInstanceId())
			dn := a.GetDisplayName()
			out := oapi.ApiInstance{
				ApiInstanceId: id,
				DisplayName:   dn,
				Annotations:   cloneAnn(a.GetAnnotations()),
			}
			if ar := a.GetApiRef(); ar != nil {
				aid := openapi_types.UUID(ar.ApiID)
				out.Api = &aid
			}
			if si := a.GetSystemInstance(); si != nil {
				iid := openapi_types.UUID(si.InstanceId)
				out.SystemInstance = &iid
			}
			return out
		}
	}
	return oapi.ApiInstance{
		ApiInstanceId: openapi_types.UUID(uuid.Nil),
		DisplayName:   " ",
	}
}

func minimalComponent(ev *events.Event) oapi.Component {
	if o, ok := firstObject(ev); ok {
		if c, ok := o.(component.Component); ok {
			id := openapi_types.UUID(c.GetComponentId())
			dn := c.GetDisplayName()
			desc := c.GetDescription()
			out := oapi.Component{
				ComponentId: &id,
				DisplayName: dn,
				Description: &desc,
				Annotations: cloneAnn(c.GetAnnotations()),
			}
			if sys := c.GetSystem(); sys != nil {
				out.System = openapi_types.UUID(sys.SystemId)
			}
			return out
		}
	}
	dn := "-"
	return oapi.Component{DisplayName: dn, System: openapi_types.UUID(uuid.Nil)}
}

func minimalComponentInstance(ev *events.Event) oapi.ComponentInstance {
	if o, ok := firstObject(ev); ok {
		if c, ok := o.(component.ComponentInstance); ok {
			id := openapi_types.UUID(c.GetInstanceId())
			dn := c.GetDisplayName()
			var comp, sys openapi_types.UUID
			if cr := c.GetComponentRef(); cr != nil {
				comp = openapi_types.UUID(cr.ComponentId)
			}
			if si := c.GetSystemInstance(); si != nil {
				sys = openapi_types.UUID(si.InstanceId)
			}
			return oapi.ComponentInstance{
				ComponentInstanceId: id,
				DisplayName:         dn,
				Component:           comp,
				SystemInstance:      sys,
				Consumes:            []openapi_types.UUID{},
				Provides:            []openapi_types.UUID{},
				Annotations:         cloneAnn(c.GetAnnotations()),
			}
		}
	}
	return oapi.ComponentInstance{
		ComponentInstanceId: openapi_types.UUID(uuid.Nil),
		DisplayName:         " ",
		Component:           openapi_types.UUID(uuid.Nil),
		SystemInstance:      openapi_types.UUID(uuid.Nil),
		Consumes:            []openapi_types.UUID{},
		Provides:            []openapi_types.UUID{},
	}
}

func cloneAnn(a annotations.Annotations) *[]oapi.Annotation {
	if a == nil {
		return nil
	}
	out := make([]oapi.Annotation, 0)
	for k := range a.GetKeys() {
		out = append(out, oapi.Annotation{Key: k, Value: a.GetValue(k)})
	}
	return &out
}
