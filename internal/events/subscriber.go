package events

import (
	"context"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
)

type subscriber struct {
	url    string
	status string
	id     uuid.UUID
}

var _ events.Subscriber = (*subscriber)(nil)

func NewSubscriber(url string) events.Subscriber {
	return &subscriber{
		url: url,
	}
}

// GetId implements [events.Subscriber].
func (s *subscriber) GetId() uuid.UUID {
	return s.id
}

// GetStatus implements [events.Subscriber].
func (s *subscriber) GetStatus() string {
	return s.status
}

// GetURL implements [events.Subscriber].
func (s *subscriber) GetURL() string {
	return s.url
}

// Notify implements [events.Subscriber].
func (s *subscriber) Notify(ctx context.Context, event *events.Event) error {
	panic("unimplemented")
}
