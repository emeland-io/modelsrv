// Package component provides the Component and ComponentInstance domain types
// for phase 1 of the EmELand model (system structure and instances).
//
// # OpenAPI gaps
//
// TODO: The OpenAPI [ComponentInstance] schema includes consumes and provides
// (lists of API instance UUIDs), but the domain [ComponentInstance] type does
// not model those fields yet. Wire payloads and GET responses may carry them
// while the model silently ignores them until a follow-up issue adds domain
// fields, converters, and replication support.
package component
