package filesensor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"go.emeland.io/modelsrv/pkg/events"
	"gopkg.in/yaml.v3"
)

// DocumentKind is the resource discriminator for a [Document]; values are [events.ResourceType] and match the YAML `kind` field.
type DocumentKind events.ResourceType

var documentKinds = map[events.ResourceType]struct{}{
	events.ContextResource:     {},
	events.ContextTypeResource: {},
	events.NodeResource:        {},
	events.NodeTypeResource:    {},

	events.SystemResource:            {},
	events.SystemInstanceResource:    {},
	events.APIResource:               {},
	events.APIInstanceResource:       {},
	events.ComponentResource:         {},
	events.ComponentInstanceResource: {},

	events.OrgUnitResource:  {},
	events.GroupResource:    {},
	events.IdentityResource: {},
	events.ProductResource:  {},

	events.FindingResource:     {},
	events.FindingTypeResource: {},

	events.ArtifactResource:         {},
	events.ArtifactInstanceResource: {},
}

// ResourceType returns the underlying [events.ResourceType].
func (k DocumentKind) ResourceType() events.ResourceType {
	return events.ResourceType(k)
}

// UnmarshalYAML accepts only resource kinds valid for a top-level document (or empty for blank placeholders).
func (k *DocumentKind) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	s = strings.TrimSpace(s)
	if s == "" {
		*k = DocumentKind(events.UnknownResourceType)
		return nil
	}
	rt := events.ParseResourceType(s)
	if rt == events.UnknownResourceType {
		return fmt.Errorf("unsupported kind %q", s)
	}
	if _, ok := documentKinds[rt]; !ok {
		return fmt.Errorf("unsupported kind %q", s)
	}
	*k = DocumentKind(rt)
	return nil
}

// Document is one top-level resource in a multi-doc YAML stream.
type Document struct {
	Version string         `yaml:"version"`
	Kind    DocumentKind   `yaml:"kind"`
	Spec    map[string]any `yaml:"spec"`
}

// DecodeDocuments decodes a multi-document YAML stream into separate [Document] values.
func DecodeDocuments(data []byte) ([]Document, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))

	var docs []Document
	for {
		var doc Document
		err := dec.Decode(&doc)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		// Ignore completely empty YAML documents (e.g. stray `---` separators).
		if strings.TrimSpace(doc.Version) == "" && doc.Kind.ResourceType() == events.UnknownResourceType && doc.Spec == nil {
			continue
		}
		if strings.TrimSpace(doc.Version) == "" {
			return nil, fmt.Errorf("document %d: missing version", len(docs))
		}
		if doc.Kind.ResourceType() == events.UnknownResourceType {
			return nil, fmt.Errorf("document %d: missing kind", len(docs))
		}
		if doc.Spec == nil {
			return nil, fmt.Errorf("document %d: missing spec", len(docs))
		}
		docs = append(docs, doc)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no YAML documents found")
	}
	return docs, nil
}

// ValidVersion reports whether v uses an accepted emeland.io API version prefix.
func ValidVersion(v string) bool {
	v = strings.TrimSpace(v)
	return strings.HasPrefix(v, "emeland.io/")
}
