package model

//go:generate mockgen -destination=../mocks/mock_api.go -package=mocks . API

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

// ensure API interface is implemented correctly
var _ API = (*apiData)(nil)

// ensure events.EventSink interface is implemented correctly
var _ events.EventSink = (*apiData)(nil)

type API interface {
	GetApiId() uuid.UUID

	GetDisplayName() string
	SetDisplayName(name string)

	GetDescription() string
	SetDescription(desc string)

	GetVersion() Version
	SetVersion(ver Version)

	GetType() ApiType
	SetType(t ApiType)

	GetSystem() *SystemRef
	SetSystemByRef(system System)

	GetAnnotations() Annotations

	Register() bool
}

type apiData struct {
	sink         events.EventSink
	isRegistered bool

	ApiId       uuid.UUID
	DisplayName string
	Description string

	Version Version
	Type    ApiType
	System  *SystemRef

	Annotations Annotations
}

type ApiRef struct {
	API    API
	ApiID  uuid.UUID
	ApiRef *EntityVersion
}

func NewAPI(sink events.EventSink, id uuid.UUID) API {
	retval := &apiData{
		sink:         sink,
		isRegistered: false,
		ApiId:        id,
	}

	retval.Annotations = NewAnnotations(sink)

	return retval
}

func (a *apiData) Register() bool {
	a.isRegistered = true

	return true
}

// GetAnnotations implements [API].
func (a *apiData) GetAnnotations() Annotations {
	return a.Annotations
}

// GetApiId implements [API].
func (a *apiData) GetApiId() uuid.UUID {
	return a.ApiId
}

// GetDescription implements [API].
func (a *apiData) GetDescription() string {
	return a.Description
}

// GetDisplayName implements [API].
func (a *apiData) GetDisplayName() string {
	return a.DisplayName
}

// GetSystem implements [API].
func (a *apiData) GetSystem() *SystemRef {
	return a.System
}

// GetType implements [API].
func (a *apiData) GetType() ApiType {
	return a.Type
}

// GetVersion implements [API].
func (a *apiData) GetVersion() Version {
	return a.Version
}

// SetDescription implements [API].
func (a *apiData) SetDescription(desc string) {
	a.Description = desc

	if a.isRegistered {
		a.sink.Receive(events.APIResource, events.UpdateOperation, a.ApiId, a)
	}
}

// SetDisplayName implements [API].
func (a *apiData) SetDisplayName(name string) {
	a.DisplayName = name

	if a.isRegistered {
		a.sink.Receive(events.APIResource, events.UpdateOperation, a.ApiId, a)
	}
}

// SetSystemByRef implements [API].
func (a *apiData) SetSystemByRef(system System) {
	if system == nil {
		return
	}

	a.System = &SystemRef{
		System:   system,
		SystemId: system.GetSystemId(),
	}
	if a.isRegistered {
		a.sink.Receive(events.APIResource, events.UpdateOperation, a.ApiId, a)
	}
}

// SetType implements [API].
func (a *apiData) SetType(t ApiType) {
	a.Type = t

	if a.isRegistered {
		a.sink.Receive(events.APIResource, events.UpdateOperation, a.ApiId, a)
	}
}

// SetVersion implements [API].
func (a *apiData) SetVersion(ver Version) {
	a.Version = ver

	if a.isRegistered {
		a.sink.Receive(events.APIResource, events.UpdateOperation, a.ApiId, a)
	}
}

// Receive implements [events.EventSink].
func (a *apiData) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, object ...any) error {
	if resType != events.AnnotationsResource {
		return fmt.Errorf("unsupported resource type %v in API event sink. Only Annotations are supported", resType)
	}

	// all changes to annotations are automatically reflected in the parent object as updates
	if a.isRegistered {
		a.sink.Receive(events.APIResource, events.UpdateOperation, a.ApiId, a)
	}

	return nil
}

// MakeTestAPI creates a fully configured test API using the provided sink, ID, name, and API type. Meant to be helper function, not for production use.
func MakeTestAPI(sink events.EventSink, id uuid.UUID, name string, apiType ApiType, version Version) API {
	api := NewAPI(sink, id)
	api.SetDisplayName(name)
	api.SetDescription("a test api")
	api.SetType(apiType)
	api.SetVersion(version)

	return api
}

// MakeTestAPIForModel creates a fully configured test API using the model's event sink. Meant to be helper function, not for production use.
func MakeTestAPIForModel(m Model, id uuid.UUID, name string, apiType ApiType, version Version) API {
	return MakeTestAPI(m.GetSink(), id, name, apiType, version)
}
