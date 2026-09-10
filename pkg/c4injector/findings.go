package c4injector

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/eventfilter"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/component"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.emeland.io/modelsrv/pkg/model/system"
)

// Same namespace as phase0 so (subject, kind) finding IDs stay deterministic and unique.
var findingNamespace = uuid.MustParse("7a3f2c1e-4b8d-5e9f-a0b1-c2d3e4f56789")

const findingDisplayName = "System Structure documentation"

// docFindingKinds are FindingKinds owned by this injector.
var docFindingKinds = map[finding.FindingKind]struct{}{
	finding.DescriptionMissing:                     {},
	finding.ConcreteSystemHasNoComponents:          {},
	finding.AbstractSystemHasComponents:            {},
	finding.APIHasNoProvider:                       {},
	finding.APIHasMultipleProviders:                {},
	finding.ProvidedAPISystemMismatch:              {},
	finding.SystemInstanceContextMissing:           {},
	finding.ApiInstanceMissingForComponentInstance: {},
	finding.ApiInstanceSystemInstanceMissing:       {},
}

func findingID(subjectID uuid.UUID, kind finding.FindingKind) uuid.UUID {
	key := append(subjectID[:], []byte(kind)...)
	return uuid.NewSHA1(findingNamespace, key)
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
		log.Printf("c4injector: AddFindingType kind=%s id=%s: %v", kind, id, err)
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
		log.Printf("c4injector: backfill FindingType description kind=%s: %v", kind, err)
	}
}

func upsertFinding(m model.Model, kind finding.FindingKind, description string, resources []*common.ResourceRef) {
	if len(resources) == 0 || resources[0] == nil {
		return
	}
	subjectID := resources[0].ResourceId
	id := findingID(subjectID, kind)
	f := finding.NewFinding(id)
	f.SetFindingTypeById(ensureFindingType(m, kind))
	f.SetDisplayName(findingDisplayName)
	f.SetDescription(description)
	f.SetResources(resources)
	if err := m.AddFinding(f); err != nil {
		log.Printf("c4injector: AddFinding id=%s kind=%s: %v", id, kind, err)
	}
}

func deleteFinding(m model.Model, subjectID uuid.UUID, kind finding.FindingKind) {
	id := findingID(subjectID, kind)
	if m.GetFindingById(id) == nil {
		return
	}
	if err := m.DeleteFindingById(id); err != nil && !errors.Is(err, common.ErrFindingNotFound) {
		log.Printf("c4injector: DeleteFindingById id=%s kind=%s: %v", id, kind, err)
	}
}

// EnsureWellKnownFindingTypes registers C4 documentation FindingTypes.
func EnsureWellKnownFindingTypes(m model.Model) {
	for kind := range docFindingKinds {
		ensureFindingType(m, kind)
	}
}

// NewFilter returns an event filter that maintains C4 documentation Findings.
func NewFilter() eventfilter.Filter {
	return eventfilter.Filter{
		DisplayName: "c4-doc-injector",
		Description: "Raises System Structure documentation findings for C4-PlantUML diagrams.",
		Fn:          filterFunc(),
	}
}

func filterFunc() eventfilter.FilterFunc {
	return func(m model.Model, ev events.Event) []events.Event {
		switch ev.ResourceType {
		case events.SystemResource, events.APIResource, events.ComponentResource,
			events.SystemInstanceResource, events.APIInstanceResource, events.ComponentInstanceResource:
			Reconcile(m)
		}
		return []events.Event{ev}
	}
}

// Reconcile evaluates all C4 documentation findings against the current model.
func Reconcile(m model.Model) {
	systems, err := m.GetSystems()
	if err != nil {
		log.Printf("c4injector: Reconcile GetSystems: %v", err)
		return
	}
	apis, err := m.GetApis()
	if err != nil {
		log.Printf("c4injector: Reconcile GetApis: %v", err)
		return
	}
	comps, err := m.GetComponents()
	if err != nil {
		log.Printf("c4injector: Reconcile GetComponents: %v", err)
		return
	}

	compsBySystem := map[uuid.UUID][]component.Component{}
	providersByAPI := map[uuid.UUID][]component.Component{}
	for _, c := range comps {
		sid := refSystemID(c.GetSystem())
		compsBySystem[sid] = append(compsBySystem[sid], c)
		for _, pref := range c.GetProvides() {
			aid := refAPIID(pref)
			if aid != uuid.Nil {
				providersByAPI[aid] = append(providersByAPI[aid], c)
			}
		}
		checkComponent(m, c)
	}

	seenSystems := map[uuid.UUID]struct{}{}
	for _, s := range systems {
		seenSystems[s.GetSystemId()] = struct{}{}
		checkSystem(m, s, compsBySystem[s.GetSystemId()])
	}

	seenAPIs := map[uuid.UUID]struct{}{}
	for _, a := range apis {
		seenAPIs[a.GetApiId()] = struct{}{}
		checkAPI(m, a, providersByAPI[a.GetApiId()])
	}

	seenComps := map[uuid.UUID]struct{}{}
	for _, c := range comps {
		seenComps[c.GetComponentId()] = struct{}{}
	}

	seenSysInst, seenCompInst, seenAPIInst := reconcileInstances(m)

	all, err := m.GetFindings()
	if err != nil {
		return
	}
	for _, f := range all {
		kind := findingKindOf(m, f)
		if _, ok := docFindingKinds[kind]; !ok {
			continue
		}
		res := f.GetResources()
		if len(res) == 0 || res[0] == nil {
			continue
		}
		sid := res[0].ResourceId
		switch res[0].ResourceType {
		case events.SystemResource:
			if _, ok := seenSystems[sid]; !ok {
				_ = m.DeleteFindingById(f.GetFindingId())
			}
		case events.APIResource:
			if _, ok := seenAPIs[sid]; !ok {
				_ = m.DeleteFindingById(f.GetFindingId())
			}
		case events.ComponentResource:
			if _, ok := seenComps[sid]; !ok {
				_ = m.DeleteFindingById(f.GetFindingId())
			}
		case events.SystemInstanceResource:
			if _, ok := seenSysInst[sid]; !ok {
				_ = m.DeleteFindingById(f.GetFindingId())
			}
		case events.ComponentInstanceResource:
			if _, ok := seenCompInst[sid]; !ok {
				_ = m.DeleteFindingById(f.GetFindingId())
			}
		case events.APIInstanceResource:
			if _, ok := seenAPIInst[sid]; !ok {
				_ = m.DeleteFindingById(f.GetFindingId())
			}
		}
	}
}

func reconcileInstances(m model.Model) (seenSysInst, seenCompInst, seenAPIInst map[uuid.UUID]struct{}) {
	seenSysInst = map[uuid.UUID]struct{}{}
	seenCompInst = map[uuid.UUID]struct{}{}
	seenAPIInst = map[uuid.UUID]struct{}{}

	sysInsts, err := m.GetSystemInstances()
	if err != nil {
		log.Printf("c4injector: Reconcile GetSystemInstances: %v", err)
		return
	}
	compInsts, err := m.GetComponentInstances()
	if err != nil {
		log.Printf("c4injector: Reconcile GetComponentInstances: %v", err)
		return
	}
	apiInsts, err := m.GetApiInstances()
	if err != nil {
		log.Printf("c4injector: Reconcile GetApiInstances: %v", err)
		return
	}

	apiInstIndex := map[apiInstKey]struct{}{}
	for _, ai := range apiInsts {
		siID := refSystemInstanceID(ai.GetSystemInstance())
		apiID := refAPIIDFromPtr(ai.GetApiRef())
		if siID != uuid.Nil && apiID != uuid.Nil {
			apiInstIndex[apiInstKey{si: siID, api: apiID}] = struct{}{}
		}
	}

	for _, si := range sysInsts {
		id := si.GetInstanceId()
		seenSysInst[id] = struct{}{}
		checkSystemInstance(m, si)
	}

	for _, ci := range compInsts {
		id := ci.GetInstanceId()
		seenCompInst[id] = struct{}{}
		checkComponentInstance(m, ci, apiInstIndex)
	}

	for _, ai := range apiInsts {
		id := ai.GetInstanceId()
		seenAPIInst[id] = struct{}{}
		checkApiInstance(m, ai)
	}
	return
}

func checkApiInstance(m model.Model, ai mdlapi.ApiInstance) {
	id := ai.GetInstanceId()
	name := displayOrID(ai.GetDisplayName(), id)
	if refSystemInstanceID(ai.GetSystemInstance()) == uuid.Nil {
		upsertFinding(m, finding.ApiInstanceSystemInstanceMissing,
			fmt.Sprintf("ApiInstance %q has no SystemInstance; it has no boundary on the C4 deployment diagram.", name),
			[]*common.ResourceRef{{ResourceId: id, ResourceType: events.APIInstanceResource}})
	} else {
		deleteFinding(m, id, finding.ApiInstanceSystemInstanceMissing)
	}
}

func checkSystemInstance(m model.Model, si system.SystemInstance) {
	id := si.GetInstanceId()
	name := displayOrID(si.GetDisplayName(), id)
	cid := refContextID(si.GetContextRef())
	if cid == uuid.Nil {
		upsertFinding(m, finding.SystemInstanceContextMissing,
			fmt.Sprintf("SystemInstance %q has no Context; it cannot be placed on C4 diagrams.", name),
			[]*common.ResourceRef{{ResourceId: id, ResourceType: events.SystemInstanceResource}})
	} else {
		deleteFinding(m, id, finding.SystemInstanceContextMissing)
	}
}

type apiInstKey struct {
	si  uuid.UUID
	api uuid.UUID
}

func checkComponentInstance(m model.Model, ci component.ComponentInstance, apiInstIndex map[apiInstKey]struct{}) {
	id := ci.GetInstanceId()
	name := displayOrID(ci.GetDisplayName(), id)
	siID := refSystemInstanceID(ci.GetSystemInstance())
	typeComp := resolveComponentType(m, ci)
	if typeComp == nil || siID == uuid.Nil {
		deleteFinding(m, id, finding.ApiInstanceMissingForComponentInstance)
		return
	}

	missing := map[uuid.UUID]struct{}{}
	collect := func(refs []mdlapi.ApiRef) {
		for _, r := range refs {
			aid := refAPIID(r)
			if aid == uuid.Nil {
				continue
			}
			if _, ok := apiInstIndex[apiInstKey{si: siID, api: aid}]; !ok {
				missing[aid] = struct{}{}
			}
		}
	}
	collect(typeComp.GetProvides())
	collect(typeComp.GetConsumes())

	if len(missing) == 0 {
		deleteFinding(m, id, finding.ApiInstanceMissingForComponentInstance)
		return
	}

	resources := []*common.ResourceRef{{ResourceId: id, ResourceType: events.ComponentInstanceResource}}
	var names []string
	for aid := range missing {
		resources = append(resources, &common.ResourceRef{ResourceId: aid, ResourceType: events.APIResource})
		if a := m.GetApiById(aid); a != nil {
			names = append(names, displayOrID(a.GetDisplayName(), aid))
		} else {
			names = append(names, aid.String())
		}
	}
	upsertFinding(m, finding.ApiInstanceMissingForComponentInstance,
		fmt.Sprintf("ComponentInstance %q references API(s) %s with no ApiInstance in the same SystemInstance.", name, strings.Join(names, ", ")),
		resources)
}

func checkSystem(m model.Model, s system.System, comps []component.Component) {
	id := s.GetSystemId()
	name := displayOrID(s.GetDisplayName(), id)

	if strings.TrimSpace(s.GetDescription()) == "" {
		upsertFinding(m, finding.DescriptionMissing,
			fmt.Sprintf("System %q has an empty description.", name),
			[]*common.ResourceRef{{ResourceId: id, ResourceType: events.SystemResource}})
	} else {
		deleteFinding(m, id, finding.DescriptionMissing)
	}

	if s.GetAbstract() {
		deleteFinding(m, id, finding.ConcreteSystemHasNoComponents)
		if len(comps) > 0 {
			upsertFinding(m, finding.AbstractSystemHasComponents,
				fmt.Sprintf("Abstract System %q has %d Component(s); abstract systems must be API-only black boxes.", name, len(comps)),
				[]*common.ResourceRef{{ResourceId: id, ResourceType: events.SystemResource}})
		} else {
			deleteFinding(m, id, finding.AbstractSystemHasComponents)
		}
		return
	}

	deleteFinding(m, id, finding.AbstractSystemHasComponents)
	if len(comps) == 0 {
		upsertFinding(m, finding.ConcreteSystemHasNoComponents,
			fmt.Sprintf("System %q has no Components; its C4 Container diagram boundary would be empty.", name),
			[]*common.ResourceRef{{ResourceId: id, ResourceType: events.SystemResource}})
	} else {
		deleteFinding(m, id, finding.ConcreteSystemHasNoComponents)
	}
}

func checkAPI(m model.Model, a mdlapi.API, providers []component.Component) {
	id := a.GetApiId()
	name := displayOrID(a.GetDisplayName(), id)

	if strings.TrimSpace(a.GetDescription()) == "" {
		upsertFinding(m, finding.DescriptionMissing,
			fmt.Sprintf("API %q has an empty description.", name),
			[]*common.ResourceRef{{ResourceId: id, ResourceType: events.APIResource}})
	} else {
		deleteFinding(m, id, finding.DescriptionMissing)
	}

	sysID := refSystemID(a.GetSystem())
	abstract := false
	if sysID != uuid.Nil {
		if s := m.GetSystemById(sysID); s != nil {
			abstract = s.GetAbstract()
		}
	}

	if abstract {
		deleteFinding(m, id, finding.APIHasNoProvider)
		deleteFinding(m, id, finding.APIHasMultipleProviders)
		return
	}

	switch len(providers) {
	case 0:
		deleteFinding(m, id, finding.APIHasMultipleProviders)
		upsertFinding(m, finding.APIHasNoProvider,
			fmt.Sprintf("API %q has no providing Component.", name),
			[]*common.ResourceRef{{ResourceId: id, ResourceType: events.APIResource}})
	case 1:
		deleteFinding(m, id, finding.APIHasNoProvider)
		deleteFinding(m, id, finding.APIHasMultipleProviders)
	default:
		deleteFinding(m, id, finding.APIHasNoProvider)
		resources := []*common.ResourceRef{{ResourceId: id, ResourceType: events.APIResource}}
		for _, p := range providers {
			resources = append(resources, &common.ResourceRef{
				ResourceId:   p.GetComponentId(),
				ResourceType: events.ComponentResource,
			})
		}
		upsertFinding(m, finding.APIHasMultipleProviders,
			fmt.Sprintf("API %q is provided by %d Components; exactly one provider is required.", name, len(providers)),
			resources)
	}
}

func checkComponent(m model.Model, c component.Component) {
	id := c.GetComponentId()
	name := displayOrID(c.GetDisplayName(), id)

	if strings.TrimSpace(c.GetDescription()) == "" {
		upsertFinding(m, finding.DescriptionMissing,
			fmt.Sprintf("Component %q has an empty description.", name),
			[]*common.ResourceRef{{ResourceId: id, ResourceType: events.ComponentResource}})
	} else {
		deleteFinding(m, id, finding.DescriptionMissing)
	}

	compSys := refSystemID(c.GetSystem())
	mismatch := false
	var badAPI uuid.UUID
	for _, pref := range c.GetProvides() {
		aid := refAPIID(pref)
		if aid == uuid.Nil {
			continue
		}
		a := m.GetApiById(aid)
		if a == nil && pref.API != nil {
			a = pref.API
		}
		if a == nil {
			continue
		}
		apiSys := refSystemID(a.GetSystem())
		if apiSys != uuid.Nil && compSys != uuid.Nil && apiSys != compSys {
			mismatch = true
			badAPI = aid
			break
		}
	}
	if mismatch {
		upsertFinding(m, finding.ProvidedAPISystemMismatch,
			fmt.Sprintf("Component %q provides API %s which belongs to a different System.", name, badAPI),
			[]*common.ResourceRef{
				{ResourceId: id, ResourceType: events.ComponentResource},
				{ResourceId: badAPI, ResourceType: events.APIResource},
			})
	} else {
		deleteFinding(m, id, finding.ProvidedAPISystemMismatch)
	}
}

func findingKindOf(m model.Model, f finding.Finding) finding.FindingKind {
	ftid := f.GetFindingTypeId()
	if ft := m.GetFindingTypeById(ftid); ft != nil {
		return finding.FindingKind(ft.GetDisplayName())
	}
	for kind := range docFindingKinds {
		if finding.TypeIDForKind(kind) == ftid {
			return kind
		}
	}
	return ""
}
