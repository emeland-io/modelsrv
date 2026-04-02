/*
Copyright 2025 Lutz Behnke <lutz.behnke@gmx.de>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
//nolint:errcheck
package oapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	eventmgr "go.emeland.io/modelsrv/internal/events"
	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/component"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.emeland.io/modelsrv/pkg/model/node"
	"go.emeland.io/modelsrv/pkg/model/system"
)

var ctx context.Context
var cancel context.CancelFunc
var backend model.Model
var eventMgr events.EventManager

// Phase 0 test data
var contextId uuid.UUID = uuid.New()
var parentContextId uuid.UUID = uuid.New()
var contextTypeId uuid.UUID = uuid.New()
var nodeId uuid.UUID = uuid.New()
var nodeTypeId uuid.UUID = uuid.New()

// Phase 1 test data
var apiInstanceId uuid.UUID = uuid.New()
var apiId uuid.UUID = uuid.New()
var componentInstanceId uuid.UUID = uuid.New()
var componentId uuid.UUID = uuid.New()
var systemId uuid.UUID = uuid.New()
var systemInstanceId uuid.UUID = uuid.New()

// Phase 2 test data

// Phase 3 test data

// Phase 4 test data
var findingId uuid.UUID = uuid.New()
var findingTypeId uuid.UUID = uuid.New()

// Phase 5 test data

// Phase 6 test data

// Phase 7 test data

// Phase 8 test data

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "OAPI Suite")
}

var _ = BeforeSuite(func() {
	var err error
	//logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	sink := events.NewListSink()
	backend, err = model.NewModel(sink)
	Expect(err).NotTo(HaveOccurred())

	contextType := mdlctx.NewContextType(backend.GetSink(), contextTypeId)
	contextType.SetDisplayName("Test Context Type")
	contextType.SetDescription("A test context type for testing purposes")
	err = backend.AddContextType(contextType)
	Expect(err).NotTo(HaveOccurred())

	testContext := mdlctx.NewContext(backend.GetSink(), contextId)
	testContext.SetParentById(parentContextId)
	testContext.SetDisplayName("the real test context")
	// TODO: not implemented yet
	// testContext.SetTypeById(contextTypeId)
	err = backend.AddContext(testContext)
	Expect(err).NotTo(HaveOccurred())

	parentContext := mdlctx.NewContext(backend.GetSink(), parentContextId)
	err = backend.AddContext(parentContext)
	Expect(err).NotTo(HaveOccurred())

	testNode := node.NewNode(backend.GetSink(), nodeId)
	err = backend.AddNode(testNode)
	Expect(err).NotTo(HaveOccurred())

	nodeType := node.NewNodeType(backend.GetSink(), nodeTypeId)
	nodeType.SetDisplayName("Test Node Type")
	nodeType.SetDescription("A test node type for testing purposes")
	err = backend.AddNodeType(nodeType)
	Expect(err).NotTo(HaveOccurred())

	api := mdlapi.NewAPI(backend.GetSink(), apiId)
	api.SetDisplayName("First API")
	api.SetVersion(common.Version{
		Version:        "1.0.0",
		AvailableFrom:  mustParseDate("2023-01-01"),
		DeprecatedFrom: mustParseDate("2024-01-01"),
		TerminatedFrom: mustParseDate("2025-01-01"),
	})
	err = backend.AddApi(api)
	Expect(err).NotTo(HaveOccurred())

	apiInstance := mdlapi.NewApiInstance(backend.GetSink(), apiInstanceId)
	apiInstance.SetDisplayName("First API Instance")
	apiInstance.SetApiRef(backend.ApiRefByID(apiId))
	err = backend.AddApiInstance(apiInstance)
	Expect(err).NotTo(HaveOccurred())

	comp := component.NewComponent(backend.GetSink(), componentId)
	comp.SetDisplayName("First Component")
	comp.SetVersion(common.Version{
		Version:        "1.0.0",
		AvailableFrom:  mustParseDate("2023-01-01"),
		DeprecatedFrom: mustParseDate("2024-01-01"),
		TerminatedFrom: mustParseDate("2025-01-01"),
	})
	err = backend.AddComponent(comp)
	Expect(err).NotTo(HaveOccurred())

	componentInstance := component.NewComponentInstance(backend.GetSink(), componentInstanceId)
	componentInstance.SetDisplayName("First Component Instance")
	componentInstance.SetComponentRef(&component.ComponentRef{
		ComponentId: componentId,
	})
	err = backend.AddComponentInstance(componentInstance)
	Expect(err).NotTo(HaveOccurred())

	firstSystem := model.MakeTestSystem(
		backend.GetSink(),
		systemId,
		"First System",
		common.Version{
			Version:        "1.0.0",
			AvailableFrom:  mustParseDate("2023-01-01"),
			DeprecatedFrom: mustParseDate("2024-01-01"),
			TerminatedFrom: mustParseDate("2025-01-01"),
		},
	)

	err = backend.AddSystem(firstSystem)
	Expect(err).NotTo(HaveOccurred())

	systemInstance := system.NewSystemInstance(backend.GetSink(), systemInstanceId)
	systemInstance.SetDisplayName("First System Instance")
	systemInstance.SetSystemRef(&system.SystemRef{
		SystemId: systemId,
	})
	systemInstance.SetContextRef(&mdlctx.ContextRef{
		ContextId: contextId,
	})
	err = backend.AddSystemInstance(systemInstance)
	Expect(err).NotTo(HaveOccurred())

	fd := finding.NewFinding(backend.GetSink(), findingId)
	fd.SetSummary("First Finding")
	fd.SetDescription("This is the first test finding.")
	fd.SetResources([]*common.ResourceRef{
		{
			ResourceType: events.ParseResourceType("API"),
			ResourceId:   apiId,
		},
		{
			ResourceType: events.ParseResourceType("Component"),
			ResourceId:   componentId,
		},
	})
	err = backend.AddFinding(fd, fd.GetSummary())
	Expect(err).NotTo(HaveOccurred())

	findingType := finding.NewFindingType(backend.GetSink(), findingTypeId)
	findingType.SetDisplayName("Test Finding Type")
	findingType.SetDescription("A test finding type for testing purposes")
	err = backend.AddFindingType(findingType)
	Expect(err).NotTo(HaveOccurred())

	eventMgr, err = eventmgr.NewEventManager()
	Expect(err).NotTo(HaveOccurred())

	By("bootstrapping test environment")
})

func mustParseDate(dateStr string) *time.Time {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		panic(fmt.Sprintf("failed to parse date %s: %v", dateStr, err))
	}
	return &t
}

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
})

var _ = Describe("calling the modelsrv API functions", func() {
	var handler http.Handler

	BeforeEach(func() {
		By("setting up http listener")
		server := oapi.NewApiServer(backend, eventMgr, "http://localhost")
		strict := oapi.NewApiHandler(server)

		r := mux.NewRouter()

		// get an `http.Handler` that we can use
		handler = oapi.HandlerFromMuxWithBaseURL(strict, r, "")
	})

	AfterEach(func() {
	})

	ctx := context.Background()

	It("should call GET on /events/query/{sequenceId} for sequenceId 0", func() {
		err := eventMgr.IncrementSequenceId(ctx) // make sure sequenceId 0 does not exist
		Expect(err).NotTo(HaveOccurred())

		url := fmt.Sprintf("http://localhost/events/query/%d", 0)
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()

		//ctx := context.Background()
		Expect(resp.StatusCode).To(Equal(http.StatusPermanentRedirect))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).To(Equal(0))
		// TODO: check reference to location returned in header

	})

	It("should call GET on /events/query/{sequenceId} for valid sequenceId", func() {
		sequenceId, err := eventMgr.GetCurrentSequenceId(ctx)
		Expect(err).NotTo(HaveOccurred())

		url := fmt.Sprintf("http://localhost/events/query/%d", sequenceId)
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()

		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).To(Equal(0))

	})

	It("should call POST on /events/register to add a subscriber", func() {
		url := "http://localhost/events/register"

		postData := []byte(`{"callbackUrl":"http://remote-server.example.com/emeland/"}`)
		req := httptest.NewRequest("POST", url, bytes.NewBuffer(postData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusCreated))

		Expect(len(eventMgr.GetSubscribers())).To(Equal(1))
		Expect(eventMgr.GetSubscribers()[0].GetURL()).To(Equal("http://remote-server.example.com/emeland/"))
	})

	It("should call POST on /events/unregister to remove a subscriber", func() {
		url := "http://localhost/events/unregister"

		// first try to remove a non-existing subscriber
		postData := []byte(`{"callbackUrl":"http://invalid-server.example.com/emeland/"}`)
		req := httptest.NewRequest("POST", url, bytes.NewBuffer(postData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()

		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		resp.Body.Close()

		// now remove the existing subscriber
		err := eventMgr.AddSubscriber("http://remote-server.example.com/emeland/")
		Expect(err).NotTo(HaveOccurred())

		postData = []byte(`{"callbackUrl":"http://remote-server.example.com/emeland/"}`)
		req = httptest.NewRequest("POST", url, bytes.NewBuffer(postData))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp = w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		Expect(len(eventMgr.GetSubscribers())).To(Equal(0))

		defer resp.Body.Close()
	})

	It("should call GET on /landscape/api-instances", func() {
		url := "http://localhost/landscape/api-instances"
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var instanceArr oapi.InstanceList
		err = json.Unmarshal(body, &instanceArr)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(instanceArr)).To(Equal(1))

		Expect(*(instanceArr[0].InstanceId)).To(Equal(apiInstanceId))
		Expect(*(instanceArr[0].Reference)).To(Equal(fmt.Sprintf("http://localhost/landscape/api-instances/%s", apiInstanceId.String())))
	})

	It("should call GET on /landscape/api-instances/{apiInstanceId}", func() {
		url := fmt.Sprintf("http://localhost/landscape/api-instances/%s", apiInstanceId.String())
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var instance oapi.ApiInstance
		err = json.Unmarshal(body, &instance)
		Expect(err).NotTo(HaveOccurred())
		Expect(instance.ApiInstanceId).To(Equal(apiInstanceId))
	})

	It("should call GET on /landscape/apis", func() {
		url := "http://localhost/landscape/apis"
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var instanceArr oapi.InstanceList
		err = json.Unmarshal(body, &instanceArr)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(instanceArr)).To(Equal(1))
		Expect(*(instanceArr[0].InstanceId)).To(Equal(apiId))

		Expect(instanceArr[0].Reference).NotTo(BeNil())
		Expect(*(instanceArr[0].Reference)).To(Equal(fmt.Sprintf("http://localhost/landscape/apis/%s", apiId.String())))
	})

	It("should call GET on /landscape/apis/{apiId}", func() {
		url := fmt.Sprintf("http://localhost/landscape/apis/%s", apiId.String())
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var instance oapi.API
		err = json.Unmarshal(body, &instance)
		Expect(err).NotTo(HaveOccurred())
		Expect(*(instance.ApiId)).To(Equal(apiId))
	})

	It("should call GET on /landscape/componentInstances", func() {
		url := "http://localhost/landscape/component-instances"
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var instanceArr oapi.InstanceList
		err = json.Unmarshal(body, &instanceArr)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(instanceArr)).To(Equal(1))
		Expect(*(instanceArr[0].InstanceId)).To(Equal(componentInstanceId))

		Expect(instanceArr[0].Reference).NotTo(BeNil())
		Expect(*(instanceArr[0].Reference)).To(Equal(fmt.Sprintf("http://localhost/landscape/component-instances/%s", componentInstanceId.String())))
	})

	It("should call GET on /landscape/component-instances/{componentInstanceId}", func() {
		url := fmt.Sprintf("http://localhost/landscape/component-instances/%s", componentInstanceId.String())
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var instance oapi.ComponentInstance
		err = json.Unmarshal(body, &instance)
		Expect(err).NotTo(HaveOccurred())
		Expect(instance.ComponentInstanceId).To(Equal(componentInstanceId))
	})

	It("should call GET on /landscape/components", func() {
		url := "http://localhost/landscape/components"
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var instanceArr oapi.InstanceList
		err = json.Unmarshal(body, &instanceArr)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(instanceArr)).To(Equal(1))
		Expect(*(instanceArr[0].InstanceId)).To(Equal(componentId))

		Expect(instanceArr[0].Reference).NotTo(BeNil())
		Expect(*(instanceArr[0].Reference)).To(Equal(fmt.Sprintf("http://localhost/landscape/components/%s", componentId.String())))
	})

	It("should call GET on /landscape/components/{componentId}", func() {
		url := fmt.Sprintf("http://localhost/landscape/components/%s", componentId.String())
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var instance oapi.Component
		err = json.Unmarshal(body, &instance)
		Expect(err).NotTo(HaveOccurred())
		Expect(*(instance.ComponentId)).To(Equal(componentId))
	})

	It("should call GET on /landscape/system-instances", func() {
		url := "http://localhost/landscape/system-instances"
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var instanceArr oapi.InstanceList
		err = json.Unmarshal(body, &instanceArr)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(instanceArr)).To(Equal(1))

		Expect(*(instanceArr[0].InstanceId)).To(Equal(systemInstanceId))
		Expect(*(instanceArr[0].Reference)).To(Equal(fmt.Sprintf("http://localhost/landscape/system-instances/%s", systemInstanceId.String())))
	})

	It("should call GET on /landscape/system-instances/{systemInstanceId}", func() {
		url := fmt.Sprintf("http://localhost/landscape/system-instances/%s", systemInstanceId.String())
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var instance oapi.SystemInstance
		err = json.Unmarshal(body, &instance)
		Expect(err).NotTo(HaveOccurred())
		Expect(instance.SystemInstanceId).To(Equal(systemInstanceId))
		Expect(*(instance.Context)).To(Equal(contextId))
	})

	It("should call GET on /landscape/systems", func() {
		url := "http://localhost/landscape/systems"
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var instanceArr oapi.InstanceList
		err = json.Unmarshal(body, &instanceArr)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(instanceArr)).To(Equal(1))

		Expect(*(instanceArr[0].InstanceId)).To(Equal(systemId))
		Expect(*(instanceArr[0].Reference)).To(Equal(fmt.Sprintf("http://localhost/landscape/systems/%s", systemId.String())))

	})

	It("should call GET on /landscape/systems/{systemId}", func() {
		url := fmt.Sprintf("http://localhost/landscape/systems/%s", systemId.String())
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var instance oapi.System
		err = json.Unmarshal(body, &instance)
		Expect(err).NotTo(HaveOccurred())
		Expect(*(instance.SystemId)).To(Equal(systemId))
	})

	It("should call GET on /landscape/findings", func() {
		url := "http://localhost/landscape/findings"
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var findingArr oapi.InstanceList
		err = json.Unmarshal(body, &findingArr)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(findingArr)).To(Equal(1))

		Expect(*(findingArr[0].InstanceId)).To(Equal(findingId))
		Expect(*(findingArr[0].Reference)).To(Equal(fmt.Sprintf("http://localhost/landscape/findings/%s", findingId.String())))

	})

	It("should call GET on /landscape/findings/{findingsId}", func() {
		url := fmt.Sprintf("http://localhost/landscape/findings/%s", findingId.String())
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var finding oapi.Finding
		err = json.Unmarshal(body, &finding)
		Expect(err).NotTo(HaveOccurred())
		Expect(finding.FindingId).To(Equal(findingId))

		Expect(len(finding.Resources)).To(Equal(2))
	})

	It("should call GET on /landscape/findingTypes", func() {
		url := "http://localhost/landscape/findingTypes"
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var findingArr oapi.InstanceList
		err = json.Unmarshal(body, &findingArr)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(findingArr)).To(Equal(1))

		Expect(*(findingArr[0].InstanceId)).To(Equal(findingTypeId))
		Expect(*(findingArr[0].Reference)).To(Equal(fmt.Sprintf("http://localhost/landscape/findingTypes/%s", findingTypeId.String())))

	})

	It("should call GET on /landscape/findingTypes/{findingTypeId}", func() {
		url := fmt.Sprintf("http://localhost/landscape/findingTypes/%s", findingTypeId.String())
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var findingType oapi.FindingType
		err = json.Unmarshal(body, &findingType)
		Expect(err).NotTo(HaveOccurred())
		Expect(*findingType.FindingTypeId).To(Equal(findingTypeId))

	})

})
