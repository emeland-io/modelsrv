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

type Subscriber interface {
	// Notify notifies the subscriber of an event.
	Notify(ctx context.Context, event *Event) error

	// GetURL returns the URL of the subscriber, under which it accepts events.
	GetURL() string

	// GetId returns the unique ID of the subscriber.
	GetId() uuid.UUID

	// GetStatus returns the current status of the subscriber (e.g. active, inactive, etc.)
	GetStatus() string
}

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
	GetSubscribers() []Subscriber
	// AddSubscriber adds a new subscriber by url.
	AddSubscriber(subUrl string) error
	// RemoveSubscriber removes a subscriber by URL.
	RemoveSubscriber(subUrl string) error
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

	//Phase 5
	FindingResource:     "Finding",
	FindingTypeResource: "FindingType",

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
	Object       any
}

func (e Event) String() string {
	if e.Operation == DeleteOperation {
		return fmt.Sprintf("%s: %s %s", e.Operation.String(), e.ResourceType.String(), e.ResourceId.String())
	} else {
		return fmt.Sprintf("%s: %s %s: %v", e.Operation.String(), e.ResourceType.String(), e.ResourceId.String(), e.Object)
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
		Object:       objects,
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
