package endpoint

import (
	"testing"

	eventmgr "go.emeland.io/modelsrv/internal/events"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
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

func TestStartWebListener_ObserverSeesStartLog(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core).Sugar()

	sink := events.NewDummySink()
	backend, err := model.NewModel(sink)
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}
	eventMgr, err := eventmgr.NewEventManager()
	if err != nil {
		t.Fatalf("failed to create event manager: %v", err)
	}

	err = StartWebListener(backend, eventMgr, "127.0.0.1:0", WebListenerOptions{Logger: logger})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer StopWebListener()

	found := false
	for _, entry := range logs.All() {
		if entry.Message == "starting web endpoint" {
			found = true
			ctx := entry.ContextMap()
			if _, ok := ctx["address"]; !ok {
				t.Error("expected 'address' field in start log")
			}
			break
		}
	}
	if !found {
		t.Error("expected 'starting web endpoint' log entry on observer")
	}
}

func TestStartWebListener_NilLoggerNoOutput(t *testing.T) {
	sink := events.NewDummySink()
	backend, err := model.NewModel(sink)
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}
	eventMgr, err := eventmgr.NewEventManager()
	if err != nil {
		t.Fatalf("failed to create event manager: %v", err)
	}

	// nil logger should not panic and produce no output
	err = StartWebListener(backend, eventMgr, "127.0.0.1:0", WebListenerOptions{})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	StopWebListener()
}
