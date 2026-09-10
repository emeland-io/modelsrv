package c4injector

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/component"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/system"
)

func TestBuildContextView_Nesting(t *testing.T) {
	m := newTestModel(t)
	ct := mdlctx.NewContextType(uuid.New())
	ct.SetDisplayName("Environment")
	require.NoError(t, m.AddContextType(ct))

	parent := mdlctx.NewContext(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	parent.SetDisplayName("prod")
	parent.SetContextTypeByRef(ct)
	require.NoError(t, m.AddContext(parent))

	child := mdlctx.NewContext(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	child.SetDisplayName("eu-west")
	child.SetContextTypeByRef(ct)
	child.SetParentByRef(parent)
	require.NoError(t, m.AddContext(child))

	v, err := BuildContextView(m, Landscape{Name: "EL", Description: "summary"})
	require.NoError(t, err)
	require.Equal(t, "EL", v.Landscape.Name)
	require.Len(t, v.Roots, 1)
	require.Equal(t, "prod", v.Roots[0].DisplayName)
	require.Equal(t, "Environment", v.Roots[0].TypeName)
	require.Len(t, v.Roots[0].Children, 1)
	require.Equal(t, "eu-west", v.Roots[0].Children[0].DisplayName)
}

func TestBuildContainerView_AbstractAndEdges(t *testing.T) {
	m := newTestModel(t)

	sysA := system.NewSystem(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	sysA.SetDisplayName("Alpha")
	sysA.SetDescription("alpha")
	require.NoError(t, m.AddSystem(sysA))

	sysB := system.NewSystem(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	sysB.SetDisplayName("Beta")
	sysB.SetDescription("beta")
	sysB.SetAbstract(true)
	require.NoError(t, m.AddSystem(sysB))

	apiB := mdlapi.NewAPI(uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"))
	apiB.SetDisplayName("BetaAPI")
	apiB.SetDescription("beta api")
	apiB.SetType(mdlapi.OpenAPI)
	apiB.SetSystemByRef(sysB)
	require.NoError(t, m.AddApi(apiB))

	compA := component.NewComponent(uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"))
	compA.SetDisplayName("AlphaWorker")
	compA.SetDescription("worker")
	compA.SetSystem(&system.SystemRef{System: sysA, SystemId: sysA.GetSystemId()})
	compA.SetConsumes([]mdlapi.ApiRef{{API: apiB, ApiID: apiB.GetApiId()}})
	require.NoError(t, m.AddComponent(compA))

	v, err := BuildContainerView(m)
	require.NoError(t, err)
	// Only the concrete System gets a boundary; the abstract one stays a black box.
	require.Len(t, v.Systems, 1)
	require.Equal(t, sysA.GetSystemId(), v.Systems[0].ID)
	require.Len(t, v.Systems[0].Containers, 1)
	require.Equal(t, "AlphaWorker", v.Systems[0].Containers[0].DisplayName)
	require.Len(t, v.Externals, 1)
	require.Equal(t, sysB.GetSystemId(), v.Externals[0].ID)
	require.True(t, v.Externals[0].Abstract)

	require.Len(t, v.Edges, 1)
	require.Equal(t, compA.GetComponentId(), v.Edges[0].FromID)
	require.Equal(t, sysB.GetSystemId(), v.Edges[0].ToID)
	require.Equal(t, "container", v.Edges[0].FromKind)
	require.Equal(t, "system", v.Edges[0].ToKind)
	require.Equal(t, "BetaAPI", v.Edges[0].Label)
}

// Containers inside one System still talk over an API, so the edge belongs on level 2.
func TestBuildContainerView_SameSystemEdgeKept(t *testing.T) {
	m := newTestModel(t)
	sys := system.NewSystem(uuid.New())
	sys.SetDisplayName("App")
	sys.SetDescription("app")
	require.NoError(t, m.AddSystem(sys))

	api := mdlapi.NewAPI(uuid.New())
	api.SetDisplayName("InternalAPI")
	api.SetDescription("internal")
	api.SetSystemByRef(sys)
	require.NoError(t, m.AddApi(api))

	provider := component.NewComponent(uuid.New())
	provider.SetDisplayName("Provider")
	provider.SetDescription("provides")
	provider.SetSystem(&system.SystemRef{System: sys, SystemId: sys.GetSystemId()})
	provider.SetProvides([]mdlapi.ApiRef{{API: api, ApiID: api.GetApiId()}})
	require.NoError(t, m.AddComponent(provider))

	consumer := component.NewComponent(uuid.New())
	consumer.SetDisplayName("Consumer")
	consumer.SetDescription("consumes")
	consumer.SetSystem(&system.SystemRef{System: sys, SystemId: sys.GetSystemId()})
	consumer.SetConsumes([]mdlapi.ApiRef{{API: api, ApiID: api.GetApiId()}})
	require.NoError(t, m.AddComponent(consumer))

	v, err := BuildContainerView(m)
	require.NoError(t, err)
	require.Len(t, v.Systems, 1)
	require.Len(t, v.Systems[0].Containers, 2)
	require.Len(t, v.Edges, 1)
	require.Equal(t, consumer.GetComponentId(), v.Edges[0].FromID)
	require.Equal(t, provider.GetComponentId(), v.Edges[0].ToID)
	require.Equal(t, "container", v.Edges[0].FromKind)
	require.Equal(t, "container", v.Edges[0].ToKind)
	require.Empty(t, v.Externals)
}

func TestBuildDeploymentView_ProjectsApiInstance(t *testing.T) {
	m := newTestModel(t)
	sys := system.NewSystem(uuid.New())
	sys.SetDisplayName("S")
	sys.SetDescription("d")
	require.NoError(t, m.AddSystem(sys))

	api := mdlapi.NewAPI(uuid.New())
	api.SetDisplayName("A")
	api.SetDescription("api")
	api.SetSystemByRef(sys)
	require.NoError(t, m.AddApi(api))

	provider := component.NewComponent(uuid.New())
	provider.SetDisplayName("P")
	provider.SetDescription("p")
	provider.SetSystem(&system.SystemRef{System: sys, SystemId: sys.GetSystemId()})
	provider.SetProvides([]mdlapi.ApiRef{{API: api, ApiID: api.GetApiId()}})
	require.NoError(t, m.AddComponent(provider))

	consumer := component.NewComponent(uuid.New())
	consumer.SetDisplayName("C")
	consumer.SetDescription("c")
	consumer.SetSystem(&system.SystemRef{System: sys, SystemId: sys.GetSystemId()})
	consumer.SetConsumes([]mdlapi.ApiRef{{API: api, ApiID: api.GetApiId()}})
	require.NoError(t, m.AddComponent(consumer))

	ct := mdlctx.NewContextType(uuid.New())
	ct.SetDisplayName("Env")
	require.NoError(t, m.AddContextType(ct))
	ctx := mdlctx.NewContext(uuid.New())
	ctx.SetDisplayName("prod")
	ctx.SetContextTypeByRef(ct)
	require.NoError(t, m.AddContext(ctx))

	si := system.NewSystemInstance(uuid.New())
	si.SetDisplayName("si")
	si.SetSystemRef(&system.SystemRef{System: sys, SystemId: sys.GetSystemId()})
	si.SetContextRef(&mdlctx.ContextRef{Context: ctx, ContextId: ctx.GetContextId()})
	require.NoError(t, m.AddSystemInstance(si))

	pi := component.NewComponentInstance(uuid.New())
	pi.SetDisplayName("pi")
	pi.SetComponentRef(&component.ComponentRef{Component: provider, ComponentId: provider.GetComponentId()})
	pi.SetSystemInstance(&system.SystemInstanceRef{SystemInstance: si, InstanceId: si.GetInstanceId()})
	require.NoError(t, m.AddComponentInstance(pi))

	ci := component.NewComponentInstance(uuid.New())
	ci.SetDisplayName("ci")
	ci.SetComponentRef(&component.ComponentRef{Component: consumer, ComponentId: consumer.GetComponentId()})
	ci.SetSystemInstance(&system.SystemInstanceRef{SystemInstance: si, InstanceId: si.GetInstanceId()})
	require.NoError(t, m.AddComponentInstance(ci))

	v, err := BuildDeploymentView(m)
	require.NoError(t, err)
	require.Empty(t, v.Edges)

	ai := mdlapi.NewApiInstance(uuid.New())
	ai.SetDisplayName("ai")
	ai.SetApiRefByRef(api)
	ai.SetSystemInstanceByRef(si)
	require.NoError(t, m.AddApiInstance(ai))

	v, err = BuildDeploymentView(m)
	require.NoError(t, err)
	require.Len(t, v.Contexts, 1)
	require.Len(t, v.Contexts[0].SystemInstances, 1)
	require.Len(t, v.Contexts[0].SystemInstances[0].Containers, 2)
	require.Len(t, v.Edges, 1)
	require.Equal(t, ci.GetInstanceId(), v.Edges[0].FromID)
	require.Equal(t, pi.GetInstanceId(), v.Edges[0].ToID)
	require.Equal(t, "A", v.Edges[0].Label)
}

func TestBuildContainerView_ComponentsAsC4Containers(t *testing.T) {
	m := newTestModel(t)
	sys := system.NewSystem(uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	sys.SetDisplayName("S")
	sys.SetDescription("d")
	require.NoError(t, m.AddSystem(sys))

	api := mdlapi.NewAPI(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	api.SetDisplayName("OrderAPI")
	api.SetDescription("orders")
	api.SetType(mdlapi.OpenAPI)
	api.SetSystemByRef(sys)
	require.NoError(t, m.AddApi(api))

	extSys := system.NewSystem(uuid.MustParse("22222222-2222-2222-2222-222222222222"))
	extSys.SetDisplayName("Beta")
	extSys.SetDescription("external")
	extSys.SetAbstract(true)
	require.NoError(t, m.AddSystem(extSys))
	extAPI := mdlapi.NewAPI(uuid.New())
	extAPI.SetDisplayName("BetaAPI")
	extAPI.SetDescription("beta")
	extAPI.SetSystemByRef(extSys)
	require.NoError(t, m.AddApi(extAPI))

	provider := component.NewComponent(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	provider.SetDisplayName("AlphaWorker")
	provider.SetDescription("worker")
	provider.SetSystem(&system.SystemRef{System: sys, SystemId: sys.GetSystemId()})
	provider.SetProvides([]mdlapi.ApiRef{{API: api, ApiID: api.GetApiId()}})
	provider.SetConsumes([]mdlapi.ApiRef{{API: extAPI, ApiID: extAPI.GetApiId()}})
	require.NoError(t, m.AddComponent(provider))

	other := component.NewComponent(uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"))
	other.SetDisplayName("Other")
	other.SetDescription("other")
	other.SetSystem(&system.SystemRef{System: sys, SystemId: sys.GetSystemId()})
	require.NoError(t, m.AddComponent(other))

	// Component under abstract System must be omitted from L2.
	absComp := component.NewComponent(uuid.New())
	absComp.SetDisplayName("Hidden")
	absComp.SetDescription("hidden")
	absComp.SetSystem(&system.SystemRef{System: extSys, SystemId: extSys.GetSystemId()})
	require.NoError(t, m.AddComponent(absComp))

	v, err := BuildContainerView(m)
	require.NoError(t, err)
	require.Len(t, v.Systems, 1)
	require.Equal(t, sys.GetSystemId(), v.Systems[0].ID)
	require.Len(t, v.Systems[0].Containers, 2)
	// The provided API type lands in the C4 Container technology slot.
	require.Equal(t, "OpenAPI", v.Systems[0].Containers[0].Technology)
	require.Len(t, v.Edges, 1)
	require.Equal(t, provider.GetComponentId(), v.Edges[0].FromID)
	require.Equal(t, extSys.GetSystemId(), v.Edges[0].ToID)
	require.Equal(t, "BetaAPI", v.Edges[0].Label)

	out := RenderContainer(v)
	require.Contains(t, out, "System_Boundary(sys_11111111_1111_1111_1111_111111111111,")
	require.Contains(t, out, "Container(container_bbbbbbbb_bbbb_bbbb_bbbb_bbbbbbbbbbbb,")
	require.Contains(t, out, "Container(container_cccccccc_cccc_cccc_cccc_cccccccccccc,")
	require.Contains(t, out, "System_Ext(sys_22222222_2222_2222_2222_222222222222,")
	require.Contains(t, out, "Rel(container_bbbbbbbb_bbbb_bbbb_bbbb_bbbbbbbbbbbb,")
	require.NotContains(t, out, "Hidden")
}

// A ComponentInstance pointing at a SystemInstance that is not in the model used to
// be dropped silently: its ref was non-nil, so it missed the loose bucket, and no
// SystemInstance iteration ever reached it.
func TestBuildDeploymentView_DanglingSystemInstanceRefStaysVisible(t *testing.T) {
	m := newTestModel(t)

	ci := component.NewComponentInstance(uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"))
	ci.SetDisplayName("ci-dangling-si")
	ci.SetSystemInstance(&system.SystemInstanceRef{InstanceId: uuid.MustParse("f0f0f0f0-f0f0-f0f0-f0f0-f0f0f0f0f0f0")})
	require.NoError(t, m.AddComponentInstance(ci))

	ai := mdlapi.NewApiInstance(uuid.MustParse("dadadada-dada-dada-dada-dadadadadada"))
	ai.SetDisplayName("ai-dangling-si")
	ai.SetSystemInstance(&system.SystemInstanceRef{InstanceId: uuid.MustParse("f0f0f0f0-f0f0-f0f0-f0f0-f0f0f0f0f0f0")})
	require.NoError(t, m.AddApiInstance(ai))

	v, err := BuildDeploymentView(m)
	require.NoError(t, err)
	require.Len(t, v.Contexts, 1)
	require.Equal(t, uuid.Nil, v.Contexts[0].ID)
	require.Len(t, v.Contexts[0].LooseContainers, 1)
	require.Equal(t, "ci-dangling-si", v.Contexts[0].LooseContainers[0].DisplayName)
	require.Len(t, v.Contexts[0].LooseEndpoints, 1)
	require.Equal(t, "ai-dangling-si", v.Contexts[0].LooseEndpoints[0].DisplayName)

	out := RenderDeployment(v)
	require.Contains(t, out, "Container(ci_eeeeeeee_eeee_eeee_eeee_eeeeeeeeeeee,")
	require.Contains(t, out, "Container(ai_dadadada_dada_dada_dada_dadadadadada,")
}

// An ApiInstance with no API ref cannot carry an edge but must still be drawn.
func TestBuildDeploymentView_ApiInstanceWithoutApiRefStillRendered(t *testing.T) {
	m := newTestModel(t)

	ai := mdlapi.NewApiInstance(uuid.MustParse("abababab-abab-abab-abab-abababababab"))
	ai.SetDisplayName("ai-no-api")
	require.NoError(t, m.AddApiInstance(ai))

	v, err := BuildDeploymentView(m)
	require.NoError(t, err)
	require.Len(t, v.Contexts, 1)
	require.Len(t, v.Contexts[0].LooseEndpoints, 1)
	require.Equal(t, "ai-no-api", v.Contexts[0].LooseEndpoints[0].DisplayName)
	require.Empty(t, v.Contexts[0].LooseEndpoints[0].APIName)
	require.Empty(t, v.Edges)

	require.Contains(t, RenderDeployment(v), `$tags="endpoint"`)
}

func TestBuildDeploymentView_LooseAndUntypedComponentInstances(t *testing.T) {
	m := newTestModel(t)

	loose := component.NewComponentInstance(uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"))
	loose.SetDisplayName("orphan-ci")
	require.NoError(t, m.AddComponentInstance(loose))

	v, err := BuildDeploymentView(m)
	require.NoError(t, err)
	require.Len(t, v.Contexts, 1)
	require.Equal(t, uuid.Nil, v.Contexts[0].ID)
	require.Len(t, v.Contexts[0].LooseContainers, 1)
	require.Equal(t, "orphan-ci", v.Contexts[0].LooseContainers[0].DisplayName)
	require.Empty(t, v.Contexts[0].SystemInstances)

	out := RenderDeployment(v)
	require.Contains(t, out, "Container(ci_dddddddd_dddd_dddd_dddd_dddddddddddd,")
}

func TestRenderContext_PlantUMLShape(t *testing.T) {
	v := ContextView{
		Landscape: Landscape{Name: "EL", Description: "summary line"},
		Roots: []ContextNode{{
			ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			DisplayName: "prod", TypeName: "Environment",
			Children: []ContextNode{{
				ID:          uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
				DisplayName: `eu "west"`, TypeName: "Region",
			}},
		}},
	}
	out := RenderContext(v)
	require.True(t, strings.HasPrefix(out, "@startuml\n"))
	require.True(t, strings.HasSuffix(strings.TrimSpace(out), "@enduml"))
	require.Contains(t, out, "!include <C4/C4_Context>")
	require.Contains(t, out, "Enterprise_Boundary(landscape,")
	require.Contains(t, out, "Boundary(ctx_aaaaaaaa_aaaa_aaaa_aaaa_aaaaaaaaaaaa,")
	require.Contains(t, out, `\"west\"`)
	require.Contains(t, out, "note as landscape_summary")
}

func TestRenderContainer_PlantUMLShape(t *testing.T) {
	v := ContainerView{
		Systems: []SystemBoundary{{
			ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			DisplayName: "App", Description: "app",
			Containers: []C4Container{{
				ID:          uuid.MustParse("22222222-2222-2222-2222-222222222222"),
				DisplayName: "Web", Description: "web app", Technology: "OpenAPI",
				InstanceCount: 2,
			}},
		}},
		Externals: []ExternalNode{{
			ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			DisplayName: "Ext", Description: "external", Kind: "system", Abstract: true,
		}},
		Edges: []RelEdge{{
			FromID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			ToID:     uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			FromKind: "container", ToKind: "system",
			Label: "calls", Tech: "OpenAPI",
		}},
	}
	out := RenderContainer(v)
	require.Contains(t, out, "!include <C4/C4_Container>")
	require.Contains(t, out, "System_Boundary(sys_11111111_1111_1111_1111_111111111111,")
	// The PlantUML line break must survive escaping as \n, not \\n.
	require.Contains(t, out, `Container(container_22222222_2222_2222_2222_222222222222, "Web", "OpenAPI", "web app\n\n[2 instances]")`)
	require.Contains(t, out, "System_Ext(sys_33333333_3333_3333_3333_333333333333,")
	require.Contains(t, out, "Rel(container_22222222_2222_2222_2222_222222222222, sys_33333333_3333_3333_3333_333333333333,")
}

func TestAliasAndQuote(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	require.Equal(t, "ctx_aaaaaaaa_bbbb_cccc_dddd_eeeeeeeeeeee", alias("ctx", id))
	require.Equal(t, `"hello \"world\""`, quote(`hello "world"`))
}
