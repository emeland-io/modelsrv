package eventforwarder

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/internal/util"
	"go.emeland.io/modelsrv/pkg/client"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/annotations"
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
			jsonStr, err := json.Marshal(convertContextToDTO(obj.(mdlctx.Context)))
			if err != nil {
				return fmt.Errorf("failed to marshal context: %w", err)
			}
			objJsons = append(objJsons, string(jsonStr))
		default:
			return fmt.Errorf("unknown type %v", obj)
			// objJsons = append(objJsons, "json_string_placeholder")
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

func convertContextToDTO(context mdlctx.Context) client.Context {
	description := context.GetDescription()

	retval := client.Context{
		ContextId:   context.GetContextId(),
		DisplayName: context.GetDisplayName(),
		Description: &description,
		Annotations: convertAnnotationsToDTO(context.GetAnnotations()),
	}

	return retval
}

func convertAnnotationsToDTO(modelAnnons annotations.Annotations) *[]client.Annotation {

	retval := make([]client.Annotation, 0)
	for key := range modelAnnons.GetKeys() {
		retval = append(retval, client.Annotation{Key: key, Value: modelAnnons.GetValue(key)})
	}
	return &retval
}
