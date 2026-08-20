package endpoint

import (
	"fmt"
	"net/http"
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

	err = StartWebListener(backend, eventMgr, "127.0.0.1:0", WebListenerOptions{})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	StopWebListener()
}

func TestRequestLogging_APICallEmitsInfoLog(t *testing.T) {
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

	addr := WebListenerAddr()
	url := fmt.Sprintf("http://%s/api/systems", addr.String())
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	_ = resp.Body.Close()

	found := false
	for _, entry := range logs.All() {
		if entry.Message == "http request" {
			found = true
			if entry.Level != zapcore.InfoLevel {
				t.Errorf("expected INFO level, got %v", entry.Level)
			}
			ctx := entry.ContextMap()
			if ctx["method"] != "GET" {
				t.Errorf("expected method=GET, got %v", ctx["method"])
			}
			if ctx["path"] != "/api/systems" {
				t.Errorf("expected path=/api/systems, got %v", ctx["path"])
			}
			break
		}
	}
	if !found {
		t.Error("no 'http request' log entry found")
	}
}

func TestRequestLogging_NilLoggerDoesNotPanic(t *testing.T) {
	sink := events.NewDummySink()
	backend, err := model.NewModel(sink)
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}
	eventMgr, err := eventmgr.NewEventManager()
	if err != nil {
		t.Fatalf("failed to create event manager: %v", err)
	}

	err = StartWebListener(backend, eventMgr, "127.0.0.1:0", WebListenerOptions{})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer StopWebListener()

	addr := WebListenerAddr()
	url := fmt.Sprintf("http://%s/api/systems", addr.String())
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	_ = resp.Body.Close()
}

func TestRequestLogging_SwaggerLogsAtDebug(t *testing.T) {
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

	addr := WebListenerAddr()
	url := fmt.Sprintf("http://%s/swagger/index.html", addr.String())
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	_ = resp.Body.Close()

	for _, entry := range logs.All() {
		if entry.Message == "http request" {
			if entry.Level != zapcore.DebugLevel {
				t.Errorf("expected DEBUG for /swagger, got %v", entry.Level)
			}
			return
		}
	}
	t.Error("no 'http request' log entry found for /swagger")
}
