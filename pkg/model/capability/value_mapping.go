package capability

import "github.com/google/uuid"

// ValueMapping maps a ValidValue on the requiring side of a Dependency to a
// ValidValue on the depended-on Capability.
type ValueMapping struct {
	FromValidValueId uuid.UUID `json:"fromValidValueId"`
	ToValidValueId   uuid.UUID `json:"toValidValueId"`
}
