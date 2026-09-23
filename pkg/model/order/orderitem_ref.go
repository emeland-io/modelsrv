package order

import "github.com/google/uuid"

// OrderItemRef references an [OrderItem] by resolved object and/or id.
type OrderItemRef struct {
	OrderItem   OrderItem
	OrderItemId uuid.UUID `json:"orderItemId"`
}

// ResolvedOrderItem returns the embedded [OrderItem] when present, or nil.
func (r *OrderItemRef) ResolvedOrderItem() OrderItem {
	if r == nil {
		return nil
	}
	return r.OrderItem
}

// EffectiveOrderItemID returns the order item id from the embedded object or from
// [OrderItemRef.OrderItemId].
func (r *OrderItemRef) EffectiveOrderItemID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.OrderItem != nil {
		return r.OrderItem.GetOrderItemId()
	}
	return r.OrderItemId
}

// BoundValueRef references a [BoundValue] by resolved object and/or id.
type BoundValueRef struct {
	BoundValue   BoundValue
	BoundValueId uuid.UUID `json:"boundValueId"`
}

// ResolvedBoundValue returns the embedded [BoundValue] when present, or nil.
func (r *BoundValueRef) ResolvedBoundValue() BoundValue {
	if r == nil {
		return nil
	}
	return r.BoundValue
}

// EffectiveBoundValueID returns the bound value id from the embedded object or from
// [BoundValueRef.BoundValueId].
func (r *BoundValueRef) EffectiveBoundValueID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.BoundValue != nil {
		return r.BoundValue.GetBoundValueId()
	}
	return r.BoundValueId
}
