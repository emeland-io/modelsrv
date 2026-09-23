package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	mdlcapability "go.emeland.io/modelsrv/pkg/model/capability"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlparameter "go.emeland.io/modelsrv/pkg/model/parameter"
)

func seedCapability(t *testing.T, m model.Model, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	c := mdlcapability.NewCapability(id)
	c.SetDisplayName(name)
	require.NoError(t, m.AddCapability(c))
	return id
}

func seedCapabilityVersion(t *testing.T, m model.Model, capID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	cv := mdlcapability.NewCapabilityVersion(id)
	cv.SetDisplayName(name)
	cv.SetVersion(common.Version{Version: "1.0.0"})
	cv.SetCapabilityById(capID)
	require.NoError(t, m.AddCapabilityVersion(cv))
	return id
}

func seedParamWithValues(t *testing.T, m model.Model, name string, valueNames ...string) (uuid.UUID, []uuid.UUID) {
	t.Helper()
	paramID := uuid.New()
	param := mdlparameter.NewParameter(paramID)
	param.SetDisplayName(name)
	require.NoError(t, m.AddParameter(param))
	ids := make([]uuid.UUID, 0, len(valueNames))
	for _, vn := range valueNames {
		vid := uuid.New()
		vv := mdlparameter.NewValidValue(vid)
		vv.SetDisplayName(vn)
		vv.SetParameterById(paramID)
		require.NoError(t, m.AddValidValue(vv))
		ids = append(ids, vid)
	}
	param = m.GetParameterById(paramID)
	param.SetValues(ids)
	require.NoError(t, m.AddParameter(param))
	return paramID, ids
}

func TestAddCapabilityRejectsUnknownVersion(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	c := mdlcapability.NewCapability(uuid.New())
	c.SetDisplayName("cap")
	c.SetVersions([]mdlcapability.CapabilityVersionRef{{CapabilityVersionId: uuid.New()}})
	assert.ErrorIs(t, m.AddCapability(c), common.ErrCapabilityVersionNotFound)
}

func TestAddCapabilityVersionRequiresCapability(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	cv := mdlcapability.NewCapabilityVersion(uuid.New())
	cv.SetDisplayName("v1")
	cv.SetVersion(common.Version{Version: "1.0.0"})
	cv.SetCapabilityById(uuid.New())
	assert.ErrorIs(t, m.AddCapabilityVersion(cv), common.ErrCapabilityNotFound)
}

func TestVariantDependencySuperset(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	capA := seedCapability(t, m, "A")
	capB := seedCapability(t, m, "B")
	cvA := seedCapabilityVersion(t, m, capA, "A-v1")
	cvB := seedCapabilityVersion(t, m, capB, "B-v1")
	paramID, vids := seedParamWithValues(t, m, "region", "eu", "us", "ap")

	provider := mdlcapability.NewVariant(uuid.New())
	provider.SetDisplayName("B-full")
	provider.SetCapabilityVersionById(cvB)
	provider.SetProvides([]mdlcapability.ParameterValueSet{{
		ParameterId:   paramID,
		ValidValueIds: vids, // eu, us, ap
	}})
	require.NoError(t, m.AddVariant(provider))

	consumer := mdlcapability.NewVariant(uuid.New())
	consumer.SetDisplayName("A-needs-eu-us")
	consumer.SetCapabilityVersionById(cvA)
	consumer.SetDependencies([]mdlcapability.VariantDependency{{
		CapabilityId: capB,
		Required: []mdlcapability.ParameterValueSet{{
			ParameterId:   paramID,
			ValidValueIds: vids[:2], // eu, us — subset of provider
		}},
	}})
	require.NoError(t, m.AddVariant(consumer))

	unsatisfied := mdlcapability.NewVariant(uuid.New())
	unsatisfied.SetDisplayName("A-needs-more-than-provided")
	unsatisfied.SetCapabilityVersionById(cvA)
	unsatisfied.SetDependencies([]mdlcapability.VariantDependency{{
		CapabilityId: capB,
		Required: []mdlcapability.ParameterValueSet{{
			ParameterId:   paramID,
			ValidValueIds: vids, // all three — provider only has subset in a fresh model
		}},
	}})

	// Replace provider with a narrower set so full vids cannot be satisfied.
	narrow := mdlcapability.NewVariant(provider.GetVariantId())
	narrow.SetDisplayName("B-narrow")
	narrow.SetCapabilityVersionById(cvB)
	narrow.SetProvides([]mdlcapability.ParameterValueSet{{
		ParameterId:   paramID,
		ValidValueIds: vids[:1],
	}})
	require.NoError(t, m.AddVariant(narrow))
	assert.ErrorIs(t, m.AddVariant(unsatisfied), common.ErrVariantDependencyUnsatisfied)
}

func TestVariantDependencyCycle(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	capA := seedCapability(t, m, "A")
	capB := seedCapability(t, m, "B")
	cvA := seedCapabilityVersion(t, m, capA, "A-v1")
	cvB := seedCapabilityVersion(t, m, capB, "B-v1")
	paramID, vids := seedParamWithValues(t, m, "tier", "basic", "pro")

	va := mdlcapability.NewVariant(uuid.New())
	va.SetDisplayName("A-var")
	va.SetCapabilityVersionById(cvA)
	va.SetProvides([]mdlcapability.ParameterValueSet{{
		ParameterId:   paramID,
		ValidValueIds: vids,
	}})
	require.NoError(t, m.AddVariant(va))

	vb := mdlcapability.NewVariant(uuid.New())
	vb.SetDisplayName("B-var")
	vb.SetCapabilityVersionById(cvB)
	vb.SetProvides([]mdlcapability.ParameterValueSet{{
		ParameterId:   paramID,
		ValidValueIds: vids,
	}})
	vb.SetDependencies([]mdlcapability.VariantDependency{{
		CapabilityId: capA,
		Required: []mdlcapability.ParameterValueSet{{
			ParameterId:   paramID,
			ValidValueIds: vids[:1],
		}},
	}})
	require.NoError(t, m.AddVariant(vb))

	// Closing the cycle: A → B while B → A already exists.
	va2 := mdlcapability.NewVariant(va.GetVariantId())
	va2.SetDisplayName("A-var")
	va2.SetCapabilityVersionById(cvA)
	va2.SetProvides([]mdlcapability.ParameterValueSet{{
		ParameterId:   paramID,
		ValidValueIds: vids,
	}})
	va2.SetDependencies([]mdlcapability.VariantDependency{{
		CapabilityId: capB,
		Required: []mdlcapability.ParameterValueSet{{
			ParameterId:   paramID,
			ValidValueIds: vids[:1],
		}},
	}})
	assert.ErrorIs(t, m.AddVariant(va2), common.ErrVariantDependencyCycle)
}
