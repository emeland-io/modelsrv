package ingress_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/ingress"
	"go.emeland.io/modelsrv/pkg/model"
)

var _ = Describe("Parse YAML", func() {
	It("decodes a multi-document YAML stream via Parse", func() {
		data := []byte(`---
version: emeland.io/v1
kind: Context
spec:
  contextId: "22222222-2222-2222-2222-222222222222"
  displayName: "Production"
`)
		docs, err := ingress.Parse("ctx.yaml", data, ingress.ParseOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(docs).To(HaveLen(1))
		Expect(docs[0].Kind.ResourceType()).To(Equal(events.ContextResource))
	})
})

var _ = Describe("Parse JSON", func() {
	It("decodes a single JSON object", func() {
		data := []byte(`{
  "version": "emeland.io/v1",
  "kind": "Context",
  "spec": {
    "contextId": "22222222-2222-2222-2222-222222222222",
    "displayName": "Production"
  }
}`)
		docs, err := ingress.DecodeJSONDocuments(data)
		Expect(err).NotTo(HaveOccurred())
		Expect(docs).To(HaveLen(1))
		Expect(docs[0].Kind.ResourceType()).To(Equal(events.ContextResource))
	})

	It("decodes a JSON array", func() {
		data := []byte(`[
  {"version":"emeland.io/v1","kind":"Context","spec":{"contextId":"22222222-2222-2222-2222-222222222222","displayName":"A"}},
  {"version":"emeland.io/v1","kind":"Context","spec":{"contextId":"33333333-3333-3333-3333-333333333333","displayName":"B"}}
]`)
		docs, err := ingress.Parse("contexts.json", data, ingress.ParseOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(docs).To(HaveLen(2))
	})

	It("decodes JSONL", func() {
		data := []byte(`{"version":"emeland.io/v1","kind":"Context","spec":{"contextId":"22222222-2222-2222-2222-222222222222","displayName":"A"}}
{"version":"emeland.io/v1","kind":"Context","spec":{"contextId":"33333333-3333-3333-3333-333333333333","displayName":"B"}}
`)
		docs, err := ingress.DecodeJSONDocuments(data)
		Expect(err).NotTo(HaveOccurred())
		Expect(docs).To(HaveLen(2))
	})

	It("applies JSON into the model", func() {
		data := []byte(`{
  "version": "emeland.io/v1",
  "kind": "Context",
  "spec": {
    "contextId": "22222222-2222-2222-2222-222222222222",
    "displayName": "Production"
  }
}`)
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())
		docs, err := ingress.Parse("c.json", data, ingress.ParseOptions{})
		Expect(err).NotTo(HaveOccurred())
		res := ingress.ApplyAll(docs, m)
		Expect(res.Applied).To(Equal(1))
		Expect(m.GetContextById(uuid.MustParse("22222222-2222-2222-2222-222222222222"))).NotTo(BeNil())
	})
})

var _ = Describe("Parse CSV", func() {
	It("rejects CSV without Kind and without a kind column", func() {
		_, err := ingress.DecodeCSVDocuments([]byte("a,b\n1,2\n"), ingress.ParseOptions{
			Columns: map[string]string{"a": "displayName", "b": "description"},
		})
		Expect(err).To(MatchError(ContainSubstring("Kind")))
	})

	It("detects kind per row from resourcetype and maps uuid to the primary id field", func() {
		data := []byte("resourcetype,uuid,displayname,description\n" +
			"Context,22222222-2222-2222-2222-222222222222,Production,prod env\n" +
			"System,aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa,Orders,\n")
		docs, err := ingress.Parse("landscape.csv", data, ingress.ParseOptions{
			Format: ingress.FormatCSV,
			Columns: map[string]string{
				"resourcetype": "kind",
				"uuid":         "id",
				"displayname":  "displayName",
				"description":  "description",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(docs).To(HaveLen(2))
		Expect(docs[0].Version).To(Equal(ingress.DefaultCSVVersion))
		Expect(docs[0].Kind.ResourceType()).To(Equal(events.ContextResource))
		Expect(docs[0].Spec["contextId"]).To(Equal("22222222-2222-2222-2222-222222222222"))
		Expect(docs[0].Spec["displayName"]).To(Equal("Production"))
		Expect(docs[0].Spec["description"]).To(Equal("prod env"))
		Expect(docs[1].Kind.ResourceType()).To(Equal(events.SystemResource))
		Expect(docs[1].Spec["systemId"]).To(Equal("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
		Expect(docs[1].Spec["displayName"]).To(Equal("Orders"))
	})

	It("uses DefaultCSVColumns when Columns is empty", func() {
		data := []byte("resourcetype,uuid,displayname,description,annotations\n" +
			`Context,22222222-2222-2222-2222-222222222222,Production,,"{""owner"":""ops""}"` + "\n")
		docs, err := ingress.DecodeCSVDocuments(data, ingress.ParseOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(docs).To(HaveLen(1))
		Expect(docs[0].Spec["contextId"]).To(Equal("22222222-2222-2222-2222-222222222222"))
		ann := docs[0].Spec["annotations"].(map[string]any)
		Expect(ann["owner"]).To(Equal("ops"))
	})

	It("applies mixed CSV rows into the model", func() {
		data := []byte("resourcetype,uuid,displayname,description\n" +
			"Context,22222222-2222-2222-2222-222222222222,Production,\n")
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())
		docs, err := ingress.Parse("c.csv", data, ingress.ParseOptions{Format: ingress.FormatCSV})
		Expect(err).NotTo(HaveOccurred())
		res := ingress.ApplyAll(docs, m)
		Expect(res.Applied).To(Equal(1))
		Expect(m.GetContextById(uuid.MustParse("22222222-2222-2222-2222-222222222222"))).NotTo(BeNil())
	})
})

var _ = Describe("DetectFormat", func() {
	It("detects extensions", func() {
		Expect(ingress.DetectFormat("a.yaml")).To(Equal(ingress.FormatYAML))
		Expect(ingress.DetectFormat("a.yml")).To(Equal(ingress.FormatYAML))
		Expect(ingress.DetectFormat("a.json")).To(Equal(ingress.FormatJSON))
		Expect(ingress.DetectFormat("a.jsonl")).To(Equal(ingress.FormatJSON))
		Expect(ingress.DetectFormat("a.csv")).To(Equal(ingress.FormatCSV))
		Expect(ingress.DetectFormat("a.txt")).To(Equal(ingress.Format("")))
	})
})

var _ = Describe("FormatFromContentType", func() {
	It("maps MIME types and ignores parameters", func() {
		Expect(ingress.FormatFromContentType("application/json")).To(Equal(ingress.FormatJSON))
		Expect(ingress.FormatFromContentType("application/json; charset=utf-8")).To(Equal(ingress.FormatJSON))
		Expect(ingress.FormatFromContentType("application/x-ndjson")).To(Equal(ingress.FormatJSON))
		Expect(ingress.FormatFromContentType("text/yaml")).To(Equal(ingress.FormatYAML))
		Expect(ingress.FormatFromContentType("application/x-yaml")).To(Equal(ingress.FormatYAML))
		Expect(ingress.FormatFromContentType("text/csv")).To(Equal(ingress.FormatCSV))
		Expect(ingress.FormatFromContentType("text/html")).To(Equal(ingress.Format("")))
		Expect(ingress.FormatFromContentType("")).To(Equal(ingress.Format("")))
	})
})

var _ = Describe("ResolveFormat", func() {
	It("prefers the explicit format, then the extension, then the content type", func() {
		f, err := ingress.ResolveFormat("a.csv", ingress.ParseOptions{Format: ingress.FormatYAML, ContentType: "application/json"})
		Expect(err).NotTo(HaveOccurred())
		Expect(f).To(Equal(ingress.FormatYAML))

		f, err = ingress.ResolveFormat("a.csv", ingress.ParseOptions{ContentType: "application/json"})
		Expect(err).NotTo(HaveOccurred())
		Expect(f).To(Equal(ingress.FormatCSV))

		f, err = ingress.ResolveFormat("landscape", ingress.ParseOptions{ContentType: "application/json"})
		Expect(err).NotTo(HaveOccurred())
		Expect(f).To(Equal(ingress.FormatJSON))

		_, err = ingress.ResolveFormat("landscape", ingress.ParseOptions{ContentType: "text/html"})
		Expect(err).To(MatchError(ContainSubstring("cannot detect format")))
	})
})
