package model

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

// ensure ApiInstance interface is implemented correctly
var _ ApiInstance = (*apiInstanceData)(nil)

// ensure events.EventSink interface is implemented correctly
var _ events.EventSink = (*apiInstanceData)(nil)

type ApiInstance interface {
	GetInstanceId() uuid.UUID

	GetDisplayName() string
	SetDisplayName(string)

	GetApiRef() *ApiRef
	SetApiRefById(apiId uuid.UUID)
	SetApiRefByRef(api *API)

	GetSystemInstance() *SystemInstanceRef
	SetSystemInstanceById(instanceId uuid.UUID)
	SetSystemInstanceByRef(instance *SystemInstance)

	GetAnnotations() Annotations

	getData() *apiInstanceData
}

type apiInstanceData struct {
	model        *modelData
	isRegistered bool

	InstanceId     uuid.UUID
	DisplayName    string
	ApiRef         *ApiRef
	SystemInstance *SystemInstanceRef
	Annotations    Annotations
}

func NewApiInstance(model Model, id uuid.UUID) ApiInstance {
	retval := &apiInstanceData{
		model:        model.getData(),
		isRegistered: false,
		InstanceId:   id,
	}

	retval.Annotations = NewAnnotations(model.getData(), retval)

	return retval
}

func (a *apiInstanceData) getData() *apiInstanceData {
	return a
}

// GetAnnotations implements [ApiInstance].
func (a *apiInstanceData) GetAnnotations() Annotations {
	return a.Annotations
}

// GetInstanceId implements [ApiInstance].
func (a *apiInstanceData) GetInstanceId() uuid.UUID {
	return a.InstanceId
}

// GetDisplayName implements [ApiInstance].
func (a *apiInstanceData) GetDisplayName() string {
	return a.DisplayName
}

// SetDisplayName implements [ApiInstance].
func (a *apiInstanceData) SetDisplayName(name string) {
	a.DisplayName = name

	if a.isRegistered {
		a.model.sink.Receive(events.APIInstanceResource, events.UpdateOperation, a.InstanceId, a)
	}
}

// GetApiRef implements [ApiInstance].
func (a *apiInstanceData) GetApiRef() *ApiRef {
	return a.ApiRef
}

// SetApiRefById implements [ApiInstance].
func (a *apiInstanceData) SetApiRefById(apiId uuid.UUID) {
	a.ApiRef = &ApiRef{
		ApiID: apiId,
	}

	ptr, ok := a.model.apisByUUID[apiId]
	if ok {
		a.ApiRef.API = ptr
	}

	if a.isRegistered {
		a.model.sink.Receive(events.APIInstanceResource, events.UpdateOperation, a.InstanceId, a)
	}
}

// SetApiRefByRef implements [ApiInstance].
func (a *apiInstanceData) SetApiRefByRef(api *API) {
	if api == nil {
		return
	}
	a.ApiRef = &ApiRef{
		API:   api,
		ApiID: api.ApiId,
	}

	if a.isRegistered {
		a.model.sink.Receive(events.APIInstanceResource, events.UpdateOperation, a.InstanceId, a)
	}
}

// GetSystemInstance implements [ApiInstance].
func (a *apiInstanceData) GetSystemInstance() *SystemInstanceRef {
	return a.SystemInstance
}

// SetSystemInstanceById implements [ApiInstance].
func (a *apiInstanceData) SetSystemInstanceById(instanceId uuid.UUID) {
	a.SystemInstance = &SystemInstanceRef{
		InstanceId: instanceId,
	}

	ptr, ok := a.model.systemInstancesByUUID[instanceId]
	if ok {
		a.SystemInstance.SystemInstance = ptr
	}

	if a.isRegistered {
		a.model.sink.Receive(events.APIInstanceResource, events.UpdateOperation, a.InstanceId, a)
	}
}

// SetSystemInstanceByRef implements [ApiInstance].
func (a *apiInstanceData) SetSystemInstanceByRef(instance *SystemInstance) {
	if instance == nil {
		return
	}
	a.SystemInstance = &SystemInstanceRef{
		SystemInstance: instance,
		InstanceId:     instance.InstanceId,
	}

	if a.isRegistered {
		a.model.sink.Receive(events.APIInstanceResource, events.UpdateOperation, a.InstanceId, a)
	}
}

// Receive implements [events.EventSink].
func (a *apiInstanceData) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, object ...any) error {
	if resType != events.AnnotationsResource {
		return fmt.Errorf("unsupported resource type %v in ApiInstance event sink. Only Annotations are supported", resType)
	}

	// all changes to annotations are automatically reflected in the parent object as updates
	if a.isRegistered {
		return a.model.sink.Receive(events.APIInstanceResource, events.UpdateOperation, a.InstanceId, a)
	}

	return nil
}
