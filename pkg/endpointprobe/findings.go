package endpointprobe

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

// SHA-1 namespace so each (apiInstanceId, kind) maps to one finding id.
var certFindingNamespace = uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890")

const certFindingDisplayName = "Certificate expiry check"

var certFindingKinds = []finding.FindingKind{
	finding.CertificateExpiringSoon,
	finding.CertificateExpired,
	finding.CertificateProbeFailed,
}

func findingID(apiInstanceID uuid.UUID, kind finding.FindingKind) uuid.UUID {
	key := append(apiInstanceID[:], []byte(kind)...)
	return uuid.NewSHA1(certFindingNamespace, key)
}

func buildFinding(pub FindingPublisher, apiInstanceID uuid.UUID, kind finding.FindingKind, description string) finding.Finding {
	f := finding.NewFinding(findingID(apiInstanceID, kind))
	f.SetFindingTypeById(pub.EnsureType(kind))
	f.SetDisplayName(certFindingDisplayName)
	f.SetDescription(description)
	f.SetResources([]*common.ResourceRef{
		{ResourceId: apiInstanceID, ResourceType: events.APIInstanceResource},
	})
	return f
}

func deleteKinds(pub FindingPublisher, apiInstanceID uuid.UUID, kinds ...finding.FindingKind) error {
	for _, kind := range kinds {
		if err := pub.Delete(findingID(apiInstanceID, kind)); err != nil {
			return err
		}
	}
	return nil
}

func deleteAllCertFindings(pub FindingPublisher, apiInstanceID uuid.UUID) error {
	return deleteKinds(pub, apiInstanceID, certFindingKinds...)
}

// reconcileFinding upserts or deletes certificate findings for a probe result
// according to the decision table:
//
//	!Success                     → CertificateProbeFailed
//	Success && !HasCert          → delete all
//	cert_remaining > threshold   → delete all (resolved)
//	0 < cert_remaining ≤ thresh  → CertificateExpiringSoon
//	cert_remaining ≤ 0           → CertificateExpired
func reconcileFinding(pub FindingPublisher, threshold time.Duration, r ProbeResult) error {
	id := r.Target.ApiInstanceID

	if !r.Success {
		desc := fmt.Sprintf("CertificateProbeFailed: probe of ApiInstance %s failed", id)
		if r.Err != nil {
			desc = fmt.Sprintf("CertificateProbeFailed: probe of ApiInstance %s failed: %v", id, r.Err)
		}
		if err := pub.Upsert(buildFinding(pub, id, finding.CertificateProbeFailed, desc)); err != nil {
			return err
		}
		return deleteKinds(pub, id, finding.CertificateExpiringSoon, finding.CertificateExpired)
	}

	if !r.HasCert {
		return deleteAllCertFindings(pub, id)
	}

	remaining := r.CertRemaining

	switch {
	case remaining > threshold:
		return deleteAllCertFindings(pub, id)
	case remaining > 0:
		desc := fmt.Sprintf(
			"CertificateExpiringSoon: ApiInstance %s certificate expires in %s (threshold %s)",
			id, remaining.Round(time.Second), threshold,
		)
		if err := pub.Upsert(buildFinding(pub, id, finding.CertificateExpiringSoon, desc)); err != nil {
			return err
		}
		return deleteKinds(pub, id, finding.CertificateExpired, finding.CertificateProbeFailed)
	default:
		desc := fmt.Sprintf(
			"CertificateExpired: ApiInstance %s certificate expired %s ago",
			id, (-remaining).Round(time.Second),
		)
		if err := pub.Upsert(buildFinding(pub, id, finding.CertificateExpired, desc)); err != nil {
			return err
		}
		return deleteKinds(pub, id, finding.CertificateExpiringSoon, finding.CertificateProbeFailed)
	}
}
