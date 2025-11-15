package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var SystemNotFoundError error = fmt.Errorf("System not found")
var ApiNotFoundError error = fmt.Errorf("API not found")
var ComponentNotFoundError error = fmt.Errorf("Component not found")
var SystemInstanceNotFoundError error = fmt.Errorf("System Instance not found")
var ApiInstanceNotFoundError error = fmt.Errorf("API Instance not found")
var ComponentInstanceNotFoundError error = fmt.Errorf("Component Instance not found")

type Model interface {
	AddSystem(sys *System, name string, writer client.SubResourceWriter) error
	DeleteSystemByResourceName(s string) error
	GetSystemByResourceName(s string) *System
	GetSystemById(id uuid.UUID) *System

	AddApi(api *API, name string, writer client.SubResourceWriter) error
	DeleteApiByResourceName(s string) error
	GetApiByResourceName(s string) *API
	GetApiById(id uuid.UUID) *API

	AddComponent(comp *Component, name string, writer client.SubResourceWriter) error
	DeleteComponentByResourceName(s string) error
	GetComponentByResourceName(s string) *Component
	GetComponentById(id uuid.UUID) *Component

	AddSystemInstance(instance *SystemInstance, name string, writer client.SubResourceWriter) error
	DeleteSystemInstanceByResourceName(s string) error
	GetSystemInstanceByResourceName(s string) *SystemInstance
	GetSystemInstanceById(id uuid.UUID) *SystemInstance

	AddApiInstance(instance *APIInstance, name string, writer client.SubResourceWriter) error
	DeleteApiInstanceByResourceName(s string) error
	GetApiInstanceByResourceName(s string) *APIInstance
	GetApiInstanceById(id uuid.UUID) *APIInstance

	AddComponentInstance(instance *ComponentInstance, name string, writer client.SubResourceWriter) error
	DeleteComponentInstanceByResourceName(s string) error
	GetComponentInstanceByResourceName(s string) *ComponentInstance
	GetComponentInstanceById(id uuid.UUID) *ComponentInstance
}

type modelData struct {
	SystemsByName    map[string]*System
	APIsByName       map[string]*API
	ComponentsByName map[string]*Component

	SystemsByUUID    map[uuid.UUID]*System
	APIsByUUID       map[uuid.UUID]*API
	ComponentsByUUID map[uuid.UUID]*Component

	SystemInstancesByName    map[string]*SystemInstance
	ApiInstancesByName       map[string]*APIInstance
	ComponentInstancesByName map[string]*ComponentInstance

	SystemInstancesByUUID    map[uuid.UUID]*SystemInstance
	APIInstancesByUUID       map[uuid.UUID]*APIInstance
	ComponentInstancesByUUID map[uuid.UUID]*ComponentInstance
}

// ensure Model interface is implemented correctly
var _ Model = (*modelData)(nil)

func NewModel() (*modelData, error) {
	model := &modelData{
		SystemsByName:    make(map[string]*System),
		APIsByName:       make(map[string]*API),
		ComponentsByName: make(map[string]*Component),

		SystemsByUUID:    make(map[uuid.UUID]*System),
		APIsByUUID:       make(map[uuid.UUID]*API),
		ComponentsByUUID: make(map[uuid.UUID]*Component),

		SystemInstancesByName:    make(map[string]*SystemInstance),
		ApiInstancesByName:       make(map[string]*APIInstance),
		ComponentInstancesByName: make(map[string]*ComponentInstance),

		SystemInstancesByUUID:    make(map[uuid.UUID]*SystemInstance),
		APIInstancesByUUID:       make(map[uuid.UUID]*APIInstance),
		ComponentInstancesByUUID: make(map[uuid.UUID]*ComponentInstance),
	}

	return model, nil
}

type Version struct {
	Version        string
	AvailableFrom  *time.Time
	DeprecatedFrom *time.Time
	TerminatedFrom *time.Time
}

func (v Version) IsEqual(other Version) bool {
	if v.Version != other.Version {
		return false
	}

	if (v.AvailableFrom == nil) != (other.AvailableFrom == nil) {
		return false
	}
	if v.AvailableFrom != nil && !v.AvailableFrom.Equal(*other.AvailableFrom) {
		return false
	}

	if (v.DeprecatedFrom == nil) != (other.DeprecatedFrom == nil) {
		return false
	}
	if v.DeprecatedFrom != nil && !v.DeprecatedFrom.Equal(*other.DeprecatedFrom) {
		return false
	}

	if (v.TerminatedFrom == nil) != (other.TerminatedFrom == nil) {
		return false
	}
	if v.TerminatedFrom != nil && !v.TerminatedFrom.Equal(*other.TerminatedFrom) {
		return false
	}

	return true
}

type EntityVersion struct {
	Name    string
	Version string
}

type System struct {
	DisplayName  string
	Description  string
	SystemId     uuid.UUID
	Version      Version
	Abstract     bool
	Parent       SystemRef
	Annotations  map[string]string
	statusWriter client.SubResourceWriter
}

type SystemRef struct {
	System    *System
	SystemId  uuid.UUID
	SystemRef *EntityVersion
}

type ApiType int

const (
	Unknown ApiType = iota
	Other
	GraphQL
	gRPC
	OpenAPI
)

var ApiTypeValues = map[ApiType]string{
	Unknown: "Unknown",
	OpenAPI: "OpenAPI",
	GraphQL: "GraphQL",
	gRPC:    "gRPC",
	Other:   "Other",
}

func ParseApiType(s string) ApiType {
	for key, val := range ApiTypeValues {
		if val == s {
			return key
		}
	}
	return Unknown
}

func (t ApiType) String() string {
	if val, ok := ApiTypeValues[t]; ok {
		return val
	}
	return ApiTypeValues[Unknown]
}

type API struct {
	DisplayName  string
	Description  string
	ApiId        uuid.UUID
	Version      Version
	Type         ApiType
	System       *SystemRef
	Annotations  map[string]string
	statusWriter client.SubResourceWriter
}

type ApiRef struct {
	API    *API
	ApiID  uuid.UUID
	ApiRef *EntityVersion
}

type Component struct {
	DisplayName  string
	Description  string
	ComponentId  uuid.UUID
	Version      Version
	System       *SystemRef
	Consumes     []ApiRef
	Provides     []ApiRef
	Annotations  map[string]string
	statusWriter client.SubResourceWriter
}

type SystemInstance struct {
	DisplayName  string
	InstanceId   uuid.UUID
	SystemRef    *SystemRef
	Annotations  map[string]string
	statusWriter client.SubResourceWriter
}

type SystemInstanceRef struct {
	SystemInstance *SystemInstance
	InstanceId     uuid.UUID
}

type APIInstance struct {
	DisplayName    string
	InstanceId     uuid.UUID
	ApiRef         ApiRef
	SystemInstance *SystemInstanceRef
	Annotations    map[string]string
}

type ComponentInstance struct {
	DisplayName    string
	InstanceId     uuid.UUID
	ComponentRef   EntityVersion
	SystemInstance *SystemInstanceRef
	Annotations    map[string]string
}

// AddSystem implements Model.
func (m *modelData) AddSystem(sys *System, name string, statusWriter client.SubResourceWriter) error {
	sys.statusWriter = statusWriter

	m.SystemsByName[name] = sys

	// parse parent ref if set
	if sys.SystemId != uuid.Nil {
		m.SystemsByUUID[sys.SystemId] = sys
	}

	return nil
}

// DeleteSystemByResourceName implements Model.
func (m *modelData) DeleteSystemByResourceName(s string) error {
	_, exists := m.SystemsByName[s]
	if !exists {
		return SystemNotFoundError
	}

	delete(m.SystemsByName, s)
	return nil
}

// GetSystemByResourceName implements Model.
func (m *modelData) GetSystemByResourceName(s string) *System {
	system, exists := m.SystemsByName[s]
	if !exists {
		return nil
	}
	return system
}

// GetSystemById implements Model.
func (m *modelData) GetSystemById(id uuid.UUID) *System {
	system, exists := m.SystemsByUUID[id]
	if !exists {
		return nil
	}
	return system
}

// AddApi implements Model.
func (m *modelData) AddApi(api *API, name string, writer client.SubResourceWriter) error {
	api.statusWriter = writer

	m.APIsByName[name] = api
	if api.ApiId != uuid.Nil {
		m.APIsByUUID[api.ApiId] = api
	}
	return nil
}

// AddApiInstance implements Model.
func (m *modelData) AddApiInstance(instance *APIInstance, name string, writer client.SubResourceWriter) error {
	m.ApiInstancesByName[name] = instance
	if instance.InstanceId != uuid.Nil {
		m.APIInstancesByUUID[instance.InstanceId] = instance
	}
	return nil
}

// AddComponent implements Model.
func (m *modelData) AddComponent(comp *Component, name string, writer client.SubResourceWriter) error {
	comp.statusWriter = writer

	m.ComponentsByName[name] = comp
	if comp.ComponentId != uuid.Nil {
		m.ComponentsByUUID[comp.ComponentId] = comp
	}
	return nil
}

// AddComponentInstance implements Model.
func (m *modelData) AddComponentInstance(instance *ComponentInstance, name string, writer client.SubResourceWriter) error {
	m.ComponentInstancesByName[name] = instance
	if instance.InstanceId != uuid.Nil {
		m.ComponentInstancesByUUID[instance.InstanceId] = instance
	}
	return nil
}

// AddSystemInstance implements Model.
func (m *modelData) AddSystemInstance(instance *SystemInstance, name string, writer client.SubResourceWriter) error {
	instance.statusWriter = writer

	m.SystemInstancesByName[name] = instance

	if instance.InstanceId != uuid.Nil {
		m.SystemInstancesByUUID[instance.InstanceId] = instance
	}

	if instance.SystemRef != nil && instance.SystemRef.SystemId != uuid.Nil {
		system, exists := m.SystemsByUUID[instance.SystemRef.SystemId]
		if exists {
			instance.SystemRef.System = system
		}
		// TODO: create finding: System not found
	}
	return nil
}

// DeleteApiByResourceName implements Model.
func (m *modelData) DeleteApiByResourceName(s string) error {
	api, exists := m.APIsByName[s]
	if !exists {
		return ApiNotFoundError
	}
	delete(m.APIsByName, s)
	if api.ApiId != uuid.Nil {
		delete(m.APIsByUUID, api.ApiId)
	}
	return nil
}

// DeleteApiInstanceByResourceName implements Model.
func (m *modelData) DeleteApiInstanceByResourceName(s string) error {
	instance, exists := m.ApiInstancesByName[s]
	if !exists {
		return ApiInstanceNotFoundError
	}
	delete(m.ApiInstancesByName, s)
	if instance.InstanceId != uuid.Nil {
		delete(m.APIInstancesByUUID, instance.InstanceId)
	}
	return nil
}

// DeleteComponentByResourceName implements Model.
func (m *modelData) DeleteComponentByResourceName(s string) error {
	comp, exists := m.ComponentsByName[s]
	if !exists {
		return ComponentNotFoundError
	}
	delete(m.ComponentsByName, s)
	if comp.ComponentId != uuid.Nil {
		delete(m.ComponentsByUUID, comp.ComponentId)
	}
	return nil
}

// DeleteComponentInstanceByResourceName implements Model.
func (m *modelData) DeleteComponentInstanceByResourceName(s string) error {
	instance, exists := m.ComponentInstancesByName[s]
	if !exists {
		return ComponentInstanceNotFoundError
	}
	delete(m.ComponentInstancesByName, s)
	if instance.InstanceId != uuid.Nil {
		delete(m.ComponentInstancesByUUID, instance.InstanceId)
	}
	return nil
}

// DeleteSystemInstanceByResourceName implements Model.
func (m *modelData) DeleteSystemInstanceByResourceName(s string) error {
	instance, exists := m.SystemInstancesByName[s]
	if !exists {
		return SystemInstanceNotFoundError
	}
	delete(m.SystemInstancesByName, s)
	if instance.InstanceId != uuid.Nil {
		delete(m.SystemInstancesByUUID, instance.InstanceId)
	}
	return nil
}

// GetApiByResourceName implements Model.
func (m *modelData) GetApiByResourceName(s string) *API {
	api, exists := m.APIsByName[s]
	if !exists {
		return nil
	}
	return api
}

// GetApiById implements Model.
func (m *modelData) GetApiById(id uuid.UUID) *API {
	api, exists := m.APIsByUUID[id]
	if !exists {
		return nil
	}
	return api
}

// GetApiInstanceById implements Model.
func (m *modelData) GetApiInstanceById(id uuid.UUID) *APIInstance {
	instance, exists := m.APIInstancesByUUID[id]
	if !exists {
		return nil
	}
	return instance
}

// GetApiInstanceByResourceName implements Model.
func (m *modelData) GetApiInstanceByResourceName(s string) *APIInstance {
	instance, exists := m.ApiInstancesByName[s]
	if !exists {
		return nil
	}
	return instance
}

// GetComponentById implements Model.
func (m *modelData) GetComponentById(id uuid.UUID) *Component {
	comp, exists := m.ComponentsByUUID[id]
	if !exists {
		return nil
	}
	return comp
}

// GetComponentByResourceName implements Model.
func (m *modelData) GetComponentByResourceName(s string) *Component {
	comp, exists := m.ComponentsByName[s]
	if !exists {
		return nil
	}
	return comp
}

// GetComponentInstanceById implements Model.
func (m *modelData) GetComponentInstanceById(id uuid.UUID) *ComponentInstance {
	instance, exists := m.ComponentInstancesByUUID[id]
	if !exists {
		return nil
	}
	return instance
}

// GetComponentInstanceByResourceName implements Model.
func (m *modelData) GetComponentInstanceByResourceName(s string) *ComponentInstance {
	instance, exists := m.ComponentInstancesByName[s]
	if !exists {
		return nil
	}
	return instance
}

// GetSystemInstanceById implements Model.
func (m *modelData) GetSystemInstanceById(id uuid.UUID) *SystemInstance {
	instance, exists := m.SystemInstancesByUUID[id]
	if !exists {
		return nil
	}
	return instance
}

// GetSystemInstanceByResourceName implements Model.
func (m *modelData) GetSystemInstanceByResourceName(s string) *SystemInstance {
	instance, exists := m.SystemInstancesByName[s]
	if !exists {
		return nil
	}
	return instance
}
