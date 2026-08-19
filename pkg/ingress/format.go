package ingress

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.emeland.io/modelsrv/pkg/events"
)

// Format identifies a document encoding.
type Format string

const (
	FormatYAML Format = "yaml"
	FormatJSON Format = "json"
	FormatCSV  Format = "csv"
)

// ParseOptions controls how [Parse] interprets bytes.
// Format empty means detect from the file name, then from ContentType.
//
// For CSV: Kind may be omitted when Columns maps a header to "kind" (e.g. resourcetype).
// Version defaults to [DefaultCSVVersion] when empty. Columns empty uses [DefaultCSVColumns].
// Map a header to "id" to fill that kind's primary UUID field (contextId, systemId, …).
type ParseOptions struct {
	Format Format

	// ContentType is a transport-supplied MIME hint (HTTP Content-Type). It is a
	// fallback for names without a known extension, never an override.
	ContentType string

	// CSV (and any non-self-describing format)
	Kind      events.ResourceType // optional default when no per-row kind column
	Version   string              // optional; defaults to DefaultCSVVersion
	Delimiter rune
	Columns   map[string]string // header -> spec path; "kind" / "version" / "id" are special
}

// DetectFormat returns a format from a file name or path extension.
func DetectFormat(name string) Format {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".yaml", ".yml":
		return FormatYAML
	case ".json", ".jsonl":
		return FormatJSON
	case ".csv":
		return FormatCSV
	default:
		return ""
	}
}

// FormatFromContentType returns a format from a MIME type, ignoring parameters
// such as "; charset=utf-8". It returns "" for unrecognised types.
func FormatFromContentType(contentType string) Format {
	mime := contentType
	if i := strings.Index(mime, ";"); i >= 0 {
		mime = mime[:i]
	}
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "application/yaml", "application/x-yaml", "text/yaml", "text/x-yaml":
		return FormatYAML
	case "application/json", "text/json", "application/x-ndjson", "application/jsonl":
		return FormatJSON
	case "text/csv", "application/csv":
		return FormatCSV
	default:
		return ""
	}
}

// ResolveFormat picks a format from opts.Format, else the name extension, else
// opts.ContentType.
func ResolveFormat(name string, opts ParseOptions) (Format, error) {
	if opts.Format != "" {
		return opts.Format, nil
	}
	if f := DetectFormat(name); f != "" {
		return f, nil
	}
	if f := FormatFromContentType(opts.ContentType); f != "" {
		return f, nil
	}
	return "", fmt.Errorf("cannot detect format from %q; set ParseOptions.Format", name)
}

// Parse decodes landscape documents from data using name and opts to select the format.
func Parse(name string, data []byte, opts ParseOptions) ([]Document, error) {
	format, err := ResolveFormat(name, opts)
	if err != nil {
		return nil, err
	}
	switch format {
	case FormatYAML:
		return DecodeDocuments(data)
	case FormatJSON:
		return DecodeJSONDocuments(data)
	case FormatCSV:
		return DecodeCSVDocuments(data, opts)
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}
