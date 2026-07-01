package finding

import "github.com/google/uuid"

// FindingKind is the canonical string identifier for a category of findings.
// It is used to derive stable [FindingType] UUIDs via [TypeIDForKind] so that
// well-known finding categories can be referenced by ID even before the
// corresponding FindingType resource is registered in the model.
type FindingKind string

const (
	// ContextTypeMissing is raised when a Context references a ContextType that
	// does not exist in the model, or has no type set at all.
	ContextTypeMissing FindingKind = "ContextTypeMissing"

	// ContextParentNotFound is raised when a Context references a parent Context
	// by UUID but that parent does not exist in the model.
	ContextParentNotFound FindingKind = "ContextParentNotFound"

	// NodeTypeMissing is raised when a Node has no NodeType assigned.
	NodeTypeMissing FindingKind = "NodeTypeMissing"
)

// findingTypeNamespace is the UUID v5 namespace used to derive stable
// FindingType UUIDs from FindingKind strings.
var findingTypeNamespace = uuid.MustParse("c3d4e5f6-a7b8-9012-cdef-012345678901")

// TypeIDForKind returns the deterministic [FindingType] UUID for the given
// FindingKind.  The same kind always produces the same UUID across processes,
// so callers can use SetFindingTypeById without first registering the type in
// the model.
func TypeIDForKind(kind FindingKind) uuid.UUID {
	return uuid.NewSHA1(findingTypeNamespace, []byte(kind))
}

// DescriptionForKind returns the canonical human-readable description for a
// well-known [FindingKind]. Unknown kinds return an empty string.
func DescriptionForKind(kind FindingKind) string {
	switch kind {
	case ContextTypeMissing:
		return "A Context resource references a ContextType by UUID that is not registered in the model, or has no ContextType assigned at all."
	case ContextParentNotFound:
		return "A Context resource references a parent Context by UUID that is not registered in the model."
	case NodeTypeMissing:
		return "A Node resource has no NodeType assigned."
	default:
		return ""
	}
}
