package eventforwarder

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/internal/util"
	"go.emeland.io/modelsrv/pkg/events"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
)

type event struct {
	resourceType events.ResourceType
	operation    events.Operation
	resourceId   uuid.UUID
	objectJson   []string
}

type eventForwarder struct {
	queue *util.CircularQueue[event]
}

// ensure EventSink interface is implemented correctly
var _ events.EventSink = (*eventForwarder)(nil)

func NewEventForwarder(queueSize int) *eventForwarder {
	return &eventForwarder{
		queue: util.NewCircularQueue[event](queueSize),
	}
}

// Receive implements [events.EventSink].
func (e *eventForwarder) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, objects ...any) error {
	objJsons := make([]string, 0, len(objects))
	for _, obj := range objects {
		switch o := obj.(type) {
		case string:
			objJsons = append(objJsons, o)
		case mdlctx.Context:
			jsonStr, err := json.Marshal(oapi.ContextToDto(o))
			if err != nil {
				return fmt.Errorf("failed to marshal context: %w", err)
			}
			objJsons = append(objJsons, string(jsonStr))
		default:
			return fmt.Errorf("unknown type %v", obj)
		}
	}
	event := event{
		resourceType: resType,
		operation:    op,
		resourceId:   resourceId,
		objectJson:   objJsons,
	}
	return e.queue.Enqueue(event)
}
