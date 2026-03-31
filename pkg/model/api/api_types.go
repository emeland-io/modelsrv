package api

import (
	"fmt"
	"strings"
)

// ApiType classifies an API (aligned with the Emerging Enterprise Landscape OpenAPI enum).
type ApiType int

const (
	Unknown ApiType = iota
	OpenAPI
	GraphQL
	GRPC
	Other
)

var apiTypeValues = map[ApiType]string{
	Unknown: "Unknown",
	OpenAPI: "OpenAPI",
	GraphQL: "GraphQL",
	GRPC:    "GRPC",
	Other:   "Other",
}

// String returns the API type label used in APIs and events. Unknown is returned for invalid values.
func (t ApiType) String() string {
	if label, exists := apiTypeValues[t]; exists {
		return label
	}
	return apiTypeValues[Unknown]
}

// ParseApiType parses a string into an ApiType, ignoring case. Unknown is returned for invalid values.
// The function does not trim the input string, so leading/trailing whitespace will cause parsing to fail. Use strings.TrimSpace before calling if needed.
func ParseApiType(s string) (ApiType, error) {
	for key, val := range apiTypeValues {
		if strings.EqualFold(val, s) {
			return key, nil
		}
	}
	return Unknown, fmt.Errorf("invalid API type %q (expected OpenAPI, GraphQL, GRPC, Other, or Unknown)", s)
}
