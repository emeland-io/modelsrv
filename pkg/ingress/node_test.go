package ingress_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/ingress"
	"go.emeland.io/modelsrv/pkg/model"
)

var _ = Describe("Apply Node url and subscriptions", func() {
	newModel := func() model.Model {
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())
		return m
	}

	It("sets url and subscriptions from a Node document", func() {
		data := []byte(`version: emeland.io/v1
kind: Node
spec:
  nodeId: aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa
  displayName: edge-node
  url: https://edge.example.com/api
  subscriptions:
    - https://peer-a.example.com/api
    - https://peer-b.example.com/api
`)
		m := newModel()
		docs, err := ingress.Parse("node.yaml", data, ingress.ParseOptions{})
		Expect(err).NotTo(HaveOccurred())

		res := ingress.ApplyAll(docs, m)
		Expect(res.Failed).To(BeEmpty())
		Expect(res.Applied).To(Equal(1))

		n := m.GetNodeById(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
		Expect(n).NotTo(BeNil())
		Expect(n.GetUrl()).To(Equal("https://edge.example.com/api"))
		Expect(n.GetSubscriptions()).To(Equal([]string{
			"https://peer-a.example.com/api",
			"https://peer-b.example.com/api",
		}))
	})

	It("defaults subscriptions to an empty list when omitted", func() {
		data := []byte(`version: emeland.io/v1
kind: Node
spec:
  nodeId: bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb
  displayName: lonely-node
  url: https://lonely.example.com/api
`)
		m := newModel()
		docs, err := ingress.Parse("node.yaml", data, ingress.ParseOptions{})
		Expect(err).NotTo(HaveOccurred())

		res := ingress.ApplyAll(docs, m)
		Expect(res.Failed).To(BeEmpty())
		Expect(res.Applied).To(Equal(1))

		n := m.GetNodeById(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
		Expect(n).NotTo(BeNil())
		Expect(n.GetUrl()).To(Equal("https://lonely.example.com/api"))
		Expect(n.GetSubscriptions()).NotTo(BeNil())
		Expect(n.GetSubscriptions()).To(BeEmpty())
	})

	It("fails when the required url is missing", func() {
		data := []byte(`version: emeland.io/v1
kind: Node
spec:
  nodeId: cccccccc-cccc-cccc-cccc-cccccccccccc
  displayName: no-url-node
`)
		m := newModel()
		docs, err := ingress.Parse("node.yaml", data, ingress.ParseOptions{})
		Expect(err).NotTo(HaveOccurred())

		res := ingress.ApplyAll(docs, m)
		Expect(res.Applied).To(Equal(0))
		Expect(res.Failed).To(HaveLen(1))
		Expect(m.GetNodeById(uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"))).To(BeNil())
	})
})
