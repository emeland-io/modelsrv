package filesensor

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.emeland.io/modelsrv/pkg/model"
	"go.uber.org/zap"
)

const debounceDelay = 250 * time.Millisecond

// Start watches dir for .yaml/.yml files, applies existing files once, then watches for changes.
// It returns immediately; work runs in a background goroutine until ctx is cancelled.
func Start(ctx context.Context, dir string, m model.Model, log *zap.SugaredLogger) {
	if log == nil {
		log = zap.NewNop().Sugar()
	}
	go run(ctx, dir, m, log)
}

func run(ctx context.Context, dir string, m model.Model, log *zap.SugaredLogger) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Errorw("filesensor: could not create data directory", "dir", dir, "error", err.Error())
		return
	}

	apply := func(path string) {
		res, err := ProcessFile(path, m)
		if err != nil {
			log.Errorw("filesensor: could not read or parse YAML file", "path", path, "error", err.Error())
			return
		}
		for _, docErr := range res.Failed {
			log.Errorw("filesensor: document skipped", "path", path, "document", docErr.Index, "error", docErr.Err.Error())
		}
		if res.Applied > 0 {
			log.Infow("filesensor: applied YAML documents", "path", path, "applied", res.Applied, "skipped", len(res.Failed))
		} else if len(res.Failed) > 0 {
			log.Errorw("filesensor: no documents applied", "path", path, "skipped", len(res.Failed))
		}
	}

	if err := scanDir(dir, apply); err != nil {
		log.Errorw("filesensor: initial scan failed", "dir", dir, "error", err.Error())
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorw("filesensor: watcher create failed", "error", err.Error())
		return
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.Errorw("filesensor: watcher close failed", "error", err.Error())
		}
	}()

	if err := watcher.Add(dir); err != nil {
		log.Errorw("filesensor: watch add failed", "dir", dir, "error", err.Error())
		return
	}

	var mu sync.Mutex
	timers := make(map[string]*time.Timer)
	schedule := func(path string) {
		if !isYAMLFileName(filepath.Base(path)) {
			return
		}
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return
		}
		parent := filepath.Dir(path)
		absParent, err := filepath.Abs(parent)
		if err != nil {
			return
		}
		if absParent != absDir {
			return
		}

		mu.Lock()
		if t, ok := timers[path]; ok {
			t.Stop()
		}
		timers[path] = time.AfterFunc(debounceDelay, func() {
			mu.Lock()
			delete(timers, path)
			mu.Unlock()

			select {
			case <-ctx.Done():
				return
			default:
			}
			apply(path)
		})
		mu.Unlock()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-watcher.Events:
			if !ok {
				return
			}
			if ev.Has(fsnotify.Create) || ev.Has(fsnotify.Write) || ev.Has(fsnotify.Rename) {
				schedule(ev.Name)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Errorw("filesensor: watcher error", "error", err.Error())
		}
	}
}
