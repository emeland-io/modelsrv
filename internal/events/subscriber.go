package events

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
	forwarder *eventForwarder
	subClient *client.ModelSrvClient

	/* fwdCtrl is a channel to control the forwarder goroutine.

	Send a new maximum index of the event list to be forwarded to the subscriber. The
	forwarder will forward all events in the list up to the specified index to the
	subscriber, and then wait for the next control signal.

	Closing the channel will stop the forwarder goroutine.
	*/
	fwdCtrl chan int // channel to control the forwarder goroutine
}

var _ events.Subscriber = (*subscriber)(nil)

func NewSubscriber(url string) (events.Subscriber, error) {
	sub := &subscriber{
		url:     url,
		status:  "active",
		id:      uuid.New(),
		fwdCtrl: make(chan int),
	}
	subClient, err := client.NewModelSrvClient(url)
	if err != nil {
		return nil, err
	}
	sub.subClient = subClient

	return sub, nil
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
	s.subClient.PostEvent(event)
	return nil
}
