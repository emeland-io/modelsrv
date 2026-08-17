package filesensor

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.emeland.io/modelsrv/pkg/eventfilter/phase0"
	"go.emeland.io/modelsrv/pkg/model"
	"go.uber.org/zap"
)

// Start applies existing documents from a local directory synchronously, then watches for changes.
// The watcher runs in a background goroutine until ctx is cancelled.
func Start(ctx context.Context, dir string, m model.Model, log *zap.SugaredLogger) {
	log = ensureLog(log)
	ApplyExisting(dir, m, log)
	StartWatch(ctx, dir, m, log)
}

// ApplyExisting creates dir if needed, applies all supported files once, then runs phase0.ReconcileAll.
func ApplyExisting(dir string, m model.Model, log *zap.SugaredLogger) {
	log = ensureLog(log)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Errorw("filesensor: could not create data directory", "dir", dir, "error", err.Error())
		return
	}
	src := NewLocalSource(dir)
	ApplySource(context.Background(), src, StaticParserConfig{}, m, log)
	phase0.ReconcileAll(m)
}

// StartWatch watches a local directory for changes. It does not scan existing files.
func StartWatch(ctx context.Context, dir string, m model.Model, log *zap.SugaredLogger) {
	log = ensureLog(log)
	src := NewLocalSource(dir)
	go runWatch(ctx, src, StaticParserConfig{}, m, log)
}

// ApplySource lists all files from src and applies them once.
func ApplySource(ctx context.Context, src Source, cfg ParserConfig, m model.Model, log *zap.SugaredLogger) {
	log = ensureLog(log)
	if cfg == nil {
		cfg = StaticParserConfig{}
	}
	metas, err := src.List(ctx)
	if err != nil {
		log.Errorw("filesensor: list failed", "error", err.Error())
		return
	}
	for _, meta := range metas {
		applyOne(ctx, src, cfg, meta, m, log)
	}
}

// StartSourceWatch starts an fsnotify-style watch when src implements [Watcher].
func StartSourceWatch(ctx context.Context, src Source, cfg ParserConfig, m model.Model, log *zap.SugaredLogger) {
	log = ensureLog(log)
	if cfg == nil {
		cfg = StaticParserConfig{}
	}
	w, ok := src.(Watcher)
	if !ok {
		log.Errorw("filesensor: source does not support watch")
		return
	}
	go runWatch(ctx, w, cfg, m, log)
}

// StartSourcePoll polls src.List on interval and re-applies changed files (by ETag / LastModified).
func StartSourcePoll(ctx context.Context, src Source, cfg ParserConfig, interval time.Duration, m model.Model, log *zap.SugaredLogger) {
	log = ensureLog(log)
	if cfg == nil {
		cfg = StaticParserConfig{}
	}
	if interval <= 0 {
		interval = time.Minute
	}
	go runPoll(ctx, src, cfg, interval, m, log, true)
}

func ensureLog(log *zap.SugaredLogger) *zap.SugaredLogger {
	if log == nil {
		return zap.NewNop().Sugar()
	}
	return log
}

func applyOne(ctx context.Context, src Source, cfg ParserConfig, meta FileMeta, m model.Model, log *zap.SugaredLogger) {
	name := meta.Name
	opts, err := cfg.OptionsFor(name)
	if err != nil {
		log.Errorw("filesensor: parser config", "name", name, "error", err.Error())
		return
	}
	if opts.ContentType == "" {
		opts.ContentType = meta.ContentType
	}
	data, err := src.Read(ctx, name)
	if err != nil {
		log.Errorw("filesensor: could not read file", "name", name, "error", err.Error())
		return
	}
	res, err := ProcessBytes(name, data, opts, m)
	if err != nil {
		log.Errorw("filesensor: could not parse file", "name", name, "error", err.Error())
		return
	}
	for _, docErr := range res.Failed {
		log.Errorw("filesensor: document skipped", "name", name, "document", docErr.Index, "error", docErr.Err.Error())
	}
	if res.Applied > 0 {
		log.Infow("filesensor: applied documents", "name", name, "applied", res.Applied, "skipped", len(res.Failed))
	} else if len(res.Failed) > 0 {
		log.Errorw("filesensor: no documents applied", "name", name, "skipped", len(res.Failed))
	}
}

func runWatch(ctx context.Context, w Watcher, cfg ParserConfig, m model.Model, log *zap.SugaredLogger) {
	src, ok := w.(Source)
	if !ok {
		log.Errorw("filesensor: watcher is not a Source")
		return
	}
	ch, err := w.Watch(ctx)
	if err != nil {
		log.Errorw("filesensor: watch failed", "error", err.Error())
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case change, ok := <-ch:
			if !ok {
				return
			}
			if change.Op == OpDelete {
				continue // v1: ignore deletes
			}
			applyOne(ctx, src, cfg, FileMeta{Name: change.Name}, m, log)
		}
	}
}

// runPoll re-applies files whose ETag (or last-modified time) changed since the
// previous pass. When applyFirst is false the first pass only records tokens,
// so a caller that already applied the source does not emit a duplicate round
// of events at startup.
func runPoll(ctx context.Context, src Source, cfg ParserConfig, interval time.Duration, m model.Model, log *zap.SugaredLogger, applyFirst bool) {
	seen := map[string]string{} // name -> etag or last-modified token
	tick := func(apply bool) {
		metas, err := src.List(ctx)
		if err != nil {
			log.Errorw("filesensor: poll list failed", "error", err.Error())
			return
		}
		for _, meta := range metas {
			token := meta.ETag
			if token == "" && !meta.LastModified.IsZero() {
				token = meta.LastModified.UTC().Format(time.RFC3339Nano)
			}
			if prev, ok := seen[meta.Name]; ok && prev == token && token != "" {
				continue
			}
			if apply {
				applyOne(ctx, src, cfg, meta, m, log)
			}
			if token != "" {
				seen[meta.Name] = token
			} else {
				seen[meta.Name] = fmt.Sprintf("seen-%d", time.Now().UnixNano())
			}
		}
	}

	tick(applyFirst)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			tick(true)
		}
	}
}

// OpenSource is one configured source after its backend client has been built.
type OpenSource struct {
	Config SourceConfig
	Source Source
	Parser ParserConfig
}

// OpenSources builds the backend client and parser config for every source in cfg.
// Callers open once and pass the result to both [ApplySources] and [StartSources],
// so credentials are resolved and validated a single time at startup.
func OpenSources(ctx context.Context, cfg *Config) ([]OpenSource, error) {
	if cfg == nil {
		return nil, nil
	}
	out := make([]OpenSource, 0, len(cfg.Sources))
	for i, sc := range cfg.Sources {
		src, parser, err := sc.Open(ctx)
		if err != nil {
			return nil, fmt.Errorf("source[%d] (%s): %w", i, sc.URI, err)
		}
		out = append(out, OpenSource{Config: sc, Source: src, Parser: parser})
	}
	return out, nil
}

// ApplySources applies every opened source once, then reconciles.
func ApplySources(ctx context.Context, sources []OpenSource, m model.Model, log *zap.SugaredLogger) {
	log = ensureLog(log)
	if len(sources) == 0 {
		return
	}
	for _, s := range sources {
		ApplySource(ctx, s.Source, s.Parser, m, log)
	}
	phase0.ReconcileAll(m)
}

// StartSources starts a watch or poll loop per source. It assumes the sources were
// already applied by [ApplySources], so poll loops record the current state without
// re-applying it.
func StartSources(ctx context.Context, sources []OpenSource, m model.Model, log *zap.SugaredLogger) {
	log = ensureLog(log)
	for _, s := range sources {
		if s.Config.Watch {
			if _, ok := s.Source.(Watcher); ok {
				StartSourceWatch(ctx, s.Source, s.Parser, m, log)
				continue
			}
			log.Warnw("filesensor: watch requested but source does not support it; falling back to poll", "uri", s.Config.URI)
		} else if s.Config.Poll <= 0 {
			continue
		}
		interval := s.Config.Poll
		if interval <= 0 {
			interval = time.Minute
		}
		go runPoll(ctx, s.Source, s.Parser, interval, m, log, false)
	}
}
