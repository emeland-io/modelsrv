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

// EventManager manages event sequence IDs and event sinks.
type EventManager interface {
	// GetCurrentEventSequenceId returns the current event sequence ID as a string.
	GetCurrentSequenceId(ctx context.Context) (uint64, error)
	IncrementSequenceId(ctx context.Context) error

	// SetSinkFactory sets the factory function to create new EventSinks.
	SetSinkFactory(factory func() (EventSink, error))
	// GetSink returns a new EventSink created by the sink factory set with [SetSinkFactory].
	// If no factory has been set, a default [ListSink] is returned.
	GetSink() (EventSink, error)

	// GetSubscribers returns a list of current subscribers.
	GetSubscribers() []string
	// AddSubscriber adds a new subscriber by URL.
	AddSubscriber(url string) error
	// RemoveSubscriber removes a subscriber by URL.
	RemoveSubscriber(url string) error
}

var _ EventManager = (*eventManager)(nil)

type eventManager struct {
	sequenceNumber uint64
	subscribers    []string
	sinkFactory    func() (EventSink, error)
}

func NewEventManager() (EventManager, error) {
	retval := &eventManager{
		sequenceNumber: 0,
		sinkFactory:    func() (EventSink, error) { return NewListSink(), nil },
	}
	return retval, nil

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

// GetSink returns a new EventSink created by the sink factory set with [SetSinkFactory].
// If no factory has been set, a default [ListSink] is returned.
//
// SetSinkFactory implements [EventManager].
func (e *eventManager) SetSinkFactory(factory func() (EventSink, error)) {
	e.sinkFactory = factory
}

// GetSink implements [EventManager].
func (e *eventManager) GetSink() (EventSink, error) {
	return NewListSink(), nil
}

// AddSubscriber implements [EventManager].
// adding the same subscriber URL again will result in only one entry in the subscriber list.
func (e *eventManager) AddSubscriber(url string) error {
	for _, sub := range e.subscribers {
		if sub == url {
			// already exists
			return nil
		}
	}
	e.subscribers = append(e.subscribers, url)
	return nil
}

// GetSubscribers implements [EventManager].
func (e *eventManager) GetSubscribers() []string {
	return e.subscribers
}

// RemoveSubscriber implements [EventManager].
// TODO: this function requires O(n) time. If the subscriber list becomes long, consider using a map for O(1) removal.
func (e *eventManager) RemoveSubscriber(url string) error {
	for i, sub := range e.subscribers {
		if sub == url {
			// remove subscriber
			e.subscribers = append(e.subscribers[:i], e.subscribers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("subscriber %s not found", url)
}

type ResourceType int

const (
	UnknownResourceType ResourceType = iota

	NodeResource
	NodeTypeResource

	// Phase 0
	ContextResource
	ContextTypeResource
	// Phase 1
	SystemResource
	SystemInstanceResource
	APIResource
	APIInstanceResource
	ComponentResource
	ComponentInstanceResource

	//Phase 5
	FindingResource
	FindingTypeResource

	// Value objects
	AnnotationsResource
)

var resourceTypeValues = map[ResourceType]string{
	UnknownResourceType: "UnknownResourceType",

	NodeResource:     "Node",
	NodeTypeResource: "NodeType",

	// Phase 0: Contexts
	ContextResource:     "Context",
	ContextTypeResource: "ContextType",

	// Phase 1:
	SystemResource:            "System",
	SystemInstanceResource:    "SystemInstance",
	APIResource:               "API",
	APIInstanceResource:       "APIInstance",
	ComponentResource:         "Component",
	ComponentInstanceResource: "ComponentInstance",

	// Value objects
	AnnotationsResource: "Annotations",
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

type Event struct {
	ResourceType ResourceType
	Operation    Operation
	ResourceId   uuid.UUID
	Objects      []any
}

func (e Event) String() string {
	if e.Operation == DeleteOperation {
		return fmt.Sprintf("%s: %s %s", e.Operation.String(), e.ResourceType.String(), e.ResourceId.String())
	} else {
		return fmt.Sprintf("%s: %s %s: %v", e.Operation.String(), e.ResourceType.String(), e.ResourceId.String(), e.Objects)
	}
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
	eventsTxts []string
	events     []Event
}

var _ EventSink = (*ListSink)(nil)

func NewListSink() *ListSink {
	return &ListSink{
		eventsTxts: make([]string, 0),
		events:     make([]Event, 0),
	}
}

// Receive implements [EventSink].
func (l *ListSink) Receive(resType ResourceType, op Operation, resourceId uuid.UUID, objects ...any) error {
	currEvent := Event{
		ResourceType: resType,
		Operation:    op,
		ResourceId:   resourceId,
		Objects:      objects,
	}
	l.events = append(l.events, currEvent)

	l.eventsTxts = append(l.eventsTxts, currEvent.String())

	return nil
}

func (l *ListSink) PrintList() {
	for str := range l.eventsTxts {
		fmt.Println(str)
	}
}

func (l *ListSink) GetList() []string {
	return l.eventsTxts
}

func (l *ListSink) GetEvents() []Event {
	return l.events
}
