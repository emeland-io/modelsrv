package endpointprobe

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/pkg/backend"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

// fakePublisher records Upsert/Delete calls for decision-table assertions.
type fakePublisher struct {
	upserts []finding.Finding
	deletes []uuid.UUID
}

func (f *fakePublisher) Upsert(find finding.Finding) error {
	f.upserts = append(f.upserts, find)
	return nil
}

func (f *fakePublisher) Delete(id uuid.UUID) error {
	f.deletes = append(f.deletes, id)
	return nil
}

func (f *fakePublisher) EnsureType(kind finding.FindingKind) uuid.UUID {
	return finding.TypeIDForKind(kind)
}

func (f *fakePublisher) upsertedKinds() []finding.FindingKind {
	var kinds []finding.FindingKind
	for _, u := range f.upserts {
		for _, k := range certFindingKinds {
			if u.GetFindingId() == findingID(u.GetResources()[0].ResourceId, k) {
				kinds = append(kinds, k)
				break
			}
		}
	}
	return kinds
}

func (f *fakePublisher) deletedKinds(apiInstanceID uuid.UUID) []finding.FindingKind {
	var kinds []finding.FindingKind
	for _, id := range f.deletes {
		for _, k := range certFindingKinds {
			if id == findingID(apiInstanceID, k) {
				kinds = append(kinds, k)
			}
		}
	}
	return kinds
}

func probeResult(apiInstanceID uuid.UUID, success, hasCert bool, remaining time.Duration, err error) ProbeResult {
	return ProbeResult{
		Target:        ProbeTarget{ApiInstanceID: apiInstanceID},
		Success:       success,
		HasCert:       hasCert,
		CertRemaining: remaining,
		Err:           err,
		ProbedAt:      time.Now(),
	}
}

func findingsOfKind(m model.Model, kind finding.FindingKind) []finding.Finding {
	typeID := finding.TypeIDForKind(kind)
	all, err := m.GetFindings()
	Expect(err).NotTo(HaveOccurred())
	var out []finding.Finding
	for _, f := range all {
		if f.GetFindingTypeId() == typeID {
			out = append(out, f)
		}
	}
	return out
}

var testApiInstanceID = uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

var _ = Describe("Certificate findings", func() {
	const threshold = 720 * time.Hour

	Describe("TypeIDForKind stability", func() {
		It("returns stable UUIDs for the three certificate kinds", func() {
			kinds := []finding.FindingKind{
				finding.CertificateExpiringSoon,
				finding.CertificateExpired,
				finding.CertificateProbeFailed,
			}
			for _, kind := range kinds {
				first := finding.TypeIDForKind(kind)
				second := finding.TypeIDForKind(kind)
				Expect(first).To(Equal(second), "kind %s should be deterministic", kind)
				Expect(first).NotTo(Equal(uuid.Nil))
			}
			Expect(finding.TypeIDForKind(finding.CertificateExpiringSoon)).NotTo(Equal(
				finding.TypeIDForKind(finding.CertificateExpired),
			))
			Expect(finding.TypeIDForKind(finding.CertificateExpired)).NotTo(Equal(
				finding.TypeIDForKind(finding.CertificateProbeFailed),
			))
		})

		It("returns non-empty descriptions for the three certificate kinds", func() {
			Expect(finding.DescriptionForKind(finding.CertificateExpiringSoon)).NotTo(BeEmpty())
			Expect(finding.DescriptionForKind(finding.CertificateExpired)).NotTo(BeEmpty())
			Expect(finding.DescriptionForKind(finding.CertificateProbeFailed)).NotTo(BeEmpty())
		})
	})

	Describe("findingID determinism", func() {
		It("derives the same UUID for the same (apiInstanceId, kind)", func() {
			a := findingID(testApiInstanceID, finding.CertificateExpiringSoon)
			b := findingID(testApiInstanceID, finding.CertificateExpiringSoon)
			Expect(a).To(Equal(b))
		})

		It("derives different UUIDs for different kinds on the same ApiInstance", func() {
			soon := findingID(testApiInstanceID, finding.CertificateExpiringSoon)
			expired := findingID(testApiInstanceID, finding.CertificateExpired)
			failed := findingID(testApiInstanceID, finding.CertificateProbeFailed)
			Expect(soon).NotTo(Equal(expired))
			Expect(soon).NotTo(Equal(failed))
			Expect(expired).NotTo(Equal(failed))
		})

		It("derives different UUIDs for the same kind on different ApiInstances", func() {
			other := uuid.MustParse("11111111-2222-3333-4444-555555555555")
			Expect(findingID(testApiInstanceID, finding.CertificateExpired)).NotTo(Equal(
				findingID(other, finding.CertificateExpired),
			))
		})
	})

	Describe("reconcileFinding decision table", func() {
		var pub *fakePublisher

		BeforeEach(func() {
			pub = &fakePublisher{}
		})

		DescribeTable("maps probe outcome to upsert/delete actions",
			func(result ProbeResult, wantUpsert finding.FindingKind, wantDelete []finding.FindingKind) {
				Expect(reconcileFinding(pub, threshold, result)).To(Succeed())

				if wantUpsert == "" {
					Expect(pub.upserts).To(BeEmpty())
				} else {
					Expect(pub.upsertedKinds()).To(Equal([]finding.FindingKind{wantUpsert}))
					Expect(pub.upserts[0].GetResources()).To(HaveLen(1))
					Expect(pub.upserts[0].GetResources()[0].ResourceId).To(Equal(testApiInstanceID))
					Expect(pub.upserts[0].GetResources()[0].ResourceType).To(Equal(events.APIInstanceResource))
					Expect(pub.upserts[0].GetFindingTypeId()).To(Equal(finding.TypeIDForKind(wantUpsert)))
				}

				Expect(pub.deletedKinds(testApiInstanceID)).To(ConsistOf(wantDelete))
			},
			Entry("probe failed → CertificateProbeFailed",
				probeResult(testApiInstanceID, false, false, 0, errors.New("connection refused")),
				finding.CertificateProbeFailed,
				[]finding.FindingKind{finding.CertificateExpiringSoon, finding.CertificateExpired},
			),
			Entry("success without cert → delete all",
				probeResult(testApiInstanceID, true, false, 0, nil),
				finding.FindingKind(""),
				[]finding.FindingKind{
					finding.CertificateExpiringSoon,
					finding.CertificateExpired,
					finding.CertificateProbeFailed,
				},
			),
			Entry("cert remaining above threshold → delete all (resolved)",
				probeResult(testApiInstanceID, true, true, 1000*time.Hour, nil),
				finding.FindingKind(""),
				[]finding.FindingKind{
					finding.CertificateExpiringSoon,
					finding.CertificateExpired,
					finding.CertificateProbeFailed,
				},
			),
			Entry("cert remaining within threshold → CertificateExpiringSoon",
				probeResult(testApiInstanceID, true, true, 24*time.Hour, nil),
				finding.CertificateExpiringSoon,
				[]finding.FindingKind{finding.CertificateExpired, finding.CertificateProbeFailed},
			),
			Entry("cert remaining exactly at threshold → CertificateExpiringSoon",
				probeResult(testApiInstanceID, true, true, threshold, nil),
				finding.CertificateExpiringSoon,
				[]finding.FindingKind{finding.CertificateExpired, finding.CertificateProbeFailed},
			),
			Entry("cert remaining ≤ 0 → CertificateExpired",
				probeResult(testApiInstanceID, true, true, -time.Hour, nil),
				finding.CertificateExpired,
				[]finding.FindingKind{finding.CertificateExpiringSoon, finding.CertificateProbeFailed},
			),
			Entry("cert remaining exactly 0 → CertificateExpired",
				probeResult(testApiInstanceID, true, true, 0, nil),
				finding.CertificateExpired,
				[]finding.FindingKind{finding.CertificateExpiringSoon, finding.CertificateProbeFailed},
			),
		)
	})

	Describe("ModelFindingPublisher integration", func() {
		var (
			b   backend.Backend
			pub *ModelFindingPublisher
			m   model.Model
		)

		BeforeEach(func() {
			var err error
			b, err = backend.New()
			Expect(err).NotTo(HaveOccurred())
			m = b.GetModel()
			pub = NewModelFindingPublisher(m)
		})

		It("makes findings appear in the model after an expiring probe (GET /landscape/findings source)", func() {
			result := probeResult(testApiInstanceID, true, true, 24*time.Hour, nil)
			Expect(reconcileFinding(pub, threshold, result)).To(Succeed())

			finds := findingsOfKind(m, finding.CertificateExpiringSoon)
			Expect(finds).To(HaveLen(1))
			Expect(finds[0].GetFindingId()).To(Equal(findingID(testApiInstanceID, finding.CertificateExpiringSoon)))
			Expect(finds[0].GetResources()).To(HaveLen(1))
			Expect(finds[0].GetResources()[0].ResourceId).To(Equal(testApiInstanceID))
			Expect(finds[0].GetResources()[0].ResourceType).To(Equal(events.APIInstanceResource))

			ft := m.GetFindingTypeById(finding.TypeIDForKind(finding.CertificateExpiringSoon))
			Expect(ft).NotTo(BeNil(), "FindingType should be ensured before upsert")
			Expect(ft.GetDisplayName()).To(Equal(string(finding.CertificateExpiringSoon)))
			Expect(ft.GetDescription()).To(Equal(finding.DescriptionForKind(finding.CertificateExpiringSoon)))
		})

		It("EnsureWellKnownFindingTypes pre-registers all certificate FindingTypes", func() {
			EnsureWellKnownFindingTypes(m)
			for _, kind := range certFindingKinds {
				ft := m.GetFindingTypeById(finding.TypeIDForKind(kind))
				Expect(ft).NotTo(BeNil(), "kind %s", kind)
				Expect(ft.GetDisplayName()).To(Equal(string(kind)))
				Expect(ft.GetDescription()).To(Equal(finding.DescriptionForKind(kind)))
			}
		})

		It("is idempotent: re-probing the same state does not duplicate findings", func() {
			result := probeResult(testApiInstanceID, true, true, 24*time.Hour, nil)
			Expect(reconcileFinding(pub, threshold, result)).To(Succeed())
			Expect(reconcileFinding(pub, threshold, result)).To(Succeed())

			finds := findingsOfKind(m, finding.CertificateExpiringSoon)
			Expect(finds).To(HaveLen(1))
			all, err := m.GetFindings()
			Expect(err).NotTo(HaveOccurred())
			certCount := 0
			for _, f := range all {
				for _, k := range certFindingKinds {
					if f.GetFindingId() == findingID(testApiInstanceID, k) {
						certCount++
					}
				}
			}
			Expect(certCount).To(Equal(1), fmt.Sprintf("expected one cert finding, got %d", certCount))
		})

		It("removes the finding when the condition is resolved", func() {
			expiring := probeResult(testApiInstanceID, true, true, 24*time.Hour, nil)
			Expect(reconcileFinding(pub, threshold, expiring)).To(Succeed())
			Expect(findingsOfKind(m, finding.CertificateExpiringSoon)).To(HaveLen(1))

			healthy := probeResult(testApiInstanceID, true, true, 1000*time.Hour, nil)
			Expect(reconcileFinding(pub, threshold, healthy)).To(Succeed())

			Expect(findingsOfKind(m, finding.CertificateExpiringSoon)).To(BeEmpty())
			Expect(findingsOfKind(m, finding.CertificateExpired)).To(BeEmpty())
			Expect(findingsOfKind(m, finding.CertificateProbeFailed)).To(BeEmpty())
		})

		It("replaces CertificateExpiringSoon with CertificateExpired when remaining drops to ≤ 0", func() {
			expiring := probeResult(testApiInstanceID, true, true, 24*time.Hour, nil)
			Expect(reconcileFinding(pub, threshold, expiring)).To(Succeed())
			Expect(findingsOfKind(m, finding.CertificateExpiringSoon)).To(HaveLen(1))

			expired := probeResult(testApiInstanceID, true, true, -time.Minute, nil)
			Expect(reconcileFinding(pub, threshold, expired)).To(Succeed())

			Expect(findingsOfKind(m, finding.CertificateExpiringSoon)).To(BeEmpty())
			Expect(findingsOfKind(m, finding.CertificateExpired)).To(HaveLen(1))
		})
	})
})
