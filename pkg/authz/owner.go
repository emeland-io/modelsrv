package authz

import (
	"strings"

	"go.emeland.io/modelsrv/pkg/model/annotations"
)

// OwnerIdentitiesKey is the annotation key for identity owners (OIDC subject values).
const OwnerIdentitiesKey = "emeland.io/owner-identities"

// OwnerGroupsKey is the annotation key for group owners (group id values).
const OwnerGroupsKey = "emeland.io/owner-groups"

// OwnerIdentities returns identity owner ids from annotations.
func OwnerIdentities(a annotations.Annotations) []string {
	if a == nil {
		return nil
	}
	return parseList(a.GetValue(OwnerIdentitiesKey))
}

// OwnerGroups returns group owner ids from annotations.
func OwnerGroups(a annotations.Annotations) []string {
	if a == nil {
		return nil
	}
	return parseList(a.GetValue(OwnerGroupsKey))
}

// HasOwner reports whether the resource has at least one owner identity or group set.
func HasOwner(a annotations.Annotations) bool {
	return len(OwnerIdentities(a)) > 0 || len(OwnerGroups(a)) > 0
}

func parseList(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
