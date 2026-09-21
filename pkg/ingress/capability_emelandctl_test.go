/*
Copyright © 2025 Lutz Behnke

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package ingress_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/ingress"
	"go.emeland.io/modelsrv/pkg/model"
)

// This mirrors the YAML that `emelandctl create capability` emits: versions with
// RFC3339 lifecycle dates, plus the forward-looking dependencies/variants
// scaffold (emeland-io/modelsrv#177) which the ingress currently ignores.
var _ = Describe("Apply capability from emelandctl", func() {
	It("applies a multi-version capability with lifecycle dates", func() {
		data := []byte(`version: emeland.io/v1
kind: Capability
spec:
  capabilityId: 11111111-1111-1111-1111-111111111111
  displayName: Mail Service
  versions:
    - capabilityVersionId: 22222222-2222-2222-2222-222222222222
      dependencies: []
      variants:
        - variantId: 33333333-3333-3333-3333-333333333333
          displayName: default
          dependencies: []
      version:
        version: "1.0.0"
        availableFrom: "2026-01-01T00:00:00Z"
        deprecatedFrom: "2027-01-01T00:00:00Z"
        terminatedFrom: "2028-01-01T00:00:00Z"
    - capabilityVersionId: 44444444-4444-4444-4444-444444444444
      dependencies: []
      variants:
        - variantId: 55555555-5555-5555-5555-555555555555
          displayName: default
          dependencies: []
      version:
        version: "2.0.0"
        availableFrom: "2027-06-01T00:00:00Z"
`)
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		docs, err := ingress.Parse("capability.yaml", data, ingress.ParseOptions{})
		Expect(err).NotTo(HaveOccurred())

		res := ingress.ApplyAll(docs, m)
		Expect(res.Applied).To(Equal(1))

		cap := m.GetCapabilityById(uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		Expect(cap).NotTo(BeNil())
		Expect(cap.GetDisplayName()).To(Equal("Mail Service"))

		versions := cap.GetVersions()
		Expect(versions).To(HaveLen(2))

		Expect(versions[0].Version.Version).To(Equal("1.0.0"))
		Expect(versions[0].Version.AvailableFrom).NotTo(BeNil())
		Expect(versions[0].Version.DeprecatedFrom).NotTo(BeNil())
		Expect(versions[0].Version.TerminatedFrom).NotTo(BeNil())

		Expect(versions[1].Version.Version).To(Equal("2.0.0"))
		Expect(versions[1].Version.AvailableFrom).NotTo(BeNil())
		// v2 only carried availableFrom; the scoping must not leak v1 dates.
		Expect(versions[1].Version.DeprecatedFrom).To(BeNil())
		Expect(versions[1].Version.TerminatedFrom).To(BeNil())
	})
})
