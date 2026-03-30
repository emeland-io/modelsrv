//go:generate go run generate_events.go
package model

//go:generate ../../bin/mockgen -destination=../mocks/mock_model.go -package=mocks . Model
//go:generate ../../bin/mockgen -destination=../mocks/mock_finding_type.go -package=mocks . FindingType

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
	ErrNodeNotFound              error = fmt.Errorf("Node not found")
	ErrNodeTypeNotFound          error = fmt.Errorf("Node Type not found")
	ErrContextNotFound           error = fmt.Errorf("Context not found")
	ErrContextTypeNotFound       error = fmt.Errorf("Context Type not found")
	ErrSystemNotFound            error = fmt.Errorf("System not found")
	ErrSystemInstanceNotFound    error = fmt.Errorf("System Instance not found")
	ErrApiNotFound               error = fmt.Errorf("API not found")
	ErrApiInstanceNotFound       error = fmt.Errorf("API Instance not found")
	ErrComponentNotFound         error = fmt.Errorf("Component not found")
	ErrComponentInstanceNotFound error = fmt.Errorf("Component Instance not found")
	ErrFindingNotFound           error = fmt.Errorf("Finding not found")
	ErrFindingTypeNotFound       error = fmt.Errorf("Finding Type not found")

	ErrUUIDNotSet error = fmt.Errorf("resource identifier UUID not set")
)

type Model interface {
	EventApplier
	GetSink() events.EventSink

	AddNode(node Node) error
	DeleteNodeById(id uuid.UUID) error
	GetNodes() ([]Node, error)
	GetNodeById(id uuid.UUID) Node

	AddNodeType(nodeType NodeType) error
	DeleteNodeTypeById(id uuid.UUID) error
	GetNodeTypes() ([]NodeType, error)
	GetNodeTypeById(id uuid.UUID) NodeType

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

	AddApi(api API) error
	DeleteApiById(id uuid.UUID) error
	GetApis() ([]API, error)
	GetApiById(id uuid.UUID) API

	// ApiRefByID builds an [ApiRef] for a registered API, or nil if not found.
	ApiRefByID(apiId uuid.UUID) *ApiRef

	AddComponent(comp Component) error
	DeleteComponentById(id uuid.UUID) error
	GetComponents() ([]Component, error)
	GetComponentById(id uuid.UUID) Component

	AddSystemInstance(instance SystemInstance) error
	DeleteSystemInstanceById(id uuid.UUID) error
	GetSystemInstances() ([]SystemInstance, error)
	GetSystemInstanceById(id uuid.UUID) SystemInstance

	// SystemInstanceRefByID builds a [SystemInstanceRef] for a registered instance, or nil if not found.
	SystemInstanceRefByID(instanceId uuid.UUID) *SystemInstanceRef

	AddApiInstance(instance ApiInstance) error
	DeleteApiInstanceById(id uuid.UUID) error
	GetApiInstances() ([]ApiInstance, error)
	GetApiInstanceById(id uuid.UUID) ApiInstance

	AddComponentInstance(instance ComponentInstance) error
	DeleteComponentInstanceById(id uuid.UUID) error
	GetComponentInstances() ([]ComponentInstance, error)
	GetComponentInstanceById(id uuid.UUID) ComponentInstance

	AddFinding(finding Finding, name string) error
	DeleteFindingById(id uuid.UUID) error
	GetFindings() ([]Finding, error)
	GetFindingById(id uuid.UUID) Finding

	AddFindingType(findingType FindingType) error
	DeleteFindingTypeById(id uuid.UUID) error
	GetFindingTypes() ([]FindingType, error)
	GetFindingTypeById(id uuid.UUID) FindingType
}

type modelData struct {
	sink events.EventSink

	nodeTypesByUUID map[uuid.UUID]NodeType
	nodesByUUID     map[uuid.UUID]Node

	contextsByUUID     map[uuid.UUID]Context
	contextTypesByUUID map[uuid.UUID]ContextType
	contextsCache      []Context

	systemsByUUID    map[uuid.UUID]System
	apisByUUID       map[uuid.UUID]API
	componentsByUUID map[uuid.UUID]Component

	systemInstancesByUUID    map[uuid.UUID]SystemInstance
	apiInstancesByUUID       map[uuid.UUID]ApiInstance
	componentInstancesByUUID map[uuid.UUID]ComponentInstance

	findingsByUUID     map[uuid.UUID]Finding
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

		nodesByUUID:     make(map[uuid.UUID]Node),
		nodeTypesByUUID: make(map[uuid.UUID]NodeType),

		contextsByUUID:     make(map[uuid.UUID]Context),
		contextTypesByUUID: make(map[uuid.UUID]ContextType),

		systemsByUUID:    make(map[uuid.UUID]System),
		apisByUUID:       make(map[uuid.UUID]API),
		componentsByUUID: make(map[uuid.UUID]Component),

		systemInstancesByUUID:    make(map[uuid.UUID]SystemInstance),
		apiInstancesByUUID:       make(map[uuid.UUID]ApiInstance),
		componentInstancesByUUID: make(map[uuid.UUID]ComponentInstance),

		findingsByUUID:     make(map[uuid.UUID]Finding),
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

type ComponentRef struct {
	Component    Component
	ComponentId  uuid.UUID
	ComponentRef *EntityVersion
}

type SystemInstanceRef struct {
	SystemInstance SystemInstance
	InstanceId     uuid.UUID
}

type ResourceRef struct {
	ResourceId   uuid.UUID
	ResourceType events.ResourceType
}

// GetSink implements [Model].
func (m *modelData) GetSink() events.EventSink {
	return m.sink
}

// Generic add helper for event-enabled types
func addEventEnabled[T any](
	m *modelData,
	obj T,
	getId func(T) uuid.UUID,
	setRegistered func(T),
	store map[uuid.UUID]T,
	resourceType events.ResourceType,
) error {
	id := getId(obj)
	if id == uuid.Nil {
		return ErrUUIDNotSet
	}
	op := events.CreateOperation
	if _, exists := store[id]; exists {
		op = events.UpdateOperation
	}
	setRegistered(obj)
	store[id] = obj
	if err := m.sink.Receive(resourceType, op, id, obj); err != nil {
		fmt.Println("Error receiving ", resourceType, "| ", op, " event: ", err)
	}
	return nil
}

// Generic delete helper
func deleteEventEnabled[T any](
	m *modelData,
	id uuid.UUID,
	store map[uuid.UUID]T,
	resourceType events.ResourceType,
	notFoundError error,
) error {
	_, exists := store[id]
	if !exists {
		return notFoundError
	}
	delete(store, id)
	if err := m.sink.Receive(resourceType, events.DeleteOperation, id); err != nil {
		fmt.Println("Error receiving ", resourceType, "| ", events.DeleteOperation, " event: ", err)
	}
	return nil
}

// Generic get helper
func getEventEnabled[T any](id uuid.UUID, store map[uuid.UUID]T) T {
	obj, exists := store[id]
	if !exists {
		var zero T
		return zero
	}
	return obj
}

// Generic getAll helper
func getAllEventEnabled[T any](store map[uuid.UUID]T) ([]T, error) {
	return slices.Collect(maps.Values(store)), nil
}

// AddContext implements Model.
func (m *modelData) AddContext(context Context) error {

	// invalidate the cache
	m.contextsCache = nil

	// TODO: parse parent ref if set

	if context.GetContextId() == uuid.Nil {
		return ErrUUIDNotSet
	}

	op := events.CreateOperation

	// check if this would overwrite an existing entry -> an update
	if _, ok := m.contextsByUUID[context.GetContextId()]; ok {
		op = events.UpdateOperation
	}

	if err := m.sink.Receive(events.ContextResource, op, context.GetContextId(), context); err != nil {
		fmt.Println("Error receiving ", events.ContextResource, "| ", op, " event: ", err)
	}

	m.contextsByUUID[context.GetContextId()] = context

	// mark Context as registered to activate sending events when updating fields
	context.Register()

	return nil
}

// DeleteContextById implements Model.
func (m *modelData) DeleteContextById(id uuid.UUID) error {
	_, exists := m.contextsByUUID[id]
	if !exists {
		return ErrContextNotFound
	}

	// invalidate the cache
	m.contextsCache = nil

	delete(m.contextsByUUID, id)

	if err := m.sink.Receive(events.ContextResource, events.DeleteOperation, id); err != nil {
		fmt.Println("Error receiving ", events.ContextResource, "| ", events.DeleteOperation, " event: ", err)
	}

	return nil
}

func (m *modelData) GetContextById(id uuid.UUID) Context {
	ctx, exists := m.contextsByUUID[id]
	if !exists {
		return nil
	}
	return ctx
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
	for ctx := range maps.Values(m.contextsByUUID) {
		contextArr = append(contextArr, ctx)
	}
	m.contextsCache = contextArr

	return contextArr, nil
}

// AddContextType implements [Model].
func (m *modelData) AddContextType(contextType ContextType) error {
	if contextType.GetContextTypeId() == uuid.Nil {
		return ErrUUIDNotSet
	}

	op := events.CreateOperation

	// check if this would overwrite an existing entry -> an update
	if _, ok := m.contextTypesByUUID[contextType.GetContextTypeId()]; ok {
		op = events.UpdateOperation
	}
	if err := m.sink.Receive(events.ContextTypeResource, op, contextType.GetContextTypeId(), contextType); err != nil {
		fmt.Println("Error receiving ", events.ContextTypeResource, "| ", op, " event: ", err)
	}

	m.contextTypesByUUID[contextType.GetContextTypeId()] = contextType

	// mark ContextType as registered to activate sending events when updating fields
	contextType.Register()

	return nil
}

// DeleteContextTypeById implements [Model].
func (m *modelData) DeleteContextTypeById(id uuid.UUID) error {
	_, exists := m.contextTypesByUUID[id]
	if !exists {
		return ErrContextTypeNotFound
	}

	delete(m.contextTypesByUUID, id)

	if err := m.sink.Receive(events.ContextTypeResource, events.DeleteOperation, id); err != nil {
		fmt.Println("Error receiving ", events.ContextTypeResource, "| ", events.DeleteOperation, " event: ", err)
	}

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

// AddNode implements [Model].
func (m *modelData) AddNode(node Node) error {
	return addEventEnabled(m, node, Node.GetNodeId, func(n Node) { n.Register() }, m.nodesByUUID, events.NodeResource)
}

// DeleteNodeById implements [Model].
func (m *modelData) DeleteNodeById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.nodesByUUID, events.NodeResource, ErrNodeNotFound)
}

// GetNodeById implements [Model].
func (m *modelData) GetNodeById(id uuid.UUID) Node {
	return getEventEnabled(id, m.nodesByUUID)
}

// GetNodes implements [Model].
func (m *modelData) GetNodes() ([]Node, error) {
	return getAllEventEnabled(m.nodesByUUID)
}

// AddNodeType implements [Model].
func (m *modelData) AddNodeType(nodeType NodeType) error {
	return addEventEnabled(m, nodeType, NodeType.GetNodeTypeId, func(nt NodeType) { nt.Register() }, m.nodeTypesByUUID, events.NodeTypeResource)
}

// DeleteNodeTypeById implements [Model].
func (m *modelData) DeleteNodeTypeById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.nodeTypesByUUID, events.NodeTypeResource, ErrNodeTypeNotFound)
}

// GetNodeTypeById implements [Model].
func (m *modelData) GetNodeTypeById(id uuid.UUID) NodeType {
	return getEventEnabled(id, m.nodeTypesByUUID)
}

// GetNodeTypes implements [Model].
func (m *modelData) GetNodeTypes() ([]NodeType, error) {
	return getAllEventEnabled(m.nodeTypesByUUID)
}

// AddSystem implements Model.
func (m *modelData) AddSystem(sys System) error {

	// parse parent ref if set
	if sys.GetSystemId() == uuid.Nil {
		return ErrUUIDNotSet
	}

	op := events.CreateOperation

	// check if this would overwrite an existing entry -> an update
	if _, ok := m.systemsByUUID[sys.GetSystemId()]; ok {
		op = events.UpdateOperation
	}
	if err := m.sink.Receive(events.SystemResource, op, sys.GetSystemId(), sys); err != nil {
		fmt.Println("Error receiving ", events.SystemResource, "| ", op, " event: ", err)
	}

	m.systemsByUUID[sys.GetSystemId()] = sys

	// mark System as registered to activate sending events when updating fields
	sys.Register()

	return nil
}

// DeleteSystemByResourceName implements Model.
func (m *modelData) DeleteSystemById(id uuid.UUID) error {
	_, exists := m.systemsByUUID[id]
	if !exists {
		return ErrSystemNotFound
	}

	delete(m.systemsByUUID, id)

	if err := m.sink.Receive(events.SystemResource, events.DeleteOperation, id); err != nil {
		fmt.Println("Error receiving ", events.SystemResource, "| ", events.DeleteOperation, " event: ", err)
	}

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
func (m *modelData) AddApi(api API) error {
	return addEventEnabled(m, api, API.GetApiId, func(a API) { a.Register() }, m.apisByUUID, events.APIResource)
}

// AddApiInstance implements Model.
func (m *modelData) AddApiInstance(instance ApiInstance) error {
	return addEventEnabled(m, instance, ApiInstance.GetInstanceId, func(i ApiInstance) { i.Register() }, m.apiInstancesByUUID, events.APIInstanceResource)
}

// AddComponent implements Model.
func (m *modelData) AddComponent(comp Component) error {
	return addEventEnabled(m, comp, Component.GetComponentId, func(c Component) { c.Register() }, m.componentsByUUID, events.ComponentResource)
}

// AddComponentInstance implements Model.
func (m *modelData) AddComponentInstance(instance ComponentInstance) error {
	return addEventEnabled(m, instance, ComponentInstance.GetInstanceId, func(i ComponentInstance) { i.Register() }, m.componentInstancesByUUID, events.ComponentInstanceResource)
}

// AddSystemInstance implements Model.
func (m *modelData) AddSystemInstance(instance SystemInstance) error {
	return addEventEnabled(m, instance, SystemInstance.GetInstanceId, func(i SystemInstance) { i.Register() }, m.systemInstancesByUUID, events.SystemInstanceResource)
}

// DeleteApiByResourceName implements Model.
func (m *modelData) DeleteApiById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.apisByUUID, events.APIResource, ErrApiNotFound)
}

// DeleteApiInstanceByResourceName implements Model.
func (m *modelData) DeleteApiInstanceById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.apiInstancesByUUID, events.APIInstanceResource, ErrApiInstanceNotFound)
}

// DeleteComponentByResourceName implements Model.
func (m *modelData) DeleteComponentById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.componentsByUUID, events.ComponentResource, ErrComponentNotFound)
}

// DeleteComponentInstanceByResourceName implements Model.
func (m *modelData) DeleteComponentInstanceById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.componentInstancesByUUID, events.ComponentInstanceResource, ErrComponentInstanceNotFound)
}

// DeleteSystemInstanceByResourceName implements Model.
func (m *modelData) DeleteSystemInstanceById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.systemInstancesByUUID, events.SystemInstanceResource, ErrSystemInstanceNotFound)
}

// GetApiById implements Model.
func (m *modelData) GetApiById(id uuid.UUID) API {
	return getEventEnabled(id, m.apisByUUID)
}

// ApiRefByID implements [Model].
func (m *modelData) ApiRefByID(apiId uuid.UUID) *ApiRef {
	api := m.GetApiById(apiId)
	if api == nil {
		return nil
	}
	return &ApiRef{API: api, ApiID: api.GetApiId()}
}

// GetApiInstanceById implements Model.
func (m *modelData) GetApiInstanceById(id uuid.UUID) ApiInstance {
	return getEventEnabled(id, m.apiInstancesByUUID)
}

// GetComponentById implements Model.
func (m *modelData) GetComponentById(id uuid.UUID) Component {
	return getEventEnabled(id, m.componentsByUUID)
}

// GetComponentInstanceById implements Model.
func (m *modelData) GetComponentInstanceById(id uuid.UUID) ComponentInstance {
	return getEventEnabled(id, m.componentInstancesByUUID)
}

// GetSystemInstanceById implements Model.
func (m *modelData) GetSystemInstanceById(id uuid.UUID) SystemInstance {
	return getEventEnabled(id, m.systemInstancesByUUID)
}

// SystemInstanceRefByID implements [Model].
func (m *modelData) SystemInstanceRefByID(instanceId uuid.UUID) *SystemInstanceRef {
	inst := m.GetSystemInstanceById(instanceId)
	if inst == nil {
		return nil
	}
	return &SystemInstanceRef{
		SystemInstance: inst,
		InstanceId:     inst.GetInstanceId(),
	}
}

// GetApiInstances implements Model.
func (m *modelData) GetApiInstances() ([]ApiInstance, error) {
	return getAllEventEnabled(m.apiInstancesByUUID)
}

// GetApis implements Model.
func (m *modelData) GetApis() ([]API, error) {
	return getAllEventEnabled(m.apisByUUID)
}

// GetComponentInstances implements Model.
func (m *modelData) GetComponentInstances() ([]ComponentInstance, error) {
	return getAllEventEnabled(m.componentInstancesByUUID)
}

// GetComponents implements Model.
func (m *modelData) GetComponents() ([]Component, error) {
	return getAllEventEnabled(m.componentsByUUID)
}

// GetSystemInstances implements Model.
func (m *modelData) GetSystemInstances() ([]SystemInstance, error) {
	return getAllEventEnabled(m.systemInstancesByUUID)
}

// GetFindings implements Model.
func (m modelData) GetFindings() ([]Finding, error) {
	findingArr := slices.Collect(maps.Values(m.findingsByUUID))
	return findingArr, nil
}

// AddFinding implements Model.
func (m *modelData) AddFinding(finding Finding, name string) error {
	if finding.GetFindingId() != uuid.Nil {
		finding.Register()
		m.findingsByUUID[finding.GetFindingId()] = finding
		if err := m.sink.Receive(events.FindingResource, events.CreateOperation, finding.GetFindingId(), finding); err != nil {
			fmt.Println("Error receiving ", events.FindingResource, "| ", events.CreateOperation, " event: ", err)
		}
	}
	return nil
}

// DeleteFindingById implements [Model].
func (m *modelData) DeleteFindingById(id uuid.UUID) error {
	_, exists := m.findingsByUUID[id]
	if !exists {
		return ErrFindingNotFound
	}
	delete(m.findingsByUUID, id)
	if err := m.sink.Receive(events.FindingResource, events.DeleteOperation, id); err != nil {
		fmt.Println("Error receiving ", events.FindingResource, "| ", events.DeleteOperation, " event: ", err)
	}
	return nil
}

// GetFindingById implements Model.
func (m *modelData) GetFindingById(id uuid.UUID) Finding {
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
		return ErrUUIDNotSet
	}

	op := events.CreateOperation

	// check if this would overwrite an existing entry -> an update
	if _, ok := m.findingTypesByUUID[findingType.GetFindingTypeId()]; ok {
		op = events.UpdateOperation
	}
	if err := m.sink.Receive(events.FindingTypeResource, op, findingType.GetFindingTypeId(), findingType); err != nil {
		fmt.Println("Error receiving ", events.FindingTypeResource, "| ", op, " event: ", err)
	}

	m.findingTypesByUUID[findingType.GetFindingTypeId()] = findingType

	// mark FindingType as registered to activate sending events when updating fields
	findingType.Register()

	return nil

}

// DeleteFindingTypeById implements [Model].
func (m *modelData) DeleteFindingTypeById(id uuid.UUID) error {
	_, exists := m.findingTypesByUUID[id]
	if !exists {
		return ErrFindingTypeNotFound
	}

	delete(m.findingTypesByUUID, id)

	if err := m.sink.Receive(events.FindingTypeResource, events.DeleteOperation, id); err != nil {
		fmt.Println("Error receiving ", events.FindingTypeResource, "| ", events.DeleteOperation, " event: ", err)
	}

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
