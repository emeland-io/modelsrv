package model

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

// MakeTestSystem returns a [System] wired to sink with the given id and display name for tests.
func MakeTestSystem(sink events.EventSink, id uuid.UUID, displayName string, v Version) System {
	sys := NewSystem(sink, id)
	sys.SetDisplayName(displayName)
	if v.Version != "" {
		sys.SetVersion(v)
	}
	return sys
}
