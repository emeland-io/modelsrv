// Package model defines the structure of the example mapping of the EmELand abstract model (TODO: link to EmELand book)
// and implements the required functions to manage this model.
//
// Domain types and constructors live in subpackages: api, node, system, component, finding, ctx, annotations,
// and common (shared value types and sentinel errors). This package defines [Model], [NewModel], replication,
// and test helpers such as [MakeTestSystem]. The ctx subpackage holds context entities (package name ctx avoids
// clashing with the standard library context package on import).
//
// The model tree has internal support to send events via the [events] package to publish changes to external listeners.
//
// # Architecture Comments
//
// Each interface / struct combination represents a specific entity in the EmELand abstract model.
// While entities with UUID identifiers are considered reference objects, those without UUIDs are considered value objects,
// as defined in the book "Domain-Driven Design" by Eric Evans.
// The data structures backing each of the reference objects must implement the [EventSink] interface, so that events, emitted for changes to their associated value
// objects, can be processed.
//
// Values objects call the [events.EventSink.Receive] method of their parent reference object, setting the uuid value to [uuid.Nil], to propagate events upwards
// in the model hierarchy. Reference objects, in turn, forward these events to the [events.EventSink] instance, registered with the [Model] object,
// which is at the root of the model hierarchy.
package model
