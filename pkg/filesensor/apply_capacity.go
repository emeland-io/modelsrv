package filesensor

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model"
	mdlcap "go.emeland.io/modelsrv/pkg/model/capacity"
)

func applyCapacity(spec map[string]any, m model.Model) error {
	id, err := parseUUIDField(spec, "capacityId")
	if err != nil {
		return err
	}
	name, err := displayName(spec)
	if err != nil {
		return err
	}

	typeID, err := uuidRefFromMap(spec, "resourceTypeRef", "capacityResourceTypeId")
	if err != nil {
		return err
	}
	contextID, err := uuidRefFromMap(spec, "contextRef", "contextId")
	if err != nil {
		return err
	}

	categoryRaw, ok := stringField(spec, "category")
	if !ok {
		return fmt.Errorf("category is required")
	}
	category, err := mdlcap.ParseCategory(categoryRaw)
	if err != nil {
		return err
	}

	amountRaw, ok := stringField(spec, "amount")
	if !ok {
		if n, ok := spec["amount"]; ok {
			amountRaw = fmt.Sprint(n)
		}
	}
	if amountRaw == "" {
		return fmt.Errorf("amount is required")
	}
	amount, err := mdlcap.ParseAmount(amountRaw)
	if err != nil {
		return err
	}

	cap := mdlcap.NewCapacity(id)
	cap.SetDisplayName(name)
	if desc, ok := stringField(spec, "description"); ok {
		cap.SetDescription(desc)
	}
	cap.SetCapacityResourceTypeById(typeID)
	cap.SetContextById(contextID)
	cap.SetCategory(category)
	cap.SetAmount(amount)
	if err := applyAnnotations(cap.GetAnnotations(), spec); err != nil {
		return err
	}
	return m.AddCapacity(cap)
}

func uuidRefFromMap(spec map[string]any, key, idField string) (uuid.UUID, error) {
	raw, ok := spec[key]
	if !ok || raw == nil {
		return uuid.Nil, fmt.Errorf("%q is required", key)
	}
	subm, ok := raw.(map[string]any)
	if !ok {
		return uuid.Nil, fmt.Errorf("%q must be an object", key)
	}
	return parseUUIDField(subm, idField)
}
