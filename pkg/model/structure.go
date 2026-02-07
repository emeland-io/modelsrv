package model

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

var (
	ContextNotFoundError           error = fmt.Errorf("Context not found")
	ContextTypeNotFoundError       error = fmt.Errorf("Context Type not found")
	SystemNotFoundError            error = fmt.Errorf("System not found")
	SystemInstanceNotFoundError    error = fmt.Errorf("System Instance not found")
	ApiNotFoundError               error = fmt.Errorf("API not found")
	ApiInstanceNotFoundError       error = fmt.Errorf("API Instance not found")
	ComponentNotFoundError         error = fmt.Errorf("Component not found")
	ComponentInstanceNotFoundError error = fmt.Errorf("Component Instance not found")
	FindingNotFoundError           error = fmt.Errorf("Finding not found")
	FindingTypeNotFoundError       error = fmt.Errorf("Finding Type not found")

	UUIDNotSetError error = fmt.Errorf("resource identifier UUID not set")
)

type Model interface {
	getData() *modelData

	AddContext(context Context) error
	DeleteContextById(id uuid.UUID) error
	GetContexts() ([]Context, error)
	GetContextById(id uuid.UUID) Context

	AddContextType(contextType ContextType) error
	DeleteContextTypeById(id uuid.UUID) error
	GetContextTypes() ([]ContextType, error)
	GetContextTypeById(id uuid.UUID) ContextType

	AddSystem(sys System) error
	DeleteSystemById(id uuid.UUID) error
	GetSystems() ([]System, error)
	GetSystemById(id uuid.UUID) System

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

	AddApiInstance(instance *ApiInstance) error
	DeleteApiInstanceById(id uuid.UUID) error
	GetApiInstances() ([]*ApiInstance, error)
	GetApiInstanceById(id uuid.UUID) *ApiInstance

	AddComponentInstance(instance *ComponentInstance) error
	DeleteComponentInstanceById(id uuid.UUID) error
	GetComponentInstances() ([]*ComponentInstance, error)
	GetComponentInstanceById(id uuid.UUID) *ComponentInstance

	AddFinding(finding *Finding, name string) error
	DeleteFindingById(id uuid.UUID) error
	GetFindings() ([]*Finding, error)
	GetFindingById(id uuid.UUID) *Finding

	AddFindingType(findingType FindingType) error
	DeleteFindingTypeById(id uuid.UUID) error
	GetFindingTypes() ([]FindingType, error)
	GetFindingTypeById(id uuid.UUID) FindingType
}

type modelData struct {
	sink events.EventSink

	contextsByUUID     map[uuid.UUID]*contextData
	contextTypesByUUID map[uuid.UUID]ContextType
	contextsCache      []Context
	systemsByUUID      map[uuid.UUID]System
	apisByUUID         map[uuid.UUID]*API
	componentsByUUID   map[uuid.UUID]*Component

	systemInstancesByUUID    map[uuid.UUID]*SystemInstance
	apiInstancesByUUID       map[uuid.UUID]*ApiInstance
	componentInstancesByUUID map[uuid.UUID]*ComponentInstance

	findingsByUUID     map[uuid.UUID]*Finding
	findingTypesByUUID map[uuid.UUID]FindingType
}

// ensure Model interface is implemented correctly
var _ Model = (*modelData)(nil)

func NewModel(sink events.EventSink) (*modelData, error) {
	if sink == nil {
		return nil, fmt.Errorf("event sink must not be nil")
	}

	model := &modelData{
		sink: sink,

		contextsByUUID:     make(map[uuid.UUID]*contextData),
		contextTypesByUUID: make(map[uuid.UUID]ContextType),

		systemsByUUID:    make(map[uuid.UUID]System),
		apisByUUID:       make(map[uuid.UUID]*API),
		componentsByUUID: make(map[uuid.UUID]*Component),

		systemInstancesByUUID:    make(map[uuid.UUID]*SystemInstance),
		apiInstancesByUUID:       make(map[uuid.UUID]*ApiInstance),
		componentInstancesByUUID: make(map[uuid.UUID]*ComponentInstance),

		findingsByUUID:     make(map[uuid.UUID]*Finding),
		findingTypesByUUID: make(map[uuid.UUID]FindingType),
	}

	return model, nil
}

func (m *modelData) GetCurrentEventSequenceId(ctx context.Context) (string, error) {
	return "forty-two", nil
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

type ApiType int

const (
	Unknown ApiType = iota
	Other
	GraphQL
	GRPC
	OpenAPI
)

var ApiTypeValues = map[ApiType]string{
	Unknown: "Unknown",
	OpenAPI: "OpenAPI",
	GraphQL: "GraphQL",
	GRPC:    "GRPC",
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

type ApiInstance struct {
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

type ResourceRef struct {
	ResourceId   uuid.UUID
	ResourceType events.ResourceType
}

func (m *modelData) getData() *modelData {
	return m
}

// AddContext implements Model.
func (m *modelData) AddContext(context Context) error {

	// invalidate the cache
	m.contextsCache = nil

	// TODO: parse parent ref if set

	if context.GetContextId() == uuid.Nil {
		return UUIDNotSetError
	}

	op := events.CreateOperation

	// check if this would overwrite an existing entry -> an update
	if _, ok := m.contextsByUUID[context.GetContextId()]; ok {
		op = events.UpdateOperation
	}

	m.sink.Receive(events.ContextResource, op, context.GetContextId(), context)

	m.contextsByUUID[context.GetContextId()] = context.getData()

	// mark Context as registered to activate sending events when updating fields
	context.getData().isRegistered = true

	return nil
}

// DeleteContextById implements Model.
func (m *modelData) DeleteContextById(id uuid.UUID) error {
	_, exists := m.contextsByUUID[id]
	if !exists {
		return ContextNotFoundError
	}

	// invalidate the cache
	m.contextsCache = nil

	delete(m.contextsByUUID, id)

	m.sink.Receive(events.ContextResource, events.DeleteOperation, id)

	return nil
}

// GetContextById implements Model.
func (m *modelData) GetContextById(id uuid.UUID) Context {
	context, exists := m.contextsByUUID[id]
	if !exists {
		return nil
	}
	return context
}

// GetContexts implements Model.
func (m *modelData) GetContexts() ([]Context, error) {
	/* since this operation would require O(n) and is therfore potentially quite costly, lets cache that
	   Any write operations to contextsByUUID must invalidate that
	*/
	if m.contextsCache != nil {
		return m.contextsCache, nil
	}

	contextArr := make([]Context, 0)
	for context := range maps.Values(m.contextsByUUID) {
		contextArr = append(contextArr, Context(context))
	}
	m.contextsCache = contextArr

	return []Context(contextArr), nil
}

// AddContextType implements [Model].
func (m *modelData) AddContextType(contextType ContextType) error {
	if contextType.GetContextTypeId() == uuid.Nil {
		return UUIDNotSetError
	}

	op := events.CreateOperation

	// check if this would overwrite an existing entry -> an update
	if _, ok := m.contextTypesByUUID[contextType.GetContextTypeId()]; ok {
		op = events.UpdateOperation
	}
	m.sink.Receive(events.ContextTypeResource, op, contextType.GetContextTypeId(), contextType)

	m.contextTypesByUUID[contextType.GetContextTypeId()] = contextType

	// mark ContextType as registered to activate sending events when updating fields
	contextType.getData().isRegistered = true

	return nil
}

// DeleteContextTypeById implements [Model].
func (m *modelData) DeleteContextTypeById(id uuid.UUID) error {
	_, exists := m.contextTypesByUUID[id]
	if !exists {
		return ContextTypeNotFoundError
	}

	delete(m.contextTypesByUUID, id)

	m.sink.Receive(events.ContextTypeResource, events.DeleteOperation, id)

	return nil
}

// GetContextTypeById implements [Model].
func (m *modelData) GetContextTypeById(id uuid.UUID) ContextType {
	contextType, exists := m.contextTypesByUUID[id]
	if !exists {
		return nil
	}
	return contextType
}

// GetContextTypes implements [Model].
func (m *modelData) GetContextTypes() ([]ContextType, error) {
	contextTypeArr := slices.Collect(maps.Values(m.contextTypesByUUID))
	return contextTypeArr, nil
}

// AddSystem implements Model.
func (m *modelData) AddSystem(sys System) error {

	// parse parent ref if set
	if sys.GetSystemId() == uuid.Nil {
		return UUIDNotSetError
	}

	op := events.CreateOperation

	// check if this would overwrite an existing entry -> an update
	if _, ok := m.systemsByUUID[sys.GetSystemId()]; ok {
		op = events.UpdateOperation
	}
	m.sink.Receive(events.SystemResource, op, sys.GetSystemId(), sys)

	m.systemsByUUID[sys.GetSystemId()] = sys

	// mark System as registered to activate sending events when updating fields
	sys.getData().isRegistered = true

	return nil
}

// DeleteSystemByResourceName implements Model.
func (m *modelData) DeleteSystemById(id uuid.UUID) error {
	_, exists := m.systemsByUUID[id]
	if !exists {
		return SystemNotFoundError
	}

	delete(m.systemsByUUID, id)

	m.sink.Receive(events.SystemResource, events.DeleteOperation, id)

	return nil
}

// GetSystems implements Model.
func (m *modelData) GetSystems() ([]System, error) {
	systemArr := slices.Collect(maps.Values(m.systemsByUUID))
	return systemArr, nil
}

// GetSystemById implements Model.
func (m *modelData) GetSystemById(id uuid.UUID) System {
	system, exists := m.systemsByUUID[id]
	if !exists {
		return nil
	}
	return system
}

// AddApi implements Model.
func (m *modelData) AddApi(api *API) error {
	if api.ApiId != uuid.Nil {
		m.apisByUUID[api.ApiId] = api
	} else {
		return UUIDNotSetError
	}
	return nil
}

// AddApiInstance implements Model.
func (m *modelData) AddApiInstance(instance *ApiInstance) error {
	if instance.InstanceId != uuid.Nil {
		m.apiInstancesByUUID[instance.InstanceId] = instance
	}
	return nil
}

// AddComponent implements Model.
func (m *modelData) AddComponent(comp *Component) error {
	if comp.ComponentId != uuid.Nil {
		m.componentsByUUID[comp.ComponentId] = comp
	}
	return nil
}

// AddComponentInstance implements Model.
func (m *modelData) AddComponentInstance(instance *ComponentInstance) error {
	if instance.InstanceId != uuid.Nil {
		m.componentInstancesByUUID[instance.InstanceId] = instance
	}
	return nil
}

// AddSystemInstance implements Model.
func (m *modelData) AddSystemInstance(instance *SystemInstance) error {
	if instance.InstanceId != uuid.Nil {
		m.systemInstancesByUUID[instance.InstanceId] = instance
	} else {
		return UUIDNotSetError
	}

	// resolve system ref if set
	if instance.SystemRef != nil && instance.SystemRef.SystemId != uuid.Nil {
		system, exists := m.systemsByUUID[instance.SystemRef.SystemId]
		if exists {
			instance.SystemRef.System = system
		}
		// TODO: create finding: System not found
	}
	return nil
}

// DeleteApiByResourceName implements Model.
func (m *modelData) DeleteApiById(id uuid.UUID) error {
	_, exists := m.apisByUUID[id]
	if !exists {
		return ApiNotFoundError
	}
	delete(m.apisByUUID, id)
	return nil
}

// DeleteApiInstanceByResourceName implements Model.
func (m *modelData) DeleteApiInstanceById(id uuid.UUID) error {
	_, exists := m.apiInstancesByUUID[id]
	if !exists {
		return ApiInstanceNotFoundError
	}
	delete(m.apiInstancesByUUID, id)
	return nil
}

// DeleteComponentByResourceName implements Model.
func (m *modelData) DeleteComponentById(id uuid.UUID) error {
	_, exists := m.componentsByUUID[id]
	if !exists {
		return ComponentNotFoundError
	}
	delete(m.componentsByUUID, id)
	return nil
}

// DeleteComponentInstanceByResourceName implements Model.
func (m *modelData) DeleteComponentInstanceById(id uuid.UUID) error {
	_, exists := m.componentInstancesByUUID[id]
	if !exists {
		return ComponentInstanceNotFoundError
	}
	delete(m.componentInstancesByUUID, id)
	return nil
}

// DeleteSystemInstanceByResourceName implements Model.
func (m *modelData) DeleteSystemInstanceById(id uuid.UUID) error {
	_, exists := m.systemInstancesByUUID[id]
	if !exists {
		return SystemInstanceNotFoundError
	}
	delete(m.systemInstancesByUUID, id)
	return nil
}

// GetApiById implements Model.
func (m *modelData) GetApiById(id uuid.UUID) *API {
	api, exists := m.apisByUUID[id]
	if !exists {
		return nil
	}
	return api
}

// GetApiInstanceById implements Model.
func (m *modelData) GetApiInstanceById(id uuid.UUID) *ApiInstance {
	instance, exists := m.apiInstancesByUUID[id]
	if !exists {
		return nil
	}
	return instance
}

// GetComponentById implements Model.
func (m *modelData) GetComponentById(id uuid.UUID) *Component {
	comp, exists := m.componentsByUUID[id]
	if !exists {
		return nil
	}
	return comp
}

// GetComponentInstanceById implements Model.
func (m *modelData) GetComponentInstanceById(id uuid.UUID) *ComponentInstance {
	instance, exists := m.componentInstancesByUUID[id]
	if !exists {
		return nil
	}
	return instance
}

// GetSystemInstanceById implements Model.
func (m *modelData) GetSystemInstanceById(id uuid.UUID) *SystemInstance {
	instance, exists := m.systemInstancesByUUID[id]
	if !exists {
		return nil
	}
	return instance
}

// GetApiInstances implements Model.
func (m *modelData) GetApiInstances() ([]*ApiInstance, error) {
	instanceArr := slices.Collect(maps.Values(m.apiInstancesByUUID))
	return instanceArr, nil
}

// GetApis implements Model.
func (m *modelData) GetApis() ([]*API, error) {
	apiArr := slices.Collect(maps.Values(m.apisByUUID))
	return apiArr, nil
}

// GetComponentInstances implements Model.
func (m *modelData) GetComponentInstances() ([]*ComponentInstance, error) {
	instanceArr := slices.Collect(maps.Values(m.componentInstancesByUUID))
	return instanceArr, nil
}

// GetComponents implements Model.
func (m *modelData) GetComponents() ([]*Component, error) {
	componentArr := slices.Collect(maps.Values(m.componentsByUUID))
	return componentArr, nil
}

// GetSystemInstances implements Model.
func (m *modelData) GetSystemInstances() ([]*SystemInstance, error) {
	instanceArr := slices.Collect(maps.Values(m.systemInstancesByUUID))
	return instanceArr, nil
}

// GetFindings implements Model.
func (m modelData) GetFindings() ([]*Finding, error) {
	findingArr := slices.Collect(maps.Values(m.findingsByUUID))
	return findingArr, nil
}

// AddFinding implements Model.
func (m *modelData) AddFinding(finding *Finding, name string) error {
	m.findingsByUUID[finding.FindingId] = finding
	return nil
}

// DeleteFindingById implements [Model].
func (m *modelData) DeleteFindingById(id uuid.UUID) error {
	panic("unimplemented")
}

// GetFindingById implements Model.
func (m *modelData) GetFindingById(id uuid.UUID) *Finding {
	finding, exists := m.findingsByUUID[id]
	if !exists {
		return nil
	}
	return finding
}

// AddFindingType implements [Model].
func (m *modelData) AddFindingType(findingType FindingType) error {

	// parse parent ref if set
	if findingType.GetFindingTypeId() == uuid.Nil {
		return UUIDNotSetError
	}

	op := events.CreateOperation

	// check if this would overwrite an existing entry -> an update
	if _, ok := m.findingTypesByUUID[findingType.GetFindingTypeId()]; ok {
		op = events.UpdateOperation
	}
	m.sink.Receive(events.FindingTypeResource, op, findingType.GetFindingTypeId(), findingType)

	m.findingTypesByUUID[findingType.GetFindingTypeId()] = findingType

	// mark FindingType as registered to activate sending events when updating fields
	findingType.getData().isRegistered = true

	return nil

}

// DeleteFindingTypeById implements [Model].
func (m *modelData) DeleteFindingTypeById(id uuid.UUID) error {
	_, exists := m.findingTypesByUUID[id]
	if !exists {
		return FindingTypeNotFoundError
	}

	delete(m.findingTypesByUUID, id)

	m.sink.Receive(events.FindingTypeResource, events.DeleteOperation, id)

	return nil
}

// GetFindingTypeById implements [Model].
func (m *modelData) GetFindingTypeById(id uuid.UUID) FindingType {
	findingType, exists := m.findingTypesByUUID[id]
	if !exists {
		return nil
	}
	return findingType
}

// GetFindingTypes implements [Model].
func (m *modelData) GetFindingTypes() ([]FindingType, error) {
	findingTypeArr := slices.Collect(maps.Values(m.findingTypesByUUID))
	return findingTypeArr, nil
}
