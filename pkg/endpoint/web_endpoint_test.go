package endpoint

import (
	"testing"

	eventmgr "go.emeland.io/modelsrv/internal/events"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

func TestStartUIListener(t *testing.T) {
	sink := events.NewDummySink()
	backend, err := model.NewModel(sink)
	if err != nil {
		t.Fatalf("failed to create model backend: %v", err)
	}

	eventMgr, err := eventmgr.NewEventManager()
	if err != nil {
		t.Fatalf("failed to create event manager: %v", err)
	}

	err = StartWebListener(backend, eventMgr, "127.0.0.1:0", WebListenerOptions{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	defer StopWebListener()

	addr := WebListenerAddr()
	if addr == nil {
		t.Fatal("expected non-nil listener address")
	}
	t.Logf("listening on %s", addr.String())
}
