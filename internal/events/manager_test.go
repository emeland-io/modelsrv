package eventmgr_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	eventmgr "go.emeland.io/modelsrv/internal/events"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/system"
)

func newPushCountingServer() (*httptest.Server, *int32) {
	var n int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/events/push", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		atomic.AddInt32(&n, 1)
		w.WriteHeader(http.StatusOK)
	})
	return httptest.NewServer(mux), &n
}

func emitSystemCreate(sink events.EventSink, id uuid.UUID) error {
	sys := system.NewSystem(sink, id)
	sys.SetDisplayName("bdd-system")
	return sink.Receive(events.SystemResource, events.CreateOperation, id, sys)
}

var _ = Describe("EventManager", func() {
	var (
		ctx context.Context
		em  events.EventManager
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		em, err = eventmgr.NewEventManager()
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("NewEventManager", func() {
		It("starts with sequence zero and a usable sink", func() {
			seq, err := em.GetCurrentSequenceId(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(seq).To(Equal(uint64(0)))

			sink, err := em.GetSink()
			Expect(err).NotTo(HaveOccurred())
			Expect(sink).NotTo(BeNil())
		})
	})

	Describe("recording sink", func() {
		It("increments sequence for each Receive", func() {
			sink, err := em.GetSink()
			Expect(err).NotTo(HaveOccurred())
			id := uuid.New()
			Expect(emitSystemCreate(sink, id)).To(Succeed())
			seq, err := em.GetCurrentSequenceId(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(seq).To(Equal(uint64(1)))

			id2 := uuid.New()
			Expect(emitSystemCreate(sink, id2)).To(Succeed())
			seq, err = em.GetCurrentSequenceId(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(seq).To(Equal(uint64(2)))
		})
	})

	Describe("subscribers", func() {
		It("replays prior events synchronously when a subscriber is added", func() {
			sink, err := em.GetSink()
			Expect(err).NotTo(HaveOccurred())
			id1 := uuid.New()
			id2 := uuid.New()
			Expect(emitSystemCreate(sink, id1)).To(Succeed())
			Expect(emitSystemCreate(sink, id2)).To(Succeed())

			srv, count := newPushCountingServer()
			defer srv.Close()

			Expect(em.AddSubscriber(srv.URL + "/api")).To(Succeed())
			Expect(atomic.LoadInt32(count)).To(Equal(int32(2)))
		})

		It("delivers no replay when there were no prior events", func() {
			srv, count := newPushCountingServer()
			defer srv.Close()

			Expect(em.AddSubscriber(srv.URL + "/api")).To(Succeed())
			Expect(atomic.LoadInt32(count)).To(Equal(int32(0)))
		})

		It("notifies subscribers asynchronously for new events after registration", func() {
			srv, count := newPushCountingServer()
			defer srv.Close()
			Expect(em.AddSubscriber(srv.URL + "/api")).To(Succeed())

			sink, err := em.GetSink()
			Expect(err).NotTo(HaveOccurred())
			Expect(emitSystemCreate(sink, uuid.New())).To(Succeed())

			Eventually(func() int32 {
				return atomic.LoadInt32(count)
			}, "2s", "10ms").Should(Equal(int32(1)))
		})

		It("is idempotent when registering the same callback URL twice", func() {
			srv, count := newPushCountingServer()
			defer srv.Close()
			base := srv.URL + "/api"

			Expect(em.AddSubscriber(base)).To(Succeed())
			Expect(em.AddSubscriber(base)).To(Succeed())
			Expect(em.GetSubscribers()).To(HaveLen(1))

			sink, err := em.GetSink()
			Expect(err).NotTo(HaveOccurred())
			Expect(emitSystemCreate(sink, uuid.New())).To(Succeed())

			Eventually(func() int32 {
				return atomic.LoadInt32(count)
			}, "2s", "10ms").Should(Equal(int32(1)))
		})

		It("removes a subscriber by URL", func() {
			srv, _ := newPushCountingServer()
			defer srv.Close()
			base := srv.URL + "/api"
			Expect(em.AddSubscriber(base)).To(Succeed())
			Expect(em.RemoveSubscriber(base)).To(Succeed())
			Expect(em.GetSubscribers()).To(BeEmpty())
		})

		It("returns an error when removing an unknown subscriber URL", func() {
			Expect(em.RemoveSubscriber("http://no-such.example/api")).NotTo(Succeed())
		})
	})
})

var _ = Describe("EnumeratedListSink", func() {
	It("records events in order and exposes them by index", func() {
		sink := eventmgr.NewEnumeratedListSink()
		id := uuid.New()
		Expect(sink.Receive(events.SystemResource, events.CreateOperation, id)).To(Succeed())
		Expect(sink.GetEventCount()).To(Equal(1))
		ev, err := sink.GetEventByIndex(0)
		Expect(err).NotTo(HaveOccurred())
		Expect(ev.ResourceType).To(Equal(events.SystemResource))
		Expect(ev.Operation).To(Equal(events.CreateOperation))
		Expect(ev.ResourceId).To(Equal(id))
	})

	It("returns an error for out-of-range index", func() {
		sink := eventmgr.NewEnumeratedListSink()
		_, err := sink.GetEventByIndex(0)
		Expect(err).To(HaveOccurred())
	})
})
