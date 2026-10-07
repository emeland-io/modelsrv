package model

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	mdlcapability "go.emeland.io/modelsrv/pkg/model/capability"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlorder "go.emeland.io/modelsrv/pkg/model/order"
	mdlparameter "go.emeland.io/modelsrv/pkg/model/parameter"
)

// Phase 3 ordering resources follow the same Add posture as observability:
// no Add-time reference validation. A Variant may reference a CapabilityVersion
// that is not yet present (e.g. delivered earlier in a replication stream).
// This keeps event apply order-tolerant; dangling references can be surfaced
// later via findings. Superset checks run only when both sides are present.

// ValidValueModel provides CRUD operations for [parameter.ValidValue] resources.
type ValidValueModel interface {
	AddValidValue(validValue mdlparameter.ValidValue) error
	DeleteValidValueById(id uuid.UUID) error
	GetValidValues() ([]mdlparameter.ValidValue, error)
	GetValidValueById(id uuid.UUID) mdlparameter.ValidValue
}

// CapabilityVersionModel provides CRUD operations for [capability.CapabilityVersion] resources.
type CapabilityVersionModel interface {
	AddCapabilityVersion(capabilityVersion mdlcapability.CapabilityVersion) error
	DeleteCapabilityVersionById(id uuid.UUID) error
	GetCapabilityVersions() ([]mdlcapability.CapabilityVersion, error)
	GetCapabilityVersionById(id uuid.UUID) mdlcapability.CapabilityVersion
}

// VariantModel provides CRUD operations for [capability.Variant] resources.
type VariantModel interface {
	AddVariant(variant mdlcapability.Variant) error
	DeleteVariantById(id uuid.UUID) error
	GetVariants() ([]mdlcapability.Variant, error)
	GetVariantById(id uuid.UUID) mdlcapability.Variant
}

// DependencyModel provides CRUD operations for [capability.Dependency] resources.
type DependencyModel interface {
	AddDependency(dependency mdlcapability.Dependency) error
	DeleteDependencyById(id uuid.UUID) error
	GetDependencies() ([]mdlcapability.Dependency, error)
	GetDependencyById(id uuid.UUID) mdlcapability.Dependency
}

// OrderModel provides CRUD operations for [order.Order] resources.
type OrderModel interface {
	AddOrder(order mdlorder.Order) error
	DeleteOrderById(id uuid.UUID) error
	GetOrders() ([]mdlorder.Order, error)
	GetOrderById(id uuid.UUID) mdlorder.Order
}

// OrderItemModel provides CRUD operations for [order.OrderItem] resources.
type OrderItemModel interface {
	AddOrderItem(orderItem mdlorder.OrderItem) error
	DeleteOrderItemById(id uuid.UUID) error
	GetOrderItems() ([]mdlorder.OrderItem, error)
	GetOrderItemById(id uuid.UUID) mdlorder.OrderItem
}

// BoundValueModel provides CRUD operations for [order.BoundValue] resources.
type BoundValueModel interface {
	AddBoundValue(boundValue mdlorder.BoundValue) error
	DeleteBoundValueById(id uuid.UUID) error
	GetBoundValues() ([]mdlorder.BoundValue, error)
	GetBoundValueById(id uuid.UUID) mdlorder.BoundValue
}

func (m *modelData) AddValidValue(v mdlparameter.ValidValue) error {
	return addEventEnabled(m, v, mdlparameter.ValidValue.GetValidValueId,
		func(x mdlparameter.ValidValue, s events.EventSink) { x.Register(s) },
		m.validValuesByUUID, events.ValidValueResource)
}

func (m *modelData) DeleteValidValueById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.validValuesByUUID, events.ValidValueResource, common.ErrValidValueNotFound)
}

func (m *modelData) GetValidValueById(id uuid.UUID) mdlparameter.ValidValue {
	return getEventEnabled(m, id, m.validValuesByUUID)
}

func (m *modelData) GetValidValues() ([]mdlparameter.ValidValue, error) {
	return getAllEventEnabled(m, m.validValuesByUUID)
}

func (m *modelData) AddCapabilityVersion(v mdlcapability.CapabilityVersion) error {
	return addEventEnabled(m, v, mdlcapability.CapabilityVersion.GetCapabilityVersionId,
		func(x mdlcapability.CapabilityVersion, s events.EventSink) { x.Register(s) },
		m.capabilityVersionsByUUID, events.CapabilityVersionResource)
}

func (m *modelData) DeleteCapabilityVersionById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.capabilityVersionsByUUID, events.CapabilityVersionResource, common.ErrCapabilityVersionNotFound)
}

func (m *modelData) GetCapabilityVersionById(id uuid.UUID) mdlcapability.CapabilityVersion {
	return getEventEnabled(m, id, m.capabilityVersionsByUUID)
}

func (m *modelData) GetCapabilityVersions() ([]mdlcapability.CapabilityVersion, error) {
	return getAllEventEnabled(m, m.capabilityVersionsByUUID)
}

func (m *modelData) AddVariant(v mdlcapability.Variant) error {
	return addEventEnabled(m, v, mdlcapability.Variant.GetVariantId,
		func(x mdlcapability.Variant, s events.EventSink) { x.Register(s) },
		m.variantsByUUID, events.VariantResource)
}

func (m *modelData) DeleteVariantById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.variantsByUUID, events.VariantResource, common.ErrVariantNotFound)
}

func (m *modelData) GetVariantById(id uuid.UUID) mdlcapability.Variant {
	return getEventEnabled(m, id, m.variantsByUUID)
}

func (m *modelData) GetVariants() ([]mdlcapability.Variant, error) {
	return getAllEventEnabled(m, m.variantsByUUID)
}

func (m *modelData) AddDependency(d mdlcapability.Dependency) error {
	return addEventEnabled(m, d, mdlcapability.Dependency.GetDependencyId,
		func(x mdlcapability.Dependency, s events.EventSink) { x.Register(s) },
		m.dependenciesByUUID, events.DependencyResource)
}

func (m *modelData) DeleteDependencyById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.dependenciesByUUID, events.DependencyResource, common.ErrDependencyNotFound)
}

func (m *modelData) GetDependencyById(id uuid.UUID) mdlcapability.Dependency {
	return getEventEnabled(m, id, m.dependenciesByUUID)
}

func (m *modelData) GetDependencies() ([]mdlcapability.Dependency, error) {
	return getAllEventEnabled(m, m.dependenciesByUUID)
}

func (m *modelData) AddOrder(o mdlorder.Order) error {
	return addEventEnabled(m, o, mdlorder.Order.GetOrderId,
		func(x mdlorder.Order, s events.EventSink) { x.Register(s) },
		m.ordersByUUID, events.OrderResource)
}

func (m *modelData) DeleteOrderById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.ordersByUUID, events.OrderResource, common.ErrOrderNotFound)
}

func (m *modelData) GetOrderById(id uuid.UUID) mdlorder.Order {
	return getEventEnabled(m, id, m.ordersByUUID)
}

func (m *modelData) GetOrders() ([]mdlorder.Order, error) {
	return getAllEventEnabled(m, m.ordersByUUID)
}

func (m *modelData) AddOrderItem(oi mdlorder.OrderItem) error {
	return addEventEnabled(m, oi, mdlorder.OrderItem.GetOrderItemId,
		func(x mdlorder.OrderItem, s events.EventSink) { x.Register(s) },
		m.orderItemsByUUID, events.OrderItemResource)
}

func (m *modelData) DeleteOrderItemById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.orderItemsByUUID, events.OrderItemResource, common.ErrOrderItemNotFound)
}

func (m *modelData) GetOrderItemById(id uuid.UUID) mdlorder.OrderItem {
	return getEventEnabled(m, id, m.orderItemsByUUID)
}

func (m *modelData) GetOrderItems() ([]mdlorder.OrderItem, error) {
	return getAllEventEnabled(m, m.orderItemsByUUID)
}

func (m *modelData) AddBoundValue(bv mdlorder.BoundValue) error {
	return addEventEnabled(m, bv, mdlorder.BoundValue.GetBoundValueId,
		func(x mdlorder.BoundValue, s events.EventSink) { x.Register(s) },
		m.boundValuesByUUID, events.BoundValueResource)
}

func (m *modelData) DeleteBoundValueById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.boundValuesByUUID, events.BoundValueResource, common.ErrBoundValueNotFound)
}

func (m *modelData) GetBoundValueById(id uuid.UUID) mdlorder.BoundValue {
	return getEventEnabled(m, id, m.boundValuesByUUID)
}

func (m *modelData) GetBoundValues() ([]mdlorder.BoundValue, error) {
	return getAllEventEnabled(m, m.boundValuesByUUID)
}
