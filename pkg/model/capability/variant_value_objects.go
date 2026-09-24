package capability

import "github.com/google/uuid"

// ParameterValueSet is a set of valid values bound to a parameter.
// It is a nested value object, not a landscape resource.
type ParameterValueSet struct {
	ParameterId   uuid.UUID
	ValidValueIds []uuid.UUID
}

// VariantDependency is a dependency of a Variant on parameter value sets of another Capability.
// It is a nested value object, not a landscape resource.
type VariantDependency struct {
	CapabilityId uuid.UUID
	Required     []ParameterValueSet
}
