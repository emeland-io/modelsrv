package ingress

import (
	"fmt"

	"go.emeland.io/modelsrv/pkg/model"
	mdlcapability "go.emeland.io/modelsrv/pkg/model/capability"
	mdlorder "go.emeland.io/modelsrv/pkg/model/order"
	mdlparameter "go.emeland.io/modelsrv/pkg/model/parameter"
)

func applyValidValue(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "validValueId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	paramID, err := parseUUIDField(spec, "parameter")
	if err != nil {
		return err
	}
	v := mdlparameter.NewValidValue(id)
	v.SetDisplayName(name)
	v.SetParameterById(paramID)
	if err := applyAnnotations(v.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddValidValue(v)
}

func applyCapabilityVersion(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "capabilityVersionId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	capID, err := parseUUIDField(spec, "capability")
	if err != nil {
		return err
	}
	v := mdlcapability.NewCapabilityVersion(id)
	v.SetDisplayName(name)
	v.SetCapabilityById(capID)
	if ver, err := parseVersionSpec(spec["version"]); err != nil {
		return err
	} else {
		v.SetVersion(ver)
	}
	if err := applyAnnotations(v.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddCapabilityVersion(v)
}

func applyVariant(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "variantId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	cvID, err := parseUUIDField(spec, "capabilityVersion")
	if err != nil {
		return err
	}
	v := mdlcapability.NewVariant(id)
	v.SetDisplayName(name)
	v.SetCapabilityVersionById(cvID)
	requires, err := parseUUIDStringList(spec, "requires")
	if err != nil {
		return err
	}
	v.SetRequires(requires)
	if err := applyAnnotations(v.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddVariant(v)
}

func applyDependency(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "dependencyId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	variantID, err := parseUUIDField(spec, "variant")
	if err != nil {
		return err
	}
	capID, err := parseUUIDField(spec, "capability")
	if err != nil {
		return err
	}
	d := mdlcapability.NewDependency(id)
	d.SetDisplayName(name)
	d.SetVariantById(variantID)
	d.SetCapabilityById(capID)
	if mappings, err := parseValueMappings(spec); err != nil {
		return err
	} else {
		d.SetMappings(mappings)
	}
	if err := applyAnnotations(d.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddDependency(d)
}

func parseValueMappings(spec map[string]any) ([]mdlcapability.ValueMapping, error) {
	raw, ok := spec["mappings"]
	if !ok || raw == nil {
		return nil, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("mappings must be an array")
	}
	out := make([]mdlcapability.ValueMapping, 0, len(arr))
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("mappings[%d] must be an object", i)
		}
		fromID, err := parseUUIDField(m, "fromValidValueId")
		if err != nil {
			return nil, fmt.Errorf("mappings[%d]: %w", i, err)
		}
		toID, err := parseUUIDField(m, "toValidValueId")
		if err != nil {
			return nil, fmt.Errorf("mappings[%d]: %w", i, err)
		}
		out = append(out, mdlcapability.ValueMapping{
			FromValidValueId: fromID,
			ToValidValueId:   toID,
		})
	}
	return out, nil
}

func applyOrder(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "orderId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	orgUnitID, err := parseUUIDField(spec, "orgUnit")
	if err != nil {
		return err
	}
	o := mdlorder.NewOrder(id)
	o.SetDisplayName(name)
	o.SetOrgUnitById(orgUnitID)
	if err := applyAnnotations(o.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddOrder(o)
}

func applyOrderItem(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "orderItemId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	orderID, err := parseUUIDField(spec, "order")
	if err != nil {
		return err
	}
	capID, err := parseUUIDField(spec, "capability")
	if err != nil {
		return err
	}
	cvID, err := parseUUIDField(spec, "capabilityVersion")
	if err != nil {
		return err
	}
	variantID, err := parseUUIDField(spec, "variant")
	if err != nil {
		return err
	}
	oi := mdlorder.NewOrderItem(id)
	oi.SetDisplayName(name)
	oi.SetOrderById(orderID)
	oi.SetCapabilityRef(&mdlcapability.CapabilityRef{CapabilityId: capID})
	oi.SetCapabilityVersionRef(&mdlcapability.CapabilityVersionRef{CapabilityVersionId: cvID})
	oi.SetVariantRef(&mdlcapability.VariantRef{VariantId: variantID})
	contexts, err := parseUUIDStringList(spec, "contexts")
	if err != nil {
		return err
	}
	oi.SetContexts(contexts)
	if err := applyAnnotations(oi.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddOrderItem(oi)
}

func applyBoundValue(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "boundValueId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	orderItemID, err := parseUUIDField(spec, "orderItem")
	if err != nil {
		return err
	}
	paramID, err := parseUUIDField(spec, "parameter")
	if err != nil {
		return err
	}
	vvID, err := parseUUIDField(spec, "validValue")
	if err != nil {
		return err
	}
	bv := mdlorder.NewBoundValue(id)
	bv.SetDisplayName(name)
	bv.SetOrderItemById(orderItemID)
	bv.SetParameterRef(&mdlparameter.ParameterRef{ParameterId: paramID})
	bv.SetValidValueRef(&mdlparameter.ValidValueRef{ValidValueId: vvID})
	if err := applyAnnotations(bv.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddBoundValue(bv)
}
