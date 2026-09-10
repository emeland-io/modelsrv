package c4injector

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/component"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.emeland.io/modelsrv/pkg/model/system"
)

type discardSink struct{}

func (discardSink) Receive(events.ResourceType, events.Operation, uuid.UUID, ...any) error {
	return nil
}

func newTestModel(t *testing.T) model.Model {
	t.Helper()
	m, err := model.NewModel(discardSink{})
	require.NoError(t, err)
	return m
}

func findingsOfKind(m model.Model, kind finding.FindingKind) []finding.Finding {
	typeID := finding.TypeIDForKind(kind)
	all, _ := m.GetFindings()
	var out []finding.Finding
	for _, f := range all {
		if f.GetFindingTypeId() == typeID {
			out = append(out, f)
		}
	}
	return out
}

func TestReconcile_DescriptionMissing(t *testing.T) {
	m := newTestModel(t)
	sys := system.NewSystem(uuid.New())
	sys.SetDisplayName("S")
	require.NoError(t, m.AddSystem(sys))

	Reconcile(m)
	require.Len(t, findingsOfKind(m, finding.DescriptionMissing), 1)
	id := findingID(sys.GetSystemId(), finding.DescriptionMissing)

	sys.SetDescription("documented")
	require.NoError(t, m.AddSystem(sys))
	Reconcile(m)
	require.Empty(t, findingsOfKind(m, finding.DescriptionMissing))
	require.Nil(t, m.GetFindingById(id))
}

func TestReconcile_ConcreteSystemHasNoComponents(t *testing.T) {
	m := newTestModel(t)
	sys := system.NewSystem(uuid.New())
	sys.SetDisplayName("S")
	sys.SetDescription("desc")
	require.NoError(t, m.AddSystem(sys))

	Reconcile(m)
	require.Len(t, findingsOfKind(m, finding.ConcreteSystemHasNoComponents), 1)

	comp := component.NewComponent(uuid.New())
	comp.SetDisplayName("C")
	comp.SetDescription("c")
	comp.SetSystem(&system.SystemRef{System: sys, SystemId: sys.GetSystemId()})
	require.NoError(t, m.AddComponent(comp))
	Reconcile(m)
	require.Empty(t, findingsOfKind(m, finding.ConcreteSystemHasNoComponents))
}

func TestReconcile_AbstractSystemHasComponents(t *testing.T) {
	m := newTestModel(t)
	sys := system.NewSystem(uuid.New())
	sys.SetDisplayName("Abs")
	sys.SetDescription("desc")
	sys.SetAbstract(true)
	require.NoError(t, m.AddSystem(sys))

	comp := component.NewComponent(uuid.New())
	comp.SetDisplayName("C")
	comp.SetDescription("c")
	comp.SetSystem(&system.SystemRef{System: sys, SystemId: sys.GetSystemId()})
	require.NoError(t, m.AddComponent(comp))

	Reconcile(m)
	require.Len(t, findingsOfKind(m, finding.AbstractSystemHasComponents), 1)
	require.Empty(t, findingsOfKind(m, finding.ConcreteSystemHasNoComponents))
}

func TestReconcile_APIProviders(t *testing.T) {
	m := newTestModel(t)
	sys := system.NewSystem(uuid.New())
	sys.SetDisplayName("S")
	sys.SetDescription("desc")
	require.NoError(t, m.AddSystem(sys))

	api := mdlapi.NewAPI(uuid.New())
	api.SetDisplayName("A")
	api.SetDescription("api")
	api.SetSystemByRef(sys)
	require.NoError(t, m.AddApi(api))

	Reconcile(m)
	require.Len(t, findingsOfKind(m, finding.APIHasNoProvider), 1)

	c1 := component.NewComponent(uuid.New())
	c1.SetDisplayName("P1")
	c1.SetDescription("p1")
	c1.SetSystem(&system.SystemRef{System: sys, SystemId: sys.GetSystemId()})
	c1.SetProvides([]mdlapi.ApiRef{{API: api, ApiID: api.GetApiId()}})
	require.NoError(t, m.AddComponent(c1))
	Reconcile(m)
	require.Empty(t, findingsOfKind(m, finding.APIHasNoProvider))
	require.Empty(t, findingsOfKind(m, finding.APIHasMultipleProviders))

	c2 := component.NewComponent(uuid.New())
	c2.SetDisplayName("P2")
	c2.SetDescription("p2")
	c2.SetSystem(&system.SystemRef{System: sys, SystemId: sys.GetSystemId()})
	c2.SetProvides([]mdlapi.ApiRef{{API: api, ApiID: api.GetApiId()}})
	require.NoError(t, m.AddComponent(c2))
	Reconcile(m)
	require.Len(t, findingsOfKind(m, finding.APIHasMultipleProviders), 1)
}

func TestReconcile_ProvidedAPISystemMismatch(t *testing.T) {
	m := newTestModel(t)
	sysA := system.NewSystem(uuid.New())
	sysA.SetDisplayName("A")
	sysA.SetDescription("a")
	require.NoError(t, m.AddSystem(sysA))
	sysB := system.NewSystem(uuid.New())
	sysB.SetDisplayName("B")
	sysB.SetDescription("b")
	require.NoError(t, m.AddSystem(sysB))

	api := mdlapi.NewAPI(uuid.New())
	api.SetDisplayName("ApiB")
	api.SetDescription("api")
	api.SetSystemByRef(sysB)
	require.NoError(t, m.AddApi(api))

	comp := component.NewComponent(uuid.New())
	comp.SetDisplayName("InA")
	comp.SetDescription("c")
	comp.SetSystem(&system.SystemRef{System: sysA, SystemId: sysA.GetSystemId()})
	comp.SetProvides([]mdlapi.ApiRef{{API: api, ApiID: api.GetApiId()}})
	require.NoError(t, m.AddComponent(comp))

	Reconcile(m)
	require.Len(t, findingsOfKind(m, finding.ProvidedAPISystemMismatch), 1)
}

func TestReconcile_FindingUUIDStable(t *testing.T) {
	m := newTestModel(t)
	sys := system.NewSystem(uuid.New())
	sys.SetDisplayName("S")
	require.NoError(t, m.AddSystem(sys))
	Reconcile(m)
	first := findingsOfKind(m, finding.DescriptionMissing)
	require.Len(t, first, 1)
	Reconcile(m)
	second := findingsOfKind(m, finding.DescriptionMissing)
	require.Len(t, second, 1)
	require.Equal(t, first[0].GetFindingId(), second[0].GetFindingId())
}

func TestReconcile_SystemInstanceContextMissing(t *testing.T) {
	m := newTestModel(t)
	sys := system.NewSystem(uuid.New())
	sys.SetDisplayName("S")
	sys.SetDescription("d")
	require.NoError(t, m.AddSystem(sys))

	si := system.NewSystemInstance(uuid.New())
	si.SetDisplayName("si")
	si.SetSystemRef(&system.SystemRef{System: sys, SystemId: sys.GetSystemId()})
	require.NoError(t, m.AddSystemInstance(si))

	Reconcile(m)
	require.Len(t, findingsOfKind(m, finding.SystemInstanceContextMissing), 1)

	ct := mdlctx.NewContextType(uuid.New())
	ct.SetDisplayName("Env")
	require.NoError(t, m.AddContextType(ct))
	ctx := mdlctx.NewContext(uuid.New())
	ctx.SetDisplayName("prod")
	ctx.SetContextTypeByRef(ct)
	require.NoError(t, m.AddContext(ctx))
	si.SetContextRef(&mdlctx.ContextRef{Context: ctx, ContextId: ctx.GetContextId()})
	require.NoError(t, m.AddSystemInstance(si))
	Reconcile(m)
	require.Empty(t, findingsOfKind(m, finding.SystemInstanceContextMissing))
}

func TestReconcile_ApiInstanceSystemInstanceMissing(t *testing.T) {
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

	ai := mdlapi.NewApiInstance(uuid.New())
	ai.SetDisplayName("ai")
	ai.SetApiRefByRef(api)
	require.NoError(t, m.AddApiInstance(ai))

	Reconcile(m)
	require.Len(t, findingsOfKind(m, finding.ApiInstanceSystemInstanceMissing), 1)

	si := system.NewSystemInstance(uuid.New())
	si.SetDisplayName("si")
	si.SetSystemRef(&system.SystemRef{System: sys, SystemId: sys.GetSystemId()})
	require.NoError(t, m.AddSystemInstance(si))
	ai.SetSystemInstanceByRef(si)
	require.NoError(t, m.AddApiInstance(ai))

	Reconcile(m)
	require.Empty(t, findingsOfKind(m, finding.ApiInstanceSystemInstanceMissing))
}

func TestReconcile_ApiInstanceMissingForComponentInstance(t *testing.T) {
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

	comp := component.NewComponent(uuid.New())
	comp.SetDisplayName("C")
	comp.SetDescription("c")
	comp.SetSystem(&system.SystemRef{System: sys, SystemId: sys.GetSystemId()})
	comp.SetProvides([]mdlapi.ApiRef{{API: api, ApiID: api.GetApiId()}})
	require.NoError(t, m.AddComponent(comp))

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

	ci := component.NewComponentInstance(uuid.New())
	ci.SetDisplayName("ci")
	ci.SetComponentRef(&component.ComponentRef{Component: comp, ComponentId: comp.GetComponentId()})
	ci.SetSystemInstance(&system.SystemInstanceRef{SystemInstance: si, InstanceId: si.GetInstanceId()})
	require.NoError(t, m.AddComponentInstance(ci))

	Reconcile(m)
	require.Len(t, findingsOfKind(m, finding.ApiInstanceMissingForComponentInstance), 1)

	ai := mdlapi.NewApiInstance(uuid.New())
	ai.SetDisplayName("ai")
	ai.SetApiRefByRef(api)
	ai.SetSystemInstanceByRef(si)
	require.NoError(t, m.AddApiInstance(ai))
	Reconcile(m)
	require.Empty(t, findingsOfKind(m, finding.ApiInstanceMissingForComponentInstance))
}

func TestFilter_PassesThrough(t *testing.T) {
	m := newTestModel(t)
	f := NewFilter()

	sys := system.NewSystem(uuid.New())
	sys.SetDisplayName("S")
	sys.SetDescription("d")
	require.NoError(t, m.AddSystem(sys))

	ev := events.Event{
		ResourceType: events.SystemResource,
		Operation:    events.CreateOperation,
		ResourceId:   sys.GetSystemId(),
		Objects:      []any{sys},
	}
	out := f.Fn(m, ev)
	require.Len(t, out, 1)
	require.Len(t, findingsOfKind(m, finding.ConcreteSystemHasNoComponents), 1)
}
