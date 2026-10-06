package capability

import (
	"fmt"

	"github.com/google/uuid"
)

// DependencyCoverageError reports that a Variant's required ValidValues are not
// covered by the Capability offered by a Dependency.
type DependencyCoverageError struct {
	Missing []uuid.UUID
}

func (e *DependencyCoverageError) Error() string {
	return fmt.Sprintf("dependency does not cover required values: %v", e.Missing)
}

// CheckDependencyCoverage verifies that for every ValidValue the Variant requires,
// the depended-on Capability offers a ValidValue with the same Parameter.
//
// When any of variant, capability, or the ValidValue lookup is unavailable
// (missing neighbor), the check is skipped and returns nil. This keeps
// ingress and replication order-tolerant.
//
// requiresByID and offersByID map ValidValueId -> ParameterId for the
// required and offered sides respectively. Callers that do not yet have
// both sides resolved should pass nil maps (or omit entries) so the check
// is treated as incomplete and skipped.
func CheckDependencyCoverage(
	variant Variant,
	capability Capability,
	requiresByParam map[uuid.UUID]uuid.UUID, // ValidValueId -> ParameterId
	offersByParam map[uuid.UUID]uuid.UUID, // ValidValueId -> ParameterId
) error {
	if variant == nil || capability == nil {
		return nil
	}
	if requiresByParam == nil || offersByParam == nil {
		return nil
	}

	// Build the set of ParameterIds covered by the capability's offers.
	offeredParams := make(map[uuid.UUID]struct{}, len(capability.GetOffers()))
	for _, offerID := range capability.GetOffers() {
		paramID, ok := offersByParam[offerID]
		if !ok {
			// Offer references a ValidValue that is not yet present; skip.
			return nil
		}
		offeredParams[paramID] = struct{}{}
	}

	var missing []uuid.UUID
	for _, reqID := range variant.GetRequires() {
		paramID, ok := requiresByParam[reqID]
		if !ok {
			// Required ValidValue not yet present; skip.
			return nil
		}
		if _, ok := offeredParams[paramID]; !ok {
			missing = append(missing, reqID)
		}
	}
	if len(missing) > 0 {
		return &DependencyCoverageError{Missing: missing}
	}
	return nil
}
