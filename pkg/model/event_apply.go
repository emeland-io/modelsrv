package model

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

// EventApplier applies replicated [events.Event] records to local state using the same Add*/Delete* paths
// as normal mutations so the recording sink records once.
type EventApplier interface {
	Apply(ev events.Event) error
}

var _ EventApplier = (*modelData)(nil)

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
	return errors.Is(err, ErrSystemNotFound) ||
		errors.Is(err, ErrSystemInstanceNotFound) ||
		errors.Is(err, ErrApiNotFound) ||
		errors.Is(err, ErrApiInstanceNotFound) ||
		errors.Is(err, ErrComponentNotFound) ||
		errors.Is(err, ErrComponentInstanceNotFound)
}

func (m *modelData) applyReplicationUpsert(rt events.ResourceType, obj any) error {
	switch rt {
	case events.SystemResource:
		s, ok := obj.(System)
		if !ok {
			return fmt.Errorf("replication object for System is %T, want System", obj)
		}
		return m.AddSystem(s)
	case events.SystemInstanceResource:
		s, ok := obj.(SystemInstance)
		if !ok {
			return fmt.Errorf("replication object for SystemInstance is %T, want SystemInstance", obj)
		}
		return m.AddSystemInstance(s)
	case events.APIResource:
		a, ok := obj.(API)
		if !ok {
			return fmt.Errorf("replication object for API is %T, want API", obj)
		}
		return m.AddApi(a)
	case events.APIInstanceResource:
		a, ok := obj.(ApiInstance)
		if !ok {
			return fmt.Errorf("replication object for ApiInstance is %T, want ApiInstance", obj)
		}
		return m.AddApiInstance(a)
	case events.ComponentResource:
		c, ok := obj.(Component)
		if !ok {
			return fmt.Errorf("replication object for Component is %T, want Component", obj)
		}
		return m.AddComponent(c)
	case events.ComponentInstanceResource:
		c, ok := obj.(ComponentInstance)
		if !ok {
			return fmt.Errorf("replication object for ComponentInstance is %T, want ComponentInstance", obj)
		}
		return m.AddComponentInstance(c)
	default:
		return fmt.Errorf("unsupported resource type for upsert: %s", rt)
	}
}
