package eventfilter_test

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"

	"go.emeland.io/modelsrv/pkg/eventfilter"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/mocks"
	"go.emeland.io/modelsrv/pkg/model"
)

// makeEvent is a test helper that builds a minimal events.Event.
func makeEvent(rt events.ResourceType) events.Event {
	return events.Event{
		ResourceType: rt,
		Operation:    events.CreateOperation,
		ResourceId:   uuid.New(),
	}
}

// passThroughFn is a FilterFunc that returns its input event unchanged.
func passThroughFn(_ model.Model, ev events.Event) []events.Event {
	return []events.Event{ev}
}

var _ = Describe("Chain", func() {
	var (
		ctrl      *gomock.Controller
		mockModel *mocks.MockModel
		chain     eventfilter.Chain
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockModel = mocks.NewMockModel(ctrl)
		chain = eventfilter.NewChain(mockModel)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("Apply with no registered filters", func() {
		It("returns the original event unchanged", func() {
			ev := makeEvent(events.SystemResource)
			result := chain.Apply(ev)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(Equal(ev))
		})
	})

	Describe("Apply with a single pass-through filter", func() {
		It("returns the original event unchanged", func() {
			chain.Register(passThroughFn)
			ev := makeEvent(events.APIResource)
			result := chain.Apply(ev)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(Equal(ev))
		})
	})

	Describe("Apply with a suppressing filter", func() {
		It("returns an empty slice", func() {
			chain.Register(func(_ model.Model, _ events.Event) []events.Event {
				return nil
			})
			ev := makeEvent(events.NodeResource)
			result := chain.Apply(ev)
			Expect(result).To(BeEmpty())
		})
	})

	Describe("Apply with an expanding filter", func() {
		It("returns multiple events", func() {
			extra := makeEvent(events.FindingResource)
			chain.Register(func(_ model.Model, ev events.Event) []events.Event {
				return []events.Event{ev, extra}
			})
			ev := makeEvent(events.SystemResource)
			result := chain.Apply(ev)
			Expect(result).To(HaveLen(2))
			Expect(result[0]).To(Equal(ev))
			Expect(result[1]).To(Equal(extra))
		})
	})

	Describe("Apply with multiple filters (flatMap composition)", func() {
		It("applies each filter to every event in the current batch in order", func() {
			finding := makeEvent(events.FindingResource)
			// filter 1: expand ev → [ev, finding]
			chain.Register(func(_ model.Model, ev events.Event) []events.Event {
				return []events.Event{ev, finding}
			})
			// filter 2: pass-through
			chain.Register(passThroughFn)

			ev := makeEvent(events.APIResource)
			result := chain.Apply(ev)
			// filter 1 produces [ev, finding]; filter 2 receives both and returns both unchanged
			Expect(result).To(HaveLen(2))
			Expect(result[0]).To(Equal(ev))
			Expect(result[1]).To(Equal(finding))
		})

		It("suppressing in the second filter empties the batch", func() {
			chain.Register(passThroughFn)
			chain.Register(func(_ model.Model, _ events.Event) []events.Event {
				return nil
			})
			result := chain.Apply(makeEvent(events.ContextResource))
			Expect(result).To(BeEmpty())
		})
	})

	Describe("Register", func() {
		It("returns a distinct FilterID on each call", func() {
			id1 := chain.Register(passThroughFn)
			id2 := chain.Register(passThroughFn)
			id3 := chain.Register(passThroughFn)
			Expect(id1).NotTo(Equal(id2))
			Expect(id2).NotTo(Equal(id3))
			Expect(id1).NotTo(Equal(id3))
		})
	})

	Describe("Unregister", func() {
		It("removes only the filter with the given ID", func() {
			var called []string

			id1 := chain.Register(func(_ model.Model, ev events.Event) []events.Event {
				called = append(called, "A")
				return []events.Event{ev}
			})
			_ = chain.Register(func(_ model.Model, ev events.Event) []events.Event {
				called = append(called, "B")
				return []events.Event{ev}
			})

			chain.Unregister(id1)
			chain.Apply(makeEvent(events.SystemResource))
			Expect(called).To(Equal([]string{"B"}))
		})

		It("is a no-op for an unknown FilterID", func() {
			chain.Register(passThroughFn)
			unknownID := eventfilter.FilterID(uuid.New())
			Expect(func() { chain.Unregister(unknownID) }).NotTo(Panic())
			result := chain.Apply(makeEvent(events.NodeResource))
			Expect(result).To(HaveLen(1))
		})
	})

	Describe("FilterFunc receives the model", func() {
		It("passes the model injected at NewChain to each filter", func() {
			var received model.Model
			chain.Register(func(m model.Model, ev events.Event) []events.Event {
				received = m
				return []events.Event{ev}
			})
			chain.Apply(makeEvent(events.SystemResource))
			Expect(received).To(BeIdenticalTo(mockModel))
		})
	})

	Describe("Concurrency", func() {
		It("does not race when Register and Apply run concurrently", func() {
			const goroutines = 20
			var wg sync.WaitGroup
			wg.Add(goroutines * 2)

			for i := 0; i < goroutines; i++ {
				go func() {
					defer wg.Done()
					chain.Register(passThroughFn)
				}()
				go func() {
					defer wg.Done()
					chain.Apply(makeEvent(events.APIResource))
				}()
			}
			wg.Wait()
		})
	})
})

var _ = Describe("FilteringSink", func() {
	var (
		ctrl       *gomock.Controller
		mockModel  *mocks.MockModel
		downstream *events.ListSink
		chain      eventfilter.Chain
		sink       events.EventSink
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockModel = mocks.NewMockModel(ctrl)
		downstream = events.NewListSink()
		chain = eventfilter.NewChain(mockModel)
		sink = eventfilter.NewFilteringSink(chain, downstream)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("passes the event through to downstream when the chain is empty", func() {
		id := uuid.New()
		Expect(sink.Receive(events.SystemResource, events.CreateOperation, id)).To(Succeed())
		evs := downstream.GetEvents()
		Expect(evs).To(HaveLen(1))
		Expect(evs[0].ResourceType).To(Equal(events.SystemResource))
		Expect(evs[0].Operation).To(Equal(events.CreateOperation))
		Expect(evs[0].ResourceId).To(Equal(id))
	})

	It("suppresses the event when the chain returns empty", func() {
		chain.Register(func(_ model.Model, _ events.Event) []events.Event { return nil })
		Expect(sink.Receive(events.NodeResource, events.CreateOperation, uuid.New())).To(Succeed())
		Expect(downstream.GetEvents()).To(BeEmpty())
	})

	It("forwards all expanded events to downstream in order", func() {
		extra := makeEvent(events.FindingResource)
		chain.Register(func(_ model.Model, ev events.Event) []events.Event {
			return []events.Event{ev, extra}
		})
		id := uuid.New()
		Expect(sink.Receive(events.APIResource, events.UpdateOperation, id)).To(Succeed())
		evs := downstream.GetEvents()
		Expect(evs).To(HaveLen(2))
		Expect(evs[0].ResourceId).To(Equal(id))
		Expect(evs[1]).To(Equal(extra))
	})

	It("stops forwarding and returns the error on downstream failure", func() {
		errSink := &errorSink{err: fmt.Errorf("downstream error")}
		failingSink := eventfilter.NewFilteringSink(chain, errSink)

		extra := makeEvent(events.FindingResource)
		chain.Register(func(_ model.Model, ev events.Event) []events.Event {
			return []events.Event{ev, extra}
		})
		err := failingSink.Receive(events.SystemResource, events.CreateOperation, uuid.New())
		Expect(err).To(MatchError("downstream error"))
		Expect(errSink.calls).To(Equal(1))
	})
})

// errorSink is a test double that always returns an error.
type errorSink struct {
	err   error
	calls int
}

func (e *errorSink) Receive(_ events.ResourceType, _ events.Operation, _ uuid.UUID, _ ...any) error {
	e.calls++
	return e.err
}
