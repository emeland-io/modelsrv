package endpointprobe

import (
	"log"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

// FindingPublisher upserts or deletes Finding resources after a probe cycle.
type FindingPublisher interface {
	Upsert(f finding.Finding) error
	Delete(id uuid.UUID) error
	// EnsureType registers or reuses the FindingType for kind and returns its id
	// (same pattern as phase0 ensureFindingType).
	EnsureType(kind finding.FindingKind) uuid.UUID
}

// ModelFindingPublisher applies Finding create/delete events via [model.Model.Apply],
// the same pipeline used by POST /events/push.
type ModelFindingPublisher struct {
	Model model.Model
}

// NewModelFindingPublisher returns a FindingPublisher backed by m.
func NewModelFindingPublisher(m model.Model) *ModelFindingPublisher {
	return &ModelFindingPublisher{Model: m}
}

// Upsert implements [FindingPublisher].
func (p *ModelFindingPublisher) Upsert(f finding.Finding) error {
	return p.Model.Apply(events.Event{
		ResourceType: events.FindingResource,
		Operation:    events.CreateOperation,
		ResourceId:   f.GetFindingId(),
		Objects:      []any{f},
	})
}

// Delete implements [FindingPublisher].
func (p *ModelFindingPublisher) Delete(id uuid.UUID) error {
	return p.Model.Apply(events.Event{
		ResourceType: events.FindingResource,
		Operation:    events.DeleteOperation,
		ResourceId:   id,
	})
}

// EnsureType implements [FindingPublisher].
func (p *ModelFindingPublisher) EnsureType(kind finding.FindingKind) uuid.UUID {
	return ensureFindingType(p.Model, kind)
}

// ensureFindingType returns the FindingType id for kind: existing match by name,
// else create with [finding.TypeIDForKind] (mirrors phase0).
func ensureFindingType(m model.Model, kind finding.FindingKind) uuid.UUID {
	name := string(kind)
	if ft := m.GetFindingTypeByName(name); ft != nil {
		backfillFindingTypeDescription(m, ft, kind)
		return ft.GetFindingTypeId()
	}

	id := finding.TypeIDForKind(kind)
	if ft := m.GetFindingTypeById(id); ft != nil {
		backfillFindingTypeDescription(m, ft, kind)
		return id
	}

	ft := finding.NewFindingType(id)
	ft.SetDisplayName(name)
	if desc := finding.DescriptionForKind(kind); desc != "" {
		ft.SetDescription(desc)
	}
	if err := m.AddFindingType(ft); err != nil {
		log.Printf("endpointprobe: AddFindingType kind=%s id=%s: %v", kind, id, err)
	}
	return id
}

func backfillFindingTypeDescription(m model.Model, ft finding.FindingType, kind finding.FindingKind) {
	desc := finding.DescriptionForKind(kind)
	if desc == "" || ft.GetDescription() != "" {
		return
	}
	updated := finding.NewFindingType(ft.GetFindingTypeId())
	updated.SetDisplayName(ft.GetDisplayName())
	updated.SetDescription(desc)
	if err := m.AddFindingType(updated); err != nil {
		log.Printf("endpointprobe: backfill FindingType description kind=%s id=%s: %v", kind, ft.GetFindingTypeId(), err)
	}
}

// EnsureWellKnownFindingTypes registers or backfills the canonical certificate
// FindingType resources (display name and description) in the model.
func EnsureWellKnownFindingTypes(m model.Model) {
	for _, kind := range certFindingKinds {
		ensureFindingType(m, kind)
	}
}
