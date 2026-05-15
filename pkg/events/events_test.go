package events_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.emeland.io/modelsrv/pkg/events"
)

var _ = Describe("ResourceType", func() {
	Describe("ParseResourceType", func() {
		It("maps known string labels back to the typed constant", func() {
			Expect(events.ParseResourceType("System")).To(Equal(events.SystemResource))
			Expect(events.ParseResourceType("APIInstance")).To(Equal(events.APIInstanceResource))
			Expect(events.ParseResourceType("Finding")).To(Equal(events.FindingResource))
			Expect(events.ParseResourceType("Product")).To(Equal(events.ProductResource))
		})

		It("returns UnknownResourceType for an unknown label", func() {
			Expect(events.ParseResourceType("no-such")).To(Equal(events.UnknownResourceType))
		})
	})

	Describe("String", func() {
		It("returns the wire-style name for known types", func() {
			Expect(events.SystemResource.String()).To(Equal("System"))
			Expect(events.APIResource.String()).To(Equal("API"))
		})

		It("returns UnknownResourceType label for an out-of-range value", func() {
			Expect(events.ResourceType(9999).String()).To(Equal("UnknownResourceType"))
		})
	})

	Describe("WireKind", func() {
		It("uses ApiInstance for APIInstanceResource to match OpenAPI enum", func() {
			Expect(events.APIInstanceResource.WireKind()).To(Equal("ApiInstance"))
		})

		It("matches String for other phase-1 kinds", func() {
			Expect(events.SystemResource.WireKind()).To(Equal("System"))
			Expect(events.ComponentResource.WireKind()).To(Equal("Component"))
		})
	})

	Describe("ParseWireKind", func() {
		It("round-trips WireKind for phase-1 kinds", func() {
			for _, rt := range []events.ResourceType{
				events.SystemResource,
				events.SystemInstanceResource,
				events.APIResource,
				events.APIInstanceResource,
				events.ComponentResource,
				events.ComponentInstanceResource,
			} {
				Expect(events.ParseWireKind(rt.WireKind())).To(Equal(rt))
			}
		})

		It("returns UnknownResourceType for unknown kind strings", func() {
			Expect(events.ParseWireKind("NoSuchKind")).To(Equal(events.UnknownResourceType))
		})

		It("does not treat Annotations as a standalone replication kind", func() {
			Expect(events.ParseWireKind("Annotations")).To(Equal(events.UnknownResourceType))
		})

		It("maps Node and Context kinds", func() {
			Expect(events.ParseWireKind("Node")).To(Equal(events.NodeResource))
			Expect(events.ParseWireKind("Context")).To(Equal(events.ContextResource))
			Expect(events.ParseWireKind("Product")).To(Equal(events.ProductResource))
		})
	})
})

var _ = Describe("Operation", func() {
	Describe("ParseOperation", func() {
		It("maps internal operation names", func() {
			Expect(events.ParseOperation("CreateOperation")).To(Equal(events.CreateOperation))
			Expect(events.ParseOperation("DeleteOperation")).To(Equal(events.DeleteOperation))
		})

		It("returns UnknownOperation for unknown strings", func() {
			Expect(events.ParseOperation("Create")).To(Equal(events.UnknownOperation))
		})
	})

	Describe("String", func() {
		It("returns the internal label", func() {
			Expect(events.CreateOperation.String()).To(Equal("CreateOperation"))
		})
	})

	Describe("WireOperation", func() {
		It("returns short JSON labels", func() {
			Expect(events.CreateOperation.WireOperation()).To(Equal("Create"))
			Expect(events.UpdateOperation.WireOperation()).To(Equal("Update"))
			Expect(events.DeleteOperation.WireOperation()).To(Equal("Delete"))
			Expect(events.UnknownOperation.WireOperation()).To(Equal("Unknown"))
		})
	})

	Describe("ParseWireOperation", func() {
		It("round-trips WireOperation for known operations", func() {
			for _, op := range []events.Operation{
				events.CreateOperation,
				events.UpdateOperation,
				events.DeleteOperation,
			} {
				Expect(events.ParseWireOperation(op.WireOperation())).To(Equal(op))
			}
		})

		It("returns UnknownOperation for unknown strings", func() {
			Expect(events.ParseWireOperation("CreateOperation")).To(Equal(events.UnknownOperation))
		})
	})
})

var _ = Describe("Event", func() {
	Describe("String", func() {
		It("formats delete events without object payload", func() {
			id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
			e := events.Event{
				ResourceType: events.SystemResource,
				Operation:    events.DeleteOperation,
				ResourceId:   id,
			}
			Expect(e.String()).To(ContainSubstring("DeleteOperation"))
			Expect(e.String()).To(ContainSubstring(id.String()))
		})

		It("formats non-delete events with objects", func() {
			id := uuid.New()
			e := events.Event{
				ResourceType: events.APIResource,
				Operation:    events.CreateOperation,
				ResourceId:   id,
				Objects:      []any{"payload"},
			}
			s := e.String()
			Expect(s).To(ContainSubstring("CreateOperation"))
			Expect(s).To(ContainSubstring("payload"))
		})
	})
})

var _ = Describe("ListSink", func() {
	It("records events and mirrors String() into GetList", func() {
		sink := events.NewListSink()
		id := uuid.New()
		Expect(sink.Receive(events.ContextResource, events.CreateOperation, id)).To(Succeed())

		evs := sink.GetEvents()
		Expect(evs).To(HaveLen(1))
		Expect(evs[0].ResourceType).To(Equal(events.ContextResource))
		Expect(evs[0].Operation).To(Equal(events.CreateOperation))
		Expect(evs[0].ResourceId).To(Equal(id))

		list := sink.GetList()
		Expect(list).To(HaveLen(1))
		Expect(list[0]).To(Equal(evs[0].String()))
	})

	It("appends in order", func() {
		sink := events.NewListSink()
		id1 := uuid.New()
		id2 := uuid.New()
		Expect(sink.Receive(events.NodeResource, events.CreateOperation, id1)).To(Succeed())
		Expect(sink.Receive(events.NodeResource, events.UpdateOperation, id2)).To(Succeed())
		Expect(sink.GetEvents()[0].Operation).To(Equal(events.CreateOperation))
		Expect(sink.GetEvents()[1].Operation).To(Equal(events.UpdateOperation))
	})
})

var _ = Describe("NewDummySink", func() {
	It("accepts Receive without storing or erroring", func() {
		sink := events.NewDummySink()
		id := uuid.New()
		Expect(sink.Receive(events.SystemResource, events.CreateOperation, id, "x")).To(Succeed())
	})
})
