/*
Copyright 2025 Lutz Behnke <lutz.behnke@gmx.de>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package events

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type EventManager interface {
	// GetCurrentEventSequenceId returns the current event sequence ID as a string.
	GetCurrentSequenceId(ctx context.Context) (uint64, error)
	IncrementSequenceId(ctx context.Context) error
	GetSink() (EventSink, error)
}

var _ EventManager = (*eventManager)(nil)

type eventManager struct {
	sequenceNumber uint64
}

func NewEventManager() (EventManager, error) {
	return &eventManager{}, nil
}

// GetCurrentSequenceId implements EventManager.
func (e *eventManager) GetCurrentSequenceId(ctx context.Context) (uint64, error) {
	return e.sequenceNumber, nil
}

// IncrementSequenceId implements EventManager.
func (e *eventManager) IncrementSequenceId(ctx context.Context) error {
	e.sequenceNumber++
	return nil
}

// GetSink implements [EventManager].
func (e *eventManager) GetSink() (EventSink, error) {
	return NewListSink(), nil
}

type ResourceType int

const (
	UnknownResourceType ResourceType = iota
	// Phase 0
	ContextResource
	// Phase 1
	SystemResource
	SystemInstanceResource
	APIResource
	APIInstanceResource
	ComponentResource
	ComponentInstanceResource
)

var resourceTypeValues = map[ResourceType]string{
	UnknownResourceType: "UnknownResourceType",

	// Phase 0: Contexts
	ContextResource: "Context",

	// Phase 1:
	SystemResource:            "System",
	SystemInstanceResource:    "SystemInstance",
	APIResource:               "API",
	APIInstanceResource:       "APIInstance",
	ComponentResource:         "Component",
	ComponentInstanceResource: "ComponentInstance",
}

func ParseResourceType(s string) ResourceType {
	for key, val := range resourceTypeValues {
		if val == s {
			return key
		}
	}
	return UnknownResourceType
}

func (t ResourceType) String() string {
	if val, ok := resourceTypeValues[t]; ok {
		return val
	}
	return resourceTypeValues[UnknownResourceType]
}

type Operation int

const (
	UnknownOperation Operation = iota
	CreateOperation
	UpdateOperation
	DeleteOperation
)

var operationValues = map[Operation]string{
	UnknownOperation: "UnknownOperation",
	CreateOperation:  "CreateOperation",
	UpdateOperation:  "UpdateOperation",
	DeleteOperation:  "DeleteOperation",
}

func ParseOperation(s string) Operation {
	for key, val := range operationValues {
		if val == s {
			return key
		}
	}
	return UnknownOperation
}

func (o Operation) String() string {
	if val, ok := operationValues[o]; ok {
		return val
	}
	return operationValues[UnknownOperation]
}

type EventSink interface {
	Receive(resType ResourceType, op Operation, resourceId uuid.UUID, object ...any) error
}

type dummySink struct {
}

// ensure Model interface is implemented correctly
var _ EventSink = (*dummySink)(nil)

func NewDummySink() EventSink {
	return &dummySink{}
}

// Receive implements [EventSink].
func (d *dummySink) Receive(resType ResourceType, op Operation, resourceId uuid.UUID, object ...any) error {
	// just do nothing

	return nil
}

type ListSink struct {
	events []string
}

var _ EventSink = (*ListSink)(nil)

func NewListSink() *ListSink {
	return &ListSink{
		events: make([]string, 0),
	}
}

// Receive implements [EventSink].
func (l *ListSink) Receive(resType ResourceType, op Operation, resourceId uuid.UUID, object ...any) error {
	var eventString string
	if op == DeleteOperation {
		eventString = fmt.Sprintf("%s: %s %s", op.String(), resType.String(), resourceId.String())
	} else {
		eventString = fmt.Sprintf("%s: %s %s: %s", op.String(), resType.String(), resourceId.String(), object)
	}

	l.events = append(l.events, eventString)

	return nil
}

func (l *ListSink) PrintList() {
	for str := range l.events {
		fmt.Println(str)
	}
}

func (l *ListSink) GetList() []string {
	return l.events
}
