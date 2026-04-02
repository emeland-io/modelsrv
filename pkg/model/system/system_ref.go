package system

import "github.com/google/uuid"

// SystemRef links to a [System] by resolved object and/or id (e.g. API payloads may only carry SystemId).
// When both are set, SystemId should match System.GetSystemId().
type SystemRef struct {
	System   System
	SystemId uuid.UUID
}

// ResolvedSystem returns the embedded [System] when present, or nil.
func (r *SystemRef) ResolvedSystem() System {
	if r == nil {
		return nil
	}
	return r.System
}

// SystemInstanceRef references a [SystemInstance] by resolved object and/or id.
type SystemInstanceRef struct {
	SystemInstance SystemInstance
	InstanceId     uuid.UUID
}
