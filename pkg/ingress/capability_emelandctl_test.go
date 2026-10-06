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

// Mirrors the split documents that `emelandctl create capability --version …`
// emits: a Capability plus one CapabilityVersion per --version.
var _ = Describe("Apply capability from emelandctl", func() {
	It("applies a capability and multi-version CapabilityVersion documents", func() {
		data := []byte(`version: emeland.io/v1
kind: Capability
spec:
  capabilityId: 11111111-1111-1111-1111-111111111111
  displayName: Mail Service
---
version: emeland.io/v1
kind: CapabilityVersion
spec:
  capabilityVersionId: 22222222-2222-2222-2222-222222222222
  displayName: "1.0.0"
  capability: 11111111-1111-1111-1111-111111111111
  version:
    version: "1.0.0"
    availableFrom: "2026-01-01T00:00:00Z"
    deprecatedFrom: "2027-01-01T00:00:00Z"
    terminatedFrom: "2028-01-01T00:00:00Z"
---
version: emeland.io/v1
kind: CapabilityVersion
spec:
  capabilityVersionId: 44444444-4444-4444-4444-444444444444
  displayName: "2.0.0"
  capability: 11111111-1111-1111-1111-111111111111
  version:
    version: "2.0.0"
    availableFrom: "2027-06-01T00:00:00Z"
`)
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		docs, err := ingress.Parse("capability.yaml", data, ingress.ParseOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(docs).To(HaveLen(3))

		res := ingress.ApplyAll(docs, m)
		Expect(res.Applied).To(Equal(3))

		cap := m.GetCapabilityById(uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		Expect(cap).NotTo(BeNil())
		Expect(cap.GetDisplayName()).To(Equal("Mail Service"))

		v1 := m.GetCapabilityVersionById(uuid.MustParse("22222222-2222-2222-2222-222222222222"))
		Expect(v1).NotTo(BeNil())
		Expect(v1.GetVersion().Version).To(Equal("1.0.0"))
		Expect(v1.GetVersion().AvailableFrom).NotTo(BeNil())
		Expect(v1.GetVersion().DeprecatedFrom).NotTo(BeNil())
		Expect(v1.GetVersion().TerminatedFrom).NotTo(BeNil())
		Expect(v1.GetCapabilityId()).To(Equal(uuid.MustParse("11111111-1111-1111-1111-111111111111")))

		v2 := m.GetCapabilityVersionById(uuid.MustParse("44444444-4444-4444-4444-444444444444"))
		Expect(v2).NotTo(BeNil())
		Expect(v2.GetVersion().Version).To(Equal("2.0.0"))
		Expect(v2.GetVersion().AvailableFrom).NotTo(BeNil())
		Expect(v2.GetVersion().DeprecatedFrom).To(BeNil())
		Expect(v2.GetVersion().TerminatedFrom).To(BeNil())
	})
})
