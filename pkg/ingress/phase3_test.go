package ingress_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/ingress"
	"go.emeland.io/modelsrv/pkg/model"
)

var _ = Describe("Apply Phase 3 ordering resources", func() {
	It("applies Parameter, ValidValue, Variant, Dependency, Order, OrderItem, and BoundValue", func() {
		data := []byte(`version: emeland.io/v1
kind: Parameter
spec:
  parameterId: aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa
  displayName: Concurrent Users
---
version: emeland.io/v1
kind: ValidValue
spec:
  validValueId: bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb
  displayName: "10000-users"
  parameter: aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa
---
version: emeland.io/v1
kind: Capability
spec:
  capabilityId: cccccccc-cccc-cccc-cccc-cccccccccccc
  displayName: Mail Service
  offers:
    - bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb
---
version: emeland.io/v1
kind: CapabilityVersion
spec:
  capabilityVersionId: dddddddd-dddd-dddd-dddd-dddddddddddd
  displayName: "1.0.0"
  capability: cccccccc-cccc-cccc-cccc-cccccccccccc
  version:
    version: "1.0.0"
---
version: emeland.io/v1
kind: Variant
spec:
  variantId: eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee
  displayName: default
  capabilityVersion: dddddddd-dddd-dddd-dddd-dddddddddddd
  requires:
    - bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb
---
version: emeland.io/v1
kind: Dependency
spec:
  dependencyId: ffffffff-ffff-ffff-ffff-ffffffffffff
  displayName: needs-storage
  variant: eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee
  capability: cccccccc-cccc-cccc-cccc-cccccccccccc
  mappings:
    - fromValidValueId: bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb
      toValidValueId: bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb
---
version: emeland.io/v1
kind: Order
spec:
  orderId: 22222222-2222-2222-2222-222222222222
  displayName: Mail for Platform
  orgUnit: 11111111-1111-1111-1111-111111111111
---
version: emeland.io/v1
kind: OrderItem
spec:
  orderItemId: 33333333-3333-3333-3333-333333333333
  displayName: Mail Service item
  order: 22222222-2222-2222-2222-222222222222
  capability: cccccccc-cccc-cccc-cccc-cccccccccccc
  capabilityVersion: dddddddd-dddd-dddd-dddd-dddddddddddd
  variant: eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee
---
version: emeland.io/v1
kind: BoundValue
spec:
  boundValueId: 44444444-4444-4444-4444-444444444444
  displayName: users=10000
  orderItem: 33333333-3333-3333-3333-333333333333
  parameter: aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa
  validValue: bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb
`)
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		docs, err := ingress.Parse("phase3.yaml", data, ingress.ParseOptions{})
		Expect(err).NotTo(HaveOccurred())

		res := ingress.ApplyAll(docs, m)
		Expect(res.Failed).To(BeEmpty())
		Expect(res.Applied).To(Equal(9))

		Expect(m.GetParameterById(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))).NotTo(BeNil())
		Expect(m.GetValidValueById(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")).GetDisplayName()).To(Equal("10000-users"))
		Expect(m.GetCapabilityById(uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")).GetOffers()).To(HaveLen(1))
		Expect(m.GetVariantById(uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")).GetRequires()).To(HaveLen(1))
		Expect(m.GetDependencyById(uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")).GetMappings()).To(HaveLen(1))
		Expect(m.GetOrderById(uuid.MustParse("22222222-2222-2222-2222-222222222222")).GetOrgUnitId()).To(Equal(uuid.MustParse("11111111-1111-1111-1111-111111111111")))
		Expect(m.GetOrderItemById(uuid.MustParse("33333333-3333-3333-3333-333333333333"))).NotTo(BeNil())
		Expect(m.GetBoundValueById(uuid.MustParse("44444444-4444-4444-4444-444444444444")).GetOrderItemId()).To(Equal(uuid.MustParse("33333333-3333-3333-3333-333333333333")))
	})
})
