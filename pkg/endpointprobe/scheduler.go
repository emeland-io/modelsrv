package endpointprobe

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.uber.org/zap"
)

// ApiInstanceClient lists and fetches ApiInstances from modelsrv.
type ApiInstanceClient interface {
	GetApiInstances() ([]common.InstanceListItem, error)
	GetApiInstanceById(id uuid.UUID) (api.ApiInstance, error)
}

// EventHook is an optional callback invoked after each probe (CERT-004 stub).
type EventHook func(ProbeResult)

const defaultDebounce = 5 * time.Second

// Scheduler periodically scans ApiInstances and fans out probes to a worker pool.
type Scheduler struct {
	Client              ApiInstanceClient
	Prober              *Prober
	Metrics             *Metrics
	Interval            time.Duration
	Debounce            time.Duration
	MaxConcurrentProbes int
	Logger              *zap.SugaredLogger
	EventHook           EventHook

	triggerOnce sync.Once
	trigger     chan struct{}
	wg          sync.WaitGroup
}

// Notify requests a debounced rescan. Safe to call before Run; never blocks the caller.
func (s *Scheduler) Notify() {
	s.initTrigger()
	select {
	case s.trigger <- struct{}{}:
	default:
	}
}

func (s *Scheduler) initTrigger() {
	s.triggerOnce.Do(func() {
		s.trigger = make(chan struct{}, 1)
	})
}

func (s *Scheduler) debounceDuration() time.Duration {
	if s.Debounce > 0 {
		return s.Debounce
	}
	return defaultDebounce
}

// Run executes probe cycles until ctx is cancelled. The first cycle runs immediately.
func (s *Scheduler) Run(ctx context.Context) {
	s.initTrigger()

	s.wg.Add(1)
	defer s.wg.Done()

	s.runOnce(ctx)

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	debounce := s.debounceDuration()
	var debounceTimer *time.Timer
	var debounceC <-chan time.Time

	stopDebounceTimer := func() {
		if debounceTimer == nil {
			return
		}
		if !debounceTimer.Stop() {
			select {
			case <-debounceTimer.C:
			default:
			}
		}
		debounceTimer = nil
		debounceC = nil
	}

	for {
		select {
		case <-ctx.Done():
			stopDebounceTimer()
			return
		case <-ticker.C:
			stopDebounceTimer()
			s.runOnce(ctx)
		case <-s.trigger:
			if debounceTimer != nil {
				if !debounceTimer.Stop() {
					select {
					case <-debounceTimer.C:
					default:
					}
				}
			}
			debounceTimer = time.NewTimer(debounce)
			debounceC = debounceTimer.C
		case <-debounceC:
			debounceTimer = nil
			debounceC = nil
			s.runOnce(ctx)
		}
	}
}

// Wait blocks until Run returns.
func (s *Scheduler) Wait() {
	s.wg.Wait()
}

func (s *Scheduler) runOnce(ctx context.Context) {
	targets, err := s.scan(ctx)
	if err != nil {
		s.Logger.Errorw("scan failed", "error", err)
		return
	}

	if len(targets) == 0 {
		s.Logger.Debug("no probe targets in this cycle")
		return
	}

	s.Logger.Infow("starting probe cycle", "targets", len(targets))

	sem := make(chan struct{}, s.MaxConcurrentProbes)
	var probeWG sync.WaitGroup

	for _, target := range targets {
		if ctx.Err() != nil {
			break
		}

		probeWG.Add(1)
		sem <- struct{}{}

		go func(t ProbeTarget) {
			defer probeWG.Done()
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			result := s.Prober.Probe(ctx, t)
			if s.Metrics != nil {
				s.Metrics.Record(result)
			}
			if s.EventHook != nil {
				s.EventHook(result)
			}

			if result.Success {
				s.Logger.Debugw("probe succeeded",
					"url", t.URL,
					"apiInstanceId", t.ApiInstanceID,
					"hasCert", result.HasCert,
				)
			} else {
				s.Logger.Warnw("probe failed",
					"url", t.URL,
					"apiInstanceId", t.ApiInstanceID,
					"error", result.Err,
				)
			}
		}(target)
	}

	probeWG.Wait()
	s.Logger.Infow("probe cycle complete", "targets", len(targets))
}

func (s *Scheduler) scan(ctx context.Context) ([]ProbeTarget, error) {
	items, err := s.Client.GetApiInstances()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	targets := make([]ProbeTarget, 0, len(items))

	for _, item := range items {
		if ctx.Err() != nil {
			break
		}

		ai, err := s.Client.GetApiInstanceById(item.Id)
		if err != nil {
			s.Logger.Warnw("failed to fetch api instance",
				"apiInstanceId", item.Id,
				"error", err,
			)
			continue
		}

		target, ok, err := TargetFromApiInstance(ai)
		if err != nil {
			s.Logger.Warnw("invalid endpoint annotations",
				"apiInstanceId", item.Id,
				"error", err,
			)
			continue
		}
		if !ok {
			continue
		}

		if _, dup := seen[target.DedupeKey]; dup {
			continue
		}
		seen[target.DedupeKey] = struct{}{}
		targets = append(targets, target)
	}

	return targets, nil
}
