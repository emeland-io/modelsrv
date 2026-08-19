package filesensor

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/ingress"
	"gopkg.in/yaml.v3"
)

// Config is the multi-source file-sensor configuration.
type Config struct {
	Sources []SourceConfig `yaml:"sources"`
}

// SourceConfig describes one origin (local, HTTP, or S3) and optional per-glob parser options.
type SourceConfig struct {
	URI     string                  `yaml:"uri"`
	Watch   bool                    `yaml:"watch"`
	Poll    time.Duration           `yaml:"poll"`
	Timeout time.Duration           `yaml:"timeout"` // HTTP client timeout; default [DefaultHTTPTimeout]
	Files   map[string]FileParseCfg `yaml:"files"`
}

// FileParseCfg is YAML-friendly parser options for a glob.
type FileParseCfg struct {
	Format    string            `yaml:"format"`
	Kind      string            `yaml:"kind"`
	Version   string            `yaml:"version"`
	Delimiter string            `yaml:"delimiter"`
	Columns   map[string]string `yaml:"columns"`
}

// LoadConfigFile reads a YAML sensor config from path.
func LoadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseConfig(data)
}

// ParseConfig unmarshals a sensor config from YAML bytes.
func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if len(cfg.Sources) == 0 {
		return nil, fmt.Errorf("config: no sources")
	}
	return &cfg, nil
}

// Open builds a Source and ParserConfig for this SourceConfig.
func (sc SourceConfig) Open(ctx context.Context) (Source, ParserConfig, error) {
	parser, err := sc.parserConfig()
	if err != nil {
		return nil, nil, err
	}
	uri := strings.TrimSpace(sc.URI)
	if uri == "" {
		return nil, nil, fmt.Errorf("uri is required")
	}

	switch {
	case strings.HasPrefix(uri, "file://"):
		dir := strings.TrimPrefix(uri, "file://")
		if dir == "" {
			return nil, nil, fmt.Errorf("file URI missing path: %q", uri)
		}
		if err := requireLocalDir(dir); err != nil {
			return nil, nil, err
		}
		return NewLocalSource(dir), parser, nil
	case strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://"):
		src := NewHTTPSource(uri)
		src.Client = httpClientWithTimeout(sc.Timeout)
		return src, parser, nil
	case strings.HasPrefix(uri, "s3://"):
		src, err := NewS3SourceFromURI(ctx, uri)
		if err != nil {
			return nil, nil, err
		}
		return src, parser, nil
	default:
		// Bare path treated as local directory (shorthand).
		if u, err := url.Parse(uri); err == nil && u.Scheme != "" && u.Scheme != "file" {
			return nil, nil, fmt.Errorf("unsupported source scheme %q", u.Scheme)
		}
		if err := requireLocalDir(uri); err != nil {
			return nil, nil, err
		}
		return NewLocalSource(uri), parser, nil
	}
}

func requireLocalDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("local source directory %q does not exist", dir)
		}
		return fmt.Errorf("stat local source directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("local source path %q is not a directory", dir)
	}
	return nil
}

func (sc SourceConfig) parserConfig() (ParserConfig, error) {
	if len(sc.Files) == 0 {
		return StaticParserConfig{}, nil
	}
	rules := make([]GlobRule, 0, len(sc.Files))
	for pattern, fc := range sc.Files {
		opts, err := fc.toParseOptions()
		if err != nil {
			return nil, fmt.Errorf("files[%q]: %w", pattern, err)
		}
		rules = append(rules, GlobRule{Pattern: pattern, Opts: opts})
	}
	sortGlobRules(rules)
	return GlobParserConfig{Rules: rules}, nil
}

// sortGlobRules imposes a stable, specific-first order on rules built from a map,
// so that overlapping patterns resolve the same way on every process start.
// Literal names beat wildcards; among equals, the longer pattern wins.
func sortGlobRules(rules []GlobRule) {
	sort.Slice(rules, func(i, j int) bool {
		a, b := rules[i].Pattern, rules[j].Pattern
		if wa, wb := hasWildcard(a), hasWildcard(b); wa != wb {
			return wb
		}
		if len(a) != len(b) {
			return len(a) > len(b)
		}
		return a < b
	})
}

func hasWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

func (fc FileParseCfg) toParseOptions() (ingress.ParseOptions, error) {
	opts := ingress.ParseOptions{
		Version: strings.TrimSpace(fc.Version),
		Columns: fc.Columns,
	}
	switch strings.ToLower(strings.TrimSpace(fc.Format)) {
	case "":
		// leave Format empty for extension detect
	case "yaml", "yml":
		opts.Format = ingress.FormatYAML
	case "json", "jsonl":
		opts.Format = ingress.FormatJSON
	case "csv":
		opts.Format = ingress.FormatCSV
	default:
		return opts, fmt.Errorf("unknown format %q", fc.Format)
	}
	if k := strings.TrimSpace(fc.Kind); k != "" {
		rt := events.ParseResourceType(k)
		if rt == events.UnknownResourceType {
			return opts, fmt.Errorf("unknown kind %q", k)
		}
		opts.Kind = rt
	}
	if d := fc.Delimiter; d != "" {
		r := []rune(d)
		if len(r) != 1 {
			return opts, fmt.Errorf("delimiter must be a single rune")
		}
		opts.Delimiter = r[0]
	}
	return opts, nil
}
