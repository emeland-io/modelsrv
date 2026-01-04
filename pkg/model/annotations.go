package model

import (
	"iter"
	"maps"
)

// ensure Annotations interface is implemented correctly
var _ Annotations = (*annotationsData)(nil)

type Annotations interface {
	Add(key string, value string)
	Delete(key string)
	GetValue(key string) string
	GetKeys() iter.Seq[string]
	getData() *annotationsData
}

type annotationsData struct {
	model   *modelData
	records map[string]string
}

func NewAnnotations(model *modelData) Annotations {
	return &annotationsData{
		model:   model,
		records: make(map[string]string),
	}
}

// Add implements [Annotations].
func (a *annotationsData) Add(key string, value string) {
	a.records[key] = value
}

// Delete implements [Annotations].
func (a *annotationsData) Delete(key string) {
	delete(a.records, key)
}

// GetValue implements [Annotations].
func (a *annotationsData) GetValue(key string) string {
	return a.records[key]
}

// GetKeys implements [Annotations].
func (a *annotationsData) GetKeys() iter.Seq[string] {
	return maps.Keys(a.records)
}

func (a *annotationsData) getData() *annotationsData {
	return a
}
