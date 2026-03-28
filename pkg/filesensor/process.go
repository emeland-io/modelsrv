package filesensor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.emeland.io/modelsrv/pkg/model"
)

func isYAMLFileName(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

// DocumentError records a single document that could not be applied.
type DocumentError struct {
	Index int   // 0-based index within the file
	Err   error // validation or apply error
}

func (e DocumentError) Error() string {
	return fmt.Sprintf("document %d: %v", e.Index, e.Err)
}

func (e DocumentError) Unwrap() error {
	return e.Err
}

// ProcessFileResult is the outcome of applying a multi-document YAML file.
type ProcessFileResult struct {
	Applied int             // documents successfully applied
	Failed  []DocumentError // documents skipped (logged by caller)
}

// ProcessFile reads a YAML file and applies each document to m in order.
// Invalid documents are skipped; processing continues for the rest of the file.
// A non-nil error is returned only for I/O failures or YAML stream decode errors (not per-document apply errors).
func ProcessFile(path string, m model.Model) (ProcessFileResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProcessFileResult{}, err
	}
	docs, err := DecodeDocuments(data)
	if err != nil {
		return ProcessFileResult{}, err
	}
	var out ProcessFileResult
	for i := range docs {
		if err := ApplyDocument(docs[i], m); err != nil {
			out.Failed = append(out.Failed, DocumentError{Index: i, Err: err})
			continue
		}
		out.Applied++
	}
	return out, nil
}

func scanDir(dir string, process func(path string)) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !isYAMLFileName(e.Name()) {
			continue
		}
		process(filepath.Join(dir, e.Name()))
	}
	return nil
}
