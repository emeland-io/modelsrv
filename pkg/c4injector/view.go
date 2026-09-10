package c4injector

import (
	"sort"
	"strings"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/component"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/system"
)

// ContextView is the level-1 System Context model.
type ContextView struct {
	Landscape Landscape
	Roots     []ContextNode
}

// ContextNode is an EmELand Context drawn as a nested Boundary.
type ContextNode struct {
	ID          uuid.UUID
	DisplayName string
	Description string
	TypeName    string
	Children    []ContextNode
}

// ContainerView is the level-2 Container model (types).
// EmELand Components are C4 Containers, grouped in a System_Boundary per
// concrete System; abstract Systems stay black boxes as System_Ext.
type ContainerView struct {
	Systems   []SystemBoundary
	Externals []ExternalNode
	Edges     []RelEdge
}

// SystemBoundary is a concrete System drawn as a C4 System_Boundary around its Containers.
type SystemBoundary struct {
	ID          uuid.UUID
	DisplayName string
	Description string
	Containers  []C4Container
}

// C4Container is an EmELand Component drawn as a C4 Container. Technology carries
// the provided API types: a Component's runtime boundary is its API surface.
type C4Container struct {
	ID          uuid.UUID
	DisplayName string
	Description string
	Technology  string
	// InstanceCount is how many ComponentInstances resolve to this Component.
	// Surfaced on level 2 so an undeployed Component is visible without
	// putting instances themselves on a type-level diagram.
	InstanceCount int
}

// RelEdge is a labelled relationship between two diagram nodes.
type RelEdge struct {
	FromID   uuid.UUID
	ToID     uuid.UUID
	FromKind string // "system" | "container"
	ToKind   string
	Label    string
	Tech     string
}

// DeploymentView is the deployment diagram model (instances).
type DeploymentView struct {
	Contexts []InstanceContext
	Edges    []RelEdge
}

// ExternalNode is an abstract/external System linked via consumes.
type ExternalNode struct {
	ID          uuid.UUID
	DisplayName string
	Description string
	Kind        string // "system"
	Abstract    bool
}

// InstanceContext groups SystemInstances under an EmELand Context.
type InstanceContext struct {
	ID              uuid.UUID
	DisplayName     string
	TypeName        string
	SystemInstances []InstanceSystem
	// LooseContainers are ComponentInstances whose SystemInstance is unset or
	// does not resolve to a SystemInstance in the model.
	LooseContainers []InstanceContainer
	// LooseEndpoints are ApiInstances whose SystemInstance is unset or does not
	// resolve. They would otherwise appear on no diagram at all.
	LooseEndpoints []InstanceEndpoint
}

// InstanceSystem is a SystemInstance drawn as a System_Boundary.
type InstanceSystem struct {
	ID          uuid.UUID
	DisplayName string
	SystemName  string // type System display name
	Containers  []InstanceContainer
	Endpoints   []InstanceEndpoint
}

// InstanceContainer is a ComponentInstance drawn as a C4 Container on the deployment diagram.
type InstanceContainer struct {
	ID          uuid.UUID
	DisplayName string
	Description string
}

// InstanceEndpoint is an ApiInstance drawn as an "endpoint"-tagged C4 Container.
// Technology and Address come from the emeland.io/endpoint.* annotations, which
// are the only record in the model of where a deployed API actually answers.
type InstanceEndpoint struct {
	ID          uuid.UUID
	DisplayName string
	APIName     string // type API display name, empty when the ref does not resolve
	Technology  string // protocol, with port when annotated
	Address     string // host and path when annotated
}

// BuildContextView builds the level-1 view from EmELand Context resources.
func BuildContextView(m model.Model, l Landscape) (ContextView, error) {
	if l.Name == "" {
		l = DefaultLandscape()
	}
	contexts, err := m.GetContexts()
	if err != nil {
		return ContextView{}, err
	}

	byID := make(map[uuid.UUID]mdlctx.Context, len(contexts))
	children := make(map[uuid.UUID][]uuid.UUID)
	var roots []uuid.UUID
	for _, c := range contexts {
		id := c.GetContextId()
		byID[id] = c
		pid := c.GetParentId()
		if pid == uuid.Nil {
			roots = append(roots, id)
		} else {
			children[pid] = append(children[pid], id)
		}
	}
	sort.Slice(roots, func(i, j int) bool {
		return contextSortKey(byID[roots[i]]) < contextSortKey(byID[roots[j]])
	})
	for pid := range children {
		sort.Slice(children[pid], func(i, j int) bool {
			return contextSortKey(byID[children[pid][i]]) < contextSortKey(byID[children[pid][j]])
		})
	}

	var build func(id uuid.UUID) ContextNode
	build = func(id uuid.UUID) ContextNode {
		c := byID[id]
		n := ContextNode{
			ID:          id,
			DisplayName: displayOrID(c.GetDisplayName(), id),
			Description: strings.TrimSpace(c.GetDescription()),
			TypeName:    contextTypeName(m, c),
		}
		for _, cid := range children[id] {
			n.Children = append(n.Children, build(cid))
		}
		return n
	}

	v := ContextView{Landscape: l}
	for _, rid := range roots {
		v.Roots = append(v.Roots, build(rid))
	}
	return v, nil
}

func contextSortKey(c mdlctx.Context) string {
	if c == nil {
		return ""
	}
	return displayOrID(c.GetDisplayName(), c.GetContextId()) + "|" + c.GetContextId().String()
}

func contextTypeName(m model.Model, c mdlctx.Context) string {
	tid := c.GetContextTypeId()
	if tid == uuid.Nil {
		return ""
	}
	if ct, err := c.GetContextType(); err == nil && ct != nil {
		return displayOrID(ct.GetDisplayName(), tid)
	}
	if ct := m.GetContextTypeById(tid); ct != nil {
		return displayOrID(ct.GetDisplayName(), tid)
	}
	return tid.String()
}

// BuildContainerView builds the level-2 view from System / Component / API types.
// Each concrete System becomes a C4 System_Boundary holding one Container per
// EmELand Component: Components only ever talk over APIs (OpenAPI / GraphQL /
// gRPC), so each is its own runtime boundary. Abstract Systems stay black boxes.
func BuildContainerView(m model.Model) (ContainerView, error) {
	comps, err := m.GetComponents()
	if err != nil {
		return ContainerView{}, err
	}
	apis, err := m.GetApis()
	if err != nil {
		return ContainerView{}, err
	}
	systems, err := m.GetSystems()
	if err != nil {
		return ContainerView{}, err
	}
	compInsts, err := m.GetComponentInstances()
	if err != nil {
		return ContainerView{}, err
	}

	instanceCount := map[uuid.UUID]int{}
	for _, ci := range compInsts {
		if ref := ci.GetComponentRef(); ref != nil {
			cid := ref.ComponentId
			if ref.Component != nil {
				cid = ref.Component.GetComponentId()
			}
			if cid != uuid.Nil {
				instanceCount[cid]++
			}
		}
	}

	sysByID := make(map[uuid.UUID]system.System, len(systems))
	for _, s := range systems {
		sysByID[s.GetSystemId()] = s
	}
	apiByID := make(map[uuid.UUID]mdlapi.API, len(apis))
	for _, a := range apis {
		apiByID[a.GetApiId()] = a
	}
	providerComp := map[uuid.UUID][]component.Component{}
	compsBySystem := map[uuid.UUID][]component.Component{}
	selectedIDs := map[uuid.UUID]struct{}{}

	for _, c := range comps {
		sid := refSystemID(c.GetSystem())
		if s, ok := sysByID[sid]; ok && s.GetAbstract() {
			continue // omit Components of abstract Systems
		}
		compsBySystem[sid] = append(compsBySystem[sid], c)
		selectedIDs[c.GetComponentId()] = struct{}{}
		for _, pref := range c.GetProvides() {
			if aid := refAPIID(pref); aid != uuid.Nil {
				providerComp[aid] = append(providerComp[aid], c)
			}
		}
	}

	sortedSys := append([]system.System(nil), systems...)
	sortSystems(sortedSys)

	v := ContainerView{}
	for _, s := range sortedSys {
		if s.GetAbstract() {
			continue
		}
		sid := s.GetSystemId()
		boundary := SystemBoundary{
			ID:          sid,
			DisplayName: displayOrID(s.GetDisplayName(), sid),
			Description: strings.TrimSpace(s.GetDescription()),
		}
		for _, c := range sortedComponents(compsBySystem[sid]) {
			boundary.Containers = append(boundary.Containers, C4Container{
				ID:            c.GetComponentId(),
				DisplayName:   displayOrID(c.GetDisplayName(), c.GetComponentId()),
				Description:   strings.TrimSpace(c.GetDescription()),
				Technology:    providedAPITypes(c, apiByID),
				InstanceCount: instanceCount[c.GetComponentId()],
			})
		}
		v.Systems = append(v.Systems, boundary)
	}

	externals := map[uuid.UUID]ExternalNode{}
	edgeKeys := map[string]struct{}{}
	var edges []RelEdge
	addEdge := func(fromID, toID uuid.UUID, toKind, label, tech string) {
		key := fromID.String() + "|" + toID.String() + "|" + label + "|" + toKind
		if _, ok := edgeKeys[key]; ok {
			return
		}
		edgeKeys[key] = struct{}{}
		edges = append(edges, RelEdge{
			FromID: fromID, ToID: toID,
			FromKind: "container", ToKind: toKind,
			Label: label, Tech: tech,
		})
	}

	for _, c := range comps {
		fromID := c.GetComponentId()
		if _, ok := selectedIDs[fromID]; !ok {
			continue
		}
		for _, cref := range c.GetConsumes() {
			aid := refAPIID(cref)
			if aid == uuid.Nil {
				continue
			}
			a, ok := apiByID[aid]
			if !ok {
				if cref.API != nil {
					a = cref.API
				} else {
					continue
				}
			}
			label := displayOrID(a.GetDisplayName(), aid)
			tech := a.GetType().String()

			var toID uuid.UUID
			toKind := "system"
			for _, p := range providerComp[aid] {
				pid := p.GetComponentId()
				if pid == fromID {
					continue
				}
				if _, in := selectedIDs[pid]; !in {
					continue
				}
				toID = pid
				toKind = "container"
				break
			}
			if toID == uuid.Nil {
				sid := refSystemID(a.GetSystem())
				if sid == uuid.Nil {
					continue
				}
				toID = sid
				toKind = "system"
				if s, ok := sysByID[sid]; ok {
					externals[sid] = ExternalNode{
						ID:          sid,
						DisplayName: displayOrID(s.GetDisplayName(), sid),
						Description: strings.TrimSpace(s.GetDescription()),
						Kind:        "system",
						Abstract:    s.GetAbstract(),
					}
				} else {
					externals[sid] = ExternalNode{
						ID: sid, DisplayName: sid.String(), Kind: "system", Abstract: true,
					}
				}
			}
			addEdge(fromID, toID, toKind, label, tech)
		}
	}

	extList := make([]ExternalNode, 0, len(externals))
	for _, e := range externals {
		extList = append(extList, e)
	}
	sort.Slice(extList, func(i, j int) bool {
		if extList[i].DisplayName != extList[j].DisplayName {
			return extList[i].DisplayName < extList[j].DisplayName
		}
		return extList[i].ID.String() < extList[j].ID.String()
	})
	v.Externals = extList
	sortRelEdges(edges)
	v.Edges = edges
	return v, nil
}

// BuildDeploymentView builds the deployment diagram from instance resources.
func BuildDeploymentView(m model.Model) (DeploymentView, error) {
	sysInsts, err := m.GetSystemInstances()
	if err != nil {
		return DeploymentView{}, err
	}
	compInsts, err := m.GetComponentInstances()
	if err != nil {
		return DeploymentView{}, err
	}
	apiInsts, err := m.GetApiInstances()
	if err != nil {
		return DeploymentView{}, err
	}
	contexts, err := m.GetContexts()
	if err != nil {
		return DeploymentView{}, err
	}

	ctxByID := make(map[uuid.UUID]mdlctx.Context, len(contexts))
	for _, c := range contexts {
		ctxByID[c.GetContextId()] = c
	}

	// Instances may reference a SystemInstance that is not (yet) in the model.
	// Such references are indistinguishable from a typo, so the instance is
	// grouped as loose rather than dropped — see the loose buckets below.
	knownSI := make(map[uuid.UUID]struct{}, len(sysInsts))
	for _, si := range sysInsts {
		knownSI[si.GetInstanceId()] = struct{}{}
	}
	placed := func(ref *system.SystemInstanceRef) (uuid.UUID, bool) {
		id := refSystemInstanceID(ref)
		if id == uuid.Nil {
			return uuid.Nil, false
		}
		_, ok := knownSI[id]
		return id, ok
	}

	// apiInstBy (systemInstanceID, apiTypeID) -> ApiInstance, for edge projection.
	type key struct {
		si  uuid.UUID
		api uuid.UUID
	}
	apiInstIndex := map[key]mdlapi.ApiInstance{}
	apiInstsBySI := map[uuid.UUID][]mdlapi.ApiInstance{}
	var looseAIs []mdlapi.ApiInstance
	for _, ai := range apiInsts {
		siID, ok := placed(ai.GetSystemInstance())
		if !ok {
			looseAIs = append(looseAIs, ai)
			continue
		}
		apiInstsBySI[siID] = append(apiInstsBySI[siID], ai)
		// Only a resolvable API ref can carry an edge, but the ApiInstance is
		// rendered either way.
		if apiID := refAPIIDFromPtr(ai.GetApiRef()); apiID != uuid.Nil {
			apiInstIndex[key{si: siID, api: apiID}] = ai
		}
	}

	compInstsBySI := map[uuid.UUID][]component.ComponentInstance{}
	var looseCIs []component.ComponentInstance
	for _, ci := range compInsts {
		siID, ok := placed(ci.GetSystemInstance())
		if !ok {
			looseCIs = append(looseCIs, ci)
			continue
		}
		compInstsBySI[siID] = append(compInstsBySI[siID], ci)
	}

	sysInstsByCtx := map[uuid.UUID][]system.SystemInstance{}
	var unscoped []system.SystemInstance
	for _, si := range sysInsts {
		cid := refContextID(si.GetContextRef())
		if cid == uuid.Nil {
			unscoped = append(unscoped, si)
			continue
		}
		sysInstsByCtx[cid] = append(sysInstsByCtx[cid], si)
	}

	var edges []RelEdge
	edgeKeys := map[string]struct{}{}
	addEdge := func(from, to uuid.UUID, label, tech string) {
		k := from.String() + "|" + to.String() + "|" + label
		if _, ok := edgeKeys[k]; ok {
			return
		}
		edgeKeys[k] = struct{}{}
		edges = append(edges, RelEdge{
			FromID: from, ToID: to,
			FromKind: "container", ToKind: "container",
			Label: label, Tech: tech,
		})
	}

	buildSI := func(si system.SystemInstance) InstanceSystem {
		is := InstanceSystem{
			ID:          si.GetInstanceId(),
			DisplayName: displayOrID(si.GetDisplayName(), si.GetInstanceId()),
		}
		if ref := si.GetSystemRef(); ref != nil {
			if ref.System != nil {
				is.SystemName = displayOrID(ref.System.GetDisplayName(), ref.System.GetSystemId())
			} else if ref.SystemId != uuid.Nil {
				if s := m.GetSystemById(ref.SystemId); s != nil {
					is.SystemName = displayOrID(s.GetDisplayName(), ref.SystemId)
				} else {
					is.SystemName = ref.SystemId.String()
				}
			}
		}

		is.Endpoints = buildEndpoints(m, apiInstsBySI[si.GetInstanceId()])

		cis := append([]component.ComponentInstance(nil), compInstsBySI[si.GetInstanceId()]...)
		sort.Slice(cis, func(i, j int) bool {
			ni := displayOrID(cis[i].GetDisplayName(), cis[i].GetInstanceId())
			nj := displayOrID(cis[j].GetDisplayName(), cis[j].GetInstanceId())
			if ni != nj {
				return ni < nj
			}
			return cis[i].GetInstanceId().String() < cis[j].GetInstanceId().String()
		})

		for _, ci := range cis {
			desc := ""
			var typeComp component.Component
			if cref := ci.GetComponentRef(); cref != nil {
				cid := cref.ComponentId
				if cref.Component != nil {
					cid = cref.Component.GetComponentId()
					typeComp = cref.Component
				}
				if typeComp == nil {
					typeComp = m.GetComponentById(cid)
				}
				if typeComp != nil {
					desc = strings.TrimSpace(typeComp.GetDescription())
				}
			}
			is.Containers = append(is.Containers, InstanceContainer{
				ID:          ci.GetInstanceId(),
				DisplayName: displayOrID(ci.GetDisplayName(), ci.GetInstanceId()),
				Description: desc,
			})

			if typeComp == nil {
				continue
			}
			siID := si.GetInstanceId()
			// Project type-level provides/consumes onto instances in the same SystemInstance.
			// Consumer CI --uses--> Provider CI (via matching ApiInstances).
			for _, pref := range typeComp.GetProvides() {
				apiTypeID := refAPIID(pref)
				if apiTypeID == uuid.Nil {
					continue
				}
				if _, ok := apiInstIndex[key{si: siID, api: apiTypeID}]; !ok {
					continue
				}
				// Find other CIs in same SI whose type consumes this API.
				for _, other := range cis {
					if other.GetInstanceId() == ci.GetInstanceId() {
						continue
					}
					otherComp := resolveComponentType(m, other)
					if otherComp == nil {
						continue
					}
					for _, cref := range otherComp.GetConsumes() {
						if refAPIID(cref) != apiTypeID {
							continue
						}
						a := m.GetApiById(apiTypeID)
						label, tech := apiTypeID.String(), ""
						if a != nil {
							label = displayOrID(a.GetDisplayName(), apiTypeID)
							tech = a.GetType().String()
						}
						addEdge(other.GetInstanceId(), ci.GetInstanceId(), label, tech)
					}
				}
			}
		}
		return is
	}

	v := DeploymentView{}
	// Contexts that have system instances, plus empty contexts? Only those with instances.
	ctxIDs := make([]uuid.UUID, 0, len(sysInstsByCtx))
	for cid := range sysInstsByCtx {
		ctxIDs = append(ctxIDs, cid)
	}
	sort.Slice(ctxIDs, func(i, j int) bool {
		ni, nj := ctxIDs[i].String(), ctxIDs[j].String()
		if c, ok := ctxByID[ctxIDs[i]]; ok {
			ni = displayOrID(c.GetDisplayName(), ctxIDs[i])
		}
		if c, ok := ctxByID[ctxIDs[j]]; ok {
			nj = displayOrID(c.GetDisplayName(), ctxIDs[j])
		}
		if ni != nj {
			return ni < nj
		}
		return ctxIDs[i].String() < ctxIDs[j].String()
	})

	for _, cid := range ctxIDs {
		ic := InstanceContext{ID: cid, DisplayName: cid.String()}
		if c, ok := ctxByID[cid]; ok {
			ic.DisplayName = displayOrID(c.GetDisplayName(), cid)
			ic.TypeName = contextTypeName(m, c)
		}
		sis := append([]system.SystemInstance(nil), sysInstsByCtx[cid]...)
		sort.Slice(sis, func(i, j int) bool {
			ni := displayOrID(sis[i].GetDisplayName(), sis[i].GetInstanceId())
			nj := displayOrID(sis[j].GetDisplayName(), sis[j].GetInstanceId())
			if ni != nj {
				return ni < nj
			}
			return sis[i].GetInstanceId().String() < sis[j].GetInstanceId().String()
		})
		for _, si := range sis {
			ic.SystemInstances = append(ic.SystemInstances, buildSI(si))
		}
		v.Contexts = append(v.Contexts, ic)
	}

	// Unscoped system instances (no context) and ComponentInstances with no
	// SystemInstance go under a synthetic "unscoped" group (uuid.Nil).
	if len(unscoped) > 0 || len(looseCIs) > 0 || len(looseAIs) > 0 {
		ic := InstanceContext{ID: uuid.Nil, DisplayName: "(no Context)", TypeName: ""}
		sort.Slice(unscoped, func(i, j int) bool {
			ni := displayOrID(unscoped[i].GetDisplayName(), unscoped[i].GetInstanceId())
			nj := displayOrID(unscoped[j].GetDisplayName(), unscoped[j].GetInstanceId())
			if ni != nj {
				return ni < nj
			}
			return unscoped[i].GetInstanceId().String() < unscoped[j].GetInstanceId().String()
		})
		for _, si := range unscoped {
			ic.SystemInstances = append(ic.SystemInstances, buildSI(si))
		}
		sort.Slice(looseCIs, func(i, j int) bool {
			ni := displayOrID(looseCIs[i].GetDisplayName(), looseCIs[i].GetInstanceId())
			nj := displayOrID(looseCIs[j].GetDisplayName(), looseCIs[j].GetInstanceId())
			if ni != nj {
				return ni < nj
			}
			return looseCIs[i].GetInstanceId().String() < looseCIs[j].GetInstanceId().String()
		})
		for _, ci := range looseCIs {
			desc := ""
			if typeComp := resolveComponentType(m, ci); typeComp != nil {
				desc = strings.TrimSpace(typeComp.GetDescription())
			}
			ic.LooseContainers = append(ic.LooseContainers, InstanceContainer{
				ID:          ci.GetInstanceId(),
				DisplayName: displayOrID(ci.GetDisplayName(), ci.GetInstanceId()),
				Description: desc,
			})
		}
		ic.LooseEndpoints = buildEndpoints(m, looseAIs)
		v.Contexts = append(v.Contexts, ic)
	}

	sortRelEdges(edges)
	v.Edges = edges
	return v, nil
}

// Endpoint annotation keys, mirroring pkg/endpointprobe.
const (
	annEndpointProtocol = "emeland.io/endpoint.protocol"
	annEndpointHost     = "emeland.io/endpoint.host"
	annEndpointPort     = "emeland.io/endpoint.port"
	annEndpointPath     = "emeland.io/endpoint.path"
)

// buildEndpoints turns ApiInstances into sorted diagram nodes. An ApiInstance
// with no endpoint annotations still renders, with an empty address.
func buildEndpoints(m model.Model, ais []mdlapi.ApiInstance) []InstanceEndpoint {
	if len(ais) == 0 {
		return nil
	}
	sorted := append([]mdlapi.ApiInstance(nil), ais...)
	sort.Slice(sorted, func(i, j int) bool {
		ni := displayOrID(sorted[i].GetDisplayName(), sorted[i].GetInstanceId())
		nj := displayOrID(sorted[j].GetDisplayName(), sorted[j].GetInstanceId())
		if ni != nj {
			return ni < nj
		}
		return sorted[i].GetInstanceId().String() < sorted[j].GetInstanceId().String()
	})

	out := make([]InstanceEndpoint, 0, len(sorted))
	for _, ai := range sorted {
		e := InstanceEndpoint{
			ID:          ai.GetInstanceId(),
			DisplayName: displayOrID(ai.GetDisplayName(), ai.GetInstanceId()),
		}
		if aid := refAPIIDFromPtr(ai.GetApiRef()); aid != uuid.Nil {
			if a := m.GetApiById(aid); a != nil {
				e.APIName = displayOrID(a.GetDisplayName(), aid)
			} else {
				e.APIName = aid.String()
			}
		}
		ann := ai.GetAnnotations()
		if ann != nil {
			e.Technology = strings.TrimSpace(ann.GetValue(annEndpointProtocol))
			if port := strings.TrimSpace(ann.GetValue(annEndpointPort)); port != "" {
				if e.Technology == "" {
					e.Technology = "port " + port
				} else {
					e.Technology += ":" + port
				}
			}
			e.Address = strings.TrimSpace(ann.GetValue(annEndpointHost)) +
				strings.TrimSpace(ann.GetValue(annEndpointPath))
		}
		out = append(out, e)
	}
	return out
}

func resolveComponentType(m model.Model, ci component.ComponentInstance) component.Component {
	ref := ci.GetComponentRef()
	if ref == nil {
		return nil
	}
	if ref.Component != nil {
		return ref.Component
	}
	if ref.ComponentId != uuid.Nil {
		return m.GetComponentById(ref.ComponentId)
	}
	return nil
}

func sortedComponents(comps []component.Component) []component.Component {
	sorted := append([]component.Component(nil), comps...)
	sort.Slice(sorted, func(i, j int) bool {
		ni := displayOrID(sorted[i].GetDisplayName(), sorted[i].GetComponentId())
		nj := displayOrID(sorted[j].GetDisplayName(), sorted[j].GetComponentId())
		if ni != nj {
			return ni < nj
		}
		return sorted[i].GetComponentId().String() < sorted[j].GetComponentId().String()
	})
	return sorted
}

func providedAPITypes(c component.Component, apiByID map[uuid.UUID]mdlapi.API) string {
	var parts []string
	for _, pref := range c.GetProvides() {
		aid := refAPIID(pref)
		if a, ok := apiByID[aid]; ok {
			parts = append(parts, a.GetType().String())
		}
	}
	sort.Strings(parts)
	// dedupe
	out := parts[:0]
	var prev string
	for _, p := range parts {
		if p == prev {
			continue
		}
		out = append(out, p)
		prev = p
	}
	return strings.Join(out, ", ")
}

func sortSystems(systems []system.System) {
	sort.Slice(systems, func(i, j int) bool {
		ni := displayOrID(systems[i].GetDisplayName(), systems[i].GetSystemId())
		nj := displayOrID(systems[j].GetDisplayName(), systems[j].GetSystemId())
		if ni != nj {
			return ni < nj
		}
		return systems[i].GetSystemId().String() < systems[j].GetSystemId().String()
	})
}

func sortRelEdges(edges []RelEdge) {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].FromID != edges[j].FromID {
			return edges[i].FromID.String() < edges[j].FromID.String()
		}
		if edges[i].ToID != edges[j].ToID {
			return edges[i].ToID.String() < edges[j].ToID.String()
		}
		return edges[i].Label < edges[j].Label
	})
}

func refSystemID(ref *system.SystemRef) uuid.UUID {
	if ref == nil {
		return uuid.Nil
	}
	if ref.System != nil {
		return ref.System.GetSystemId()
	}
	return ref.SystemId
}

func refAPIID(ref mdlapi.ApiRef) uuid.UUID {
	if ref.API != nil {
		return ref.API.GetApiId()
	}
	return ref.ApiID
}

func refAPIIDFromPtr(ref *mdlapi.ApiRef) uuid.UUID {
	if ref == nil {
		return uuid.Nil
	}
	return refAPIID(*ref)
}

func refSystemInstanceID(ref *system.SystemInstanceRef) uuid.UUID {
	if ref == nil {
		return uuid.Nil
	}
	if ref.SystemInstance != nil {
		return ref.SystemInstance.GetInstanceId()
	}
	return ref.InstanceId
}

func refContextID(ref *mdlctx.ContextRef) uuid.UUID {
	if ref == nil {
		return uuid.Nil
	}
	if ref.Context != nil {
		return ref.Context.GetContextId()
	}
	return ref.ContextId
}

func displayOrID(name string, id uuid.UUID) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return id.String()
}
