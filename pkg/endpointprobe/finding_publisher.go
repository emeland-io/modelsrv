package endpointprobe

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.uber.org/zap"
)

// FindingPublisher upserts or deletes Finding resources after a probe cycle.
type FindingPublisher interface {
	Upsert(f finding.Finding) error
	Delete(id uuid.UUID) error
	// TypeID returns the cached FindingType id for kind.
	TypeID(kind finding.FindingKind) uuid.UUID
}

// ModelFindingPublisher applies Finding create/delete events via [model.Model.Apply],
// the same pipeline used by POST /events/push.
type ModelFindingPublisher struct {
	Model   model.Model
	Logger  *zap.SugaredLogger
	typeIDs map[finding.FindingKind]uuid.UUID
}

// NewModelFindingPublisher returns a FindingPublisher backed by m.
// It registers the well-known certificate FindingTypes and caches their IDs
// so probe cycles do not re-look them up.
func NewModelFindingPublisher(m model.Model, logger *zap.SugaredLogger) *ModelFindingPublisher {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	p := &ModelFindingPublisher{
		Model:   m,
		Logger:  logger,
		typeIDs: make(map[finding.FindingKind]uuid.UUID, len(certFindingKinds)),
	}
	for _, kind := range certFindingKinds {
		p.typeIDs[kind] = ensureFindingType(m, logger, kind)
	}
	return p
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

// TypeID implements [FindingPublisher].
func (p *ModelFindingPublisher) TypeID(kind finding.FindingKind) uuid.UUID {
	if id, ok := p.typeIDs[kind]; ok {
		return id
	}
	id := ensureFindingType(p.Model, p.Logger, kind)
	p.typeIDs[kind] = id
	return id
}

// ensureFindingType returns the FindingType id for kind: existing match by name,
// else create with [finding.TypeIDForKind] (mirrors phase0).
func ensureFindingType(m model.Model, logger *zap.SugaredLogger, kind finding.FindingKind) uuid.UUID {
	name := string(kind)
	if ft := m.GetFindingTypeByName(name); ft != nil {
		backfillFindingTypeDescription(m, logger, ft, kind)
		return ft.GetFindingTypeId()
	}

	id := finding.TypeIDForKind(kind)
	if ft := m.GetFindingTypeById(id); ft != nil {
		backfillFindingTypeDescription(m, logger, ft, kind)
		return id
	}

	ft := finding.NewFindingType(id)
	ft.SetDisplayName(name)
	if desc := finding.DescriptionForKind(kind); desc != "" {
		ft.SetDescription(desc)
	}
	if err := m.AddFindingType(ft); err != nil {
		logger.Warnw("AddFindingType failed", "kind", kind, "id", id, "error", err)
	}
	return id
}

func backfillFindingTypeDescription(m model.Model, logger *zap.SugaredLogger, ft finding.FindingType, kind finding.FindingKind) {
	desc := finding.DescriptionForKind(kind)
	if desc == "" || ft.GetDescription() != "" {
		return
	}
	updated := finding.NewFindingType(ft.GetFindingTypeId())
	updated.SetDisplayName(ft.GetDisplayName())
	updated.SetDescription(desc)
	if err := m.AddFindingType(updated); err != nil {
		logger.Warnw("backfill FindingType description failed",
			"kind", kind,
			"id", ft.GetFindingTypeId(),
			"error", err,
		)
	}
}

// EnsureWellKnownFindingTypes registers or backfills the canonical certificate
// FindingType resources (display name and description) in the model.
func EnsureWellKnownFindingTypes(m model.Model, logger *zap.SugaredLogger) {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	for _, kind := range certFindingKinds {
		ensureFindingType(m, logger, kind)
	}
}
