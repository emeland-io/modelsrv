package eventmgr

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/client"
	"go.emeland.io/modelsrv/pkg/events"
	"go.uber.org/zap"
)

type subscriber struct {
	url       string
	status    string
	id        uuid.UUID
	subClient *client.ModelSrvClient
	log       *zap.SugaredLogger
}

var _ events.Subscriber = (*subscriber)(nil)

func NewSubscriber(url string, log *zap.SugaredLogger) (events.Subscriber, error) {
	if log == nil {
		log = zap.NewNop().Sugar()
	}
	sub := &subscriber{
		url:    url,
		status: "active",
		id:     uuid.New(),
		log:    log,
	}
	sc, err := client.NewModelSrvClient(url)
	if err != nil {
		return nil, err
	}
	sc.SetLogger(log)
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
	if event == nil {
		s.log.Debugw("subscriber push", "url", s.url, "event", "nil")
		return s.subClient.PostEvent(ctx, event)
	}
	s.log.Debugw("subscriber push",
		"url", s.url,
		"subscriberId", s.id.String(),
		"kind", event.ResourceType.WireKind(),
		"operation", event.Operation.WireOperation(),
		"resourceId", event.ResourceId.String(),
		"objects", len(event.Objects),
	)
	started := time.Now()
	err := s.subClient.PostEvent(ctx, event)
	elapsed := time.Since(started)
	if err != nil {
		s.log.Debugw("subscriber push failed",
			"url", s.url,
			"subscriberId", s.id.String(),
			"kind", event.ResourceType.WireKind(),
			"operation", event.Operation.WireOperation(),
			"resourceId", event.ResourceId.String(),
			"elapsed", elapsed,
			"error", err,
		)
		return err
	}
	s.log.Debugw("subscriber push accepted",
		"url", s.url,
		"subscriberId", s.id.String(),
		"kind", event.ResourceType.WireKind(),
		"operation", event.Operation.WireOperation(),
		"resourceId", event.ResourceId.String(),
		"elapsed", elapsed,
		"status", 200,
	)
	return nil
}
