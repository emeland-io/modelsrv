package model

import "github.com/google/uuid"

// Custom setter methods for ApiInstance that provide convenience functions
// beyond the generated SetApiRef and SetSystemInstance methods.

// SetApiRefById sets the API reference by looking up the API by ID.
func (a *apiinstanceData) SetApiRefById(apiId uuid.UUID) {
	api := a.model.GetApiById(apiId)
	if api == nil {
		// API not found, set nil reference
		a.SetApiRef(nil)
		return
	}

	a.SetApiRef(&ApiRef{
		API:   api,
		ApiID: api.GetApiId(),
	})
}

// SetApiRefByRef sets the API reference from an API object.
func (a *apiinstanceData) SetApiRefByRef(api API) {
	a.SetApiRef(&ApiRef{
		API:   api,
		ApiID: api.GetApiId(),
	})
}

// SetSystemInstanceById sets the system instance reference by looking up the instance by ID.
func (a *apiinstanceData) SetSystemInstanceById(instanceId uuid.UUID) {
	instance := a.model.GetSystemInstanceById(instanceId)
	if instance == nil {
		// Instance not found, set nil reference
		a.SetSystemInstance(nil)
		return
	}

	a.SetSystemInstance(&SystemInstanceRef{
		SystemInstance: instance,
		InstanceId:     instance.GetInstanceId(),
	})
}

// SetSystemInstanceByRef sets the system instance reference from a SystemInstance object.
func (a *apiinstanceData) SetSystemInstanceByRef(instance SystemInstance) {
	a.SetSystemInstance(&SystemInstanceRef{
		SystemInstance: instance,
		InstanceId:     instance.GetInstanceId(),
	})
}
