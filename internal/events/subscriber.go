package eventmgr

import (
	"context"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/client"
	"go.emeland.io/modelsrv/pkg/events"
)

type subscriber struct {
	url       string
	status    string
	id        uuid.UUID
	subClient *client.ModelSrvClient
}

var _ events.Subscriber = (*subscriber)(nil)

func NewSubscriber(url string) (events.Subscriber, error) {
	sub := &subscriber{
		url:    url,
		status: "active",
		id:     uuid.New(),
	}
	sc, err := client.NewModelSrvClient(url)
	if err != nil {
		return nil, err
	}
	sub.subClient = sc
	return sub, nil
}

func (s *subscriber) GetId() uuid.UUID {
	return s.id
}

func (s *subscriber) GetStatus() string {
	return s.status
}

func (s *subscriber) GetURL() string {
	return s.url
}

func (s *subscriber) Notify(ctx context.Context, event *events.Event) error {
	return s.subClient.PostEvent(ctx, event)
}
