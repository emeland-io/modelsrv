package product

import (
	"time"

	"github.com/google/uuid"
)

// ProductionVersion describes a line of releases of a product: lifecycle dates and artefact identifiers.
// It is distinct from [go.emeland.io/modelsrv/pkg/model/common.Version], which additionally carries a semantic version string.
type ProductionVersion struct {
	AvailableFrom  *time.Time  `json:"availableFrom,omitempty"`
	DeprecatedFrom *time.Time  `json:"deprecatedFrom,omitempty"`
	TerminatedFrom *time.Time  `json:"terminatedFrom,omitempty"`
	Artefacts      []uuid.UUID `json:"artefacts,omitempty"`
}
