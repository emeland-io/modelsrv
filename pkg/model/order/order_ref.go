package order

import "github.com/google/uuid"

// OrderRef references an [Order] by resolved object and/or id.
type OrderRef struct {
	Order   Order
	OrderId uuid.UUID
}

// ResolvedOrder returns the embedded [Order] when present, or nil.
func (r *OrderRef) ResolvedOrder() Order {
	if r == nil {
		return nil
	}
	return r.Order
}

// EffectiveOrderID returns the order id from the embedded object or from [OrderRef.OrderId].
func (r *OrderRef) EffectiveOrderID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.Order != nil {
		return r.Order.GetOrderId()
	}
	return r.OrderId
}
