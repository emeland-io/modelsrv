package filesensor_test

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/filesensor"
	"go.emeland.io/modelsrv/pkg/model"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
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
		docs, err := filesensor.DecodeDocuments([]byte(data))
		Expect(err).NotTo(HaveOccurred())
		Expect(docs).To(HaveLen(2))
		Expect(docs[0].Kind.ResourceType()).To(Equal(events.ContextResource))
		Expect(docs[1].Kind.ResourceType()).To(Equal(events.SystemResource))
		Expect(filesensor.ValidVersion(docs[0].Version)).To(BeTrue())
		Expect(filesensor.ValidVersion(docs[1].Version)).To(BeTrue())
	})

	It("rejects a document that omits kind", func() {
		data := `---
version: emeland.io/v1
spec:
  contextId: "22222222-2222-2222-2222-222222222222"
`
		_, err := filesensor.DecodeDocuments([]byte(data))
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
		Expect(ctx.GetAnnotations().GetValue(filesensor.AnnotationContextTypeID)).To(Equal("11111111-1111-1111-1111-111111111111"))

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
			doc := filesensor.Document{
				Version: "emeland.io/v1",
				Kind:    filesensor.DocumentKind(events.APIResource),
				Spec: map[string]any{
					"apiId":       "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"displayName": "X",
					"system":      "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					"type":        "NotARealType",
				},
			}
			Expect(filesensor.ApplyDocument(doc, m)).To(MatchError(ContainSubstring("invalid API type")))
		})
	})

	Context("when the kind is not supported", func() {
		It("returns an error", func() {
			doc := filesensor.Document{
				Version: "emeland.io/v1",
				Kind:    filesensor.DocumentKind(events.AnnotationsResource),
				Spec: map[string]any{
					"componentId": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"displayName": "X",
				},
			}
			Expect(filesensor.ApplyDocument(doc, m)).To(MatchError(ContainSubstring("unsupported kind")))
		})
	})

	Context("when systemId is used instead of system for an API", func() {
		It("accepts the document after the system exists", func() {
			Expect(filesensor.ApplyDocument(filesensor.Document{
				Version: "emeland.io/v1",
				Kind:    filesensor.DocumentKind(events.SystemResource),
				Spec: map[string]any{
					"systemId":    "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					"displayName": "S",
				},
			}, m)).To(Succeed())

			Expect(filesensor.ApplyDocument(filesensor.Document{
				Version: "emeland.io/v1",
				Kind:    filesensor.DocumentKind(events.APIResource),
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
