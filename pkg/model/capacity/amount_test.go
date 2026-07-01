package capacity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mdlcap "go.emeland.io/modelsrv/pkg/model/capacity"
)

func TestParseCategory(t *testing.T) {
	cat, err := mdlcap.ParseCategory("provided")
	require.NoError(t, err)
	assert.Equal(t, mdlcap.CategoryProvided, cat)

	_, err = mdlcap.ParseCategory("invalid")
	assert.Error(t, err)
}

func TestParseAmount(t *testing.T) {
	amt, err := mdlcap.ParseAmount("64")
	require.NoError(t, err)
	assert.Equal(t, mdlcap.Amount("64"), amt)

	_, err = mdlcap.ParseAmount("-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-negative")
}
