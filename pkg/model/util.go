package model

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/system"
)

// MakeTestSystem returns a [system.System] with the given id and display name for tests.
func MakeTestSystem(id uuid.UUID, displayName string, v common.Version) system.System {
	sys := system.NewSystem(id)
	sys.SetDisplayName(displayName)
	if v.Version != "" {
		sys.SetVersion(v)
	}
	return sys
}
