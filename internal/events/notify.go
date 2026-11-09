package eventmgr

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"go.emeland.io/modelsrv/pkg/events"
	"go.uber.org/zap"
)

const (
	// notifyQueueDepth bounds how many events may await delivery to one
	// subscriber. Beyond that the queued events are abandoned in favour of
	// origin-count reconcile, which converges the subscriber on current
	// state without retaining an unbounded backlog.
	notifyQueueDepth = 256

	notifyMaxAttempts    = 5
	notifyAttemptTimeout = 3 * time.Second
	notifyBaseBackoff    = 100 * time.Millisecond
	notifyMaxBackoff     = 5 * time.Second
	notifyResyncDelay    = 2 * time.Second
)

// notifyReconcileInterval is how often a notifier asks its subscriber for
// per-origin live counts. Overridable in tests via SetReconcileIntervalForTest.
// Atomic because notifiers from earlier tests may still be starting while a
// later test overrides it.
var notifyReconcileInterval atomic.Int64

func init() { notifyReconcileInterval.Store(int64(10 * time.Second)) }

// SetReconcileIntervalForTest overrides the periodic origin-count tick.
// Call before AddSubscriber so new notifiers pick up the interval. Returns a
// restore function for defer.
func SetReconcileIntervalForTest(d time.Duration) (restore func()) {
	prev := notifyReconcileInterval.Swap(int64(d))
	return func() { notifyReconcileInterval.Store(prev) }
}

// notifier delivers events to one subscriber from a single goroutine, so
// they arrive in the order the model produced them: concurrent delivery
// could land a Create after the Delete that supersedes it.
//
// Delivery is bounded by the queue rather than by blocking the caller. A
// slow or unreachable subscriber must not stall model mutations, nor
// delivery to the other subscribers.
//
// Catch-up after queue overflow, exhausted retries, or a peer that fell
// behind is owned by origin-count reconciliation: ask how many live
// resources the subscriber holds per OriginURL, and resend that origin's
// live events when the counts differ.
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
// scheduled for origin-count reconcile instead.
func (n *notifier) enqueue(ev events.Event) {
	select {
	case n.queue <- ev:
		n.logger.Debugw("queued event for subscriber",
			"url", n.sub.GetURL(),
			"subscriberId", n.sub.GetId().String(),
			"kind", ev.ResourceType.WireKind(),
			"operation", ev.Operation.WireOperation(),
			"resourceId", ev.ResourceId.String(),
			"queued", len(n.queue),
			"queueDepth", notifyQueueDepth,
		)
	default:
		n.resync.Store(true)
		n.logger.Warnw("subscriber fell behind; scheduling origin-count reconcile",
			"url", n.sub.GetURL(),
			"subscriberId", n.sub.GetId().String(),
			"kind", ev.ResourceType.WireKind(),
			"operation", ev.Operation.WireOperation(),
			"resourceId", ev.ResourceId.String(),
			"queueDepth", notifyQueueDepth,
		)
	}
}

func (n *notifier) run() {
	ticker := time.NewTicker(time.Duration(notifyReconcileInterval.Load()))
	defer ticker.Stop()
	var resyncRetry <-chan time.Time
	for {
		select {
		case <-n.stop:
			return
		case ev := <-n.queue:
			if !n.deliver(ev) {
				n.resync.Store(true)
			}
		case <-ticker.C:
			n.resync.Store(true)
		case <-resyncRetry:
		}

		resyncRetry = nil
		// Clear before reconciling: an enqueue that overflows while a
		// reconcile is in flight must schedule another round rather than
		// be swallowed by a success that predates it.
		if n.resync.CompareAndSwap(true, false) && !n.reconcile() {
			n.resync.Store(true)
			resyncRetry = time.After(notifyResyncDelay)
		}
	}
}

// reconcile compares per-origin live counts with the subscriber and resends
// any origin whose counts differ (or whose count GET fails). Queued events
// are discarded first: they predate the snapshot, and replaying one
// afterwards would resurrect a resource the snapshot has already established
// as deleted.
func (n *notifier) reconcile() bool {
	n.drainQueue()
	snapshot := n.snapshot()
	byOrigin := make(map[string][]events.Event)
	for i := range snapshot {
		origin := snapshot[i].OriginURL
		byOrigin[origin] = append(byOrigin[origin], snapshot[i])
	}

	n.logger.Debugw("reconciling subscriber by origin count",
		"url", n.sub.GetURL(),
		"subscriberId", n.sub.GetId().String(),
		"origins", len(byOrigin),
		"resources", len(snapshot),
	)

	ok := true
	for origin, evs := range byOrigin {
		ctx, cancel := context.WithTimeout(context.Background(), notifyAttemptTimeout)
		remote, err := n.sub.CountOrigin(ctx, origin)
		cancel()
		local := len(evs)
		if err == nil && remote == local {
			n.logger.Debugw("origin count matches; skip resend",
				"url", n.sub.GetURL(),
				"originUrl", origin,
				"count", local,
			)
			continue
		}
		if err != nil {
			n.logger.Warnw("origin count failed; resending origin",
				"url", n.sub.GetURL(),
				"originUrl", origin,
				"local", local,
				"error", err,
			)
		} else {
			n.logger.Infow("origin count mismatch; resending origin",
				"url", n.sub.GetURL(),
				"originUrl", origin,
				"local", local,
				"remote", remote,
			)
		}
		for i := range evs {
			n.logger.Debugw("reconcile pushing resource",
				"url", n.sub.GetURL(),
				"originUrl", origin,
				"index", i,
				"of", len(evs),
				"kind", evs[i].ResourceType.WireKind(),
				"resourceId", evs[i].ResourceId.String(),
			)
			if !n.deliver(evs[i]) {
				n.logger.Warnw("reconcile aborted; will retry",
					"url", n.sub.GetURL(),
					"originUrl", origin,
					"delivered", i,
					"resources", len(evs),
				)
				ok = false
				break
			}
		}
	}
	return ok
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

// deliver reports whether ev was handled for this subscriber. Transient
// failures are retried with exponential backoff; a permanent rejection is
// skipped so a reconcile can continue with later resources. Exhausting retries
// is not data loss: the caller schedules origin-count reconcile.
func (n *notifier) deliver(ev events.Event) bool {
	for attempt := 1; ; attempt++ {
		n.logger.Debugw("pushing event to subscriber",
			"url", n.sub.GetURL(),
			"subscriberId", n.sub.GetId().String(),
			"attempt", attempt,
			"maxAttempts", notifyMaxAttempts,
			"timeout", notifyAttemptTimeout.String(),
			"kind", ev.ResourceType.WireKind(),
			"operation", ev.Operation.WireOperation(),
			"resourceId", ev.ResourceId.String(),
			"objects", len(ev.Objects),
		)
		started := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), notifyAttemptTimeout)
		err := n.sub.Notify(ctx, &ev)
		cancel()
		elapsed := time.Since(started)
		if err == nil {
			n.logger.Debugw("subscriber accepted event",
				"url", n.sub.GetURL(),
				"subscriberId", n.sub.GetId().String(),
				"attempt", attempt,
				"elapsed", elapsed.String(),
				"kind", ev.ResourceType.WireKind(),
				"operation", ev.Operation.WireOperation(),
				"resourceId", ev.ResourceId.String(),
			)
			return true
		}
		if !notifyRetryable(err) {
			n.logger.Warnw("subscriber rejected event permanently; skipping",
				"url", n.sub.GetURL(),
				"subscriberId", n.sub.GetId().String(),
				"attempt", attempt,
				"elapsed", elapsed.String(),
				"kind", ev.ResourceType.WireKind(),
				"operation", ev.Operation.WireOperation(),
				"resourceId", ev.ResourceId.String(),
				"error", err,
			)
			return true
		}
		if attempt >= notifyMaxAttempts {
			n.logger.Errorw("subscriber notify exhausted retries",
				"url", n.sub.GetURL(),
				"subscriberId", n.sub.GetId().String(),
				"attempts", attempt,
				"elapsed", elapsed.String(),
				"kind", ev.ResourceType.WireKind(),
				"operation", ev.Operation.WireOperation(),
				"resourceId", ev.ResourceId.String(),
				"error", err,
			)
			return false
		}
		n.logger.Warnw("subscriber notify failed; retrying",
			"url", n.sub.GetURL(),
			"subscriberId", n.sub.GetId().String(),
			"attempt", attempt,
			"elapsed", elapsed.String(),
			"backoff", notifyBackoff(attempt).String(),
			"kind", ev.ResourceType.WireKind(),
			"operation", ev.Operation.WireOperation(),
			"resourceId", ev.ResourceId.String(),
			"error", err,
		)
		select {
		case <-time.After(notifyBackoff(attempt)):
		case <-n.stop:
			return false
		}
	}
}

type permanentError interface {
	Permanent() bool
}

func notifyRetryable(err error) bool {
	if err == nil {
		return false
	}
	var perm permanentError
	if errors.As(err, &perm) && perm.Permanent() {
		return false
	}
	msg := err.Error()
	return !strings.Contains(msg, "replication decode:") && !strings.Contains(msg, "replication apply:")
}

func notifyBackoff(attempt int) time.Duration {
	d := notifyBaseBackoff * time.Duration(1<<(attempt-1))
	if d > notifyMaxBackoff {
		return notifyMaxBackoff
	}
	return d
}
