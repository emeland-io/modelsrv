package eventmgr

import (
	"context"
	"sync/atomic"
	"time"

	"go.emeland.io/modelsrv/pkg/events"
	"go.uber.org/zap"
)

const (
	// notifyQueueDepth bounds how many events may await delivery to one
	// subscriber. Beyond that the queued events are abandoned in favour of
	// a latest-state resync, which converges the subscriber on current
	// state without retaining an unbounded backlog.
	notifyQueueDepth = 256

	notifyMaxAttempts    = 5
	notifyAttemptTimeout = 3 * time.Second
	notifyBaseBackoff    = 100 * time.Millisecond
	notifyMaxBackoff     = 5 * time.Second
	notifyResyncDelay    = 2 * time.Second
)

// notifier delivers events to one subscriber from a single goroutine, so
// they arrive in the order the model produced them: concurrent delivery
// could land a Create after the Delete that supersedes it.
//
// Delivery is bounded by the queue rather than by blocking the caller. A
// slow or unreachable subscriber must not stall model mutations, nor
// delivery to the other subscribers.
type notifier struct {
	sub      events.Subscriber
	snapshot func() []events.Event
	logger   *zap.SugaredLogger

	queue  chan events.Event
	resync atomic.Bool
	stop   chan struct{}
}

func newNotifier(sub events.Subscriber, snapshot func() []events.Event, logger *zap.SugaredLogger) *notifier {
	return &notifier{
		sub:      sub,
		snapshot: snapshot,
		logger:   logger,
		queue:    make(chan events.Event, notifyQueueDepth),
		stop:     make(chan struct{}),
	}
}

func (n *notifier) start() { go n.run() }

func (n *notifier) stopDelivery() { close(n.stop) }

// enqueue hands ev to the delivery goroutine without blocking. A subscriber
// that has fallen a full queue behind loses the individual events and is
// scheduled for a resync instead.
func (n *notifier) enqueue(ev events.Event) {
	select {
	case n.queue <- ev:
	default:
		n.resync.Store(true)
		n.logger.Warnw("subscriber fell behind; scheduling state resync",
			"url", n.sub.GetURL(),
			"queueDepth", notifyQueueDepth,
		)
	}
}

func (n *notifier) run() {
	var resyncRetry <-chan time.Time
	for {
		select {
		case <-n.stop:
			return
		case ev := <-n.queue:
			if !n.deliver(ev) {
				n.resync.Store(true)
			}
		case <-resyncRetry:
		}

		resyncRetry = nil
		// Clear before resyncing: an enqueue that overflows while the
		// snapshot is in flight must schedule another round rather than
		// be swallowed by a success that predates it.
		if n.resync.CompareAndSwap(true, false) && !n.deliverSnapshot() {
			n.resync.Store(true)
			resyncRetry = time.After(notifyResyncDelay)
		}
	}
}

// deliverSnapshot brings the subscriber back to current state after events
// were dropped. Queued events are discarded first: they predate the
// snapshot, and replaying one afterwards would resurrect a resource the
// snapshot has already established as deleted.
func (n *notifier) deliverSnapshot() bool {
	n.drainQueue()
	snapshot := n.snapshot()
	n.logger.Infow("resyncing subscriber from current state",
		"url", n.sub.GetURL(),
		"resources", len(snapshot),
	)
	for i := range snapshot {
		if !n.deliver(snapshot[i]) {
			return false
		}
	}
	return true
}

func (n *notifier) drainQueue() {
	for {
		select {
		case <-n.queue:
		default:
			return
		}
	}
}

// deliver reports whether ev reached the subscriber, retrying transient
// failures with exponential backoff. Giving up is not data loss: the caller
// schedules a resync, which re-sends current state.
func (n *notifier) deliver(ev events.Event) bool {
	for attempt := 1; ; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), notifyAttemptTimeout)
		err := n.sub.Notify(ctx, &ev)
		cancel()
		if err == nil {
			return true
		}
		if attempt >= notifyMaxAttempts {
			n.logger.Errorw("subscriber notify exhausted retries",
				"url", n.sub.GetURL(),
				"attempts", attempt,
				"error", err,
			)
			return false
		}
		n.logger.Warnw("subscriber notify failed; retrying",
			"url", n.sub.GetURL(),
			"attempt", attempt,
			"error", err,
		)
		select {
		case <-time.After(notifyBackoff(attempt)):
		case <-n.stop:
			return false
		}
	}
}

func notifyBackoff(attempt int) time.Duration {
	d := notifyBaseBackoff * time.Duration(1<<(attempt-1))
	if d > notifyMaxBackoff {
		return notifyMaxBackoff
	}
	return d
}
