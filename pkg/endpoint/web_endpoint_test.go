package endpoint

import (
	"testing"

	"gitlab.com/emeland/modelsrv/pkg/events"
	"gitlab.com/emeland/modelsrv/pkg/model"
)

func TestStartUIListener(t *testing.T) {
	sink := events.NewDummySink()
	backend, err := model.NewModel(sink)
	if err != nil {
		t.Fatalf("failed to create model backend: %v", err)
	}

	eventMgr, err := events.NewEventManager()
	if err != nil {
		t.Fatalf("failed to create event manager: %v", err)
	}

	// This test should verify that StartUIListener returns nil and starts a server
	// For now, just check that it returns nil (since actual server start is async)
	err = StarWebListener(backend, eventMgr, "127.0.0.1:24000")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
