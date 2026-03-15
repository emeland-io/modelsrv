package test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	ievents "go.emeland.io/modelsrv/internal/events"
	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/client"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/test/matchers"
	"go.uber.org/zap"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type modelsrvInstance struct {
	url       string
	backend   model.Model
	eventMgr  events.EventManager
	handler   http.Handler
	webserver *http.Server
	logger    *zap.SugaredLogger
	client    *client.ModelSrvClient
}

var aliceSrv, bobSrv *modelsrvInstance

func newModelsrvInstance(port string) (*modelsrvInstance, error) {
	baseUrl := fmt.Sprintf("http://localhost:%s/api", port)
	sink := events.NewListSink()
	backend, err := model.NewModel(sink)
	if err != nil {
		return nil, err
	}

	eventMgr, err := ievents.NewEventManager()
	if err != nil {
		return nil, err
	}

	server := oapi.NewApiServer(backend, eventMgr, baseUrl)
	strict := oapi.NewApiHandler(server)

	By("attaching the model to a listener")
	r := mux.NewRouter()

	// get an `http.Handler` that we can use
	handler := oapi.HandlerFromMuxWithBaseURL(strict, r, "/api")

	webServer := &http.Server{
		Handler: handler,
		Addr:    fmt.Sprintf(":%s", port),
	}

	setupLogger := zap.NewExample().Sugar()
	go runListener(setupLogger, webServer)

	client, err := client.NewModelSrvClient(baseUrl)
	if err != nil {
		return nil, err
	}

	return &modelsrvInstance{
		url:       baseUrl,
		backend:   backend,
		eventMgr:  eventMgr,
		handler:   handler,
		webserver: webServer,
		logger:    setupLogger,
		client:    client,
	}, nil
}

func runListener(log *zap.SugaredLogger, server *http.Server) {

	err := server.ListenAndServe()
	if err != nil {
		log.Error(err, ". Ended server with error")
	} else {
		log.Info("Ended server.")
	}
}

func (instance *modelsrvInstance) stop() {
	instance.webserver.Shutdown(context.Background())
}

var _ = Describe("forwarding events between two modelsrv instances", func() {

	BeforeEach(func() {
		var err error

		By("creating two modelsrv instances")
		aliceSrv, err = newModelsrvInstance("24001")
		Expect(err).To(Succeed())
		Expect(aliceSrv).ToNot(BeNil())

		Eventually(func() error {
			err := aliceSrv.client.GetTest()
			return err
		}, "10s", "500ms").Should(Succeed())

		Expect(aliceSrv.client.GetTest()).To(Succeed())

		bobSrv, err = newModelsrvInstance("24002")
		Expect(err).To(Succeed())
		Expect(bobSrv).ToNot(BeNil())

		Eventually(func() error {
			err := bobSrv.client.GetTest()
			return err
		}, "10s", "500ms").Should(Succeed())

	})

	AfterEach(func() {
	})

	It("should transfer changes to resources on one instance to the other", func() {

		By("registering instance bob as a subscriber to instance alice")
		err := aliceSrv.client.Register(bobSrv.url)
		Expect(err).To(Succeed())

		Eventually(func() []events.Subscriber {
			subs := aliceSrv.eventMgr.GetSubscribers()
			return subs
		}, "10s", "500ms").ShouldNot(BeEmpty())

		By("expecting to see bob in alice's list of subscribers")
		Eventually(func() []events.Subscriber {
			fmt.Printf("Seeing subscribers: %#v\n", aliceSrv.eventMgr.GetSubscribers())
			return aliceSrv.eventMgr.GetSubscribers()
		}, "10s", "500ms").Should(ContainElement(matchers.MatchSubscriberUrl("http://localhost:24002/api")))

		By("adding a new Context to instance alice")
		contextId := uuid.New()
		ctx := model.NewContext(aliceSrv.backend, contextId)
		ctx.SetDisplayName("Test Context")

		err = aliceSrv.backend.AddContext(ctx)
		Expect(err).To(Succeed())

		By("expecting to see the new Context in instance bob")
		Eventually(func() error {
			_, err := bobSrv.client.GetContextById(contextId)
			return err
		}, "10s", "500ms").Should(Succeed())
	})
})
