// Package c4injector is an EmELand Injector that emits C4-PlantUML diagrams from
// landscape resources and serves them as text/plain .puml over HTTP.
//
// Abstraction mapping (see docs/adr/c4-abstraction-mapping.md):
//   - Context (level 1): hardcoded landscape summary + EmELand Context boundaries
//   - Container (level 2): EmELand Components as C4 Containers inside a
//     System_Boundary per concrete System; abstract Systems as System_Ext
//   - Component (level 3): not implemented (HTTP 404) — an EmELand Component is
//     already a C4 Container, and Components have no sub-parts
//   - Deployment: SystemInstance / ComponentInstance / ApiInstance (not a C4 level).
//     Every instance is drawn; those with an unset or unresolvable SystemInstance
//     are grouped under a synthetic "(no Context)" boundary rather than dropped.
//   - Code (level 4): not implemented (HTTP 404)
//
// Documentation gaps are not recorded as Findings; the injector only depicts
// what is present in the landscape.
package c4injector
