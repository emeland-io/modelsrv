package client_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gitlab.com/emeland/modelsrv/pkg/client"
	"gitlab.com/emeland/modelsrv/pkg/endpoint"
	"gitlab.com/emeland/modelsrv/pkg/events"
	"gitlab.com/emeland/modelsrv/pkg/model"
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
		testModel, err = model.NewModel()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(testModel).NotTo(BeNil())

		testEvents, err = events.NewEventManager()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(testEvents).NotTo(BeNil())

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
			context, err := testClient.GetContextById(uuid.New())
			Expect(err).Should(Equal(model.ContextNotFoundError))
			Expect(context).To(BeNil())

			// now try with a valid ID
			context, err = testClient.GetContextById(contextId)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(context).NotTo(BeNil())

			Expect(context.ContextId).To(Equal(contextId))
			Expect(context.DisplayName).To(Equal("Test Context"))
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
			Expect(err).Should(Equal(model.SystemNotFoundError))
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
			Expect(err).Should(Equal(model.SystemInstanceNotFoundError))
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
			Expect(err).Should(Equal(model.ApiNotFoundError))
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
			Expect(err).Should(Equal(model.ApiInstanceNotFoundError))
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
			Expect(err).Should(Equal(model.ComponentNotFoundError))
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
			Expect(err).Should(Equal(model.ComponentInstanceNotFoundError))
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

	context := &model.Context{
		DisplayName: "Test Context",
		ContextId:   contextId,
	}
	err := target.AddContext(context)
	if err != nil {
		return err
	}

	system := &model.System{
		DisplayName: "Test System",
		SystemId:    systemId,
	}
	err = target.AddSystem(system)
	if err != nil {
		return err
	}

	component := &model.Component{
		DisplayName: "Test Component",
		ComponentId: componentId,
		System: &model.SystemRef{
			System: system,
		},
	}
	err = target.AddComponent(component)
	if err != nil {
		return err
	}

	componentInstance := &model.ComponentInstance{
		DisplayName: "Test ComponentInstance",
		InstanceId:  componentInstanceId,
		ComponentRef: &model.ComponentRef{
			Component: component,
		},
	}
	err = target.AddComponentInstance(componentInstance)
	if err != nil {
		return err
	}

	api := &model.API{
		DisplayName: "Test API",
		ApiId:       apiId,
		System: &model.SystemRef{
			System: system,
		},
	}
	err = target.AddApi(api)
	if err != nil {
		return err
	}

	apiInstance := &model.APIInstance{
		DisplayName: "Test ApiInstance",
		InstanceId:  apiInstanceId,
		ApiRef: &model.ApiRef{
			API: api,
		},
	}
	err = target.AddApiInstance(apiInstance)
	if err != nil {
		return err
	}

	err = target.AddApi(api)
	if err != nil {
		return err
	}

	// TODO: set context in which this instance exists
	systemInstance := &model.SystemInstance{
		DisplayName: "Test SystemInstances",
		InstanceId:  systemInstanceId,
		SystemRef: &model.SystemRef{
			System: system,
		},
	}
	err = target.AddSystemInstance(systemInstance)
	if err != nil {
		return err
	}
	return nil
}
