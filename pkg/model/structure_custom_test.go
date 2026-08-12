package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/model"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.emeland.io/modelsrv/pkg/model/system"
)

func TestAddNilUUIDReturnsError(t *testing.T) {
	m, _ := newStoreModel(t)

	ctx := mdlctx.NewContext(uuid.Nil)
	ctx.SetDisplayName("missing-id")
	err := m.AddContext(ctx)
	require.ErrorIs(t, err, common.ErrUUIDNotSet)
}

func TestAddUpdatesExistingResource(t *testing.T) {
	m, _ := newStoreModel(t)

	id := uuid.New()
	first := system.NewSystem(id)
	first.SetDisplayName("v1")
	require.NoError(t, m.AddSystem(first))

	second := system.NewSystem(id)
	second.SetDisplayName("v2")
	require.NoError(t, m.AddSystem(second))

	got := m.GetSystemById(id)
	require.NotNil(t, got)
	assert.Equal(t, "v2", got.GetDisplayName())
}

func TestGetFindingTypeByName(t *testing.T) {
	m, _ := newStoreModel(t)

	id := uuid.New()
	ft := finding.NewFindingType(id)
	ft.SetDisplayName("integrity-violation")
	require.NoError(t, m.AddFindingType(ft))

	assert.Nil(t, m.GetFindingTypeByName(""))
	assert.Nil(t, m.GetFindingTypeByName("unknown"))
	assert.Equal(t, ft, m.GetFindingTypeByName("integrity-violation"))
	assert.Equal(t, ft, m.GetFindingTypeById(id))
}

func TestDeleteContextById(t *testing.T) {
	m, _ := newStoreModel(t)

	missing := uuid.New()
	err := m.DeleteContextById(missing)
	require.ErrorIs(t, err, common.ErrContextNotFound)

	id := uuid.New()
	ctx := mdlctx.NewContext(id)
	ctx.SetDisplayName("ctx-to-delete")
	require.NoError(t, m.AddContext(ctx))
	require.NotNil(t, m.GetContextById(id))

	require.NoError(t, m.DeleteContextById(id))
	assert.Nil(t, m.GetContextById(id))
	require.ErrorIs(t, m.DeleteContextById(id), common.ErrContextNotFound)
}

func TestDeleteFindingTypeById(t *testing.T) {
	m, _ := newStoreModel(t)

	id := uuid.New()
	ft := finding.NewFindingType(id)
	ft.SetDisplayName("finding-type")
	require.NoError(t, m.AddFindingType(ft))

	require.NoError(t, m.DeleteFindingTypeById(id))
	assert.Nil(t, m.GetFindingTypeById(id))
	require.ErrorIs(t, m.DeleteFindingTypeById(id), common.ErrFindingTypeNotFound)
}

func TestApiRefByID(t *testing.T) {
	m, _ := newStoreModel(t)

	assert.Nil(t, m.ApiRefByID(uuid.New()))

	apiID := uuid.New()
	api := mdlapi.NewAPI(apiID)
	api.SetDisplayName("orders-api")
	require.NoError(t, m.AddApi(api))

	ref := m.ApiRefByID(apiID)
	require.NotNil(t, ref)
	assert.Equal(t, api, ref.API)
	assert.Equal(t, apiID, ref.ApiID)
}

func TestSystemInstanceRefByID(t *testing.T) {
	m, _ := newStoreModel(t)

	assert.Nil(t, m.SystemInstanceRefByID(uuid.New()))

	instanceID := uuid.New()
	sys := system.NewSystem(uuid.New())
	sys.SetDisplayName("backend")
	require.NoError(t, m.AddSystem(sys))

	inst := system.NewSystemInstance(instanceID)
	inst.SetDisplayName("backend-prod")
	inst.SetSystemRef(&system.SystemRef{System: sys})
	require.NoError(t, m.AddSystemInstance(inst))

	ref := m.SystemInstanceRefByID(instanceID)
	require.NotNil(t, ref)
	assert.Equal(t, inst, ref.SystemInstance)
	assert.Equal(t, instanceID, ref.InstanceId)
}

func TestGetContextsAndGetApis(t *testing.T) {
	m, _ := newStoreModel(t)

	contexts, err := m.GetContexts()
	require.NoError(t, err)
	assert.Empty(t, contexts)

	apis, err := m.GetApis()
	require.NoError(t, err)
	assert.Empty(t, apis)

	ctID := uuid.New()
	ct := mdlctx.NewContextType(ctID)
	ct.SetDisplayName("env")
	require.NoError(t, m.AddContextType(ct))

	ctxID := uuid.New()
	ctx := mdlctx.NewContext(ctxID)
	ctx.SetDisplayName("production")
	ctx.SetContextTypeById(ctID)
	require.NoError(t, m.AddContext(ctx))

	contexts, err = m.GetContexts()
	require.NoError(t, err)
	require.Len(t, contexts, 1)
	assert.Equal(t, ctxID, contexts[0].GetContextId())

	sysID := uuid.New()
	sys := system.NewSystem(sysID)
	sys.SetDisplayName("orders")
	require.NoError(t, m.AddSystem(sys))

	apiID := uuid.New()
	api := mdlapi.NewAPI(apiID)
	api.SetDisplayName("orders-api")
	api.SetSystem(&system.SystemRef{SystemId: sysID})
	require.NoError(t, m.AddApi(api))

	apis, err = m.GetApis()
	require.NoError(t, err)
	require.Len(t, apis, 1)
	assert.Equal(t, apiID, apis[0].GetApiId())
}

func TestGetFindingsReferencingResource(t *testing.T) {
	m, _ := newStoreModel(t)

	subjectID := uuid.New()
	missingID := uuid.New()
	otherID := uuid.New()

	assert.Nil(t, m.GetFindingsReferencingResource(uuid.Nil))
	assert.Nil(t, m.GetFindingsReferencingResource(missingID))

	fid := uuid.New()
	f := finding.NewFinding(fid)
	f.SetDisplayName("dangling")
	f.SetResources([]*common.ResourceRef{
		{ResourceId: subjectID},
		{ResourceId: missingID},
	})
	require.NoError(t, m.AddFinding(f))

	bySubject := m.GetFindingsReferencingResource(subjectID)
	require.Len(t, bySubject, 1)
	assert.Equal(t, fid, bySubject[0].GetFindingId())

	byMissing := m.GetFindingsReferencingResource(missingID)
	require.Len(t, byMissing, 1)
	assert.Equal(t, fid, byMissing[0].GetFindingId())

	assert.Nil(t, m.GetFindingsReferencingResource(otherID))

	// Update Resources: drop missingID, cite otherID instead.
	updated := finding.NewFinding(fid)
	updated.SetDisplayName("dangling")
	updated.SetResources([]*common.ResourceRef{
		{ResourceId: subjectID},
		{ResourceId: otherID},
	})
	require.NoError(t, m.AddFinding(updated))

	assert.Nil(t, m.GetFindingsReferencingResource(missingID))
	byOther := m.GetFindingsReferencingResource(otherID)
	require.Len(t, byOther, 1)
	assert.Equal(t, fid, byOther[0].GetFindingId())
	require.Len(t, m.GetFindingsReferencingResource(subjectID), 1)

	require.NoError(t, m.DeleteFindingById(fid))
	assert.Nil(t, m.GetFindingsReferencingResource(subjectID))
	assert.Nil(t, m.GetFindingsReferencingResource(otherID))
}

func TestNewModelRejectsNilSink(t *testing.T) {
	_, err := model.NewModel(nil)
	require.Error(t, err)
}
