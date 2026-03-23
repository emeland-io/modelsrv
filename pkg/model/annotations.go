package model

import (
	"iter"
	"maps"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

//go:generate mockgen -destination=../mocks/mock_annotations.go -package=mocks . Annotations

// ensure Annotations interface is implemented correctly
var _ Annotations = (*annotationsData)(nil)

type Annotations interface {
	Add(key string, value string)
	Delete(key string)
	GetValue(key string) string
	GetKeys() iter.Seq[string]
}

type annotationsData struct {
	sink    events.EventSink
	records map[string]string
}

func NewAnnotations(sink events.EventSink) Annotations {
	return &annotationsData{
		sink:    sink,
		records: make(map[string]string),
	}
}

// Add implements [Annotations].
func (a *annotationsData) Add(key string, value string) {
	if currval, exists := a.records[key]; exists {
		if currval == value {
			// no change
			return
		} else {
			// updating existing value
			a.records[key] = value
			a.sink.Receive(events.AnnotationsResource, events.UpdateOperation, uuid.Nil, map[string]string{key: value})
			return
		}
	}

	a.records[key] = value
	a.sink.Receive(events.AnnotationsResource, events.CreateOperation, uuid.Nil, map[string]string{key: value})
}

// Delete implements [Annotations].
func (a *annotationsData) Delete(key string) {
	delete(a.records, key)
	a.sink.Receive(events.AnnotationsResource, events.DeleteOperation, uuid.Nil, key)
}

// GetValue implements [Annotations].
func (a *annotationsData) GetValue(key string) string {
	return a.records[key]
}

// GetKeys implements [Annotations].
func (a *annotationsData) GetKeys() iter.Seq[string] {
	return maps.Keys(a.records)
}
