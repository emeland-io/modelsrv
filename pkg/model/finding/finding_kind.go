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

	// CertificateExpiringSoon is raised when an ApiInstance TLS certificate
	// will expire within the configured warning threshold.
	CertificateExpiringSoon FindingKind = "CertificateExpiringSoon"

	// CertificateExpired is raised when an ApiInstance TLS certificate
	// has already expired (remaining lifetime ≤ 0).
	CertificateExpired FindingKind = "CertificateExpired"

	// CertificateProbeFailed is raised when a certificate probe against an
	// ApiInstance endpoint fails (connection error, timeout, etc.).
	CertificateProbeFailed FindingKind = "CertificateProbeFailed"

	// ReferencedResourceNotFound is raised when a subject cites a resource UUID
	// that is not registered in the local model. Resources layout: [subject, missing].
	ReferencedResourceNotFound FindingKind = "ReferencedResourceNotFound"

	// MissingResourceReference is raised when a subject lacks a required EmELand
	// reference (e.g. ApiInstance without an API ref). Resources layout: [subject].
	MissingResourceReference FindingKind = "MissingResourceReference"

	// DescriptionMissing is raised when a System, API, or Component has an empty
	// description (documentation needed for C4 System Structure diagrams).
	DescriptionMissing FindingKind = "DescriptionMissing"

	// ConcreteSystemHasNoComponents is raised when a non-abstract System has no
	// Components (its C4 Container diagram boundary would be empty).
	ConcreteSystemHasNoComponents FindingKind = "ConcreteSystemHasNoComponents"

	// AbstractSystemHasComponents is raised when an abstract System has Components
	// (contradicts the black-box rule and pollutes Container diagrams).
	AbstractSystemHasComponents FindingKind = "AbstractSystemHasComponents"

	// APIHasNoProvider is raised when a non-abstract System's API is not listed in
	// any Component.provides (book: exactly one provider).
	APIHasNoProvider FindingKind = "APIHasNoProvider"

	// APIHasMultipleProviders is raised when more than one Component lists the same
	// API in provides. Resources layout: [API, provider Components...].
	APIHasMultipleProviders FindingKind = "APIHasMultipleProviders"

	// ProvidedAPISystemMismatch is raised when a Component provides an API whose
	// owning System is not the Component's System. Resources: [Component, API].
	ProvidedAPISystemMismatch FindingKind = "ProvidedAPISystemMismatch"

	// SystemInstanceContextMissing is raised when a SystemInstance has no Context
	// ref and therefore cannot be placed on C4 Context or deployment diagrams.
	SystemInstanceContextMissing FindingKind = "SystemInstanceContextMissing"

	// ApiInstanceMissingForComponentInstance is raised when a ComponentInstance's
	// type provides or consumes an API that has no ApiInstance in the same
	// SystemInstance, so no deployment edge can be drawn.
	// Resources: [ComponentInstance, API...].
	ApiInstanceMissingForComponentInstance FindingKind = "ApiInstanceMissingForComponentInstance"

	// ApiInstanceSystemInstanceMissing is raised when an ApiInstance has no
	// SystemInstance ref, so it has no boundary to be drawn inside on the
	// deployment diagram. Mirrors SystemInstanceContextMissing one level down.
	ApiInstanceSystemInstanceMissing FindingKind = "ApiInstanceSystemInstanceMissing"
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
	case CertificateExpiringSoon:
		return "An ApiInstance TLS certificate will expire within the configured warning threshold."
	case CertificateExpired:
		return "An ApiInstance TLS certificate has already expired."
	case CertificateProbeFailed:
		return "A certificate probe against an ApiInstance endpoint failed."
	case ReferencedResourceNotFound:
		return "A resource references another resource by UUID that is not registered in the local model."
	case MissingResourceReference:
		return "A resource lacks a required EmELand reference to another resource."
	case DescriptionMissing:
		return "A System, API, or Component has an empty description required for System Structure documentation."
	case ConcreteSystemHasNoComponents:
		return "A non-abstract System has no Components, so its C4 Container diagram boundary would be empty."
	case AbstractSystemHasComponents:
		return "An abstract System has Components; abstract systems are black boxes with APIs only."
	case APIHasNoProvider:
		return "An API belonging to a non-abstract System is not provided by any Component."
	case APIHasMultipleProviders:
		return "More than one Component provides the same API; each API must have exactly one provider."
	case ProvidedAPISystemMismatch:
		return "A Component provides an API that belongs to a different System than the Component."
	case SystemInstanceContextMissing:
		return "A SystemInstance has no Context reference and cannot be placed on C4 Context or deployment diagrams."
	case ApiInstanceMissingForComponentInstance:
		return "A ComponentInstance's type provides or consumes an API that has no ApiInstance in the same SystemInstance."
	case ApiInstanceSystemInstanceMissing:
		return "An ApiInstance has no SystemInstance reference, so it has no boundary to be placed in on the C4 deployment diagram."
	default:
		return ""
	}
}
