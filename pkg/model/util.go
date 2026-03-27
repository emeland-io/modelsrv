package model

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/system"
)

// MakeTestSystem returns a [system.System] wired to sink with the given id and display name for tests.
func MakeTestSystem(sink events.EventSink, id uuid.UUID, displayName string, v common.Version) system.System {
	sys := system.NewSystem(sink, id)
	sys.SetDisplayName(displayName)
	if v.Version != "" {
		sys.SetVersion(v)
	}
	return sys
}
