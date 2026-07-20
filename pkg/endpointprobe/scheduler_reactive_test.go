package endpointprobe

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/pkg/backend"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/api"
	"go.uber.org/zap"
)

func newTLSServer() *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func apiInstanceForServer(server *httptest.Server) api.ApiInstance {
	parsed, err := url.Parse(server.URL)
	Expect(err).NotTo(HaveOccurred())

	host, port, err := net.SplitHostPort(parsed.Host)
	Expect(err).NotTo(HaveOccurred())

	ai := api.NewApiInstance(uuid.New())
	ai.GetAnnotations().Add(annProtocol, parsed.Scheme)
	ai.GetAnnotations().Add(annHost, host)
	ai.GetAnnotations().Add(annPort, port)
	return ai
}

func startReactiveScheduler(instances []api.ApiInstance) (*Scheduler, *atomic.Int32, context.CancelFunc) {
	var probeCount atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	sched := &Scheduler{
		Client:              &fakeApiInstanceClient{instances: instances},
		Prober:              NewProber(5 * time.Second),
		Interval:            time.Hour,
		Debounce:            50 * time.Millisecond,
		MaxConcurrentProbes: 1,
		Logger:              zap.NewNop().Sugar(),
		EventHook: func(ProbeResult) {
			probeCount.Add(1)
		},
	}

	go sched.Run(ctx)
	return sched, &probeCount, cancel
}

func waitForInitialScan(probeCount *atomic.Int32) {
	Eventually(func() int32 {
		return probeCount.Load()
	}, "1s", "10ms").Should(BeNumerically(">=", 1))
}

var _ = Describe("Reactive scheduler", func() {
	Describe("Notify", func() {
		var (
			server     *httptest.Server
			sched      *Scheduler
			probeCount *atomic.Int32
			cancel     context.CancelFunc
		)

		BeforeEach(func() {
			server = newTLSServer()
			ai := apiInstanceForServer(server)
			sched, probeCount, cancel = startReactiveScheduler([]api.ApiInstance{ai})
			waitForInitialScan(probeCount)
		})

		AfterEach(func() {
			cancel()
			sched.Wait()
			server.Close()
		})

		It("triggers a debounced rescan after the initial scan", func() {
			before := probeCount.Load()
			sched.Notify()

			Eventually(func() int32 {
				return probeCount.Load()
			}, "1s", "10ms").Should(BeNumerically(">", before))
		})

		It("coalesces rapid notifications into a single rescan", func() {
			before := probeCount.Load()
			sched.Notify()
			sched.Notify()
			sched.Notify()

			Eventually(func() int32 {
				return probeCount.Load()
			}, "1s", "10ms").Should(Equal(before + 1))
		})
	})

	Describe("RescanFilter", func() {
		Context("when the event is unrelated to ApiInstance", func() {
			It("passes the event through unchanged", func() {
				sched := &Scheduler{Logger: zap.NewNop().Sugar()}
				filter := NewRescanFilter(sched)

				ev := events.Event{
					ResourceType: events.SystemResource,
					Operation:    events.CreateOperation,
					ResourceId:   uuid.New(),
				}

				Expect(filter.Fn(nil, ev)).To(Equal([]events.Event{ev}))
			})
		})

		Context("when an ApiInstance is updated", func() {
			var (
				server     *httptest.Server
				sched      *Scheduler
				probeCount *atomic.Int32
				cancel     context.CancelFunc
				ai         api.ApiInstance
			)

			BeforeEach(func() {
				server = newTLSServer()
				ai = apiInstanceForServer(server)
				sched, probeCount, cancel = startReactiveScheduler([]api.ApiInstance{ai})
				waitForInitialScan(probeCount)
			})

			AfterEach(func() {
				cancel()
				sched.Wait()
				server.Close()
			})

			It("passes the event through and triggers a debounced rescan", func() {
				filter := NewRescanFilter(sched)
				ev := events.Event{
					ResourceType: events.APIInstanceResource,
					Operation:    events.UpdateOperation,
					ResourceId:   ai.GetInstanceId(),
				}

				before := probeCount.Load()
				Expect(filter.Fn(nil, ev)).To(Equal([]events.Event{ev}))

				Eventually(func() int32 {
					return probeCount.Load()
				}, "1s", "10ms").Should(BeNumerically(">", before))
			})
		})

		Context("when wired through the backend filter chain", func() {
			var (
				b          backend.Backend
				server     *httptest.Server
				sched      *Scheduler
				probeCount atomic.Int32
			)

			BeforeEach(func() {
				var err error
				b, err = backend.New()
				Expect(err).NotTo(HaveOccurred())

				server = newTLSServer()

				ctx, cancel := context.WithCancel(context.Background())

				sched = &Scheduler{
					Client:              NewModelClient(b.GetModel()),
					Prober:              NewProber(5 * time.Second),
					Interval:            time.Hour,
					Debounce:            50 * time.Millisecond,
					MaxConcurrentProbes: 1,
					Logger:              zap.NewNop().Sugar(),
					EventHook: func(ProbeResult) {
						probeCount.Add(1)
					},
				}

				filterID := b.GetChain().RegisterFilter(NewRescanFilter(sched))
				DeferCleanup(func() { b.GetChain().Unregister(filterID) })

				go sched.Run(ctx)
				DeferCleanup(func() {
					cancel()
					sched.Wait()
				})

				time.Sleep(20 * time.Millisecond)
			})

			AfterEach(func() {
				server.Close()
			})

			It("rescans after a model mutation adds a probe target", func() {
				Expect(probeCount.Load()).To(Equal(int32(0)), "no targets before ApiInstance is added")

				ai := apiInstanceForServer(server)
				Expect(b.GetModel().AddApiInstance(ai)).To(Succeed())

				Eventually(func() int32 {
					return probeCount.Load()
				}, "2s", "20ms").Should(BeNumerically(">=", 1))
			})
		})
	})
})
