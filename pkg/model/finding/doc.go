// Package finding contains the [Finding] and [FindingType] domain resources.
//
// # Concepts
//
// A Finding represents a detected condition in the landscape model that
// warrants attention — typically a referential-integrity violation, a missing
// required relationship, or any other rule that a filter decides to surface.
// It is a first-class model resource: it has a stable UUID, participates in
// the event stream, and can be queried via [model.Model.GetFindings].
//
// A FindingType classifies Findings.  It is also a first-class resource with
// its own UUID, display name, and description.  The relationship mirrors the
// one between Node and NodeType, or Context and ContextType: a Finding holds a
// [FindingTypeRef] pointing to its FindingType, resolved lazily through the
// model.
//
// # FindingKind and well-known types
//
// [FindingKind] is a string constant that names a category of findings
// (e.g. [ContextTypeMissing], [NodeTypeMissing]).  For each kind,
// [TypeIDForKind] derives a deterministic, stable UUID from the kind string
// using UUID v5.  This lets filter code call [Finding.SetFindingTypeById] with
// a well-known UUID without needing the corresponding FindingType resource to
// be registered in the model beforehand.  When the FindingType is later
// registered under that UUID, [Finding.GetFindingType] will resolve it.
//
// # Finding identity and lifecycle
//
// Findings created by built-in filters (see [pkg/eventfilter/phase0]) use
// deterministic UUIDs derived from the subject resource's UUID and the
// FindingKind, so re-applying the same event produces an upsert rather than a
// duplicate finding.  When the violating condition is resolved (e.g. a missing
// ContextType is added to the model), the filter deletes the corresponding
// finding automatically.
//
// # Fields
//
// Each Finding carries:
//
//   - Summary      – a human-readable one-line description of the violation.
//   - Description  – optional longer explanation.
//   - TypeRef      – reference to the [FindingType] that classifies this finding.
//   - Resources    – a list of [common.ResourceRef] values pointing to the
//     resources involved in the violation (subject first, then any
//     referenced-but-missing resources).
//   - Annotations  – arbitrary key/value metadata.
package finding
