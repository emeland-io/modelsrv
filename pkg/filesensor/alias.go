package filesensor

import (
	"go.emeland.io/modelsrv/pkg/ingress"
	"go.emeland.io/modelsrv/pkg/model"
)

// Re-exported [ingress] types, so callers that only deal with files keep a single
// import. Parsing and applying themselves live in pkg/ingress.

// Document is one top-level resource (see [ingress.Document]).
type Document = ingress.Document

// DocumentKind is the resource discriminator (see [ingress.DocumentKind]).
type DocumentKind = ingress.DocumentKind

// DocumentError records a single document that could not be applied.
type DocumentError = ingress.DocumentError

// ProcessFileResult is the outcome of applying a multi-document file.
type ProcessFileResult = ingress.ProcessResult

// DecodeDocuments decodes a multi-document YAML stream.
func DecodeDocuments(data []byte) ([]Document, error) {
	return ingress.DecodeDocuments(data)
}

// ValidVersion reports whether v uses an accepted emeland.io API version prefix.
func ValidVersion(v string) bool {
	return ingress.ValidVersion(v)
}

// ApplyDocument validates and applies a single decoded Document to the model.
func ApplyDocument(doc Document, m model.Model) error {
	return ingress.ApplyDocument(doc, m)
}

// ParseOptions controls format parsing for a Source file (see [ingress.ParseOptions]).
type ParseOptions = ingress.ParseOptions

// Format aliases for Sensor config.
type Format = ingress.Format

const (
	FormatYAML = ingress.FormatYAML
	FormatJSON = ingress.FormatJSON
	FormatCSV  = ingress.FormatCSV
)
