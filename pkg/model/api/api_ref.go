package api

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model/common"
)

// ApiRef references an [API] by resolved object and/or id.
// ApiRef may embed an [common.EntityVersion] when the reference is versioned (e.g. contract / semantic binding).
type ApiRef struct {
	API    API
	ApiID  uuid.UUID
	ApiRef *common.EntityVersion
}
