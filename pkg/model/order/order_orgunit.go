package order

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/iam"
)

// GetOrgUnit returns the embedded [iam.OrgUnit] when present (otherwise nil).
func (o *orderData) GetOrgUnit() (iam.OrgUnit, error) {
	if o.OrgUnitRef == nil {
		return nil, nil
	}
	return o.OrgUnitRef.ResolvedOrgUnit(), nil
}

// GetOrgUnitId returns the org unit id when set.
func (o *orderData) GetOrgUnitId() uuid.UUID {
	if o.OrgUnitRef == nil {
		return uuid.Nil
	}
	return o.OrgUnitRef.EffectiveOrgUnitID()
}

// SetOrgUnitRef sets the low-level org unit reference and emits when registered.
func (o *orderData) SetOrgUnitRef(val *iam.OrgUnitRef) {
	o.OrgUnitRef = val

	if o.isRegistered {
		_ = o.sink.Receive(events.OrderResource, events.UpdateOperation, o.OrderId, o)
	}
}

// SetOrgUnitById records only the org unit id (resolved object may be nil).
func (o *orderData) SetOrgUnitById(orgUnitId uuid.UUID) {
	if orgUnitId == uuid.Nil {
		o.SetOrgUnitRef(nil)
		return
	}
	o.SetOrgUnitRef(&iam.OrgUnitRef{OrgUnitId: orgUnitId})
}
