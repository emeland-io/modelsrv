package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	mdlcapability "go.emeland.io/modelsrv/pkg/model/capability"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/iam"
	mdlorder "go.emeland.io/modelsrv/pkg/model/order"
	mdlparameter "go.emeland.io/modelsrv/pkg/model/parameter"
)

func TestBoundValueUniquenessAndVariantProvides(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	ou := iam.NewOrgUnit(uuid.New())
	ou.SetDisplayName("Org")
	require.NoError(t, m.AddOrgUnit(ou))

	capID := seedCapability(t, m, "Cap")
	cvID := seedCapabilityVersion(t, m, capID, "v1")
	paramID, vids := seedParamWithValues(t, m, "region", "eu", "us")

	variant := mdlcapability.NewVariant(uuid.New())
	variant.SetDisplayName("eu-only")
	variant.SetCapabilityVersionById(cvID)
	variant.SetProvides([]mdlcapability.ParameterValueSet{{
		ParameterId:   paramID,
		ValidValueIds: vids[:1], // eu only
	}})
	require.NoError(t, m.AddVariant(variant))

	ord := mdlorder.NewOrder(uuid.New())
	ord.SetDisplayName("Order")
	ord.SetOrgUnitById(ou.GetOrgUnitId())
	require.NoError(t, m.AddOrder(ord))

	oi := mdlorder.NewOrderItem(uuid.New())
	oi.SetDisplayName("Item")
	oi.SetOrderById(ord.GetOrderId())
	oi.SetCapabilityRef(&mdlcapability.CapabilityRef{CapabilityId: capID})
	oi.SetVariantRef(&mdlcapability.VariantRef{VariantId: variant.GetVariantId()})
	require.NoError(t, m.AddOrderItem(oi))

	bvID := uuid.New()
	bv := mdlorder.NewBoundValue(bvID)
	bv.SetDisplayName("eu")
	bv.SetOrderItemById(oi.GetOrderItemId())
	bv.SetParameterRef(&mdlparameter.ParameterRef{ParameterId: paramID})
	bv.SetValidValueRef(&mdlparameter.ValidValueRef{ValidValueId: vids[0]})
	require.NoError(t, m.AddBoundValue(bv))

	conflict := mdlorder.NewBoundValue(uuid.New())
	conflict.SetDisplayName("eu-again")
	conflict.SetOrderItemById(oi.GetOrderItemId())
	conflict.SetParameterRef(&mdlparameter.ParameterRef{ParameterId: paramID})
	conflict.SetValidValueRef(&mdlparameter.ValidValueRef{ValidValueId: vids[0]})
	assert.ErrorIs(t, m.AddBoundValue(conflict), common.ErrBoundValueConflict)

	notProvided := mdlorder.NewBoundValue(uuid.New())
	notProvided.SetDisplayName("us")
	notProvided.SetOrderItemById(oi.GetOrderItemId())
	notProvided.SetParameterRef(&mdlparameter.ParameterRef{ParameterId: paramID})
	notProvided.SetValidValueRef(&mdlparameter.ValidValueRef{ValidValueId: vids[1]})
	assert.Error(t, m.AddBoundValue(notProvided))
}

func TestOrderItemRefsMustMatchCapability(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	ou := iam.NewOrgUnit(uuid.New())
	ou.SetDisplayName("Org")
	require.NoError(t, m.AddOrgUnit(ou))

	capID := seedCapability(t, m, "Cap")
	otherCap := seedCapability(t, m, "Other")
	cvID := seedCapabilityVersion(t, m, capID, "v1")
	otherCV := seedCapabilityVersion(t, m, otherCap, "other-v")

	otherVariant := mdlcapability.NewVariant(uuid.New())
	otherVariant.SetDisplayName("other")
	otherVariant.SetCapabilityVersionById(otherCV)
	require.NoError(t, m.AddVariant(otherVariant))

	ord := mdlorder.NewOrder(uuid.New())
	ord.SetDisplayName("Order")
	ord.SetOrgUnitById(ou.GetOrgUnitId())
	require.NoError(t, m.AddOrder(ord))

	wrongVersion := mdlorder.NewOrderItem(uuid.New())
	wrongVersion.SetDisplayName("wrong version")
	wrongVersion.SetOrderById(ord.GetOrderId())
	wrongVersion.SetCapabilityRef(&mdlcapability.CapabilityRef{CapabilityId: capID})
	wrongVersion.SetCapabilityVersionRef(&mdlcapability.CapabilityVersionRef{CapabilityVersionId: otherCV})
	assert.Error(t, m.AddOrderItem(wrongVersion))

	wrongVariant := mdlorder.NewOrderItem(uuid.New())
	wrongVariant.SetDisplayName("wrong variant")
	wrongVariant.SetOrderById(ord.GetOrderId())
	wrongVariant.SetCapabilityRef(&mdlcapability.CapabilityRef{CapabilityId: capID})
	wrongVariant.SetCapabilityVersionRef(&mdlcapability.CapabilityVersionRef{CapabilityVersionId: cvID})
	wrongVariant.SetVariantRef(&mdlcapability.VariantRef{VariantId: otherVariant.GetVariantId()})
	assert.Error(t, m.AddOrderItem(wrongVariant))

	nilBound := mdlorder.NewOrderItem(uuid.New())
	nilBound.SetDisplayName("nil bound")
	nilBound.SetOrderById(ord.GetOrderId())
	nilBound.SetCapabilityRef(&mdlcapability.CapabilityRef{CapabilityId: capID})
	nilBound.SetBoundValues([]mdlorder.BoundValueRef{{}})
	assert.Error(t, m.AddOrderItem(nilBound))
}
