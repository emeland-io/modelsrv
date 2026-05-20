package model_test

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/node"
)

// TestModelConcurrentNodeTypeMutation exercises the same paths that crashed in production:
// concurrent AddNodeType (e.g. file sensor) and Apply upsert (e.g. POST /events/push).
// Run with: go test -race ./pkg/model/...
func TestModelConcurrentNodeTypeMutation(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}

	const workers = 32
	const rounds = 50
	id := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	errCh := make(chan error, workers*2*rounds*2)
	var wg sync.WaitGroup
	wg.Add(workers * 2)
	for range workers {
		go func() {
			defer wg.Done()
			for range rounds {
				nt := node.NewNodeType(sink, id)
				nt.SetDisplayName("from-add")
				if err := m.AddNodeType(nt); err != nil {
					errCh <- err
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			for range rounds {
				nt := node.NewNodeType(sink, id)
				nt.SetDisplayName("from-apply")
				if err := m.Apply(events.Event{
					ResourceType: events.NodeTypeResource,
					Operation:    events.CreateOperation,
					ResourceId:   id,
					Objects:      []any{nt},
				}); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent mutation: %v", err)
	}

	types, err := m.GetNodeTypes()
	if err != nil {
		t.Fatalf("GetNodeTypes: %v", err)
	}
	if len(types) != 1 {
		t.Fatalf("expected 1 node type, got %d", len(types))
	}
}

// TestModelConcurrentReadWrite mirrors metrics scrape (read) alongside mutation (write).
func TestModelConcurrentReadWrite(t *testing.T) {
	sink := events.NewListSink()
	m, err := model.NewModel(sink)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}

	const workers = 16
	const rounds = 100

	errCh := make(chan error, workers*rounds*2)
	var wg sync.WaitGroup
	wg.Add(workers * 2)
	for range workers {
		go func() {
			defer wg.Done()
			for range rounds {
				id := uuid.New()
				nt := node.NewNodeType(sink, id)
				nt.SetDisplayName("nt")
				if err := m.AddNodeType(nt); err != nil {
					errCh <- err
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			for range rounds {
				if _, err := m.GetNodeTypes(); err != nil {
					errCh <- err
					return
				}
				if _, err := m.GetSystems(); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent read/write: %v", err)
	}
}
