package model

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlparameter "go.emeland.io/modelsrv/pkg/model/parameter"
)

type validValueTupleKey struct {
	paramID uuid.UUID
	name    string
}

func validValueTupleKeyFrom(vv mdlparameter.ValidValue) validValueTupleKey {
	return validValueTupleKey{
		paramID: vv.GetParameterId(),
		name:    vv.GetDisplayName(),
	}
}

func validateParameter(parameter mdlparameter.Parameter, m *modelData) error {
	if parameter == nil {
		return fmt.Errorf("parameter is nil")
	}
	if parameter.GetParameterId() == uuid.Nil {
		return common.ErrUUIDNotSet
	}
	for _, vid := range parameter.GetValues() {
		vv := m.GetValidValueById(vid)
		if vv == nil {
			return common.ErrValidValueNotFound
		}
		if vv.GetParameterId() != parameter.GetParameterId() {
			return fmt.Errorf("valid value %s does not belong to parameter %s", vid, parameter.GetParameterId())
		}
	}
	return nil
}

func validateValidValue(vv mdlparameter.ValidValue, m *modelData) error {
	if vv == nil {
		return fmt.Errorf("valid value is nil")
	}
	if vv.GetValidValueId() == uuid.Nil {
		return common.ErrUUIDNotSet
	}
	paramID := vv.GetParameterId()
	if paramID == uuid.Nil {
		return fmt.Errorf("parameterRef is required")
	}
	if m.GetParameterById(paramID) == nil {
		return common.ErrParameterNotFound
	}
	return nil
}

// AddParameter implements [Model].
func (m *modelData) AddParameter(parameter mdlparameter.Parameter) error {
	if err := validateParameter(parameter, m); err != nil {
		return err
	}
	return addEventEnabled(m, parameter, mdlparameter.Parameter.GetParameterId,
		func(x mdlparameter.Parameter, s events.EventSink) { x.Register(s) },
		m.parametersByUUID, events.ParameterResource)
}

// DeleteParameterById implements [Model].
func (m *modelData) DeleteParameterById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.parametersByUUID, events.ParameterResource, common.ErrParameterNotFound)
}

// GetParameterById implements [Model].
func (m *modelData) GetParameterById(id uuid.UUID) mdlparameter.Parameter {
	return getEventEnabled(m, id, m.parametersByUUID)
}

// GetParameters implements [Model].
func (m *modelData) GetParameters() ([]mdlparameter.Parameter, error) {
	return getAllEventEnabled(m, m.parametersByUUID)
}

// AddValidValue implements [Model].
func (m *modelData) AddValidValue(vv mdlparameter.ValidValue) error {
	if err := validateValidValue(vv, m); err != nil {
		return err
	}

	op, id, err := func() (events.Operation, uuid.UUID, error) {
		m.mu.Lock()
		defer m.mu.Unlock()

		id := vv.GetValidValueId()
		key := validValueTupleKeyFrom(vv)

		if existingID, ok := m.validValuesByTuple[key]; ok && existingID != id {
			return events.UnknownOperation, uuid.Nil, common.ErrValidValueConflict
		}

		op := events.CreateOperation
		if prev, exists := m.validValuesByUUID[id]; exists {
			op = events.UpdateOperation
			prevKey := validValueTupleKeyFrom(prev)
			if prevKey != key {
				if otherID, occupied := m.validValuesByTuple[key]; occupied && otherID != id {
					return events.UnknownOperation, uuid.Nil, common.ErrValidValueConflict
				}
				delete(m.validValuesByTuple, prevKey)
			}
		}

		vv.Register(m.sink)
		m.validValuesByUUID[id] = vv
		m.validValuesByTuple[key] = id
		return op, id, nil
	}()
	if err != nil {
		return err
	}

	if err := m.sink.Receive(events.ValidValueResource, op, id, vv); err != nil {
		fmt.Println("Error receiving ", events.ValidValueResource, "| ", op, " event: ", err)
	}
	return nil
}

// DeleteValidValueById implements [Model].
func (m *modelData) DeleteValidValueById(id uuid.UUID) error {
	err := func() error {
		m.mu.Lock()
		defer m.mu.Unlock()

		prev, exists := m.validValuesByUUID[id]
		if !exists {
			return common.ErrValidValueNotFound
		}
		delete(m.validValuesByUUID, id)
		delete(m.validValuesByTuple, validValueTupleKeyFrom(prev))
		return nil
	}()
	if err != nil {
		return err
	}

	if err := m.sink.Receive(events.ValidValueResource, events.DeleteOperation, id); err != nil {
		fmt.Println("Error receiving ", events.ValidValueResource, "| ", events.DeleteOperation, " event: ", err)
	}
	return nil
}

// GetValidValueById implements [Model].
func (m *modelData) GetValidValueById(id uuid.UUID) mdlparameter.ValidValue {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vv, exists := m.validValuesByUUID[id]
	if !exists {
		return nil
	}
	return vv
}

// GetValidValues implements [Model].
func (m *modelData) GetValidValues() ([]mdlparameter.ValidValue, error) {
	return getAllEventEnabled(m, m.validValuesByUUID)
}
