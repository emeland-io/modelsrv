// Package resolvefindings provides an [eventfilter.FilterFunc] that deletes
// findings when later resource create/update events make them obsolete.
//
// Resolution paths:
//   - Structural dangling-ref (kind-agnostic): findings whose Resources list
//     cites a missing resource that now exists are deleted, including unknown
//     FindingKinds — unknown findings are never discarded merely for being unknown.
//   - Documented MissingResourceReference: when the subject gains its required
//     first-class reference (ApiInstance.ApiRef, ComponentInstance.ComponentRef).
//
// Phase-0 finding kinds are skipped; those remain under [phase0] negation.
package resolvefindings

import (
	"errors"
	"log"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/eventfilter"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

var phase0Kinds = []finding.FindingKind{
	finding.ContextTypeMissing,
	finding.ContextParentNotFound,
	finding.NodeTypeMissing,
}

// New returns the resolve-findings filter with its discoverable identity.
func New() eventfilter.Filter {
	return eventfilter.Filter{
		DisplayName: "Resolve findings",
		Description: "Deletes findings when referenced resources appear or required references are populated.",
		Fn:          filterFunc(),
	}
}

// NewFilterFunc returns the resolve-findings filter function.
func NewFilterFunc() eventfilter.FilterFunc {
	return New().Fn
}

// EnsureWellKnownFindingTypes registers or backfills the documented
// resolve-findings FindingType resources in the model.
func EnsureWellKnownFindingTypes(m model.Model) {
	for _, kind := range []finding.FindingKind{
		finding.ReferencedResourceNotFound,
		finding.MissingResourceReference,
	} {
		ensureFindingType(m, kind)
	}
}

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
		log.Printf("resolvefindings: AddFindingType kind=%s id=%s: %v", kind, id, err)
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
		log.Printf("resolvefindings: backfill FindingType description kind=%s id=%s: %v", kind, ft.GetFindingTypeId(), err)
	}
}

func filterFunc() eventfilter.FilterFunc {
	return func(m model.Model, ev events.Event) []events.Event {
		switch ev.Operation {
		case events.CreateOperation, events.UpdateOperation:
			switch ev.ResourceType {
			case events.FindingResource, events.FindingTypeResource:
				// Never suppress or rewrite finding ingress; leave unknown findings intact.
			default:
				if ev.ResourceId != uuid.Nil {
					resolveAgainstEvent(m, ev)
				}
			}
		}
		return []events.Event{ev}
	}
}

func resolveAgainstEvent(m model.Model, ev events.Event) {
	candidates := m.GetFindingsReferencingResource(ev.ResourceId)
	for _, f := range candidates {
		if f == nil {
			continue
		}
		if isPhase0Finding(m, f) {
			continue
		}
		if tryResolveDanglingRef(m, f, ev) {
			continue
		}
		tryResolveMissingResourceReference(m, f, ev)
	}
}

func isPhase0Finding(m model.Model, f finding.Finding) bool {
	typeID := f.GetFindingTypeId()
	for _, kind := range phase0Kinds {
		if typeID == finding.TypeIDForKind(kind) {
			return true
		}
	}
	if ft := m.GetFindingTypeById(typeID); ft != nil {
		name := ft.GetDisplayName()
		for _, kind := range phase0Kinds {
			if name == string(kind) {
				return true
			}
		}
	}
	return false
}

func tryResolveDanglingRef(m model.Model, f finding.Finding, ev events.Event) bool {
	refs := f.GetResources()
	if len(refs) < 2 {
		return false
	}
	for _, ref := range refs[1:] {
		if !refMatchesEvent(ref, ev) {
			continue
		}
		check := *ref
		if check.ResourceType == events.UnknownResourceType {
			check.ResourceType = ev.ResourceType
		}
		if !model.ResourceExists(m, &check) {
			continue
		}
		deleteFinding(m, f.GetFindingId())
		return true
	}
	return false
}

func refMatchesEvent(ref *common.ResourceRef, ev events.Event) bool {
	if ref == nil || ref.ResourceId != ev.ResourceId {
		return false
	}
	if ref.ResourceType != events.UnknownResourceType && ref.ResourceType != ev.ResourceType {
		return false
	}
	return true
}

func tryResolveMissingResourceReference(m model.Model, f finding.Finding, ev events.Event) {
	if !isMissingResourceReference(m, f) {
		return
	}
	refs := f.GetResources()
	if len(refs) == 0 || refs[0] == nil {
		return
	}
	subject := refs[0]
	if subject.ResourceId != ev.ResourceId {
		return
	}
	if subject.ResourceType != events.UnknownResourceType && subject.ResourceType != ev.ResourceType {
		return
	}
	if !subjectHasRequiredRef(m, subject) {
		return
	}
	deleteFinding(m, f.GetFindingId())
}

func isMissingResourceReference(m model.Model, f finding.Finding) bool {
	typeID := f.GetFindingTypeId()
	if typeID == finding.TypeIDForKind(finding.MissingResourceReference) {
		return true
	}
	if ft := m.GetFindingTypeById(typeID); ft != nil {
		return ft.GetDisplayName() == string(finding.MissingResourceReference)
	}
	return false
}

func subjectHasRequiredRef(m model.Model, subject *common.ResourceRef) bool {
	switch subject.ResourceType {
	case events.APIInstanceResource:
		ai := m.GetApiInstanceById(subject.ResourceId)
		if ai == nil {
			return false
		}
		ref := ai.GetApiRef()
		return ref != nil && ref.ApiID != uuid.Nil
	case events.ComponentInstanceResource:
		ci := m.GetComponentInstanceById(subject.ResourceId)
		if ci == nil {
			return false
		}
		ref := ci.GetComponentRef()
		return ref != nil && ref.ComponentId != uuid.Nil
	default:
		return false
	}
}

func deleteFinding(m model.Model, id uuid.UUID) {
	if m.GetFindingById(id) == nil {
		return
	}
	if err := m.DeleteFindingById(id); err != nil && !errors.Is(err, common.ErrFindingNotFound) {
		log.Printf("resolvefindings: DeleteFindingById id=%s: %v", id, err)
	}
}
