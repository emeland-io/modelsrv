//go:generate go run ../../tools/gen
package model

//go:generate ../../bin/mockgen -destination=../mocks/mock_model.go -package=mocks . Model

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/artifact"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/component"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	mdlevent "go.emeland.io/modelsrv/pkg/model/event"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.emeland.io/modelsrv/pkg/model/iam"
	"go.emeland.io/modelsrv/pkg/model/node"
	mdlprod "go.emeland.io/modelsrv/pkg/model/product"
	"go.emeland.io/modelsrv/pkg/model/system"
)

// NodeModel provides CRUD operations for [node.Node] resources.
type NodeModel interface {
	// AddNode registers a Node in the model. Returns an error if the node's UUID is nil.
	AddNode(n node.Node) error
	// DeleteNodeById removes the Node with the given id.
	DeleteNodeById(id uuid.UUID) error
	// GetNodes returns all registered Nodes.
	GetNodes() ([]node.Node, error)
	// GetNodeById returns the Node with the given id, or nil if not found.
	GetNodeById(id uuid.UUID) node.Node
}

// NodeTypeModel provides CRUD operations for [node.NodeType] resources.
type NodeTypeModel interface {
	// AddNodeType registers a NodeType in the model.
	AddNodeType(nodeType node.NodeType) error
	// DeleteNodeTypeById removes the NodeType with the given id.
	DeleteNodeTypeById(id uuid.UUID) error
	// GetNodeTypes returns all registered NodeTypes.
	GetNodeTypes() ([]node.NodeType, error)
	// GetNodeTypeById returns the NodeType with the given id, or nil if not found.
	GetNodeTypeById(id uuid.UUID) node.NodeType
}

// ContextModel provides CRUD operations for [mdlctx.Context] resources.
type ContextModel interface {
	// AddContext registers a Context in the model.
	AddContext(c mdlctx.Context) error
	// DeleteContextById removes the Context with the given id.
	DeleteContextById(id uuid.UUID) error
	// GetContexts returns all registered Contexts.
	GetContexts() ([]mdlctx.Context, error)
	// GetContextById returns the Context with the given id, or nil if not found.
	GetContextById(id uuid.UUID) mdlctx.Context
}

// ContextTypeModel provides CRUD operations for [mdlctx.ContextType] resources.
type ContextTypeModel interface {
	// AddContextType registers a ContextType in the model.
	AddContextType(contextType mdlctx.ContextType) error
	// DeleteContextTypeById removes the ContextType with the given id.
	DeleteContextTypeById(id uuid.UUID) error
	// GetContextTypes returns all registered ContextTypes.
	GetContextTypes() ([]mdlctx.ContextType, error)
	// GetContextTypeById returns the ContextType with the given id, or nil if not found.
	GetContextTypeById(id uuid.UUID) mdlctx.ContextType
}

// SystemModel provides CRUD operations for [system.System] resources.
type SystemModel interface {
	// AddSystem registers a System in the model.
	AddSystem(sys system.System) error
	// DeleteSystemById removes the System with the given id.
	DeleteSystemById(id uuid.UUID) error
	// GetSystems returns all registered Systems.
	GetSystems() ([]system.System, error)
	// GetSystemById returns the System with the given id, or nil if not found.
	GetSystemById(id uuid.UUID) system.System
}

// ApiModel provides CRUD operations for [mdlapi.API] resources.
type ApiModel interface {
	// AddApi registers an API in the model.
	AddApi(a mdlapi.API) error
	// DeleteApiById removes the API with the given id.
	DeleteApiById(id uuid.UUID) error
	// GetApis returns all registered APIs.
	GetApis() ([]mdlapi.API, error)
	// GetApiById returns the API with the given id, or nil if not found.
	GetApiById(id uuid.UUID) mdlapi.API
	// ApiRefByID builds an [mdlapi.ApiRef] for a registered API, or nil if not found.
	ApiRefByID(apiId uuid.UUID) *mdlapi.ApiRef
}

// ComponentModel provides CRUD operations for [component.Component] resources.
type ComponentModel interface {
	// AddComponent registers a Component in the model.
	AddComponent(comp component.Component) error
	// DeleteComponentById removes the Component with the given id.
	DeleteComponentById(id uuid.UUID) error
	// GetComponents returns all registered Components.
	GetComponents() ([]component.Component, error)
	// GetComponentById returns the Component with the given id, or nil if not found.
	GetComponentById(id uuid.UUID) component.Component
}

// SystemInstanceModel provides CRUD operations for [system.SystemInstance] resources.
type SystemInstanceModel interface {
	// AddSystemInstance registers a SystemInstance in the model.
	AddSystemInstance(instance system.SystemInstance) error
	// DeleteSystemInstanceById removes the SystemInstance with the given id.
	DeleteSystemInstanceById(id uuid.UUID) error
	// GetSystemInstances returns all registered SystemInstances.
	GetSystemInstances() ([]system.SystemInstance, error)
	// GetSystemInstanceById returns the SystemInstance with the given id, or nil if not found.
	GetSystemInstanceById(id uuid.UUID) system.SystemInstance
	// SystemInstanceRefByID builds a [system.SystemInstanceRef] for a registered instance, or nil if not found.
	SystemInstanceRefByID(instanceId uuid.UUID) *system.SystemInstanceRef
}

// ApiInstanceModel provides CRUD operations for [mdlapi.ApiInstance] resources.
type ApiInstanceModel interface {
	// AddApiInstance registers an ApiInstance in the model.
	AddApiInstance(instance mdlapi.ApiInstance) error
	// DeleteApiInstanceById removes the ApiInstance with the given id.
	DeleteApiInstanceById(id uuid.UUID) error
	// GetApiInstances returns all registered ApiInstances.
	GetApiInstances() ([]mdlapi.ApiInstance, error)
	// GetApiInstanceById returns the ApiInstance with the given id, or nil if not found.
	GetApiInstanceById(id uuid.UUID) mdlapi.ApiInstance
}

// ComponentInstanceModel provides CRUD operations for [component.ComponentInstance] resources.
type ComponentInstanceModel interface {
	// AddComponentInstance registers a ComponentInstance in the model.
	AddComponentInstance(instance component.ComponentInstance) error
	// DeleteComponentInstanceById removes the ComponentInstance with the given id.
	DeleteComponentInstanceById(id uuid.UUID) error
	// GetComponentInstances returns all registered ComponentInstances.
	GetComponentInstances() ([]component.ComponentInstance, error)
	// GetComponentInstanceById returns the ComponentInstance with the given id, or nil if not found.
	GetComponentInstanceById(id uuid.UUID) component.ComponentInstance
}

// FindingModel provides CRUD operations for [finding.Finding] resources.
type FindingModel interface {
	// AddFinding registers a Finding in the model with the given display name.
	AddFinding(f finding.Finding, name string) error
	// DeleteFindingById removes the Finding with the given id.
	DeleteFindingById(id uuid.UUID) error
	// GetFindings returns all registered Findings.
	GetFindings() ([]finding.Finding, error)
	// GetFindingById returns the Finding with the given id, or nil if not found.
	GetFindingById(id uuid.UUID) finding.Finding
}

// FindingTypeModel provides CRUD operations for [finding.FindingType] resources.
type FindingTypeModel interface {
	// AddFindingType registers a FindingType in the model.
	AddFindingType(findingType finding.FindingType) error
	// DeleteFindingTypeById removes the FindingType with the given id.
	DeleteFindingTypeById(id uuid.UUID) error
	// GetFindingTypes returns all registered FindingTypes.
	GetFindingTypes() ([]finding.FindingType, error)
	// GetFindingTypeById returns the FindingType with the given id, or nil if not found.
	GetFindingTypeById(id uuid.UUID) finding.FindingType
	// GetFindingTypeByName returns the first registered FindingType whose display
	// name equals name, or nil if none match. An empty name yields nil.
	GetFindingTypeByName(name string) finding.FindingType
}

// ArtifactModel provides CRUD operations for [artifact.Artifact] resources.
type ArtifactModel interface {
	// AddArtifact registers an Artifact in the model.
	AddArtifact(a artifact.Artifact) error
	// DeleteArtifactById removes the Artifact with the given id.
	DeleteArtifactById(id uuid.UUID) error
	// GetArtifacts returns all registered Artifacts.
	GetArtifacts() ([]artifact.Artifact, error)
	// GetArtifactById returns the Artifact with the given id, or nil if not found.
	GetArtifactById(id uuid.UUID) artifact.Artifact
}

// ArtifactInstanceModel provides CRUD operations for [artifact.ArtifactInstance] resources.
type ArtifactInstanceModel interface {
	// AddArtifactInstance registers an ArtifactInstance in the model.
	AddArtifactInstance(ai artifact.ArtifactInstance) error
	// DeleteArtifactInstanceById removes the ArtifactInstance with the given id.
	DeleteArtifactInstanceById(id uuid.UUID) error
	// GetArtifactInstances returns all registered ArtifactInstances.
	GetArtifactInstances() ([]artifact.ArtifactInstance, error)
	// GetArtifactInstanceById returns the ArtifactInstance with the given id, or nil if not found.
	GetArtifactInstanceById(id uuid.UUID) artifact.ArtifactInstance
}

// Model is the aggregate interface for the landscape model, composed of per-resource sub-interfaces.
type Model interface {
	mdlevent.EventApplier
	// GetSink returns the event sink used by this model for recording mutations.
	GetSink() events.EventSink

	NodeModel
	NodeTypeModel
	ContextModel
	ContextTypeModel
	SystemModel
	ApiModel
	ComponentModel
	SystemInstanceModel
	ApiInstanceModel
	ComponentInstanceModel
	FindingModel
	FindingTypeModel
	ArtifactModel
	ArtifactInstanceModel
	iam.OrgUnitModel
	iam.GroupModel
	iam.IdentityModel
	mdlprod.ProductModel
}

type modelData struct {
	sink     events.EventSink
	handlers map[events.ResourceType]resourceHandler

	nodeTypesByUUID map[uuid.UUID]node.NodeType
	nodesByUUID     map[uuid.UUID]node.Node

	contextsByUUID     map[uuid.UUID]mdlctx.Context
	contextTypesByUUID map[uuid.UUID]mdlctx.ContextType
	contextsCache      []mdlctx.Context

	systemsByUUID    map[uuid.UUID]system.System
	apisByUUID       map[uuid.UUID]mdlapi.API
	componentsByUUID map[uuid.UUID]component.Component

	systemInstancesByUUID    map[uuid.UUID]system.SystemInstance
	apiInstancesByUUID       map[uuid.UUID]mdlapi.ApiInstance
	componentInstancesByUUID map[uuid.UUID]component.ComponentInstance

	findingsByUUID     map[uuid.UUID]finding.Finding
	findingTypesByUUID map[uuid.UUID]finding.FindingType

	artifactsByUUID         map[uuid.UUID]artifact.Artifact
	artifactInstancesByUUID map[uuid.UUID]artifact.ArtifactInstance

	orgUnitsByUUID   map[uuid.UUID]iam.OrgUnit
	groupsByUUID     map[uuid.UUID]iam.Group
	identitiesByUUID map[uuid.UUID]iam.Identity

	productsByUUID map[uuid.UUID]mdlprod.Product
}

// ensure Model interface is implemented correctly
var _ Model = (*modelData)(nil)

func NewModel(sink events.EventSink) (*modelData, error) {
	if sink == nil {
		return nil, fmt.Errorf("event sink must not be nil")
	}

	model := &modelData{
		sink:     sink,
		handlers: maps.Clone(handlerRegistry),

		nodesByUUID:     make(map[uuid.UUID]node.Node),
		nodeTypesByUUID: make(map[uuid.UUID]node.NodeType),

		contextsByUUID:     make(map[uuid.UUID]mdlctx.Context),
		contextTypesByUUID: make(map[uuid.UUID]mdlctx.ContextType),

		systemsByUUID:    make(map[uuid.UUID]system.System),
		apisByUUID:       make(map[uuid.UUID]mdlapi.API),
		componentsByUUID: make(map[uuid.UUID]component.Component),

		systemInstancesByUUID:    make(map[uuid.UUID]system.SystemInstance),
		apiInstancesByUUID:       make(map[uuid.UUID]mdlapi.ApiInstance),
		componentInstancesByUUID: make(map[uuid.UUID]component.ComponentInstance),

		findingsByUUID:     make(map[uuid.UUID]finding.Finding),
		findingTypesByUUID: make(map[uuid.UUID]finding.FindingType),

		artifactsByUUID:         make(map[uuid.UUID]artifact.Artifact),
		artifactInstancesByUUID: make(map[uuid.UUID]artifact.ArtifactInstance),

		orgUnitsByUUID:   make(map[uuid.UUID]iam.OrgUnit),
		groupsByUUID:     make(map[uuid.UUID]iam.Group),
		identitiesByUUID: make(map[uuid.UUID]iam.Identity),

		productsByUUID: make(map[uuid.UUID]mdlprod.Product),
	}

	return model, nil
}

func (m *modelData) GetCurrentEventSequenceId(ctx context.Context) (string, error) {
	return "forty-two", nil
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
		return common.ErrUUIDNotSet
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
func (m *modelData) AddContext(c mdlctx.Context) error {

	// invalidate the cache
	m.contextsCache = nil

	// TODO: parse parent ref if set

	if c.GetContextId() == uuid.Nil {
		return common.ErrUUIDNotSet
	}

	op := events.CreateOperation

	// check if this would overwrite an existing entry -> an update
	if _, ok := m.contextsByUUID[c.GetContextId()]; ok {
		op = events.UpdateOperation
	}

	// Register and persist before notifying the sink so filters see consistent model state.
	c.Register()
	m.contextsByUUID[c.GetContextId()] = c
	if err := m.sink.Receive(events.ContextResource, op, c.GetContextId(), c); err != nil {
		fmt.Println("Error receiving ", events.ContextResource, "| ", op, " event: ", err)
	}

	return nil
}

// DeleteContextById implements Model.
func (m *modelData) DeleteContextById(id uuid.UUID) error {
	_, exists := m.contextsByUUID[id]
	if !exists {
		return common.ErrContextNotFound
	}

	// invalidate the cache
	m.contextsCache = nil

	delete(m.contextsByUUID, id)

	if err := m.sink.Receive(events.ContextResource, events.DeleteOperation, id); err != nil {
		fmt.Println("Error receiving ", events.ContextResource, "| ", events.DeleteOperation, " event: ", err)
	}

	return nil
}

func (m *modelData) GetContextById(id uuid.UUID) mdlctx.Context {
	dctx, exists := m.contextsByUUID[id]
	if !exists {
		return nil
	}
	return dctx
}

// GetContexts implements Model.
func (m *modelData) GetContexts() ([]mdlctx.Context, error) {
	/* since this operation would require O(n) and is therfore potentially quite costly, lets cache that
	   Any write operations to contextsByUUID must invalidate that
	*/
	if m.contextsCache != nil {
		return m.contextsCache, nil
	}

	contextArr := make([]mdlctx.Context, 0)
	for c := range maps.Values(m.contextsByUUID) {
		contextArr = append(contextArr, c)
	}
	m.contextsCache = contextArr

	return contextArr, nil
}

// AddContextType implements [Model].
func (m *modelData) AddContextType(contextType mdlctx.ContextType) error {
	if contextType.GetContextTypeId() == uuid.Nil {
		return common.ErrUUIDNotSet
	}

	op := events.CreateOperation

	// check if this would overwrite an existing entry -> an update
	if _, ok := m.contextTypesByUUID[contextType.GetContextTypeId()]; ok {
		op = events.UpdateOperation
	}

	// Register and persist before notifying the sink so filters see consistent model state.
	contextType.Register()
	m.contextTypesByUUID[contextType.GetContextTypeId()] = contextType
	if err := m.sink.Receive(events.ContextTypeResource, op, contextType.GetContextTypeId(), contextType); err != nil {
		fmt.Println("Error receiving ", events.ContextTypeResource, "| ", op, " event: ", err)
	}

	return nil
}

// DeleteContextTypeById implements [Model].
func (m *modelData) DeleteContextTypeById(id uuid.UUID) error {
	_, exists := m.contextTypesByUUID[id]
	if !exists {
		return common.ErrContextTypeNotFound
	}

	delete(m.contextTypesByUUID, id)

	if err := m.sink.Receive(events.ContextTypeResource, events.DeleteOperation, id); err != nil {
		fmt.Println("Error receiving ", events.ContextTypeResource, "| ", events.DeleteOperation, " event: ", err)
	}

	return nil
}

// GetContextTypeById implements [Model].
func (m *modelData) GetContextTypeById(id uuid.UUID) mdlctx.ContextType {
	contextType, exists := m.contextTypesByUUID[id]
	if !exists {
		return nil
	}
	return contextType
}

// GetContextTypes implements [Model].
func (m *modelData) GetContextTypes() ([]mdlctx.ContextType, error) {
	contextTypeArr := slices.Collect(maps.Values(m.contextTypesByUUID))
	return contextTypeArr, nil
}

// AddNode implements [Model].
func (m *modelData) AddNode(n node.Node) error {
	return addEventEnabled(m, n, node.Node.GetNodeId, func(x node.Node) { x.Register() }, m.nodesByUUID, events.NodeResource)
}

// DeleteNodeById implements [Model].
func (m *modelData) DeleteNodeById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.nodesByUUID, events.NodeResource, common.ErrNodeNotFound)
}

// GetNodeById implements [Model].
func (m *modelData) GetNodeById(id uuid.UUID) node.Node {
	return getEventEnabled(id, m.nodesByUUID)
}

// GetNodes implements [Model].
func (m *modelData) GetNodes() ([]node.Node, error) {
	return getAllEventEnabled(m.nodesByUUID)
}

// AddNodeType implements [Model].
func (m *modelData) AddNodeType(nodeType node.NodeType) error {
	return addEventEnabled(m, nodeType, node.NodeType.GetNodeTypeId, func(nt node.NodeType) { nt.Register() }, m.nodeTypesByUUID, events.NodeTypeResource)
}

// DeleteNodeTypeById implements [Model].
func (m *modelData) DeleteNodeTypeById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.nodeTypesByUUID, events.NodeTypeResource, common.ErrNodeTypeNotFound)
}

// GetNodeTypeById implements [Model].
func (m *modelData) GetNodeTypeById(id uuid.UUID) node.NodeType {
	return getEventEnabled(id, m.nodeTypesByUUID)
}

// GetNodeTypes implements [Model].
func (m *modelData) GetNodeTypes() ([]node.NodeType, error) {
	return getAllEventEnabled(m.nodeTypesByUUID)
}

// AddSystem implements Model.
func (m *modelData) AddSystem(sys system.System) error {

	// parse parent ref if set
	if sys.GetSystemId() == uuid.Nil {
		return common.ErrUUIDNotSet
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
		return common.ErrSystemNotFound
	}

	delete(m.systemsByUUID, id)

	if err := m.sink.Receive(events.SystemResource, events.DeleteOperation, id); err != nil {
		fmt.Println("Error receiving ", events.SystemResource, "| ", events.DeleteOperation, " event: ", err)
	}

	return nil
}

// GetSystems implements Model.
func (m *modelData) GetSystems() ([]system.System, error) {
	systemArr := slices.Collect(maps.Values(m.systemsByUUID))
	return systemArr, nil
}

// GetSystemById implements Model.
func (m *modelData) GetSystemById(id uuid.UUID) system.System {
	sys, exists := m.systemsByUUID[id]
	if !exists {
		return nil
	}
	return sys
}

// AddApi implements Model.
func (m *modelData) AddApi(a mdlapi.API) error {
	return addEventEnabled(m, a, mdlapi.API.GetApiId, func(x mdlapi.API) { x.Register() }, m.apisByUUID, events.APIResource)
}

// AddApiInstance implements Model.
func (m *modelData) AddApiInstance(instance mdlapi.ApiInstance) error {
	return addEventEnabled(m, instance, mdlapi.ApiInstance.GetInstanceId, func(i mdlapi.ApiInstance) { i.Register() }, m.apiInstancesByUUID, events.APIInstanceResource)
}

// AddComponent implements Model.
func (m *modelData) AddComponent(comp component.Component) error {
	return addEventEnabled(m, comp, component.Component.GetComponentId, func(c component.Component) { c.Register() }, m.componentsByUUID, events.ComponentResource)
}

// AddComponentInstance implements Model.
func (m *modelData) AddComponentInstance(instance component.ComponentInstance) error {
	return addEventEnabled(m, instance, component.ComponentInstance.GetInstanceId, func(i component.ComponentInstance) { i.Register() }, m.componentInstancesByUUID, events.ComponentInstanceResource)
}

// AddSystemInstance implements Model.
func (m *modelData) AddSystemInstance(instance system.SystemInstance) error {
	return addEventEnabled(m, instance, system.SystemInstance.GetInstanceId, func(i system.SystemInstance) { i.Register() }, m.systemInstancesByUUID, events.SystemInstanceResource)
}

// DeleteApiByResourceName implements Model.
func (m *modelData) DeleteApiById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.apisByUUID, events.APIResource, common.ErrApiNotFound)
}

// DeleteApiInstanceByResourceName implements Model.
func (m *modelData) DeleteApiInstanceById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.apiInstancesByUUID, events.APIInstanceResource, common.ErrApiInstanceNotFound)
}

// DeleteComponentByResourceName implements Model.
func (m *modelData) DeleteComponentById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.componentsByUUID, events.ComponentResource, common.ErrComponentNotFound)
}

// DeleteComponentInstanceByResourceName implements Model.
func (m *modelData) DeleteComponentInstanceById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.componentInstancesByUUID, events.ComponentInstanceResource, common.ErrComponentInstanceNotFound)
}

// DeleteSystemInstanceByResourceName implements Model.
func (m *modelData) DeleteSystemInstanceById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.systemInstancesByUUID, events.SystemInstanceResource, common.ErrSystemInstanceNotFound)
}

// GetApiById implements Model.
func (m *modelData) GetApiById(id uuid.UUID) mdlapi.API {
	return getEventEnabled(id, m.apisByUUID)
}

// ApiRefByID implements [Model].
func (m *modelData) ApiRefByID(apiId uuid.UUID) *mdlapi.ApiRef {
	a := m.GetApiById(apiId)
	if a == nil {
		return nil
	}
	return &mdlapi.ApiRef{API: a, ApiID: a.GetApiId()}
}

// GetApiInstanceById implements Model.
func (m *modelData) GetApiInstanceById(id uuid.UUID) mdlapi.ApiInstance {
	return getEventEnabled(id, m.apiInstancesByUUID)
}

// GetComponentById implements Model.
func (m *modelData) GetComponentById(id uuid.UUID) component.Component {
	return getEventEnabled(id, m.componentsByUUID)
}

// GetComponentInstanceById implements Model.
func (m *modelData) GetComponentInstanceById(id uuid.UUID) component.ComponentInstance {
	return getEventEnabled(id, m.componentInstancesByUUID)
}

// GetSystemInstanceById implements Model.
func (m *modelData) GetSystemInstanceById(id uuid.UUID) system.SystemInstance {
	return getEventEnabled(id, m.systemInstancesByUUID)
}

// SystemInstanceRefByID implements [Model].
func (m *modelData) SystemInstanceRefByID(instanceId uuid.UUID) *system.SystemInstanceRef {
	inst := m.GetSystemInstanceById(instanceId)
	if inst == nil {
		return nil
	}
	return &system.SystemInstanceRef{
		SystemInstance: inst,
		InstanceId:     inst.GetInstanceId(),
	}
}

// GetApiInstances implements Model.
func (m *modelData) GetApiInstances() ([]mdlapi.ApiInstance, error) {
	return getAllEventEnabled(m.apiInstancesByUUID)
}

// GetApis implements Model.
func (m *modelData) GetApis() ([]mdlapi.API, error) {
	return getAllEventEnabled(m.apisByUUID)
}

// GetComponentInstances implements Model.
func (m *modelData) GetComponentInstances() ([]component.ComponentInstance, error) {
	return getAllEventEnabled(m.componentInstancesByUUID)
}

// GetComponents implements Model.
func (m *modelData) GetComponents() ([]component.Component, error) {
	return getAllEventEnabled(m.componentsByUUID)
}

// GetSystemInstances implements Model.
func (m *modelData) GetSystemInstances() ([]system.SystemInstance, error) {
	return getAllEventEnabled(m.systemInstancesByUUID)
}

// GetFindings implements Model.
func (m modelData) GetFindings() ([]finding.Finding, error) {
	findingArr := slices.Collect(maps.Values(m.findingsByUUID))
	return findingArr, nil
}

// AddFinding implements Model.
func (m *modelData) AddFinding(f finding.Finding, name string) error {
	if f.GetFindingId() != uuid.Nil {
		f.Register()
		m.findingsByUUID[f.GetFindingId()] = f
		if err := m.sink.Receive(events.FindingResource, events.CreateOperation, f.GetFindingId(), f); err != nil {
			fmt.Println("Error receiving ", events.FindingResource, "| ", events.CreateOperation, " event: ", err)
		}
	}
	return nil
}

// DeleteFindingById implements [Model].
func (m *modelData) DeleteFindingById(id uuid.UUID) error {
	_, exists := m.findingsByUUID[id]
	if !exists {
		return common.ErrFindingNotFound
	}
	delete(m.findingsByUUID, id)
	if err := m.sink.Receive(events.FindingResource, events.DeleteOperation, id); err != nil {
		fmt.Println("Error receiving ", events.FindingResource, "| ", events.DeleteOperation, " event: ", err)
	}
	return nil
}

// GetFindingById implements Model.
func (m *modelData) GetFindingById(id uuid.UUID) finding.Finding {
	fobj, exists := m.findingsByUUID[id]
	if !exists {
		return nil
	}
	return fobj
}

// AddFindingType implements [Model].
func (m *modelData) AddFindingType(findingType finding.FindingType) error {

	// parse parent ref if set
	if findingType.GetFindingTypeId() == uuid.Nil {
		return common.ErrUUIDNotSet
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
		return common.ErrFindingTypeNotFound
	}

	delete(m.findingTypesByUUID, id)

	if err := m.sink.Receive(events.FindingTypeResource, events.DeleteOperation, id); err != nil {
		fmt.Println("Error receiving ", events.FindingTypeResource, "| ", events.DeleteOperation, " event: ", err)
	}

	return nil
}

// GetFindingTypeById implements [Model].
func (m *modelData) GetFindingTypeById(id uuid.UUID) finding.FindingType {
	findingType, exists := m.findingTypesByUUID[id]
	if !exists {
		return nil
	}
	return findingType
}

// GetFindingTypeByName implements [Model].
func (m *modelData) GetFindingTypeByName(name string) finding.FindingType {
	if name == "" {
		return nil
	}
	for _, ft := range m.findingTypesByUUID {
		if ft.GetDisplayName() == name {
			return ft
		}
	}
	return nil
}

// GetFindingTypes implements [Model].
func (m *modelData) GetFindingTypes() ([]finding.FindingType, error) {
	findingTypeArr := slices.Collect(maps.Values(m.findingTypesByUUID))
	return findingTypeArr, nil
}

// AddArtifact implements [Model].
func (m *modelData) AddArtifact(a artifact.Artifact) error {
	return addEventEnabled(m, a, artifact.Artifact.GetArtifactId, func(x artifact.Artifact) { x.Register() }, m.artifactsByUUID, events.ArtifactResource)
}

// DeleteArtifactById implements [Model].
func (m *modelData) DeleteArtifactById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.artifactsByUUID, events.ArtifactResource, common.ErrArtifactNotFound)
}

// GetArtifacts implements [Model].
func (m *modelData) GetArtifacts() ([]artifact.Artifact, error) {
	return getAllEventEnabled(m.artifactsByUUID)
}

// GetArtifactById implements [Model].
func (m *modelData) GetArtifactById(id uuid.UUID) artifact.Artifact {
	return getEventEnabled(id, m.artifactsByUUID)
}

// AddArtifactInstance implements [Model].
func (m *modelData) AddArtifactInstance(ai artifact.ArtifactInstance) error {
	return addEventEnabled(m, ai, artifact.ArtifactInstance.GetArtifactInstanceId, func(x artifact.ArtifactInstance) { x.Register() }, m.artifactInstancesByUUID, events.ArtifactInstanceResource)
}

// DeleteArtifactInstanceById implements [Model].
func (m *modelData) DeleteArtifactInstanceById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.artifactInstancesByUUID, events.ArtifactInstanceResource, common.ErrArtifactInstanceNotFound)
}

// GetArtifactInstances implements [Model].
func (m *modelData) GetArtifactInstances() ([]artifact.ArtifactInstance, error) {
	return getAllEventEnabled(m.artifactInstancesByUUID)
}

// GetArtifactInstanceById implements [Model].
func (m *modelData) GetArtifactInstanceById(id uuid.UUID) artifact.ArtifactInstance {
	return getEventEnabled(id, m.artifactInstancesByUUID)
}

// AddGroup implements [Model].
func (m *modelData) AddGroup(g iam.Group) error {
	return addEventEnabled(m, g, iam.Group.GetGroupId, func(x iam.Group) { x.Register() }, m.groupsByUUID, events.GroupResource)
}

// AddIdentity implements [Model].
func (m *modelData) AddIdentity(i iam.Identity) error {
	return addEventEnabled(m, i, iam.Identity.GetIdentityId, func(x iam.Identity) { x.Register() }, m.identitiesByUUID, events.IdentityResource)
}

// AddOrgUnit implements [Model].
func (m *modelData) AddOrgUnit(o iam.OrgUnit) error {
	return addEventEnabled(m, o, iam.OrgUnit.GetOrgUnitId, func(x iam.OrgUnit) { x.Register() }, m.orgUnitsByUUID, events.OrgUnitResource)
}

// DeleteGroup implements [Model].
func (m *modelData) DeleteGroup(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.groupsByUUID, events.GroupResource, common.ErrGroupNotFound)
}

// DeleteIdentity implements [Model].
func (m *modelData) DeleteIdentity(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.identitiesByUUID, events.IdentityResource, common.ErrIdentityNotFound)
}

// DeleteOrgUnit implements [Model].
func (m *modelData) DeleteOrgUnit(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.orgUnitsByUUID, events.OrgUnitResource, common.ErrOrgUnitNotFound)
}

// GetGroupById implements [Model].
func (m *modelData) GetGroupById(id uuid.UUID) iam.Group {
	return getEventEnabled(id, m.groupsByUUID)
}

// GetGroups implements [Model].
func (m *modelData) GetGroups() ([]iam.Group, error) {
	return getAllEventEnabled(m.groupsByUUID)
}

// GetIdentities implements [Model].
func (m *modelData) GetIdentities() ([]iam.Identity, error) {
	return getAllEventEnabled(m.identitiesByUUID)
}

// GetIdentityById implements [Model].
func (m *modelData) GetIdentityById(id uuid.UUID) iam.Identity {
	return getEventEnabled(id, m.identitiesByUUID)
}

// GetOrgUnitById implements [Model].
func (m *modelData) GetOrgUnitById(id uuid.UUID) iam.OrgUnit {
	return getEventEnabled(id, m.orgUnitsByUUID)
}

// GetOrgUnits implements [Model].
func (m *modelData) GetOrgUnits() ([]iam.OrgUnit, error) {
	return getAllEventEnabled(m.orgUnitsByUUID)
}

// AddProduct implements [Model].
func (m *modelData) AddProduct(p mdlprod.Product) error {
	return addEventEnabled(m, p, mdlprod.Product.GetProductId, func(x mdlprod.Product) { x.Register() }, m.productsByUUID, events.ProductResource)
}

// DeleteProductById implements [Model].
func (m *modelData) DeleteProductById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.productsByUUID, events.ProductResource, common.ErrProductNotFound)
}

// GetProductById implements [Model].
func (m *modelData) GetProductById(id uuid.UUID) mdlprod.Product {
	return getEventEnabled(id, m.productsByUUID)
}

// GetProducts implements [Model].
func (m *modelData) GetProducts() ([]mdlprod.Product, error) {
	return getAllEventEnabled(m.productsByUUID)
}
