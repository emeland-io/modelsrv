package endpointprobe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ConfigWriterConfig holds the inputs for [NewConfigWriter].
type ConfigWriterConfig struct {
	Path     string // required; output path for the collector config
	Opts     CollectorConfigOptions
	Debounce time.Duration // required, > 0
	Logger   *zap.SugaredLogger
}

// ConfigWriter maintains an in-memory registry of probe targets keyed by
// ApiInstance ID and flushes a rendered collector config to disk after a
// debounce window. Notify is safe to call before Run.
type ConfigWriter struct {
	path     string
	opts     CollectorConfigOptions
	debounce time.Duration
	logger   *zap.SugaredLogger

	mu      sync.Mutex
	targets map[uuid.UUID]ProbeTarget
	dirty   bool

	triggerOnce sync.Once
	trigger     chan struct{}
	wg          sync.WaitGroup
}

// NewConfigWriter validates cfg and returns a ready-to-run ConfigWriter.
func NewConfigWriter(cfg ConfigWriterConfig) (*ConfigWriter, error) {
	if cfg.Path == "" {
		return nil, errors.New("path is required")
	}
	if cfg.Debounce <= 0 {
		return nil, fmt.Errorf("debounce must be positive, got %s", cfg.Debounce)
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop().Sugar()
	}
	return &ConfigWriter{
		path:     cfg.Path,
		opts:     cfg.Opts,
		debounce: cfg.Debounce,
		logger:   cfg.Logger,
		targets:  make(map[uuid.UUID]ProbeTarget),
		dirty:    true, // force an initial flush so the file exists
	}, nil
}

// Upsert records or replaces a probe target and schedules a flush.
func (w *ConfigWriter) Upsert(t ProbeTarget) {
	w.mu.Lock()
	w.targets[t.ApiInstanceID] = t
	w.dirty = true
	w.mu.Unlock()
	w.Notify()
}

// Remove drops a probe target by ApiInstance ID and schedules a flush.
// It is a no-op when the ID is not present.
func (w *ConfigWriter) Remove(id uuid.UUID) {
	w.mu.Lock()
	_, existed := w.targets[id]
	if existed {
		delete(w.targets, id)
		w.dirty = true
	}
	w.mu.Unlock()
	if existed {
		w.Notify()
	}
}

// Notify requests a debounced flush. Safe to call before Run; never blocks.
func (w *ConfigWriter) Notify() {
	w.initTrigger()
	select {
	case w.trigger <- struct{}{}:
	default:
	}
}

func (w *ConfigWriter) initTrigger() {
	w.triggerOnce.Do(func() {
		w.trigger = make(chan struct{}, 1)
	})
}

// Run flushes immediately, then flushes again after each debounced Notify
// until ctx is cancelled. A final flush runs on exit so pending changes are
// not lost.
func (w *ConfigWriter) Run(ctx context.Context) {
	w.initTrigger()

	w.wg.Add(1)
	defer w.wg.Done()

	if err := w.Flush(); err != nil {
		w.logger.Errorw("otel config flush failed", "error", err, "path", w.path)
	}

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
			if err := w.Flush(); err != nil {
				w.logger.Errorw("otel config final flush failed", "error", err, "path", w.path)
			}
			return
		case <-w.trigger:
			if debounceTimer != nil {
				if !debounceTimer.Stop() {
					select {
					case <-debounceTimer.C:
					default:
					}
				}
			}
			debounceTimer = time.NewTimer(w.debounce)
			debounceC = debounceTimer.C
		case <-debounceC:
			debounceTimer = nil
			debounceC = nil
			if err := w.Flush(); err != nil {
				w.logger.Errorw("otel config flush failed", "error", err, "path", w.path)
			}
		}
	}
}

// Wait blocks until Run returns.
func (w *ConfigWriter) Wait() {
	w.wg.Wait()
}

// Flush snapshots the registry, renders the collector config, and writes it
// atomically. No-ops when the registry is clean. Skips the write when the
// rendered bytes match the file on disk.
func (w *ConfigWriter) Flush() error {
	w.mu.Lock()
	if !w.dirty {
		w.mu.Unlock()
		return nil
	}
	snapshot := make([]ProbeTarget, 0, len(w.targets))
	for _, t := range w.targets {
		snapshot = append(snapshot, t)
	}
	w.dirty = false
	w.mu.Unlock()

	targets := dedupeTargets(snapshot)
	raw, err := RenderCollectorConfig(targets, w.opts)
	if err != nil {
		// Leave dirty so a later flush retries after a render failure.
		w.mu.Lock()
		w.dirty = true
		w.mu.Unlock()
		return err
	}

	if existing, err := os.ReadFile(w.path); err == nil && bytes.Equal(existing, raw) {
		return nil
	}

	if err := atomicWriteFile(w.path, raw, 0644); err != nil {
		w.mu.Lock()
		w.dirty = true
		w.mu.Unlock()
		return err
	}

	w.logger.Infow("wrote otel collector config",
		"path", w.path,
		"targets", len(targets),
	)
	return nil
}

// dedupeTargets keeps one ProbeTarget per DedupeKey (host:port). When
// multiple targets share a key, the one with the lowest ApiInstanceID wins
// so the result does not depend on map iteration or event arrival order.
func dedupeTargets(targets []ProbeTarget) []ProbeTarget {
	if len(targets) == 0 {
		return nil
	}

	sorted := make([]ProbeTarget, len(targets))
	copy(sorted, targets)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].DedupeKey != sorted[j].DedupeKey {
			return sorted[i].DedupeKey < sorted[j].DedupeKey
		}
		return sorted[i].ApiInstanceID.String() < sorted[j].ApiInstanceID.String()
	})

	seen := make(map[string]struct{}, len(sorted))
	out := make([]ProbeTarget, 0, len(sorted))
	for _, t := range sorted {
		if _, ok := seen[t.DedupeKey]; ok {
			continue
		}
		seen[t.DedupeKey] = struct{}{}
		out = append(out, t)
	}
	return out
}

// atomicWriteFile writes data to path via a sibling temp file and rename.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create output directory %s: %w", dir, err)
		}
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return fmt.Errorf("write temp file %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}
	return nil
}
