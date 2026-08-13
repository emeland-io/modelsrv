package resolvefindings_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.emeland.io/modelsrv/pkg/eventfilter"
	"go.emeland.io/modelsrv/pkg/eventfilter/phase0"
	"go.emeland.io/modelsrv/pkg/eventfilter/resolvefindings"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/component"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

func newModel() model.Model {
	m, err := model.NewModel(events.NewDummySink())
	Expect(err).NotTo(HaveOccurred())
	if m == nil { // required by nilaway
		Fail("NewModel returned nil model")
	}
	return m
}

func newModelWithResolveChain() (model.Model, *events.ListSink) {
	listSink := events.NewListSink()
	chain := eventfilter.NewChain(nil)
	filtered := eventfilter.NewFilteringSink(chain, listSink)
	m, err := model.NewModel(filtered)
	Expect(err).NotTo(HaveOccurred())
	if m == nil { // required by nilaway
		Fail("NewModel returned nil model")
	}
	chain.SetModel(m)
	chain.RegisterFilter(resolvefindings.New())
	resolvefindings.EnsureWellKnownFindingTypes(m)
	return m, listSink
}

func newModelWithPhase0AndResolve() (model.Model, *events.ListSink) {
	listSink := events.NewListSink()
	chain := eventfilter.NewChain(nil)
	filtered := eventfilter.NewFilteringSink(chain, listSink)
	m, err := model.NewModel(filtered)
	Expect(err).NotTo(HaveOccurred())
	if m == nil { // required by nilaway
		Fail("NewModel returned nil model")
	}
	chain.SetModel(m)
	chain.RegisterFilter(phase0.New())
	phase0.EnsureWellKnownFindingTypes(m)
	chain.RegisterFilter(resolvefindings.New())
	resolvefindings.EnsureWellKnownFindingTypes(m)
	return m, listSink
}

func findingsOfKind(m model.Model, kind finding.FindingKind) []finding.Finding {
	wantName := string(kind)
	typeID := finding.TypeIDForKind(kind)
	all, err := m.GetFindings()
	Expect(err).NotTo(HaveOccurred())
	var out []finding.Finding
	for _, f := range all {
		ftID := f.GetFindingTypeId()
		if ftID == typeID {
			out = append(out, f)
			continue
		}
		ft := m.GetFindingTypeById(ftID)
		if ft != nil && ft.GetDisplayName() == wantName {
			out = append(out, f)
		}
	}
	return out
}

func addDanglingFinding(m model.Model, kind finding.FindingKind, subject, missing *common.ResourceRef) finding.Finding {
	id := uuid.New()
	f := finding.NewFinding(id)
	if kind != "" {
		f.SetFindingTypeById(finding.TypeIDForKind(kind))
		f.SetDisplayName(string(kind))
	} else {
		// Unknown custom kind — arbitrary type id, not in documented table.
		customType := finding.NewFindingType(uuid.New())
		customType.SetDisplayName("CustomUnknownKind")
		Expect(m.AddFindingType(customType)).To(Succeed())
		f.SetFindingTypeById(customType.GetFindingTypeId())
		f.SetDisplayName("CustomUnknownKind")
	}
	f.SetDescription("test dangling ref")
	refs := []*common.ResourceRef{subject}
	if missing != nil {
		refs = append(refs, missing)
	}
	f.SetResources(refs)
	Expect(m.AddFinding(f)).To(Succeed())
	return f
}

var _ = Describe("resolvefindings filter identity", func() {
	It("New returns the expected display name and description", func() {
		f := resolvefindings.New()
		Expect(f.DisplayName).To(Equal("Resolve findings"))
		Expect(f.Description).To(ContainSubstring("Deletes findings"))
		Expect(f.Fn).NotTo(BeNil())
	})
})

var _ = Describe("structural dangling-ref resolution", func() {
	It("clears ReferencedResourceNotFound when the missing API is added", func() {
		m, sink := newModelWithResolveChain()
		apiID := uuid.New()
		aiID := uuid.New()

		ai := mdlapi.NewApiInstance(aiID)
		ai.SetDisplayName("svc")
		ai.SetApiRef(&mdlapi.ApiRef{ApiID: apiID})
		Expect(m.AddApiInstance(ai)).To(Succeed())

		addDanglingFinding(m, finding.ReferencedResourceNotFound,
			&common.ResourceRef{ResourceId: aiID, ResourceType: events.APIInstanceResource},
			&common.ResourceRef{ResourceId: apiID, ResourceType: events.APIResource},
		)
		Expect(findingsOfKind(m, finding.ReferencedResourceNotFound)).To(HaveLen(1))

		api := mdlapi.NewAPI(apiID)
		api.SetDisplayName("Payments API")
		Expect(m.AddApi(api)).To(Succeed())

		Expect(findingsOfKind(m, finding.ReferencedResourceNotFound)).To(BeEmpty())

		var sawFindingDelete bool
		for _, e := range sink.GetEvents() {
			if e.ResourceType == events.FindingResource && e.Operation == events.DeleteOperation {
				sawFindingDelete = true
				break
			}
		}
		Expect(sawFindingDelete).To(BeTrue())
	})

	It("clears dangling ComponentInstance finding when Component appears", func() {
		m, _ := newModelWithResolveChain()
		compID := uuid.New()
		ciID := uuid.New()

		ci := component.NewComponentInstance(ciID)
		ci.SetDisplayName("deploy")
		ci.SetComponentRef(&component.ComponentRef{ComponentId: compID})
		Expect(m.AddComponentInstance(ci)).To(Succeed())

		addDanglingFinding(m, finding.ReferencedResourceNotFound,
			&common.ResourceRef{ResourceId: ciID, ResourceType: events.ComponentInstanceResource},
			&common.ResourceRef{ResourceId: compID, ResourceType: events.ComponentResource},
		)
		Expect(findingsOfKind(m, finding.ReferencedResourceNotFound)).To(HaveLen(1))

		comp := component.NewComponent(compID)
		comp.SetDisplayName("web")
		Expect(m.AddComponent(comp)).To(Succeed())

		Expect(findingsOfKind(m, finding.ReferencedResourceNotFound)).To(BeEmpty())
	})

	It("clears an unknown FindingKind when Resources cite a missing resource that appears", func() {
		m, _ := newModelWithResolveChain()
		apiID := uuid.New()
		aiID := uuid.New()

		ai := mdlapi.NewApiInstance(aiID)
		ai.SetDisplayName("svc")
		Expect(m.AddApiInstance(ai)).To(Succeed())

		f := addDanglingFinding(m, "",
			&common.ResourceRef{ResourceId: aiID, ResourceType: events.APIInstanceResource},
			&common.ResourceRef{ResourceId: apiID, ResourceType: events.APIResource},
		)
		Expect(m.GetFindingById(f.GetFindingId())).NotTo(BeNil())

		api := mdlapi.NewAPI(apiID)
		api.SetDisplayName("API")
		Expect(m.AddApi(api)).To(Succeed())

		Expect(m.GetFindingById(f.GetFindingId())).To(BeNil())
	})

	It("retains an unknown FindingKind with no resolvable Resources shape", func() {
		m, _ := newModelWithResolveChain()
		aiID := uuid.New()

		ai := mdlapi.NewApiInstance(aiID)
		ai.SetDisplayName("svc")
		Expect(m.AddApiInstance(ai)).To(Succeed())

		f := addDanglingFinding(m, "",
			&common.ResourceRef{ResourceId: aiID, ResourceType: events.APIInstanceResource},
			nil, // subject only — cannot resolve structurally
		)
		Expect(m.GetFindingById(f.GetFindingId())).NotTo(BeNil())

		api := mdlapi.NewAPI(uuid.New())
		api.SetDisplayName("unrelated")
		Expect(m.AddApi(api)).To(Succeed())

		Expect(m.GetFindingById(f.GetFindingId())).NotTo(BeNil())
	})
})

var _ = Describe("MissingResourceReference resolution", func() {
	It("clears when ApiInstance later gains an ApiRef", func() {
		m, _ := newModelWithResolveChain()
		aiID := uuid.New()

		ai := mdlapi.NewApiInstance(aiID)
		ai.SetDisplayName("gateway")
		Expect(m.AddApiInstance(ai)).To(Succeed())

		f := finding.NewFinding(uuid.New())
		f.SetFindingTypeById(finding.TypeIDForKind(finding.MissingResourceReference))
		f.SetDisplayName(string(finding.MissingResourceReference))
		f.SetDescription("missing api ref annotation")
		f.SetResources([]*common.ResourceRef{
			{ResourceId: aiID, ResourceType: events.APIInstanceResource},
		})
		Expect(m.AddFinding(f)).To(Succeed())
		Expect(findingsOfKind(m, finding.MissingResourceReference)).To(HaveLen(1))

		updated := mdlapi.NewApiInstance(aiID)
		updated.SetDisplayName("gateway")
		updated.SetApiRef(&mdlapi.ApiRef{ApiID: uuid.New()})
		Expect(m.AddApiInstance(updated)).To(Succeed())

		Expect(findingsOfKind(m, finding.MissingResourceReference)).To(BeEmpty())
	})

	It("clears when ComponentInstance later gains a ComponentRef", func() {
		m, _ := newModelWithResolveChain()
		ciID := uuid.New()

		ci := component.NewComponentInstance(ciID)
		ci.SetDisplayName("job")
		Expect(m.AddComponentInstance(ci)).To(Succeed())

		f := finding.NewFinding(uuid.New())
		f.SetFindingTypeById(finding.TypeIDForKind(finding.MissingResourceReference))
		f.SetDisplayName(string(finding.MissingResourceReference))
		f.SetResources([]*common.ResourceRef{
			{ResourceId: ciID, ResourceType: events.ComponentInstanceResource},
		})
		Expect(m.AddFinding(f)).To(Succeed())

		updated := component.NewComponentInstance(ciID)
		updated.SetDisplayName("job")
		updated.SetComponentRef(&component.ComponentRef{ComponentId: uuid.New()})
		Expect(m.AddComponentInstance(updated)).To(Succeed())

		Expect(findingsOfKind(m, finding.MissingResourceReference)).To(BeEmpty())
	})
})

var _ = Describe("phase0 non-interference", func() {
	It("does not structurally delete phase0 ContextTypeMissing findings", func() {
		m, _ := newModelWithPhase0AndResolve()
		typeID := uuid.New()

		ctx := mdlctx.NewContext(uuid.New())
		ctx.SetContextTypeById(typeID)
		Expect(m.AddContext(ctx)).To(Succeed())
		Expect(findingsOfKind(m, finding.ContextTypeMissing)).To(HaveLen(1))

		// Resolvefindings must not clear this when an unrelated resource arrives.
		api := mdlapi.NewAPI(uuid.New())
		api.SetDisplayName("unrelated")
		Expect(m.AddApi(api)).To(Succeed())
		Expect(findingsOfKind(m, finding.ContextTypeMissing)).To(HaveLen(1))

		// Phase0 still clears when the ContextType appears.
		ct := mdlctx.NewContextType(typeID)
		ct.SetDisplayName("env")
		Expect(m.AddContextType(ct)).To(Succeed())
		Expect(findingsOfKind(m, finding.ContextTypeMissing)).To(BeEmpty())
	})
})

var _ = Describe("EnsureWellKnownFindingTypes", func() {
	It("registers documented FindingTypes", func() {
		m := newModel()
		resolvefindings.EnsureWellKnownFindingTypes(m)

		Expect(m.GetFindingTypeById(finding.TypeIDForKind(finding.ReferencedResourceNotFound))).NotTo(BeNil())
		Expect(m.GetFindingTypeById(finding.TypeIDForKind(finding.MissingResourceReference))).NotTo(BeNil())
	})
})
