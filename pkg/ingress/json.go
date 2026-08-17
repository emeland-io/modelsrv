package ingress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go.emeland.io/modelsrv/pkg/events"
)

// jsonDocument is the on-wire shape for JSON landscape documents.
type jsonDocument struct {
	Version string         `json:"version"`
	Kind    string         `json:"kind"`
	Spec    map[string]any `json:"spec"`
}

// DecodeJSONDocuments decodes one JSON object, an array of objects, or JSONL into [Document] values.
func DecodeJSONDocuments(data []byte) ([]Document, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("no JSON documents found")
	}

	// Prefer a single JSON value (object or array). Fall back to JSONL on failure
	// only when the payload looks like multiple top-level values.
	if trimmed[0] == '[' {
		var raw []jsonDocument
		if err := json.Unmarshal(trimmed, &raw); err != nil {
			return nil, err
		}
		return jsonDocsToDocuments(raw)
	}

	if trimmed[0] == '{' {
		var r jsonDocument
		if err := json.Unmarshal(trimmed, &r); err == nil {
			doc, err := jsonDocToDocument(r, 0)
			if err != nil {
				return nil, err
			}
			return []Document{doc}, nil
		}
		// Pretty-printed single object already handled; remaining failure with '{'
		// may be JSONL of objects — try line-oriented decode.
	}

	return decodeJSONL(trimmed)
}

func decodeJSONL(data []byte) ([]Document, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	var docs []Document
	for i := 0; ; i++ {
		var r jsonDocument
		err := dec.Decode(&r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		doc, err := jsonDocToDocument(r, i)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no JSON documents found")
	}
	return docs, nil
}

func jsonDocsToDocuments(raw []jsonDocument) ([]Document, error) {
	docs := make([]Document, 0, len(raw))
	for i, r := range raw {
		doc, err := jsonDocToDocument(r, i)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no JSON documents found")
	}
	return docs, nil
}

func jsonDocToDocument(r jsonDocument, index int) (Document, error) {
	if strings.TrimSpace(r.Version) == "" {
		return Document{}, fmt.Errorf("document %d: missing version", index)
	}
	kindStr := strings.TrimSpace(r.Kind)
	if kindStr == "" {
		return Document{}, fmt.Errorf("document %d: missing kind", index)
	}
	rt := events.ParseResourceType(kindStr)
	if rt == events.UnknownResourceType {
		return Document{}, fmt.Errorf("document %d: unsupported kind %q", index, kindStr)
	}
	if _, ok := documentKinds[rt]; !ok {
		return Document{}, fmt.Errorf("document %d: unsupported kind %q", index, kindStr)
	}
	if r.Spec == nil {
		return Document{}, fmt.Errorf("document %d: missing spec", index)
	}
	return Document{
		Version: r.Version,
		Kind:    DocumentKind(rt),
		Spec:    r.Spec,
	}, nil
}
