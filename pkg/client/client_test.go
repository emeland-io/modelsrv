package client_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	eventmgr "go.emeland.io/modelsrv/internal/events"
	"go.emeland.io/modelsrv/pkg/client"
	"go.emeland.io/modelsrv/pkg/endpoint"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlcommon "go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/component"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/system"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting modelsrv suite\n")
	RunSpecs(t, "Client Suite")
}

var testModel model.Model
var testEvents events.EventManager
var testClient *client.ModelSrvClient

var (
	systemId            = uuid.New()
	systemInstanceId    = uuid.New()
	componentId         = uuid.New()
	componentInstanceId = uuid.New()
	contextId           = uuid.New()
	apiId               = uuid.New()
	apiInstanceId       = uuid.New()
)

var _ = Describe("Client", Ordered, func() {
	BeforeAll(func() {
		var err error
		By("starting a model server")

		testEvents, err = eventmgr.NewEventManager()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(testEvents).NotTo(BeNil())

		sink, err := testEvents.GetSink()
		Expect(err).ShouldNot(HaveOccurred())

		testModel, err = model.NewModel(sink)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(testModel).NotTo(BeNil())

		By("attaching the model to a listener")
		Expect(endpoint.StarWebListener(testModel, testEvents, "localhost:24000")).To(Succeed())

		By("creating a client")
		testClient, err = client.NewModelSrvClient("http://localhost:24000/api/")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(testClient).NotTo(BeNil())

		By("loading test data into the model")
		err = loadModel(testModel)
		Expect(err).ShouldNot(HaveOccurred())

		By("waiting for the server to be ready")
		Eventually(func() error {
			err := testClient.GetTest()
			return err
		}, "10s", "500ms").Should(Succeed())
	})

	AfterAll(func() {
		By("stopping the listener")
		endpoint.StopWebListener()
	})

	Describe("Test client functions for Contexts", func() {
		It("return a list of Contexts", func() {
			instanceList, err := testClient.GetContexts()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(instanceList).NotTo(BeNil())

			Expect(len(*instanceList)).To(BeNumerically(">", 0))
		})
		It("return a Context by ID", func() {
			// first try with an invalid ID
			context, err := testClient.GetContextById(uuid.New(), testModel)
			Expect(err).Should(Equal(mdlcommon.ErrContextNotFound))
			Expect(context).To(BeNil())

			// now try with a valid ID
			context, err = testClient.GetContextById(contextId, testModel)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(context).NotTo(BeNil())

			Expect(context.GetContextId()).To(Equal(contextId))
			Expect(context.GetDisplayName()).To(Equal("Test Context"))
		})
	})

	Describe("Test client functions for Systems", func() {
		It("return a list of systems", func() {
			instanceList, err := testClient.GetSystems()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(instanceList).NotTo(BeNil())

			Expect(len(*instanceList)).To(BeNumerically(">", 0))
		})
		It("return a System by ID", func() {
			// first try with an invalid ID
			system, err := testClient.GetSystemById(uuid.New())
			Expect(err).Should(Equal(common.ErrSystemNotFound))
			Expect(system).To(BeNil())

			// now try with a valid ID
			system, err = testClient.GetSystemById(systemId)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(system).NotTo(BeNil())

			Expect(*system.SystemId).To(Equal(systemId))
			Expect(system.DisplayName).To(Equal("Test System"))
		})
	})

	Describe("Test client functions for SystemsInstances", func() {
		It("return a list of SystemInstances", func() {
			instanceList, err := testClient.GetSystemInstances()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(instanceList).NotTo(BeNil())

			Expect(len(*instanceList)).To(BeNumerically(">", 0))
		})
		It("return a SystemInstance by ID", func() {
			// first try with an invalid ID
			systemInstance, err := testClient.GetSystemInstanceById(uuid.New())
			Expect(err).Should(Equal(common.ErrSystemInstanceNotFound))
			Expect(systemInstance).To(BeNil())

			systemInstance, err = testClient.GetSystemInstanceById(systemInstanceId)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(systemInstance).NotTo(BeNil())

			Expect(systemInstance.SystemInstanceId).To(Equal(systemInstanceId))
			Expect(systemInstance.DisplayName).To(Equal("Test SystemInstances"))
		})
	})

	Describe("Test client functions for APIs", func() {
		It("return a list of APIs", func() {
			instanceList, err := testClient.GetAPIs()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(instanceList).NotTo(BeNil())

			Expect(len(*instanceList)).To(BeNumerically(">", 0))
		})
		It("return a API by ID", func() {
			// first try with an invalid ID
			api, err := testClient.GetAPIById(uuid.New())
			Expect(err).Should(Equal(common.ErrApiNotFound))
			Expect(api).To(BeNil())

			api, err = testClient.GetAPIById(apiId)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(api).NotTo(BeNil())

			Expect(*api.ApiId).To(Equal(apiId))
			Expect(api.DisplayName).To(Equal("Test API"))
		})
	})

	Describe("Test client functions for ApiInstances", func() {
		It("return a list of APIs", func() {
			instanceList, err := testClient.GetApiInstances()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(instanceList).NotTo(BeNil())

			Expect(len(*instanceList)).To(BeNumerically(">", 0))
		})
		It("return a ApiInstance by ID", func() {
			// first try with an invalid ID
			apiInstance, err := testClient.GetApiInstanceById(uuid.New())
			Expect(err).Should(Equal(common.ErrApiInstanceNotFound))
			Expect(apiInstance).To(BeNil())

			apiInstance, err = testClient.GetApiInstanceById(apiInstanceId)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(apiInstance).NotTo(BeNil())

			Expect(apiInstance.ApiInstanceId).To(Equal(apiInstanceId))
			Expect(apiInstance.DisplayName).To(Equal("Test ApiInstance"))
		})
	})

	Describe("Test client functions for Components", func() {
		It("return a list of Components", func() {
			instanceList, err := testClient.GetComponents()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(instanceList).NotTo(BeNil())

			Expect(len(*instanceList)).To(BeNumerically(">", 0))
		})
		It("return a Component by ID", func() {
			// first try with an invalid ID
			component, err := testClient.GetComponentById(uuid.New())
			Expect(err).Should(Equal(common.ErrComponentNotFound))
			Expect(component).To(BeNil())

			component, err = testClient.GetComponentById(componentId)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(component).NotTo(BeNil())

			Expect(*component.ComponentId).To(Equal(componentId))
			Expect(component.DisplayName).To(Equal("Test Component"))
		})
	})

	Describe("Test client functions for ComponentInstances", func() {
		It("return a list of ComponentInstances", func() {
			instanceList, err := testClient.GetComponentInstances()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(instanceList).NotTo(BeNil())

			Expect(len(*instanceList)).To(BeNumerically(">", 0))
		})
		It("return a Component Instance by ID", func() {
			// first try with an invalid ID
			componentInstance, err := testClient.GetComponentInstanceById(uuid.New())
			Expect(err).Should(Equal(common.ErrComponentInstanceNotFound))
			Expect(componentInstance).To(BeNil())

			componentInstance, err = testClient.GetComponentInstanceById(componentInstanceId)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(componentInstance).NotTo(BeNil())

			Expect(componentInstance.ComponentInstanceId).To(Equal(componentInstanceId))
			Expect(componentInstance.DisplayName).To(Equal("Test ComponentInstance"))
		})
	})
})

func loadModel(target model.Model) error {
	// create simple System with single Component and API

	c := mdlctx.NewContext(target.GetSink(), contextId)
	c.SetDisplayName("Test Context")

	err := target.AddContext(c)
	if err != nil {
		return err
	}

	sys := system.NewSystem(target.GetSink(), systemId)
	sys.SetDisplayName("Test System")
	sys.SetVersion(common.Version{})
	err = target.AddSystem(sys)
	if err != nil {
		return err
	}

	comp := component.NewComponent(target.GetSink(), componentId)
	comp.SetDisplayName("Test Component")
	comp.SetSystem(&system.SystemRef{
		System: sys,
	})
	err = target.AddComponent(comp)
	if err != nil {
		return err
	}

	componentInstance := component.NewComponentInstance(target.GetSink(), componentInstanceId)
	componentInstance.SetDisplayName("Test ComponentInstance")
	componentInstance.SetComponentRef(&component.ComponentRef{
		Component: comp,
	})
	err = target.AddComponentInstance(componentInstance)
	if err != nil {
		return err
	}

	a := mdlapi.NewAPI(target.GetSink(), apiId)
	a.SetDisplayName("Test API")
	a.SetSystem(&system.SystemRef{
		System: sys,
	})
	err = target.AddApi(a)
	if err != nil {
		return err
	}

	apiInstance := mdlapi.NewApiInstance(target.GetSink(), apiInstanceId)
	apiInstance.SetDisplayName("Test ApiInstance")
	apiInstance.SetApiRefByRef(a)
	err = target.AddApiInstance(apiInstance)
	if err != nil {
		return err
	}

	// TODO: set context in which this instance exists
	systemInstance := system.NewSystemInstance(target.GetSink(), systemInstanceId)
	systemInstance.SetDisplayName("Test SystemInstances")
	systemInstance.SetSystemRef(&system.SystemRef{
		System: sys,
	})
	err = target.AddSystemInstance(systemInstance)
	if err != nil {
		return err
	}
	return nil
}
