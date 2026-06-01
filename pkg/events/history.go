package events

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// StoredEvent is an event enriched with sequence ID and timestamp for history queries.
type StoredEvent struct {
	SequenceId   uint64    `json:"sequenceId"`
	Timestamp    time.Time `json:"timestamp"`
	ResourceType string    `json:"resourceType"`
	Operation    string    `json:"operation"`
	ResourceId   uuid.UUID `json:"resourceId"`
	Objects      []any     `json:"objects,omitempty"`
}

// NewStoredEvent creates a StoredEvent with human-readable type/operation strings.
func NewStoredEvent(seq uint64, ts time.Time, rt ResourceType, op Operation, id uuid.UUID, objects []any) StoredEvent {
	return StoredEvent{
		SequenceId:   seq,
		Timestamp:    ts,
		ResourceType: rt.WireKind(),
		Operation:    op.WireOperation(),
		ResourceId:   id,
		Objects:      objects,
	}
}

// EventQuery defines filters and pagination for querying event history.
type EventQuery struct {
	Operation      *Operation
	ResourceType   *ResourceType
	ResourceId     *uuid.UUID
	SinceSeq       uint64 // return events with SequenceId > SinceSeq
	Limit          int    // max results; 0 means default (100)
	IncludePayload bool
}

// EventQuerier can query stored event history.
type EventQuerier interface {
	QueryEvents(ctx context.Context, q EventQuery) ([]StoredEvent, error)
}
