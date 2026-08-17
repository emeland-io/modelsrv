package filesensor

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"go.emeland.io/modelsrv/pkg/ingress"
	"go.emeland.io/modelsrv/pkg/model"
)

// FileMeta describes one file available from a [Source].
type FileMeta struct {
	Name         string // relative path / key
	ETag         string // optional change token
	ContentType  string // optional MIME hint, used when Name has no known extension
	LastModified time.Time
}

// Op is a change operation on a Source file.
type Op string

const (
	OpUpsert Op = "upsert"
	OpDelete Op = "delete" // v2; ignored by apply path today
)

// Change is a notification that a Source file changed.
type Change struct {
	Name string
	Op   Op
}

// Source lists and reads files from a persistence backend (local, HTTP, S3, …).
type Source interface {
	List(ctx context.Context) ([]FileMeta, error)
	Read(ctx context.Context, name string) ([]byte, error)
}

// Watcher optionally streams change notifications. Local Source implements this with fsnotify.
type Watcher interface {
	Watch(ctx context.Context) (<-chan Change, error)
}

// ParserConfig resolves [ingress.ParseOptions] for a file name (glob-based).
type ParserConfig interface {
	OptionsFor(name string) (ingress.ParseOptions, error)
}

// StaticParserConfig uses one ParseOptions for every file (extension-based format detect).
type StaticParserConfig struct {
	Opts ingress.ParseOptions
}

// OptionsFor implements [ParserConfig].
func (c StaticParserConfig) OptionsFor(name string) (ingress.ParseOptions, error) {
	return c.Opts, nil
}

// GlobParserConfig maps filepath globs to ParseOptions. Rules are evaluated in
// slice order and the first match wins, so callers building rules from an
// unordered source must sort them (see [sortGlobRules]).
type GlobParserConfig struct {
	Rules []GlobRule
}

// GlobRule binds a filepath.Match pattern to parse options.
type GlobRule struct {
	Pattern string
	Opts    ingress.ParseOptions
}

// OptionsFor implements [ParserConfig].
func (c GlobParserConfig) OptionsFor(name string) (ingress.ParseOptions, error) {
	base := filepath.Base(name)
	for _, r := range c.Rules {
		ok, err := filepath.Match(r.Pattern, base)
		if err != nil {
			return ingress.ParseOptions{}, fmt.Errorf("glob %q: %w", r.Pattern, err)
		}
		if ok {
			return r.Opts, nil
		}
		ok, err = filepath.Match(r.Pattern, name)
		if err != nil {
			return ingress.ParseOptions{}, fmt.Errorf("glob %q: %w", r.Pattern, err)
		}
		if ok {
			return r.Opts, nil
		}
	}
	// Default: auto-detect format; CSV without columns will fail in ingress.
	return ingress.ParseOptions{}, nil
}

func isSupportedFileName(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".yaml") ||
		strings.HasSuffix(lower, ".yml") ||
		strings.HasSuffix(lower, ".json") ||
		strings.HasSuffix(lower, ".jsonl") ||
		strings.HasSuffix(lower, ".csv")
}

// ProcessBytes parses data and applies documents to m.
func ProcessBytes(name string, data []byte, opts ingress.ParseOptions, m model.Model) (ProcessFileResult, error) {
	docs, err := ingress.Parse(name, data, opts)
	if err != nil {
		return ProcessFileResult{}, err
	}
	return ingress.ApplyAll(docs, m), nil
}

// ProcessFile reads a local YAML/JSON/CSV file and applies each document to m.
// Kept for tests and callers that still pass a filesystem path.
func ProcessFile(path string, m model.Model) (ProcessFileResult, error) {
	src := NewLocalSource(filepath.Dir(path))
	name := filepath.Base(path)
	data, err := src.Read(context.Background(), name)
	if err != nil {
		return ProcessFileResult{}, err
	}
	return ProcessBytes(name, data, ingress.ParseOptions{}, m)
}
