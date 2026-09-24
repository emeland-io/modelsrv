package filesensor_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/filesensor"
	"go.emeland.io/modelsrv/pkg/ingress"
	"go.emeland.io/modelsrv/pkg/model"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/iam"
)

var _ = Describe("DecodeDocuments", func() {
	It("parses a multi-document YAML stream", func() {
		data := `---
version: emeland.io/v1
kind: Context
spec:
  contextId: "22222222-2222-2222-2222-222222222222"
  displayName: "Production"
---
version: emeland.io/v1alpha1
kind: System
spec:
  systemId: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
  displayName: "Order Service"
`
		docs, err := ingress.DecodeDocuments([]byte(data))
		Expect(err).NotTo(HaveOccurred())
		Expect(docs).To(HaveLen(2))
		Expect(docs[0].Kind.ResourceType()).To(Equal(events.ContextResource))
		Expect(docs[1].Kind.ResourceType()).To(Equal(events.SystemResource))
		Expect(ingress.ValidVersion(docs[0].Version)).To(BeTrue())
		Expect(ingress.ValidVersion(docs[1].Version)).To(BeTrue())
	})

	It("rejects a document that omits kind", func() {
		data := `---
version: emeland.io/v1
spec:
  contextId: "22222222-2222-2222-2222-222222222222"
`
		_, err := ingress.DecodeDocuments([]byte(data))
		Expect(err).To(MatchError(ContainSubstring("missing kind")))
	})
})

var _ = Describe("ProcessFile", func() {
	var issueExampleYAML string

	It("loads test/fixtures/simple_system.yaml (System, Component, API)", func() {
		_, file, _, ok := runtime.Caller(0)
		Expect(ok).To(BeTrue())
		root := filepath.Join(filepath.Dir(file), "..", "..")
		path := filepath.Join(root, "test", "fixtures", "simple_system.yaml")
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		res, err := filesensor.ProcessFile(path, m)
		Expect(err).NotTo(HaveOccurred())
		Expect(res.Applied).To(Equal(3))
		Expect(res.Failed).To(BeEmpty())

		Expect(m.GetSystemById(uuid.MustParse("b4eaa9f0-0242-4a26-9496-fa2b1a3b9330"))).NotTo(BeNil())
		Expect(m.GetComponentById(uuid.MustParse("104e9a87-817d-486a-b834-5a70e8c4f68a"))).NotTo(BeNil())
		Expect(m.GetApiById(uuid.MustParse("c649f2f3-462b-4a6d-8337-0d2e7403c44d"))).NotTo(BeNil())
	})

	BeforeEach(func() {
		issueExampleYAML = `---
version: emeland.io/v1
kind: Context
spec:
  contextId: "22222222-2222-2222-2222-222222222222"
  displayName: "Production"
  parent: null
  type: "11111111-1111-1111-1111-111111111111"
---
version: emeland.io/v1
kind: System
spec:
  systemId: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
  displayName: "Order Service"
  description: "Handles order processing"
  abstract: false
---
version: emeland.io/v1
kind: API
spec:
  apiId: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
  displayName: "Order API"
  description: "REST API for orders"
  type: "OpenAPI"
  system: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
`
	})

	It("loads the issue #24 example into the model", func() {
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "model.yaml")
		Expect(os.WriteFile(path, []byte(issueExampleYAML), 0644)).To(Succeed())

		res, err := filesensor.ProcessFile(path, m)
		Expect(err).NotTo(HaveOccurred())
		Expect(res.Applied).To(Equal(3))
		Expect(res.Failed).To(BeEmpty())

		ctxID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		ctx := m.GetContextById(ctxID)
		Expect(ctx).NotTo(BeNil())
		Expect(ctx.GetDisplayName()).To(Equal("Production"))
		Expect(ctx.GetContextTypeId()).To(Equal(uuid.MustParse("11111111-1111-1111-1111-111111111111")))

		sysID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
		sys := m.GetSystemById(sysID)
		Expect(sys).NotTo(BeNil())
		Expect(sys.GetDisplayName()).To(Equal("Order Service"))

		apiID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
		api := m.GetApiById(apiID)
		Expect(api).NotTo(BeNil())
		Expect(api.GetDisplayName()).To(Equal("Order API"))
		Expect(api.GetType()).To(Equal(mdlapi.OpenAPI))
		Expect(api.GetSystem()).NotTo(BeNil())
		Expect(api.GetSystem().SystemId).To(Equal(sysID))
	})

	It("links a Context to its ContextType by id after ingestion", func() {
		data := `---
version: emeland.io/v1
kind: ContextType
spec:
  contextTypeId: "11111111-1111-1111-1111-111111111111"
  displayName: "Environment"
---
version: emeland.io/v1
kind: Context
spec:
  contextId: "22222222-2222-2222-2222-222222222222"
  displayName: "Production"
  type: "11111111-1111-1111-1111-111111111111"
`
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "context-with-type.yaml")
		Expect(os.WriteFile(path, []byte(data), 0644)).To(Succeed())

		res, err := filesensor.ProcessFile(path, m)
		Expect(err).NotTo(HaveOccurred())
		Expect(res.Applied).To(Equal(2))
		Expect(res.Failed).To(BeEmpty())

		ctxID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		typeID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		ctx := m.GetContextById(ctxID)
		Expect(ctx).NotTo(BeNil())

		// GetContextTypeId returns the stored UUID directly from the TypeRef
		Expect(ctx.GetContextTypeId()).To(Equal(typeID))

		// Resolve to the full ContextType via the model (refs only store ids after file-sensor ingestion)
		resolvedType := m.GetContextTypeById(ctx.GetContextTypeId())
		Expect(resolvedType).NotTo(BeNil())
		Expect(resolvedType.GetContextTypeId()).To(Equal(typeID))
		Expect(resolvedType.GetDisplayName()).To(Equal("Environment"))
	})

	It("applies valid documents and skips invalid ones in the same file", func() {
		data := `---
version: "emeland.io/v1"
kind: Context
spec:
  daf: "22222222-2222-2222-2222-222222222222"
  fa: "Production"
---
version: "emeland.io/v1"
kind: Context
spec:
  contextId: "11111111-2222-2222-2222-222222222222"
  displayName: "Staging"
  parent: null
  type: "11111111-1111-1111-1111-111111111111"
`
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "mixed.yaml")
		Expect(os.WriteFile(path, []byte(data), 0644)).To(Succeed())

		res, err := filesensor.ProcessFile(path, m)
		Expect(err).NotTo(HaveOccurred())
		Expect(res.Applied).To(Equal(1))
		Expect(res.Failed).To(HaveLen(1))
		Expect(res.Failed[0].Index).To(Equal(0))

		ctx := m.GetContextById(uuid.MustParse("11111111-2222-2222-2222-222222222222"))
		Expect(ctx).NotTo(BeNil())
		Expect(ctx.GetDisplayName()).To(Equal("Staging"))
	})
})

var _ = Describe("ApplyDocument", func() {
	var m model.Model

	BeforeEach(func() {
		sink := events.NewListSink()
		var err error
		m, err = model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when the API type is invalid", func() {
		It("returns an error", func() {
			doc := ingress.Document{
				Version: "emeland.io/v1",
				Kind:    ingress.DocumentKind(events.APIResource),
				Spec: map[string]any{
					"apiId":       "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"displayName": "X",
					"system":      "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					"type":        "NotARealType",
				},
			}
			Expect(ingress.ApplyDocument(doc, m)).To(MatchError(ContainSubstring("invalid API type")))
		})
	})

	Context("when the kind is not supported", func() {
		It("returns an error", func() {
			doc := ingress.Document{
				Version: "emeland.io/v1",
				Kind:    ingress.DocumentKind(events.AnnotationsResource),
				Spec: map[string]any{
					"componentId": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"displayName": "X",
				},
			}
			Expect(ingress.ApplyDocument(doc, m)).To(MatchError(ContainSubstring("unsupported kind")))
		})
	})

	Context("when systemId is used instead of system for an API", func() {
		It("accepts the document after the system exists", func() {
			Expect(ingress.ApplyDocument(ingress.Document{
				Version: "emeland.io/v1",
				Kind:    ingress.DocumentKind(events.SystemResource),
				Spec: map[string]any{
					"systemId":    "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					"displayName": "S",
				},
			}, m)).To(Succeed())

			Expect(ingress.ApplyDocument(ingress.Document{
				Version: "emeland.io/v1",
				Kind:    ingress.DocumentKind(events.APIResource),
				Spec: map[string]any{
					"apiId":       "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"displayName": "A",
					"systemId":    "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				},
			}, m)).To(Succeed())

			api := m.GetApiById(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
			Expect(api).NotTo(BeNil())
		})
	})
})

var _ = Describe("CapacityResourceType documents", func() {
	It("applies a valid CapacityResourceType YAML document", func() {
		crtID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		doc := ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.CapacityResourceTypeResource),
			Spec: map[string]any{
				"capacityResourceTypeId": crtID.String(),
				"displayName":            "CPU cores",
				"description":            "Virtual CPU cores",
				"unit":                   "cores",
			},
		}
		Expect(ingress.ApplyDocument(doc, m)).To(Succeed())

		got := m.GetCapacityResourceTypeById(crtID)
		Expect(got).NotTo(BeNil())
		Expect(got.GetDisplayName()).To(Equal("CPU cores"))
		Expect(got.GetDescription()).To(Equal("Virtual CPU cores"))
		Expect(got.GetUnit()).To(Equal("cores"))
	})
})

var _ = Describe("Capacity documents", func() {
	var (
		m     model.Model
		crtID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
		ctxID = uuid.MustParse("33333333-3333-3333-3333-333333333333")
		capID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
		ctID  = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	)

	BeforeEach(func() {
		sink := events.NewListSink()
		var err error
		m, err = model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.CapacityResourceTypeResource),
			Spec: map[string]any{
				"capacityResourceTypeId": crtID.String(),
				"displayName":            "CPU",
				"unit":                   "cores",
			},
		}, m)).To(Succeed())

		ct := mdlctx.NewContextType(ctID)
		ct.SetDisplayName("Environment")
		Expect(m.AddContextType(ct)).To(Succeed())

		ctx := mdlctx.NewContext(ctxID)
		ctx.SetDisplayName("Production")
		ctx.SetContextTypeById(ctID)
		Expect(m.AddContext(ctx)).To(Succeed())
	})

	validCapacityDoc := func() ingress.Document {
		return ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.CapacityResource),
			Spec: map[string]any{
				"capacityId":  capID.String(),
				"displayName": "Production CPU provided",
				"description": "Available CPU in production context",
				"resourceTypeRef": map[string]any{
					"capacityResourceTypeId": crtID.String(),
				},
				"contextRef": map[string]any{
					"contextId": ctxID.String(),
				},
				"category": "provided",
				"amount":   "64",
				"annotations": map[string]any{
					"emeland.io/owner-groups": "platform-team",
				},
			},
		}
	}

	It("applies a valid Capacity YAML document after dependencies exist", func() {
		Expect(ingress.ApplyDocument(validCapacityDoc(), m)).To(Succeed())
		got := m.GetCapacityById(capID)
		Expect(got).NotTo(BeNil())
		Expect(got.GetDisplayName()).To(Equal("Production CPU provided"))
		Expect(string(got.GetCategory())).To(Equal("provided"))
		Expect(string(got.GetAmount())).To(Equal("64"))
		Expect(got.GetCapacityResourceTypeId()).To(Equal(crtID))
		Expect(got.GetContextId()).To(Equal(ctxID))
	})

	It("rejects invalid category", func() {
		doc := validCapacityDoc()
		doc.Spec["category"] = "unknown"
		Expect(ingress.ApplyDocument(doc, m)).To(MatchError(ContainSubstring("invalid capacity category")))
	})

	It("rejects negative amount", func() {
		doc := validCapacityDoc()
		doc.Spec["amount"] = "-1"
		Expect(ingress.ApplyDocument(doc, m)).To(MatchError(ContainSubstring("non-negative")))
	})

	It("rejects tuple conflict when CapacityId differs", func() {
		Expect(ingress.ApplyDocument(validCapacityDoc(), m)).To(Succeed())

		doc := validCapacityDoc()
		doc.Spec["capacityId"] = uuid.New().String()
		Expect(ingress.ApplyDocument(doc, m)).To(MatchError(ContainSubstring("capacity tuple already exists")))
	})
})

var _ = Describe("ValidValue documents", func() {
	var (
		m       model.Model
		paramID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
		vvID    = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	)

	BeforeEach(func() {
		sink := events.NewListSink()
		var err error
		m, err = model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.ParameterResource),
			Spec: map[string]any{
				"parameterId": paramID.String(),
				"displayName": "Region",
			},
		}, m)).To(Succeed())
	})

	validValidValueDoc := func() ingress.Document {
		return ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.ValidValueResource),
			Spec: map[string]any{
				"validValueId": vvID.String(),
				"displayName":  "eu-west-1",
				"parameterRef": map[string]any{
					"parameterId": paramID.String(),
				},
			},
		}
	}

	It("applies a valid ValidValue YAML document after Parameter exists", func() {
		Expect(ingress.ApplyDocument(validValidValueDoc(), m)).To(Succeed())
		got := m.GetValidValueById(vvID)
		Expect(got).NotTo(BeNil())
		Expect(got.GetDisplayName()).To(Equal("eu-west-1"))
		Expect(got.GetParameterId()).To(Equal(paramID))
	})

	It("rejects missing Parameter FK", func() {
		doc := validValidValueDoc()
		doc.Spec["parameterRef"] = map[string]any{
			"parameterId": uuid.New().String(),
		}
		Expect(ingress.ApplyDocument(doc, m)).To(MatchError(ContainSubstring("parameter not found")))
	})

	It("rejects displayName uniqueness conflict", func() {
		Expect(ingress.ApplyDocument(validValidValueDoc(), m)).To(Succeed())
		doc := validValidValueDoc()
		doc.Spec["validValueId"] = uuid.New().String()
		Expect(ingress.ApplyDocument(doc, m)).To(MatchError(ContainSubstring("valid value already exists")))
	})

	It("applies Parameter values as ValidValue UUIDs", func() {
		Expect(ingress.ApplyDocument(validValidValueDoc(), m)).To(Succeed())
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.ParameterResource),
			Spec: map[string]any{
				"parameterId": paramID.String(),
				"displayName": "Region",
				"values":      []any{vvID.String()},
			},
		}, m)).To(Succeed())
		got := m.GetParameterById(paramID)
		Expect(got).NotTo(BeNil())
		Expect(got.GetValues()).To(Equal([]uuid.UUID{vvID}))
	})
})

var _ = Describe("CapabilityVersion documents", func() {
	var (
		m     model.Model
		capID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
		cvID  = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	)

	BeforeEach(func() {
		sink := events.NewListSink()
		var err error
		m, err = model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.CapabilityResource),
			Spec: map[string]any{
				"capabilityId": capID.String(),
				"displayName":  "Compute",
			},
		}, m)).To(Succeed())
	})

	validDoc := func() ingress.Document {
		return ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.CapabilityVersionResource),
			Spec: map[string]any{
				"capabilityVersionId": cvID.String(),
				"displayName":         "Compute 1.0",
				"version": map[string]any{
					"version": "1.0.0",
				},
				"capabilityRef": map[string]any{
					"capabilityId": capID.String(),
				},
			},
		}
	}

	It("applies a valid CapabilityVersion after Capability exists", func() {
		Expect(ingress.ApplyDocument(validDoc(), m)).To(Succeed())
		got := m.GetCapabilityVersionById(cvID)
		Expect(got).NotTo(BeNil())
		Expect(got.GetDisplayName()).To(Equal("Compute 1.0"))
		Expect(got.GetCapabilityId()).To(Equal(capID))
	})

	It("rejects missing Capability FK", func() {
		doc := validDoc()
		doc.Spec["capabilityRef"] = map[string]any{
			"capabilityId": uuid.New().String(),
		}
		Expect(ingress.ApplyDocument(doc, m)).To(MatchError(ContainSubstring("capability not found")))
	})
})

var _ = Describe("Order documents", func() {
	var (
		m     model.Model
		ouID  = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
		ordID = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	)

	BeforeEach(func() {
		sink := events.NewListSink()
		var err error
		m, err = model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		ou := iam.NewOrgUnit(ouID)
		ou.SetDisplayName("Platform")
		Expect(m.AddOrgUnit(ou)).To(Succeed())
	})

	It("applies a valid Order after OrgUnit exists", func() {
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.OrderResource),
			Spec: map[string]any{
				"orderId":     ordID.String(),
				"displayName": "Q1 order",
				"orgUnitRef": map[string]any{
					"orgUnitId": ouID.String(),
				},
			},
		}, m)).To(Succeed())
		got := m.GetOrderById(ordID)
		Expect(got).NotTo(BeNil())
		Expect(got.GetOrgUnitId()).To(Equal(ouID))
	})

	It("rejects missing OrgUnit FK", func() {
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.OrderResource),
			Spec: map[string]any{
				"orderId":     ordID.String(),
				"displayName": "Q1 order",
				"orgUnitRef": map[string]any{
					"orgUnitId": uuid.New().String(),
				},
			},
		}, m)).To(MatchError(ContainSubstring("organizational unit not found")))
	})
})

var _ = Describe("Variant documents", func() {
	var (
		m       model.Model
		capID   = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
		cvID    = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
		paramID = uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
		vvID    = uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
		varID   = uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	)

	BeforeEach(func() {
		sink := events.NewListSink()
		var err error
		m, err = model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.ParameterResource),
			Spec: map[string]any{
				"parameterId": paramID.String(),
				"displayName": "region",
			},
		}, m)).To(Succeed())
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.ValidValueResource),
			Spec: map[string]any{
				"validValueId": vvID.String(),
				"displayName":  "eu",
				"parameterRef": map[string]any{"parameterId": paramID.String()},
			},
		}, m)).To(Succeed())
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.CapabilityResource),
			Spec: map[string]any{
				"capabilityId": capID.String(),
				"displayName":  "Compute",
			},
		}, m)).To(Succeed())
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.CapabilityVersionResource),
			Spec: map[string]any{
				"capabilityVersionId": cvID.String(),
				"displayName":         "Compute 1.0",
				"version":             map[string]any{"version": "1.0.0"},
				"capabilityRef":       map[string]any{"capabilityId": capID.String()},
			},
		}, m)).To(Succeed())
	})

	It("applies a Variant after its CapabilityVersion exists", func() {
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.VariantResource),
			Spec: map[string]any{
				"variantId":   varID.String(),
				"displayName": "eu",
				"capabilityVersionRef": map[string]any{
					"capabilityVersionId": cvID.String(),
				},
				"provides": []any{
					map[string]any{
						"parameterId":   paramID.String(),
						"validValueIds": []any{vvID.String()},
					},
				},
			},
		}, m)).To(Succeed())
		got := m.GetVariantById(varID)
		Expect(got).NotTo(BeNil())
		Expect(got.GetCapabilityVersionId()).To(Equal(cvID))
		Expect(got.GetProvides()).To(HaveLen(1))
	})

	It("rejects a Variant with an unknown CapabilityVersion", func() {
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.VariantResource),
			Spec: map[string]any{
				"variantId":   varID.String(),
				"displayName": "eu",
				"capabilityVersionRef": map[string]any{
					"capabilityVersionId": uuid.New().String(),
				},
			},
		}, m)).To(MatchError(ContainSubstring("capability version not found")))
	})
})

var _ = Describe("OrderItem documents", func() {
	var (
		m     model.Model
		ouID  = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
		ordID = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
		capID = uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
		oiID  = uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	)

	BeforeEach(func() {
		sink := events.NewListSink()
		var err error
		m, err = model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		ou := iam.NewOrgUnit(ouID)
		ou.SetDisplayName("Platform")
		Expect(m.AddOrgUnit(ou)).To(Succeed())
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.OrderResource),
			Spec: map[string]any{
				"orderId":     ordID.String(),
				"displayName": "Q1 order",
				"orgUnitRef":  map[string]any{"orgUnitId": ouID.String()},
			},
		}, m)).To(Succeed())
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.CapabilityResource),
			Spec: map[string]any{
				"capabilityId": capID.String(),
				"displayName":  "Compute",
			},
		}, m)).To(Succeed())
	})

	validItem := func() ingress.Document {
		return ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.OrderItemResource),
			Spec: map[string]any{
				"orderItemId":   oiID.String(),
				"displayName":   "Compute line",
				"orderRef":      map[string]any{"orderId": ordID.String()},
				"capabilityRef": map[string]any{"capabilityId": capID.String()},
			},
		}
	}

	It("applies an OrderItem after Order and Capability exist", func() {
		Expect(ingress.ApplyDocument(validItem(), m)).To(Succeed())
		got := m.GetOrderItemById(oiID)
		Expect(got).NotTo(BeNil())
		Expect(got.GetOrderId()).To(Equal(ordID))
		Expect(got.GetCapabilityRef().EffectiveCapabilityID()).To(Equal(capID))
	})

	It("rejects an OrderItem with an unknown Order", func() {
		doc := validItem()
		doc.Spec["orderRef"] = map[string]any{"orderId": uuid.New().String()}
		Expect(ingress.ApplyDocument(doc, m)).To(MatchError(ContainSubstring("order not found")))
	})
})

var _ = Describe("BoundValue documents", func() {
	var (
		m       model.Model
		ouID    = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
		ordID   = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
		capID   = uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
		oiID    = uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
		paramID = uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
		vvID    = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
		bvID    = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	)

	BeforeEach(func() {
		sink := events.NewListSink()
		var err error
		m, err = model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		ou := iam.NewOrgUnit(ouID)
		ou.SetDisplayName("Platform")
		Expect(m.AddOrgUnit(ou)).To(Succeed())
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.OrderResource),
			Spec: map[string]any{
				"orderId":     ordID.String(),
				"displayName": "Q1 order",
				"orgUnitRef":  map[string]any{"orgUnitId": ouID.String()},
			},
		}, m)).To(Succeed())
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.CapabilityResource),
			Spec: map[string]any{
				"capabilityId": capID.String(),
				"displayName":  "Compute",
			},
		}, m)).To(Succeed())
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.OrderItemResource),
			Spec: map[string]any{
				"orderItemId":   oiID.String(),
				"displayName":   "Compute line",
				"orderRef":      map[string]any{"orderId": ordID.String()},
				"capabilityRef": map[string]any{"capabilityId": capID.String()},
			},
		}, m)).To(Succeed())
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.ParameterResource),
			Spec: map[string]any{
				"parameterId": paramID.String(),
				"displayName": "region",
			},
		}, m)).To(Succeed())
		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.ValidValueResource),
			Spec: map[string]any{
				"validValueId": vvID.String(),
				"displayName":  "eu",
				"parameterRef": map[string]any{"parameterId": paramID.String()},
			},
		}, m)).To(Succeed())
	})

	validBound := func() ingress.Document {
		return ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.BoundValueResource),
			Spec: map[string]any{
				"boundValueId": bvID.String(),
				"displayName":  "eu",
				"orderItemRef": map[string]any{"orderItemId": oiID.String()},
				"parameterRef": map[string]any{"parameterId": paramID.String()},
				"validValueRef": map[string]any{
					"validValueId": vvID.String(),
				},
			},
		}
	}

	It("applies a BoundValue after its OrderItem and ValidValue exist", func() {
		Expect(ingress.ApplyDocument(validBound(), m)).To(Succeed())
		got := m.GetBoundValueById(bvID)
		Expect(got).NotTo(BeNil())
		Expect(got.GetOrderItemId()).To(Equal(oiID))
		Expect(got.GetValidValueRef().EffectiveValidValueID()).To(Equal(vvID))
	})

	It("rejects a BoundValue with an unknown OrderItem", func() {
		doc := validBound()
		doc.Spec["orderItemRef"] = map[string]any{"orderItemId": uuid.New().String()}
		Expect(ingress.ApplyDocument(doc, m)).To(MatchError(ContainSubstring("order item not found")))
	})
})

var _ = Describe("Metric documents", func() {
	It("applies a valid Metric YAML document", func() {
		metricID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		doc := ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.MetricResource),
			Spec: map[string]any{
				"metricId":    metricID.String(),
				"displayName": "p99 API latency",
				"description": "End-to-end latency",
				"annotations": map[string]any{
					"emeland.io/unit": "ms",
				},
			},
		}
		Expect(ingress.ApplyDocument(doc, m)).To(Succeed())

		got := m.GetMetricById(metricID)
		Expect(got).NotTo(BeNil())
		Expect(got.GetDisplayName()).To(Equal("p99 API latency"))
		Expect(got.GetDescription()).To(Equal("End-to-end latency"))
		Expect(got.GetAnnotations().GetValue("emeland.io/unit")).To(Equal("ms"))
	})
})

var _ = Describe("Threshold documents", func() {
	var (
		m        model.Model
		metricID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
		miID     = uuid.MustParse("44444444-4444-4444-4444-444444444444")
		thID     = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	)

	BeforeEach(func() {
		sink := events.NewListSink()
		var err error
		m, err = model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.MetricResource),
			Spec: map[string]any{
				"metricId":    metricID.String(),
				"displayName": "p99 API latency",
			},
		}, m)).To(Succeed())

		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.MetricInstanceResource),
			Spec: map[string]any{
				"metricInstanceId": miID.String(),
				"displayName":      "p99 API latency for orders-api",
				"metricRef": map[string]any{
					"metricId": metricID.String(),
				},
			},
		}, m)).To(Succeed())
	})

	validThresholdDoc := func() ingress.Document {
		return ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.ThresholdResource),
			Spec: map[string]any{
				"thresholdId": thID.String(),
				"displayName": "Latency SLO breach",
				"description": "p99 must stay under 500ms",
				"metricInstanceRef": map[string]any{
					"metricInstanceId": miID.String(),
				},
				"annotations": map[string]any{
					"emeland.io/threshold.expression": "histogram_quantile(0.99, ...) > 0.5",
					"emeland.io/threshold.language":   "promql",
				},
			},
		}
	}

	It("applies a valid Threshold YAML document after MetricInstance exists", func() {
		Expect(ingress.ApplyDocument(validThresholdDoc(), m)).To(Succeed())
		got := m.GetThresholdById(thID)
		Expect(got).NotTo(BeNil())
		Expect(got.GetDisplayName()).To(Equal("Latency SLO breach"))
		Expect(got.GetMetricInstanceId()).To(Equal(miID))
		Expect(got.GetAnnotations().GetValue("emeland.io/threshold.language")).To(Equal("promql"))
	})

	It("accepts Threshold with an unresolved MetricInstance reference", func() {
		doc := validThresholdDoc()
		doc.Spec["metricInstanceRef"] = map[string]any{
			"metricInstanceId": uuid.New().String(),
		}
		Expect(ingress.ApplyDocument(doc, m)).To(Succeed())
	})
})

var _ = Describe("MetricInstance documents", func() {
	var (
		m        model.Model
		metricID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
		miID     = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	)

	BeforeEach(func() {
		sink := events.NewListSink()
		var err error
		m, err = model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.MetricResource),
			Spec: map[string]any{
				"metricId":    metricID.String(),
				"displayName": "p99 API latency",
			},
		}, m)).To(Succeed())
	})

	validMetricInstanceDoc := func() ingress.Document {
		return ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.MetricInstanceResource),
			Spec: map[string]any{
				"metricInstanceId": miID.String(),
				"displayName":      "p99 API latency for orders-api",
				"metricRef": map[string]any{
					"metricId": metricID.String(),
				},
			},
		}
	}

	It("applies a valid MetricInstance YAML document after Metric exists", func() {
		Expect(ingress.ApplyDocument(validMetricInstanceDoc(), m)).To(Succeed())
		got := m.GetMetricInstanceById(miID)
		Expect(got).NotTo(BeNil())
		Expect(got.GetDisplayName()).To(Equal("p99 API latency for orders-api"))
		Expect(got.GetMetricId()).To(Equal(metricID))
	})

	It("applies a MetricInstance with an optional subject", func() {
		doc := validMetricInstanceDoc()
		subjID := uuid.New()
		doc.Spec["subject"] = map[string]any{
			"resourceId":   subjID.String(),
			"resourceType": "Node",
		}
		Expect(ingress.ApplyDocument(doc, m)).To(Succeed())
		got := m.GetMetricInstanceById(miID)
		Expect(got).NotTo(BeNil())
		Expect(got.GetSubject()).NotTo(BeNil())
		Expect(got.GetSubject().ResourceId).To(Equal(subjID))
		Expect(got.GetSubject().ResourceType).To(Equal(events.NodeResource))
	})

	It("accepts MetricInstance with an unresolved Metric reference", func() {
		// The Metric reference is optional and not existence-checked.
		doc := validMetricInstanceDoc()
		doc.Spec["metricRef"] = map[string]any{
			"metricId": uuid.New().String(),
		}
		Expect(ingress.ApplyDocument(doc, m)).To(Succeed())
	})
})

var _ = Describe("MetricValue documents", func() {
	var (
		m        model.Model
		metricID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
		miID     = uuid.MustParse("44444444-4444-4444-4444-444444444444")
		mvID     = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	)

	BeforeEach(func() {
		sink := events.NewListSink()
		var err error
		m, err = model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.MetricResource),
			Spec: map[string]any{
				"metricId":    metricID.String(),
				"displayName": "p99 API latency",
			},
		}, m)).To(Succeed())

		Expect(ingress.ApplyDocument(ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.MetricInstanceResource),
			Spec: map[string]any{
				"metricInstanceId": miID.String(),
				"displayName":      "p99 API latency for orders-api",
				"metricRef": map[string]any{
					"metricId": metricID.String(),
				},
			},
		}, m)).To(Succeed())
	})

	validMetricValueDoc := func() ingress.Document {
		return ingress.Document{
			Version: "emeland.io/v1",
			Kind:    ingress.DocumentKind(events.MetricValueResource),
			Spec: map[string]any{
				"metricValueId": mvID.String(),
				"displayName":   "Current p99 latency",
				"metricInstanceRef": map[string]any{
					"metricInstanceId": miID.String(),
				},
				"value": "412",
			},
		}
	}

	It("applies a valid MetricValue YAML document after MetricInstance exists", func() {
		Expect(ingress.ApplyDocument(validMetricValueDoc(), m)).To(Succeed())
		got := m.GetMetricValueById(mvID)
		Expect(got).NotTo(BeNil())
		Expect(got.GetDisplayName()).To(Equal("Current p99 latency"))
		Expect(got.GetMetricInstanceId()).To(Equal(miID))
		Expect(got.GetValue()).To(Equal("412"))
	})

	It("rejects MetricValue when value is missing", func() {
		doc := validMetricValueDoc()
		delete(doc.Spec, "value")
		Expect(ingress.ApplyDocument(doc, m)).To(MatchError(ContainSubstring("value is required")))
	})

	It("accepts MetricValue with an unresolved MetricInstance reference", func() {
		doc := validMetricValueDoc()
		doc.Spec["metricInstanceRef"] = map[string]any{
			"metricInstanceId": uuid.New().String(),
		}
		Expect(ingress.ApplyDocument(doc, m)).To(Succeed())
	})
})

var _ = Describe("ApplyExisting", func() {
	It("applies YAML in dir synchronously before returning", func() {
		_, file, _, ok := runtime.Caller(0)
		Expect(ok).To(BeTrue())
		root := filepath.Join(filepath.Dir(file), "..", "..")
		src := filepath.Join(root, "test", "fixtures", "simple_system.yaml")

		dir := GinkgoT().TempDir()
		data, err := os.ReadFile(src)
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(filepath.Join(dir, "simple_system.yaml"), data, 0644)).To(Succeed())

		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		filesensor.ApplyExisting(dir, m, nil)

		Expect(m.GetSystemById(uuid.MustParse("b4eaa9f0-0242-4a26-9496-fa2b1a3b9330"))).NotTo(BeNil())
		Expect(m.GetComponentById(uuid.MustParse("104e9a87-817d-486a-b834-5a70e8c4f68a"))).NotTo(BeNil())
		Expect(m.GetApiById(uuid.MustParse("c649f2f3-462b-4a6d-8337-0d2e7403c44d"))).NotTo(BeNil())
	})
})

var _ = Describe("StartWatch", func() {
	It("does not scan existing files on start", func() {
		_, file, _, ok := runtime.Caller(0)
		Expect(ok).To(BeTrue())
		root := filepath.Join(filepath.Dir(file), "..", "..")
		src := filepath.Join(root, "test", "fixtures", "simple_system.yaml")

		dir := GinkgoT().TempDir()
		data, err := os.ReadFile(src)
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(filepath.Join(dir, "simple_system.yaml"), data, 0644)).To(Succeed())

		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		filesensor.StartWatch(ctx, dir, m, nil)

		Consistently(func() bool {
			return m.GetSystemById(uuid.MustParse("b4eaa9f0-0242-4a26-9496-fa2b1a3b9330")) == nil
		}, "300ms", "50ms").Should(BeTrue())
	})
})
