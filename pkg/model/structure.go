package model

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
)

var (
	ContextNotFoundError           error = fmt.Errorf("Context not found")
	SystemNotFoundError            error = fmt.Errorf("System not found")
	SystemInstanceNotFoundError    error = fmt.Errorf("System Instance not found")
	ApiNotFoundError               error = fmt.Errorf("API not found")
	ApiInstanceNotFoundError       error = fmt.Errorf("API Instance not found")
	ComponentNotFoundError         error = fmt.Errorf("Component not found")
	ComponentInstanceNotFoundError error = fmt.Errorf("Component Instance not found")
)
var UUIDNotSetError error = fmt.Errorf("resource identifier UUID not set")

type Model interface {
	AddContext(sys *Context) error
	DeleteContextById(id uuid.UUID) error
	GetContexts() ([]*Context, error)
	GetContextById(id uuid.UUID) *Context

	AddSystem(sys *System) error
	DeleteSystemById(id uuid.UUID) error
	GetSystems() ([]*System, error)
	GetSystemById(id uuid.UUID) *System

	AddApi(api *API) error
	DeleteApiById(id uuid.UUID) error
	GetApis() ([]*API, error)
	GetApiById(id uuid.UUID) *API

	AddComponent(comp *Component) error
	DeleteComponentById(id uuid.UUID) error
	GetComponents() ([]*Component, error)
	GetComponentById(id uuid.UUID) *Component

	AddSystemInstance(instance *SystemInstance) error
	DeleteSystemInstanceById(id uuid.UUID) error
	GetSystemInstances() ([]*SystemInstance, error)
	GetSystemInstanceById(id uuid.UUID) *SystemInstance

	AddApiInstance(instance *APIInstance) error
	DeleteApiInstanceById(id uuid.UUID) error
	GetApiInstances() ([]*APIInstance, error)
	GetApiInstanceById(id uuid.UUID) *APIInstance

	AddComponentInstance(instance *ComponentInstance) error
	DeleteComponentInstanceById(id uuid.UUID) error
	GetComponentInstances() ([]*ComponentInstance, error)
	GetComponentInstanceById(id uuid.UUID) *ComponentInstance

	AddFinding(finding *Finding, name string) error
	GetFindings() ([]*Finding, error)
	GetFindingById(id uuid.UUID) *Finding
}

type modelData struct {
	ContextsByUUID   map[uuid.UUID]*Context
	SystemsByUUID    map[uuid.UUID]*System
	APIsByUUID       map[uuid.UUID]*API
	ComponentsByUUID map[uuid.UUID]*Component

	SystemInstancesByUUID    map[uuid.UUID]*SystemInstance
	APIInstancesByUUID       map[uuid.UUID]*APIInstance
	ComponentInstancesByUUID map[uuid.UUID]*ComponentInstance

	FindingsByUUID map[uuid.UUID]*Finding
}

// ensure Model interface is implemented correctly
var _ Model = (*modelData)(nil)

func NewModel() (*modelData, error) {
	model := &modelData{
		ContextsByUUID: make(map[uuid.UUID]*Context),

		SystemsByUUID:    make(map[uuid.UUID]*System),
		APIsByUUID:       make(map[uuid.UUID]*API),
		ComponentsByUUID: make(map[uuid.UUID]*Component),

		SystemInstancesByUUID:    make(map[uuid.UUID]*SystemInstance),
		APIInstancesByUUID:       make(map[uuid.UUID]*APIInstance),
		ComponentInstancesByUUID: make(map[uuid.UUID]*ComponentInstance),

		FindingsByUUID: make(map[uuid.UUID]*Finding),
	}

	return model, nil
}

func (m *modelData) GetCurrentEventSequenceId(ctx context.Context) (string, error) {
	return "forty-two", nil
}

type Context struct {
	ContextId   uuid.UUID
	DisplayName string
	Description string
	Parent      *ContextRef
	Annotations map[string]string
}

type ContextRef struct {
	Context   *Context
	ContextId uuid.UUID
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
	DisplayName string
	Description string
	SystemId    uuid.UUID
	Version     Version
	Abstract    bool
	Parent      SystemRef
	Annotations map[string]string
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
	DisplayName string
	Description string
	ApiId       uuid.UUID
	Version     Version
	Type        ApiType
	System      *SystemRef
	Annotations map[string]string
}

type ApiRef struct {
	API    *API
	ApiID  uuid.UUID
	ApiRef *EntityVersion
}

type Component struct {
	DisplayName string
	Description string
	ComponentId uuid.UUID
	Version     Version
	System      *SystemRef
	Consumes    []ApiRef
	Provides    []ApiRef
	Annotations map[string]string
}

type ComponentRef struct {
	Component    *Component
	ComponentId  uuid.UUID
	ComponentRef *EntityVersion
}

type SystemInstance struct {
	InstanceId  uuid.UUID
	DisplayName string
	SystemRef   *SystemRef
	ContextRef  *ContextRef
	Annotations map[string]string
}

type SystemInstanceRef struct {
	SystemInstance *SystemInstance
	InstanceId     uuid.UUID
}

type APIInstance struct {
	DisplayName    string
	InstanceId     uuid.UUID
	ApiRef         *ApiRef
	SystemInstance *SystemInstanceRef
	Annotations    map[string]string
}

type ComponentInstance struct {
	DisplayName    string
	InstanceId     uuid.UUID
	ComponentRef   *ComponentRef
	SystemInstance *SystemInstanceRef
	Annotations    map[string]string
}

type Finding struct {
	FindingId   uuid.UUID
	Summary     string
	Description string
	Resources   []*ResourceRef
	Annotations map[string]string
}

type ResourceType int

const (
	UnknownResourceType ResourceType = iota
	SystemResource
	SystemInstanceResource
	APIResource
	APIInstanceResource
	ComponentResource
	ComponentInstanceResource
)

var ResourceTypeValues = map[ResourceType]string{
	UnknownResourceType:       "UnknownResourceType",
	SystemResource:            "System",
	SystemInstanceResource:    "SystemInstance",
	APIResource:               "API",
	APIInstanceResource:       "APIInstance",
	ComponentResource:         "Component",
	ComponentInstanceResource: "ComponentInstance",
}

func ParseResourceType(s string) ResourceType {
	for key, val := range ResourceTypeValues {
		if val == s {
			return key
		}
	}
	return UnknownResourceType
}

func (t ResourceType) String() string {
	if val, ok := ResourceTypeValues[t]; ok {
		return val
	}
	return ResourceTypeValues[UnknownResourceType]
}

type ResourceRef struct {
	ResourceId   uuid.UUID
	ResourceType ResourceType
}

// AddContext implements Model.
func (m *modelData) AddContext(sys *Context) error {
	// parse parent ref if set
	if sys.ContextId != uuid.Nil {
		m.ContextsByUUID[sys.ContextId] = sys
	} else {
		return UUIDNotSetError
	}

	return nil
}

// DeleteContextById implements Model.
func (m *modelData) DeleteContextById(id uuid.UUID) error {
	_, exists := m.ContextsByUUID[id]
	if !exists {
		return ContextNotFoundError
	}
	delete(m.ContextsByUUID, id)
	return nil
}

// GetContextById implements Model.
func (m *modelData) GetContextById(id uuid.UUID) *Context {
	context, exists := m.ContextsByUUID[id]
	if !exists {
		return nil
	}
	return context
}

// GetContexts implements Model.
func (m *modelData) GetContexts() ([]*Context, error) {
	contextArr := slices.Collect(maps.Values(m.ContextsByUUID))
	return contextArr, nil
}

// AddSystem implements Model.
func (m *modelData) AddSystem(sys *System) error {

	// parse parent ref if set
	if sys.SystemId != uuid.Nil {
		m.SystemsByUUID[sys.SystemId] = sys
	} else {
		return UUIDNotSetError
	}

	return nil
}

// DeleteSystemByResourceName implements Model.
func (m *modelData) DeleteSystemById(id uuid.UUID) error {
	_, exists := m.SystemsByUUID[id]
	if !exists {
		return SystemNotFoundError
	}

	delete(m.SystemsByUUID, id)
	return nil
}

// GetSystems implements Model.
func (m *modelData) GetSystems() ([]*System, error) {
	systemArr := slices.Collect(maps.Values(m.SystemsByUUID))
	return systemArr, nil
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
func (m *modelData) AddApi(api *API) error {
	if api.ApiId != uuid.Nil {
		m.APIsByUUID[api.ApiId] = api
	} else {
		return UUIDNotSetError
	}
	return nil
}

// AddApiInstance implements Model.
func (m *modelData) AddApiInstance(instance *APIInstance) error {
	if instance.InstanceId != uuid.Nil {
		m.APIInstancesByUUID[instance.InstanceId] = instance
	}
	return nil
}

// AddComponent implements Model.
func (m *modelData) AddComponent(comp *Component) error {
	if comp.ComponentId != uuid.Nil {
		m.ComponentsByUUID[comp.ComponentId] = comp
	}
	return nil
}

// AddComponentInstance implements Model.
func (m *modelData) AddComponentInstance(instance *ComponentInstance) error {
	if instance.InstanceId != uuid.Nil {
		m.ComponentInstancesByUUID[instance.InstanceId] = instance
	}
	return nil
}

// AddSystemInstance implements Model.
func (m *modelData) AddSystemInstance(instance *SystemInstance) error {
	if instance.InstanceId != uuid.Nil {
		m.SystemInstancesByUUID[instance.InstanceId] = instance
	} else {
		return UUIDNotSetError
	}

	// resolve system ref if set
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
func (m *modelData) DeleteApiById(id uuid.UUID) error {
	_, exists := m.APIsByUUID[id]
	if !exists {
		return ApiNotFoundError
	}
	delete(m.APIsByUUID, id)
	return nil
}

// DeleteApiInstanceByResourceName implements Model.
func (m *modelData) DeleteApiInstanceById(id uuid.UUID) error {
	_, exists := m.APIInstancesByUUID[id]
	if !exists {
		return ApiInstanceNotFoundError
	}
	delete(m.APIInstancesByUUID, id)
	return nil
}

// DeleteComponentByResourceName implements Model.
func (m *modelData) DeleteComponentById(id uuid.UUID) error {
	_, exists := m.ComponentsByUUID[id]
	if !exists {
		return ComponentNotFoundError
	}
	delete(m.ComponentsByUUID, id)
	return nil
}

// DeleteComponentInstanceByResourceName implements Model.
func (m *modelData) DeleteComponentInstanceById(id uuid.UUID) error {
	_, exists := m.ComponentInstancesByUUID[id]
	if !exists {
		return ComponentInstanceNotFoundError
	}
	delete(m.ComponentInstancesByUUID, id)
	return nil
}

// DeleteSystemInstanceByResourceName implements Model.
func (m *modelData) DeleteSystemInstanceById(id uuid.UUID) error {
	_, exists := m.SystemInstancesByUUID[id]
	if !exists {
		return SystemInstanceNotFoundError
	}
	delete(m.SystemInstancesByUUID, id)
	return nil
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

// GetComponentById implements Model.
func (m *modelData) GetComponentById(id uuid.UUID) *Component {
	comp, exists := m.ComponentsByUUID[id]
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

// GetSystemInstanceById implements Model.
func (m *modelData) GetSystemInstanceById(id uuid.UUID) *SystemInstance {
	instance, exists := m.SystemInstancesByUUID[id]
	if !exists {
		return nil
	}
	return instance
}

// GetApiInstances implements Model.
func (m *modelData) GetApiInstances() ([]*APIInstance, error) {
	instanceArr := slices.Collect(maps.Values(m.APIInstancesByUUID))
	return instanceArr, nil
}

// GetApis implements Model.
func (m *modelData) GetApis() ([]*API, error) {
	apiArr := slices.Collect(maps.Values(m.APIsByUUID))
	return apiArr, nil
}

// GetComponentInstances implements Model.
func (m *modelData) GetComponentInstances() ([]*ComponentInstance, error) {
	instanceArr := slices.Collect(maps.Values(m.ComponentInstancesByUUID))
	return instanceArr, nil
}

// GetComponents implements Model.
func (m *modelData) GetComponents() ([]*Component, error) {
	componentArr := slices.Collect(maps.Values(m.ComponentsByUUID))
	return componentArr, nil
}

// GetSystemInstances implements Model.
func (m *modelData) GetSystemInstances() ([]*SystemInstance, error) {
	instanceArr := slices.Collect(maps.Values(m.SystemInstancesByUUID))
	return instanceArr, nil
}

// GetFindings implements Model.
func (m modelData) GetFindings() ([]*Finding, error) {
	findingArr := slices.Collect(maps.Values(m.FindingsByUUID))
	return findingArr, nil
}

// AddFinding implements Model.
func (m *modelData) AddFinding(finding *Finding, name string) error {
	m.FindingsByUUID[finding.FindingId] = finding
	return nil
}

// GetFindingById implements Model.
func (m *modelData) GetFindingById(id uuid.UUID) *Finding {
	finding, exists := m.FindingsByUUID[id]
	if !exists {
		return nil
	}
	return finding
}
