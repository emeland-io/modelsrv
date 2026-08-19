package eventmgr_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	eventmgr "go.emeland.io/modelsrv/internal/events"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/system"
)

func newPushServer(h http.HandlerFunc) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/events/push", h)
	return httptest.NewServer(mux)
}

// pushedDisplayName reads the display name out of a POST /events/push body,
// which tests use as an ordering marker.
func pushedDisplayName(body []byte) string {
	var wire struct {
		Resource map[string]any `json:"resource"`
	}
	Expect(json.Unmarshal(body, &wire)).To(Succeed())
	name, _ := wire.Resource["displayName"].(string)
	return name
}

func newPushCountingServer() (*httptest.Server, *int32) {
	var n int32
	srv := newPushServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		atomic.AddInt32(&n, 1)
		w.WriteHeader(http.StatusOK)
	})
	return srv, &n
}

func emitSystemCreate(sink events.EventSink, id uuid.UUID) error {
	return emitNamedSystemCreate(sink, id, "bdd-system")
}

func emitNamedSystemCreate(sink events.EventSink, id uuid.UUID, displayName string) error {
	sys := system.NewSystem(id)
	sys.SetDisplayName(displayName)
	return sink.Receive(events.SystemResource, events.CreateOperation, id, sys)
}

func emitSystemDelete(sink events.EventSink, id uuid.UUID) error {
	return sink.Receive(events.SystemResource, events.DeleteOperation, id)
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

		It("replays only currently-live resources, reflecting deletes that happened before the subscriber was added", func() {
			sink, err := em.GetSink()
			Expect(err).NotTo(HaveOccurred())
			id1 := uuid.New()
			id2 := uuid.New()
			Expect(emitSystemCreate(sink, id1)).To(Succeed())
			Expect(emitSystemCreate(sink, id2)).To(Succeed())
			Expect(emitSystemDelete(sink, id1)).To(Succeed())

			srv, count := newPushCountingServer()
			defer srv.Close()

			Expect(em.AddSubscriber(srv.URL + "/api")).To(Succeed())
			Expect(atomic.LoadInt32(count)).To(Equal(int32(1)))
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

		It("delivers a burst one event at a time, in the order the model produced them", func() {
			var mu sync.Mutex
			var inFlight, maxInFlight int
			var delivered []string

			srv := newPushServer(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				body, err := io.ReadAll(r.Body)
				Expect(err).NotTo(HaveOccurred())

				mu.Lock()
				inFlight++
				if inFlight > maxInFlight {
					maxInFlight = inFlight
				}
				mu.Unlock()

				time.Sleep(5 * time.Millisecond)

				mu.Lock()
				inFlight--
				delivered = append(delivered, pushedDisplayName(body))
				mu.Unlock()
				w.WriteHeader(http.StatusOK)
			})
			defer srv.Close()

			Expect(em.AddSubscriber(srv.URL + "/api")).To(Succeed())
			sink, err := em.GetSink()
			Expect(err).NotTo(HaveOccurred())

			const burst = 24
			emitted := make([]string, 0, burst)
			for i := range burst {
				name := fmt.Sprintf("system-%02d", i)
				emitted = append(emitted, name)
				Expect(emitNamedSystemCreate(sink, uuid.New(), name)).To(Succeed())
			}

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(delivered)
			}, "5s", "10ms").Should(Equal(burst))

			mu.Lock()
			defer mu.Unlock()
			Expect(maxInFlight).To(Equal(1))
			Expect(delivered).To(Equal(emitted))
		})

		It("resyncs a subscriber from current state when its queue overflows", func() {
			release := make(chan struct{})
			var mu sync.Mutex
			seen := map[string]struct{}{}

			srv := newPushServer(func(w http.ResponseWriter, r *http.Request) {
				<-release
				body, err := io.ReadAll(r.Body)
				Expect(err).NotTo(HaveOccurred())
				mu.Lock()
				seen[pushedDisplayName(body)] = struct{}{}
				mu.Unlock()
				w.WriteHeader(http.StatusOK)
			})
			defer srv.Close()

			Expect(em.AddSubscriber(srv.URL + "/api")).To(Succeed())
			sink, err := em.GetSink()
			Expect(err).NotTo(HaveOccurred())

			// More events than the delivery queue holds, while the
			// subscriber is wedged: the excess cannot be queued.
			const total = 320
			for i := range total {
				Expect(emitNamedSystemCreate(sink, uuid.New(), fmt.Sprintf("system-%03d", i))).To(Succeed())
			}
			close(release)

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(seen)
			}, "20s", "50ms").Should(Equal(total))
		})

		It("does not stall model mutations while a subscriber is unreachable", func() {
			release := make(chan struct{})
			srv := newPushServer(func(w http.ResponseWriter, r *http.Request) {
				<-release
				w.WriteHeader(http.StatusOK)
			})
			defer srv.Close()
			defer close(release)

			Expect(em.AddSubscriber(srv.URL + "/api")).To(Succeed())
			sink, err := em.GetSink()
			Expect(err).NotTo(HaveOccurred())

			done := make(chan struct{})
			go func() {
				defer close(done)
				for range 50 {
					Expect(emitSystemCreate(sink, uuid.New())).To(Succeed())
				}
			}()

			Eventually(done, "2s").Should(BeClosed())
			seq, err := em.GetCurrentSequenceId(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(seq).To(Equal(uint64(50)))
		})

		It("retries a failed subscriber notify until it succeeds", func() {
			var attempts, successes int32
			srv := newPushServer(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				n := atomic.AddInt32(&attempts, 1)
				if n < 3 {
					http.Error(w, "unavailable", http.StatusServiceUnavailable)
					return
				}
				atomic.AddInt32(&successes, 1)
				w.WriteHeader(http.StatusOK)
			})
			defer srv.Close()

			Expect(em.AddSubscriber(srv.URL + "/api")).To(Succeed())
			sink, err := em.GetSink()
			Expect(err).NotTo(HaveOccurred())
			Expect(emitSystemCreate(sink, uuid.New())).To(Succeed())

			Eventually(func() int32 {
				return atomic.LoadInt32(&successes)
			}, "5s", "10ms").Should(Equal(int32(1)))
			Expect(atomic.LoadInt32(&attempts)).To(BeNumerically(">=", 3))
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
