package model

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlorder "go.emeland.io/modelsrv/pkg/model/order"
)

type boundValueTupleKey struct {
	orderItemID uuid.UUID
	parameterID uuid.UUID
}

func boundValueTupleKeyFrom(bv mdlorder.BoundValue) boundValueTupleKey {
	paramID := uuid.Nil
	if ref := bv.GetParameterRef(); ref != nil {
		paramID = ref.EffectiveParameterID()
	}
	return boundValueTupleKey{
		orderItemID: bv.GetOrderItemId(),
		parameterID: paramID,
	}
}

func validateOrder(o mdlorder.Order, m *modelData) error {
	if o == nil {
		return fmt.Errorf("order is nil")
	}
	if o.GetOrderId() == uuid.Nil {
		return common.ErrUUIDNotSet
	}
	orgID := o.GetOrgUnitId()
	if orgID == uuid.Nil {
		return fmt.Errorf("orgUnitRef is required")
	}
	if m.GetOrgUnitById(orgID) == nil {
		return common.ErrOrgUnitNotFound
	}
	for _, item := range o.GetItems() {
		iid := item.EffectiveOrderItemID()
		if iid == uuid.Nil {
			return fmt.Errorf("order items entry has nil orderItemId")
		}
		if m.GetOrderItemById(iid) == nil {
			return common.ErrOrderItemNotFound
		}
	}
	return nil
}

func validateOrderItem(oi mdlorder.OrderItem, m *modelData) error {
	if oi == nil {
		return fmt.Errorf("order item is nil")
	}
	if oi.GetOrderItemId() == uuid.Nil {
		return common.ErrUUIDNotSet
	}
	orderID := oi.GetOrderId()
	if orderID == uuid.Nil {
		return fmt.Errorf("orderRef is required")
	}
	if m.GetOrderById(orderID) == nil {
		return common.ErrOrderNotFound
	}
	capRef := oi.GetCapabilityRef()
	if capRef == nil || capRef.EffectiveCapabilityID() == uuid.Nil {
		return fmt.Errorf("capabilityRef is required")
	}
	capID := capRef.EffectiveCapabilityID()
	if m.GetCapabilityById(capID) == nil {
		return common.ErrCapabilityNotFound
	}
	var itemVersionID uuid.UUID
	if cvRef := oi.GetCapabilityVersionRef(); cvRef != nil {
		cvID := cvRef.EffectiveCapabilityVersionID()
		if cvID == uuid.Nil {
			return fmt.Errorf("capabilityVersionRef has nil capabilityVersionId")
		}
		cv := m.GetCapabilityVersionById(cvID)
		if cv == nil {
			return common.ErrCapabilityVersionNotFound
		}
		if cv.GetCapabilityId() != capID {
			return fmt.Errorf("capability version %s does not belong to capability %s", cvID, capID)
		}
		itemVersionID = cvID
	}
	if vRef := oi.GetVariantRef(); vRef != nil {
		vid := vRef.EffectiveVariantID()
		if vid == uuid.Nil {
			return fmt.Errorf("variantRef has nil variantId")
		}
		variant := m.GetVariantById(vid)
		if variant == nil {
			return common.ErrVariantNotFound
		}
		vcv := m.GetCapabilityVersionById(variant.GetCapabilityVersionId())
		if vcv == nil || vcv.GetCapabilityId() != capID {
			return fmt.Errorf("variant %s does not belong to capability %s", vid, capID)
		}
		if itemVersionID != uuid.Nil && variant.GetCapabilityVersionId() != itemVersionID {
			return fmt.Errorf("variant %s does not belong to capability version %s", vid, itemVersionID)
		}
	}
	for _, bvRef := range oi.GetBoundValues() {
		bvid := bvRef.EffectiveBoundValueID()
		if bvid == uuid.Nil {
			return fmt.Errorf("boundValues entry has nil boundValueId")
		}
		if m.GetBoundValueById(bvid) == nil {
			return common.ErrBoundValueNotFound
		}
	}
	return nil
}

func validateBoundValue(bv mdlorder.BoundValue, m *modelData) error {
	if bv == nil {
		return fmt.Errorf("bound value is nil")
	}
	if bv.GetBoundValueId() == uuid.Nil {
		return common.ErrUUIDNotSet
	}
	oiID := bv.GetOrderItemId()
	if oiID == uuid.Nil {
		return fmt.Errorf("orderItemRef is required")
	}
	oi := m.GetOrderItemById(oiID)
	if oi == nil {
		return common.ErrOrderItemNotFound
	}
	paramRef := bv.GetParameterRef()
	if paramRef == nil || paramRef.EffectiveParameterID() == uuid.Nil {
		return fmt.Errorf("parameterRef is required")
	}
	paramID := paramRef.EffectiveParameterID()
	if m.GetParameterById(paramID) == nil {
		return common.ErrParameterNotFound
	}
	vvRef := bv.GetValidValueRef()
	if vvRef == nil || vvRef.EffectiveValidValueID() == uuid.Nil {
		return fmt.Errorf("validValueRef is required")
	}
	vvID := vvRef.EffectiveValidValueID()
	vv := m.GetValidValueById(vvID)
	if vv == nil {
		return common.ErrValidValueNotFound
	}
	if vv.GetParameterId() != paramID {
		return fmt.Errorf("valid value %s does not belong to parameter %s", vvID, paramID)
	}
	if vRef := oi.GetVariantRef(); vRef != nil {
		vid := vRef.EffectiveVariantID()
		if vid != uuid.Nil {
			variant := m.GetVariantById(vid)
			if variant == nil {
				return common.ErrVariantNotFound
			}
			found := false
			for _, pvs := range variant.GetProvides() {
				if pvs.ParameterId != paramID {
					continue
				}
				for _, id := range pvs.ValidValueIds {
					if id == vvID {
						found = true
						break
					}
				}
			}
			if !found {
				return fmt.Errorf("bound value %s is not provided by variant %s for parameter %s", vvID, vid, paramID)
			}
		}
	}
	return nil
}

// AddOrder implements [Model].
func (m *modelData) AddOrder(o mdlorder.Order) error {
	if err := validateOrder(o, m); err != nil {
		return err
	}
	return addEventEnabled(m, o, mdlorder.Order.GetOrderId,
		func(x mdlorder.Order, s events.EventSink) { x.Register(s) },
		m.ordersByUUID, events.OrderResource)
}

// DeleteOrderById implements [Model].
func (m *modelData) DeleteOrderById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.ordersByUUID, events.OrderResource, common.ErrOrderNotFound)
}

// GetOrderById implements [Model].
func (m *modelData) GetOrderById(id uuid.UUID) mdlorder.Order {
	return getEventEnabled(m, id, m.ordersByUUID)
}

// GetOrders implements [Model].
func (m *modelData) GetOrders() ([]mdlorder.Order, error) {
	return getAllEventEnabled(m, m.ordersByUUID)
}

// AddOrderItem implements [Model].
func (m *modelData) AddOrderItem(oi mdlorder.OrderItem) error {
	if err := validateOrderItem(oi, m); err != nil {
		return err
	}
	return addEventEnabled(m, oi, mdlorder.OrderItem.GetOrderItemId,
		func(x mdlorder.OrderItem, s events.EventSink) { x.Register(s) },
		m.orderItemsByUUID, events.OrderItemResource)
}

// DeleteOrderItemById implements [Model].
func (m *modelData) DeleteOrderItemById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.orderItemsByUUID, events.OrderItemResource, common.ErrOrderItemNotFound)
}

// GetOrderItemById implements [Model].
func (m *modelData) GetOrderItemById(id uuid.UUID) mdlorder.OrderItem {
	return getEventEnabled(m, id, m.orderItemsByUUID)
}

// GetOrderItems implements [Model].
func (m *modelData) GetOrderItems() ([]mdlorder.OrderItem, error) {
	return getAllEventEnabled(m, m.orderItemsByUUID)
}

// AddBoundValue implements [Model].
func (m *modelData) AddBoundValue(bv mdlorder.BoundValue) error {
	if err := validateBoundValue(bv, m); err != nil {
		return err
	}

	op, id, err := func() (events.Operation, uuid.UUID, error) {
		m.mu.Lock()
		defer m.mu.Unlock()

		id := bv.GetBoundValueId()
		key := boundValueTupleKeyFrom(bv)

		if existingID, ok := m.boundValuesByTuple[key]; ok && existingID != id {
			return events.UnknownOperation, uuid.Nil, common.ErrBoundValueConflict
		}

		op := events.CreateOperation
		if prev, exists := m.boundValuesByUUID[id]; exists {
			op = events.UpdateOperation
			prevKey := boundValueTupleKeyFrom(prev)
			if prevKey != key {
				if otherID, occupied := m.boundValuesByTuple[key]; occupied && otherID != id {
					return events.UnknownOperation, uuid.Nil, common.ErrBoundValueConflict
				}
				delete(m.boundValuesByTuple, prevKey)
			}
		}

		bv.Register(m.sink)
		m.boundValuesByUUID[id] = bv
		m.boundValuesByTuple[key] = id
		return op, id, nil
	}()
	if err != nil {
		return err
	}

	if err := m.sink.Receive(events.BoundValueResource, op, id, bv); err != nil {
		fmt.Println("Error receiving ", events.BoundValueResource, "| ", op, " event: ", err)
	}
	return nil
}

// DeleteBoundValueById implements [Model].
func (m *modelData) DeleteBoundValueById(id uuid.UUID) error {
	err := func() error {
		m.mu.Lock()
		defer m.mu.Unlock()

		prev, exists := m.boundValuesByUUID[id]
		if !exists {
			return common.ErrBoundValueNotFound
		}
		delete(m.boundValuesByUUID, id)
		delete(m.boundValuesByTuple, boundValueTupleKeyFrom(prev))
		return nil
	}()
	if err != nil {
		return err
	}

	if err := m.sink.Receive(events.BoundValueResource, events.DeleteOperation, id); err != nil {
		fmt.Println("Error receiving ", events.BoundValueResource, "| ", events.DeleteOperation, " event: ", err)
	}
	return nil
}

// GetBoundValueById implements [Model].
func (m *modelData) GetBoundValueById(id uuid.UUID) mdlorder.BoundValue {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bv, exists := m.boundValuesByUUID[id]
	if !exists {
		return nil
	}
	return bv
}

// GetBoundValues implements [Model].
func (m *modelData) GetBoundValues() ([]mdlorder.BoundValue, error) {
	return getAllEventEnabled(m, m.boundValuesByUUID)
}
