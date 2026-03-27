package oapi_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	eventmgr "go.emeland.io/modelsrv/internal/events"
	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

func newServer() (model.Model, events.EventManager, *httptest.Server) {
	em, err := eventmgr.NewEventManager()
	Expect(err).NotTo(HaveOccurred())
	sink, err := em.GetSink()
	Expect(err).NotTo(HaveOccurred())
	m, err := model.NewModel(sink)
	Expect(err).NotTo(HaveOccurred())

	r := mux.NewRouter()
	srv := oapi.NewApiServer(m, em, "http://test")
	strict := oapi.NewApiHandler(srv)
	h := oapi.HandlerFromMuxWithBaseURL(strict, r, "/api")

	ts := httptest.NewServer(h)
	return m, em, ts
}

func postEventsRegister(upstreamAPIBase, callbackURL string) {
	body := fmt.Sprintf(`{"callbackUrl":%q}`, callbackURL)
	resp, err := http.Post(upstreamAPIBase+"/events/register", "application/json", bytes.NewReader([]byte(body)))
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close() //nolint:errcheck
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
}

var _ = Describe("phase-1 event replication (server to server)", func() {
	It("replays prior events onto the subscriber model when it registers", func() {
		mA, _, srvA := newServer()
		defer srvA.Close()

		sid := uuid.New()
		sys := model.NewSystem(mA.GetSink(), sid)
		sys.SetDisplayName("Upstream System")
		Expect(mA.AddSystem(sys)).To(Succeed())

		mB, _, srvB := newServer()
		defer srvB.Close()

		postEventsRegister(srvA.URL+"/api", srvB.URL+"/api")

		got := mB.GetSystemById(sid)
		Expect(got).NotTo(BeNil())
		Expect(got.GetDisplayName()).To(Equal("Upstream System"))
	})

	It("applies live events to a subscriber that registered earlier", func() {
		mA, _, srvA := newServer()
		defer srvA.Close()
		mB, _, srvB := newServer()
		defer srvB.Close()

		postEventsRegister(srvA.URL+"/api", srvB.URL+"/api")

		sid := uuid.New()
		sys := model.NewSystem(mA.GetSink(), sid)
		sys.SetDisplayName("Live System")
		Expect(mA.AddSystem(sys)).To(Succeed())

		Eventually(func() string {
			g := mB.GetSystemById(sid)
			if g == nil {
				return ""
			}
			return g.GetDisplayName()
		}, "2s", "20ms").Should(Equal("Live System"))
	})

	It("chains replication so a subscriber of a subscriber converges to the same state", func() {
		mA, _, srvA := newServer()
		defer srvA.Close()
		_, _, srvB := newServer()
		defer srvB.Close()
		mC, _, srvC := newServer()
		defer srvC.Close()

		postEventsRegister(srvA.URL+"/api", srvB.URL+"/api")
		postEventsRegister(srvB.URL+"/api", srvC.URL+"/api")

		sid := uuid.New()
		sys := model.NewSystem(mA.GetSink(), sid)
		sys.SetDisplayName("Chained System")
		Expect(mA.AddSystem(sys)).To(Succeed())

		Eventually(func() string {
			g := mC.GetSystemById(sid)
			if g == nil {
				return ""
			}
			return g.GetDisplayName()
		}, "3s", "20ms").Should(Equal("Chained System"))
	})
})
