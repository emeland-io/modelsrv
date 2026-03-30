package model

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/component"
	mdlevent "go.emeland.io/modelsrv/pkg/model/event"
	"go.emeland.io/modelsrv/pkg/model/system"
)

var _ mdlevent.EventApplier = (*modelData)(nil)

// Apply implements [EventApplier].
func (m *modelData) Apply(ev events.Event) error {
	switch ev.Operation {
	case events.DeleteOperation:
		return m.applyReplicationDelete(ev.ResourceType, ev.ResourceId)
	case events.CreateOperation, events.UpdateOperation:
		if len(ev.Objects) == 0 {
			return fmt.Errorf("missing resource object for %s %s", ev.ResourceType, ev.Operation)
		}
		return m.applyReplicationUpsert(ev.ResourceType, ev.Objects[0])
	default:
		return fmt.Errorf("unsupported operation %v", ev.Operation)
	}
}

func (m *modelData) applyReplicationDelete(rt events.ResourceType, id uuid.UUID) error {
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
	if err != nil && !replicationNotFound(err) {
		return err
	}
	return nil
}

func replicationNotFound(err error) bool {
	return errors.Is(err, common.ErrSystemNotFound) ||
		errors.Is(err, common.ErrSystemInstanceNotFound) ||
		errors.Is(err, common.ErrApiNotFound) ||
		errors.Is(err, common.ErrApiInstanceNotFound) ||
		errors.Is(err, common.ErrComponentNotFound) ||
		errors.Is(err, common.ErrComponentInstanceNotFound)
}

func (m *modelData) applyReplicationUpsert(rt events.ResourceType, obj any) error {
	switch rt {
	case events.SystemResource:
		s, ok := obj.(system.System)
		if !ok {
			return fmt.Errorf("replication object for System is %T, want System", obj)
		}
		return m.AddSystem(s)
	case events.SystemInstanceResource:
		s, ok := obj.(system.SystemInstance)
		if !ok {
			return fmt.Errorf("replication object for SystemInstance is %T, want SystemInstance", obj)
		}
		return m.AddSystemInstance(s)
	case events.APIResource:
		a, ok := obj.(mdlapi.API)
		if !ok {
			return fmt.Errorf("replication object for API is %T, want API", obj)
		}
		return m.AddApi(a)
	case events.APIInstanceResource:
		a, ok := obj.(mdlapi.ApiInstance)
		if !ok {
			return fmt.Errorf("replication object for ApiInstance is %T, want ApiInstance", obj)
		}
		return m.AddApiInstance(a)
	case events.ComponentResource:
		c, ok := obj.(component.Component)
		if !ok {
			return fmt.Errorf("replication object for Component is %T, want Component", obj)
		}
		return m.AddComponent(c)
	case events.ComponentInstanceResource:
		c, ok := obj.(component.ComponentInstance)
		if !ok {
			return fmt.Errorf("replication object for ComponentInstance is %T, want ComponentInstance", obj)
		}
		return m.AddComponentInstance(c)
	default:
		return fmt.Errorf("unsupported resource type for upsert: %s", rt)
	}
}
