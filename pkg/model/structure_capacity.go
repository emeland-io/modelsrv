package model

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	mdlcap "go.emeland.io/modelsrv/pkg/model/capacity"
	"go.emeland.io/modelsrv/pkg/model/common"
)

type capacityTupleKey struct {
	contextID              uuid.UUID
	capacityResourceTypeID uuid.UUID
	category               mdlcap.Category
}

func capacityTupleKeyFrom(c mdlcap.Capacity) capacityTupleKey {
	return capacityTupleKey{
		contextID:              c.GetContextId(),
		capacityResourceTypeID: c.GetCapacityResourceTypeId(),
		category:               c.GetCategory(),
	}
}

func validateCapacity(c mdlcap.Capacity, m *modelData) error {
	if c == nil {
		return fmt.Errorf("capacity is nil")
	}
	if c.GetCapacityId() == uuid.Nil {
		return common.ErrUUIDNotSet
	}
	if _, err := mdlcap.ParseCategory(string(c.GetCategory())); err != nil {
		return err
	}
	if _, err := mdlcap.ParseAmount(string(c.GetAmount())); err != nil {
		return err
	}
	typeID := c.GetCapacityResourceTypeId()
	if typeID == uuid.Nil {
		return fmt.Errorf("resourceTypeRef is required")
	}
	if m.GetCapacityResourceTypeById(typeID) == nil {
		return common.ErrCapacityResourceTypeNotFound
	}
	contextID := c.GetContextId()
	if contextID == uuid.Nil {
		return fmt.Errorf("contextRef is required")
	}
	if m.GetContextById(contextID) == nil {
		return common.ErrContextNotFound
	}
	return nil
}

// AddCapacity implements [Model].
func (m *modelData) AddCapacity(c mdlcap.Capacity) error {
	if err := validateCapacity(c, m); err != nil {
		return err
	}

	op, id, err := func() (events.Operation, uuid.UUID, error) {
		m.mu.Lock()
		defer m.mu.Unlock()

		id := c.GetCapacityId()
		key := capacityTupleKeyFrom(c)

		if existingID, ok := m.capacitiesByTuple[key]; ok && existingID != id {
			return events.UnknownOperation, uuid.Nil, common.ErrCapacityTupleConflict
		}

		op := events.CreateOperation
		if prev, exists := m.capacitiesByUUID[id]; exists {
			op = events.UpdateOperation
			prevKey := capacityTupleKeyFrom(prev)
			if prevKey != key {
				if otherID, occupied := m.capacitiesByTuple[key]; occupied && otherID != id {
					return events.UnknownOperation, uuid.Nil, common.ErrCapacityTupleConflict
				}
				delete(m.capacitiesByTuple, prevKey)
			}
		}

		c.Register(m.sink)
		m.capacitiesByUUID[id] = c
		m.capacitiesByTuple[key] = id
		return op, id, nil
	}()
	if err != nil {
		return err
	}

	if err := m.sink.Receive(events.CapacityResource, op, id, c); err != nil {
		fmt.Println("Error receiving ", events.CapacityResource, "| ", op, " event: ", err)
	}
	return nil
}

// DeleteCapacityById implements [Model].
func (m *modelData) DeleteCapacityById(id uuid.UUID) error {
	err := func() error {
		m.mu.Lock()
		defer m.mu.Unlock()

		prev, exists := m.capacitiesByUUID[id]
		if !exists {
			return common.ErrCapacityNotFound
		}
		delete(m.capacitiesByUUID, id)
		delete(m.capacitiesByTuple, capacityTupleKeyFrom(prev))
		return nil
	}()
	if err != nil {
		return err
	}

	if err := m.sink.Receive(events.CapacityResource, events.DeleteOperation, id); err != nil {
		fmt.Println("Error receiving ", events.CapacityResource, "| ", events.DeleteOperation, " event: ", err)
	}
	return nil
}

// GetCapacityById implements [Model].
func (m *modelData) GetCapacityById(id uuid.UUID) mdlcap.Capacity {
	m.mu.RLock()
	defer m.mu.RUnlock()

	c, exists := m.capacitiesByUUID[id]
	if !exists {
		return nil
	}
	return c
}

// GetCapacities implements [Model].
func (m *modelData) GetCapacities() ([]mdlcap.Capacity, error) {
	return getAllEventEnabled(m, m.capacitiesByUUID)
}

// AddCapacityResourceType implements [Model].
func (m *modelData) AddCapacityResourceType(capacityResourceType mdlcap.CapacityResourceType) error {
	return addEventEnabled(m, capacityResourceType, mdlcap.CapacityResourceType.GetCapacityResourceTypeId,
		func(x mdlcap.CapacityResourceType, s events.EventSink) { x.Register(s) },
		m.capacityResourceTypesByUUID, events.CapacityResourceTypeResource)
}

// DeleteCapacityResourceTypeById implements [Model].
func (m *modelData) DeleteCapacityResourceTypeById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.capacityResourceTypesByUUID, events.CapacityResourceTypeResource, common.ErrCapacityResourceTypeNotFound)
}

// GetCapacityResourceTypeById implements [Model].
func (m *modelData) GetCapacityResourceTypeById(id uuid.UUID) mdlcap.CapacityResourceType {
	return getEventEnabled(m, id, m.capacityResourceTypesByUUID)
}

// GetCapacityResourceTypes implements [Model].
func (m *modelData) GetCapacityResourceTypes() ([]mdlcap.CapacityResourceType, error) {
	return getAllEventEnabled(m, m.capacityResourceTypesByUUID)
}
