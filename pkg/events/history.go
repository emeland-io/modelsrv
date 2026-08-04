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
//
// SinceSeq queries reaching further back than the EventManager's configured
// retention window (see eventmgr.WithHistoryLimit) still succeed: history
// for resources that aged out of the window but still exist is synthesized
// as a Create carrying their current state, so replaying the full result in
// order reconstructs the same state a new subscriber would receive.
//
// The synthesized prefix is always delivered in full within a single
// response, even if that means returning more than Limit entries: capping
// it would let a paginating client (using the last returned SequenceId as
// the next SinceSeq) permanently skip whatever didn't fit. Once a client's
// SinceSeq reaches the retention boundary, subsequent calls page through
// the retained tail normally, honoring Limit.
//
// Operation/ResourceType/ResourceId filters apply to the synthesized
// prefix too, but the prefix only ever reports Operation "Create" — an
// Operation filter for "Update" or "Delete" will not surface resources
// whose only remaining history is synthesized, even if their real
// historical operation was different. This is inherent to bounding
// retention: preserving exact historical operation types forever would
// require unbounded memory, which is what this retention window exists to
// avoid.
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
