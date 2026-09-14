package c4injector

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
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
