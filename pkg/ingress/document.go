package ingress

import (
	"fmt"
	"strings"

	"go.emeland.io/modelsrv/pkg/events"
)

// DocumentKind is the resource discriminator for a [Document]; values are [events.ResourceType]
// and match the on-wire `kind` field (YAML/JSON) or CSV resourcetype column.
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

	events.OrgUnitResource:        {},
	events.GroupResource:          {},
	events.IdentityResource:       {},
	events.PermissionSpecResource: {},
	events.RoleSpecResource:       {},
	events.PermissionResource:     {},
	events.RoleResource:           {},
	events.BindingResource:        {},
	events.ProductResource:        {},

	events.FindingResource:     {},
	events.FindingTypeResource: {},

	events.ArtifactResource:         {},
	events.ArtifactInstanceResource: {},

	events.FilterRuleResource: {},
	events.MergeRuleResource:  {},

	events.CapabilityResource:           {},
	events.ParameterResource:            {},
	events.CapacityResourceTypeResource: {},
	events.CapacityResource:             {},

	events.MetricResource:      {},
	events.ThresholdResource:   {},
	events.MetricValueResource: {},
}

// ResourceType returns the underlying [events.ResourceType].
func (k DocumentKind) ResourceType() events.ResourceType {
	return events.ResourceType(k)
}

func parseDocumentKind(s string) (DocumentKind, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return DocumentKind(events.UnknownResourceType), nil
	}
	rt := events.ParseResourceType(s)
	if rt == events.UnknownResourceType {
		return 0, fmt.Errorf("unsupported kind %q", s)
	}
	if _, ok := documentKinds[rt]; !ok {
		return 0, fmt.Errorf("unsupported kind %q", s)
	}
	return DocumentKind(rt), nil
}

// Document is one top-level landscape resource after format decode.
type Document struct {
	Version string         `yaml:"version" json:"version"`
	Kind    DocumentKind   `yaml:"kind" json:"kind"`
	Spec    map[string]any `yaml:"spec" json:"spec"`
}

// ValidVersion reports whether v uses an accepted emeland.io API version prefix.
func ValidVersion(v string) bool {
	v = strings.TrimSpace(v)
	return strings.HasPrefix(v, "emeland.io/")
}
