package model

// ApiType classifies an API (aligned with the Emerging Enterprise Landscape OpenAPI enum).
type ApiType int

const (
	Unknown ApiType = iota
	OpenAPI
	GraphQL
	GRPC
	Other
)

// String returns the API type label used in APIs and events. Unknown is returned for invalid values.
func (t ApiType) String() string {
	switch t {
	case Unknown:
		return "Unknown"
	case OpenAPI:
		return "OpenAPI"
	case GraphQL:
		return "GraphQL"
	case GRPC:
		return "GRPC"
	case Other:
		return "Other"
	default:
		return "Unknown"
	}
}
