package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
)

func TestResourceDisplayName(t *testing.T) {
	m, _ := newStoreModel(t)

	ctxID := uuid.New()
	ctx := mdlctx.NewContext(ctxID)
	ctx.SetDisplayName("Production")
	require.NoError(t, m.AddContext(ctx))

	name := model.ResourceDisplayName(m, &common.ResourceRef{
		ResourceId:   ctxID,
		ResourceType: events.ContextResource,
	})
	require.Equal(t, "Production", name)

	require.Equal(t, "", model.ResourceDisplayName(m, &common.ResourceRef{
		ResourceId:   uuid.New(),
		ResourceType: events.ContextResource,
	}))
}
