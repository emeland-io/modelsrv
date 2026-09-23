package ingress

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model"
	mdlcapability "go.emeland.io/modelsrv/pkg/model/capability"
	mdlorder "go.emeland.io/modelsrv/pkg/model/order"
	mdlparameter "go.emeland.io/modelsrv/pkg/model/parameter"
)

func applyCapabilityVersion(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "capabilityVersionId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}
	capID, err := uuidRefFromMap(spec, "capabilityRef", "capabilityId")
	if err != nil {
		return err
	}
	cv := mdlcapability.NewCapabilityVersion(id)
	cv.SetDisplayName(name)
	cv.SetCapabilityById(capID)
	if ver, err := parseVersionSpec(spec["version"]); err != nil {
		return err
	} else {
		cv.SetVersion(ver)
	}
	if err := applyAnnotations(cv.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddCapabilityVersion(cv)
}

func parseParameterValueSet(raw any) (mdlcapability.ParameterValueSet, error) {
	m, ok := raw.(map[string]any)
	if !ok {
		return mdlcapability.ParameterValueSet{}, fmt.Errorf("parameter value set must be an object")
	}
	paramID, err := parseUUIDField(m, "parameterId")
	if err != nil {
		return mdlcapability.ParameterValueSet{}, err
	}
	out := mdlcapability.ParameterValueSet{ParameterId: paramID}
	rawIDs, ok := m["validValueIds"]
	if !ok {
		return out, nil
	}
	list, ok := rawIDs.([]any)
	if !ok {
		return mdlcapability.ParameterValueSet{}, fmt.Errorf("validValueIds must be an array")
	}
	for _, v := range list {
		s, ok := v.(string)
		if !ok {
			return mdlcapability.ParameterValueSet{}, fmt.Errorf("validValueIds entries must be UUID strings")
		}
		vid, err := uuid.Parse(s)
		if err != nil {
			return mdlcapability.ParameterValueSet{}, fmt.Errorf("invalid validValueId %q: %w", s, err)
		}
		out.ValidValueIds = append(out.ValidValueIds, vid)
	}
	return out, nil
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
	cvID, err := uuidRefFromMap(spec, "capabilityVersionRef", "capabilityVersionId")
	if err != nil {
		return err
	}
	v := mdlcapability.NewVariant(id)
	v.SetDisplayName(name)
	v.SetCapabilityVersionById(cvID)
	if raw, ok := spec["provides"]; ok {
		list, ok := raw.([]any)
		if !ok {
			return fmt.Errorf("provides must be an array")
		}
		provides := make([]mdlcapability.ParameterValueSet, 0, len(list))
		for i, item := range list {
			pvs, err := parseParameterValueSet(item)
			if err != nil {
				return fmt.Errorf("provides[%d]: %w", i, err)
			}
			provides = append(provides, pvs)
		}
		v.SetProvides(provides)
	}
	if raw, ok := spec["dependencies"]; ok {
		list, ok := raw.([]any)
		if !ok {
			return fmt.Errorf("dependencies must be an array")
		}
		deps := make([]mdlcapability.VariantDependency, 0, len(list))
		for i, item := range list {
			dm, ok := item.(map[string]any)
			if !ok {
				return fmt.Errorf("dependencies[%d] must be an object", i)
			}
			capID, err := parseUUIDField(dm, "capabilityId")
			if err != nil {
				return fmt.Errorf("dependencies[%d]: %w", i, err)
			}
			dep := mdlcapability.VariantDependency{CapabilityId: capID}
			if reqRaw, ok := dm["required"]; ok {
				reqList, ok := reqRaw.([]any)
				if !ok {
					return fmt.Errorf("dependencies[%d].required must be an array", i)
				}
				for j, reqItem := range reqList {
					pvs, err := parseParameterValueSet(reqItem)
					if err != nil {
						return fmt.Errorf("dependencies[%d].required[%d]: %w", i, j, err)
					}
					dep.Required = append(dep.Required, pvs)
				}
			}
			deps = append(deps, dep)
		}
		v.SetDependencies(deps)
	}
	if err := applyAnnotations(v.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddVariant(v)
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
	orgID, err := uuidRefFromMap(spec, "orgUnitRef", "orgUnitId")
	if err != nil {
		return err
	}
	o := mdlorder.NewOrder(id)
	o.SetDisplayName(name)
	o.SetOrgUnitById(orgID)
	if raw, ok := spec["items"]; ok {
		list, ok := raw.([]any)
		if !ok {
			return fmt.Errorf("items must be an array")
		}
		items := make([]mdlorder.OrderItemRef, 0, len(list))
		for _, item := range list {
			im, ok := item.(map[string]any)
			if !ok {
				return fmt.Errorf("items entries must be objects")
			}
			iid, err := parseUUIDField(im, "orderItemId")
			if err != nil {
				return err
			}
			items = append(items, mdlorder.OrderItemRef{OrderItemId: iid})
		}
		o.SetItems(items)
	}
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
	orderID, err := uuidRefFromMap(spec, "orderRef", "orderId")
	if err != nil {
		return err
	}
	capID, err := uuidRefFromMap(spec, "capabilityRef", "capabilityId")
	if err != nil {
		return err
	}
	oi := mdlorder.NewOrderItem(id)
	oi.SetDisplayName(name)
	oi.SetOrderById(orderID)
	oi.SetCapabilityRef(&mdlcapability.CapabilityRef{CapabilityId: capID})
	if _, ok := spec["capabilityVersionRef"]; ok {
		cvID, err := uuidRefFromMap(spec, "capabilityVersionRef", "capabilityVersionId")
		if err != nil {
			return err
		}
		oi.SetCapabilityVersionRef(&mdlcapability.CapabilityVersionRef{CapabilityVersionId: cvID})
	}
	if _, ok := spec["variantRef"]; ok {
		vid, err := uuidRefFromMap(spec, "variantRef", "variantId")
		if err != nil {
			return err
		}
		oi.SetVariantRef(&mdlcapability.VariantRef{VariantId: vid})
	}
	if raw, ok := spec["boundValues"]; ok {
		list, ok := raw.([]any)
		if !ok {
			return fmt.Errorf("boundValues must be an array")
		}
		refs := make([]mdlorder.BoundValueRef, 0, len(list))
		for _, item := range list {
			im, ok := item.(map[string]any)
			if !ok {
				return fmt.Errorf("boundValues entries must be objects")
			}
			bvid, err := parseUUIDField(im, "boundValueId")
			if err != nil {
				return err
			}
			refs = append(refs, mdlorder.BoundValueRef{BoundValueId: bvid})
		}
		oi.SetBoundValues(refs)
	}
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
	oiID, err := uuidRefFromMap(spec, "orderItemRef", "orderItemId")
	if err != nil {
		return err
	}
	paramID, err := uuidRefFromMap(spec, "parameterRef", "parameterId")
	if err != nil {
		return err
	}
	vvID, err := uuidRefFromMap(spec, "validValueRef", "validValueId")
	if err != nil {
		return err
	}
	bv := mdlorder.NewBoundValue(id)
	bv.SetDisplayName(name)
	bv.SetOrderItemById(oiID)
	bv.SetParameterRef(&mdlparameter.ParameterRef{ParameterId: paramID})
	bv.SetValidValueRef(&mdlparameter.ValidValueRef{ValidValueId: vvID})
	if err := applyAnnotations(bv.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddBoundValue(bv)
}
