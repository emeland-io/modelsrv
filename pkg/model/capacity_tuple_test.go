package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	mdlcap "go.emeland.io/modelsrv/pkg/model/capacity"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
)

func seedCapacityDeps(t *testing.T, m model.Model) (crtID, ctxID uuid.UUID) {
	t.Helper()

	crtID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ctxID = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	ctID := uuid.New()

	crt := mdlcap.NewCapacityResourceType(crtID)
	crt.SetDisplayName("CPU")
	crt.SetUnit("cores")
	require.NoError(t, m.AddCapacityResourceType(crt))

	ct := mdlctx.NewContextType(ctID)
	ct.SetDisplayName("Environment")
	require.NoError(t, m.AddContextType(ct))

	ctx := mdlctx.NewContext(ctxID)
	ctx.SetDisplayName("Production")
	ctx.SetContextTypeById(ctID)
	require.NoError(t, m.AddContext(ctx))

	return crtID, ctxID
}

func newCapacity(id, crtID, ctxID uuid.UUID, name string) mdlcap.Capacity {
	c := mdlcap.NewCapacity(id)
	c.SetDisplayName(name)
	c.SetCapacityResourceTypeById(crtID)
	c.SetContextById(ctxID)
	c.SetCategory(mdlcap.CategoryProvided)
	c.SetAmount(mdlcap.Amount("64"))
	return c
}

// Tuple-keyed upsert per docs/adr/capacity-resources.md: same tuple + CapacityId updates;
// same tuple + different CapacityId rejects without changing the existing row.
func TestCapacityTupleUpsert(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	crtID, ctxID := seedCapacityDeps(t, m)
	capID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	require.NoError(t, m.AddCapacity(newCapacity(capID, crtID, ctxID, "v1")))

	updated := newCapacity(capID, crtID, ctxID, "v2")
	require.NoError(t, m.AddCapacity(updated))
	got := m.GetCapacityById(capID)
	require.NotNil(t, got)
	assert.Equal(t, "v2", got.GetDisplayName())

	conflictID := uuid.New()
	conflict := newCapacity(conflictID, crtID, ctxID, "conflict")
	assert.ErrorIs(t, m.AddCapacity(conflict), common.ErrCapacityTupleConflict)
	assert.Nil(t, m.GetCapacityById(conflictID))
	still := m.GetCapacityById(capID)
	require.NotNil(t, still)
	assert.Equal(t, "v2", still.GetDisplayName())
}

func TestAddCapacityRejectsUnresolvedRefs(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	c := mdlcap.NewCapacity(uuid.New())
	c.SetDisplayName("orphan")
	c.SetCapacityResourceTypeById(uuid.New())
	c.SetContextById(uuid.New())
	c.SetCategory(mdlcap.CategoryRequested)
	c.SetAmount(mdlcap.Amount("1"))

	assert.ErrorIs(t, m.AddCapacity(c), common.ErrCapacityResourceTypeNotFound)
}
