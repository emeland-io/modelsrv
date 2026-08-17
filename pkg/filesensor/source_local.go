package filesensor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// LocalSource reads files from a directory on the local filesystem.
type LocalSource struct {
	Dir string
}

// NewLocalSource returns a Source rooted at dir.
func NewLocalSource(dir string) *LocalSource {
	return &LocalSource{Dir: dir}
}

// List implements [Source].
func (s *LocalSource) List(ctx context.Context) ([]FileMeta, error) {
	_ = ctx
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, err
	}
	out := make([]FileMeta, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !isSupportedFileName(e.Name()) {
			continue
		}
		info, err := e.Info()
		meta := FileMeta{Name: e.Name()}
		if err == nil {
			meta.LastModified = info.ModTime()
			meta.ETag = fmt.Sprintf("%d-%d", info.ModTime().UnixNano(), info.Size())
		}
		out = append(out, meta)
	}
	return out, nil
}

// Read implements [Source].
func (s *LocalSource) Read(ctx context.Context, name string) ([]byte, error) {
	_ = ctx
	if filepath.Base(name) != name || name == "." || name == ".." {
		return nil, fmt.Errorf("invalid file name %q", name)
	}
	return os.ReadFile(filepath.Join(s.Dir, name))
}

// Watch implements [Watcher] using fsnotify. Only Create/Write/Rename emit upserts.
func (s *LocalSource) Watch(ctx context.Context) (<-chan Change, error) {
	if err := os.MkdirAll(s.Dir, 0755); err != nil {
		return nil, err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := watcher.Add(s.Dir); err != nil {
		_ = watcher.Close()
		return nil, err
	}

	absDir, err := filepath.Abs(s.Dir)
	if err != nil {
		_ = watcher.Close()
		return nil, err
	}

	ch := make(chan Change, 16)
	go func() {
		defer close(ch)
		defer watcher.Close() //nolint:errcheck

		var mu sync.Mutex
		timers := make(map[string]*time.Timer)
		const debounceDelay = 250 * time.Millisecond

		schedule := func(path string) {
			base := filepath.Base(path)
			if !isSupportedFileName(base) {
				return
			}
			parent := filepath.Dir(path)
			absParent, err := filepath.Abs(parent)
			if err != nil || absParent != absDir {
				return
			}

			mu.Lock()
			if t, ok := timers[base]; ok {
				t.Stop()
			}
			timers[base] = time.AfterFunc(debounceDelay, func() {
				mu.Lock()
				delete(timers, base)
				mu.Unlock()
				select {
				case <-ctx.Done():
					return
				case ch <- Change{Name: base, Op: OpUpsert}:
				}
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
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return ch, nil
}
