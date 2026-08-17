package ingress

import (
	"fmt"

	"go.emeland.io/modelsrv/pkg/model"
)

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

// ProcessResult is the outcome of applying a list of decoded documents.
type ProcessResult struct {
	Applied int             // documents successfully applied
	Failed  []DocumentError // documents skipped (logged by caller)
}

// ApplyAll applies each document to m in order.
// Invalid documents are skipped; processing continues for the rest.
func ApplyAll(docs []Document, m model.Model) ProcessResult {
	var out ProcessResult
	for i := range docs {
		if err := ApplyDocument(docs[i], m); err != nil {
			out.Failed = append(out.Failed, DocumentError{Index: i, Err: err})
			continue
		}
		out.Applied++
	}
	return out
}
