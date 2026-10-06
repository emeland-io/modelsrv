package capability_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mdlcapability "go.emeland.io/modelsrv/pkg/model/capability"
)

func TestCheckDependencyCoverage_Pass(t *testing.T) {
	paramID := uuid.New()
	reqID := uuid.New()
	offerID := uuid.New()

	cap := mdlcapability.NewCapability(uuid.New())
	cap.SetOffers([]uuid.UUID{offerID})

	variant := mdlcapability.NewVariant(uuid.New())
	variant.SetRequires([]uuid.UUID{reqID})

	err := mdlcapability.CheckDependencyCoverage(
		variant, cap,
		map[uuid.UUID]uuid.UUID{reqID: paramID},
		map[uuid.UUID]uuid.UUID{offerID: paramID},
	)
	require.NoError(t, err)
}

func TestCheckDependencyCoverage_FailMissingOffer(t *testing.T) {
	paramID := uuid.New()
	reqID := uuid.New()
	otherOffer := uuid.New()
	otherParam := uuid.New()

	cap := mdlcapability.NewCapability(uuid.New())
	cap.SetOffers([]uuid.UUID{otherOffer})

	variant := mdlcapability.NewVariant(uuid.New())
	variant.SetRequires([]uuid.UUID{reqID})

	err := mdlcapability.CheckDependencyCoverage(
		variant, cap,
		map[uuid.UUID]uuid.UUID{reqID: paramID},
		map[uuid.UUID]uuid.UUID{otherOffer: otherParam},
	)
	require.Error(t, err)
	var cov *mdlcapability.DependencyCoverageError
	require.ErrorAs(t, err, &cov)
	assert.Equal(t, []uuid.UUID{reqID}, cov.Missing)
}

func TestCheckDependencyCoverage_SkipMissingNeighbor(t *testing.T) {
	variant := mdlcapability.NewVariant(uuid.New())
	variant.SetRequires([]uuid.UUID{uuid.New()})

	err := mdlcapability.CheckDependencyCoverage(variant, nil, nil, nil)
	require.NoError(t, err)

	cap := mdlcapability.NewCapability(uuid.New())
	err = mdlcapability.CheckDependencyCoverage(variant, cap, nil, nil)
	require.NoError(t, err)
}
