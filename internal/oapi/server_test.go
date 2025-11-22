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

package oapi

import (
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
	"gitlab.com/emeland/modelsrv/pkg/events"
	"gitlab.com/emeland/modelsrv/pkg/model"
)

var ctx context.Context
var cancel context.CancelFunc
var backend model.Model
var eventMgr events.EventManager

var apiInstanceId uuid.UUID = uuid.New()
var apiId uuid.UUID = uuid.New()
var componentInstanceId uuid.UUID = uuid.New()
var componentId uuid.UUID = uuid.New()
var findingId uuid.UUID = uuid.New()
var systemId uuid.UUID = uuid.New()
var systemInstanceId uuid.UUID = uuid.New()

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "OAPI Suite")
}

var _ = BeforeSuite(func() {
	var err error
	//logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	backend, err = model.NewModel()
	Expect(err).NotTo(HaveOccurred())

	api := model.API{
		ApiId:       apiId,
		DisplayName: "First API",
		Version: model.Version{
			Version:        "1.0.0",
			AvailableFrom:  mustParseDate("2023-01-01"),
			DeprecatedFrom: mustParseDate("2024-01-01"),
			TerminatedFrom: mustParseDate("2025-01-01"),
		},
		Annotations: map[string]string{},
	}
	err = backend.AddApi(&api, api.DisplayName, nil)
	Expect(err).NotTo(HaveOccurred())

	apiInstance := model.APIInstance{
		InstanceId:  apiInstanceId,
		DisplayName: "First API Instance",
		ApiRef: &model.ApiRef{
			ApiID: uuid.New(),
		},
		Annotations: map[string]string{},
	}
	err = backend.AddApiInstance(&apiInstance, apiInstance.DisplayName, nil)
	Expect(err).NotTo(HaveOccurred())

	component := model.Component{
		ComponentId: componentId,
		DisplayName: "First Component",
		Version: model.Version{
			Version:        "1.0.0",
			AvailableFrom:  mustParseDate("2023-01-01"),
			DeprecatedFrom: mustParseDate("2024-01-01"),
			TerminatedFrom: mustParseDate("2025-01-01"),
		},
		Annotations: map[string]string{},
	}
	err = backend.AddComponent(&component, component.DisplayName, nil)
	Expect(err).NotTo(HaveOccurred())

	componentInstance := model.ComponentInstance{
		InstanceId:  componentInstanceId,
		DisplayName: "First Component Instance",
		ComponentRef: &model.ComponentRef{
			ComponentId: componentId,
		},
		Annotations: map[string]string{},
	}
	err = backend.AddComponentInstance(&componentInstance, componentInstance.DisplayName, nil)
	Expect(err).NotTo(HaveOccurred())

	system := model.System{
		SystemId:    systemId,
		DisplayName: "First System",
		Version: model.Version{
			Version:        "1.0.0",
			AvailableFrom:  mustParseDate("2023-01-01"),
			DeprecatedFrom: mustParseDate("2024-01-01"),
			TerminatedFrom: mustParseDate("2025-01-01"),
		},
		Annotations: map[string]string{},
	}
	err = backend.AddSystem(&system, system.DisplayName, nil)
	Expect(err).NotTo(HaveOccurred())

	systemInstance := model.SystemInstance{
		InstanceId:  systemInstanceId,
		DisplayName: "First System Instance",
		SystemRef: &model.SystemRef{
			SystemId: systemId,
		},
		Annotations: map[string]string{},
	}
	err = backend.AddSystemInstance(&systemInstance, systemInstance.DisplayName, nil)
	Expect(err).NotTo(HaveOccurred())

	finding := model.Finding{
		FindingId:   findingId,
		Summary:     "First Finding",
		Description: "This is the first test finding.",
		Resources: []*model.ResourceRef{
			&model.ResourceRef{
				ResourceType: model.ParseResourceType("API"),
				ResourceId:   apiId,
			},
			&model.ResourceRef{
				ResourceType: model.ParseResourceType("Component"),
				ResourceId:   componentId,
			},
		},
		Annotations: map[string]string{},
	}
	err = backend.AddFinding(&finding, finding.Summary)
	Expect(err).NotTo(HaveOccurred())

	eventMgr, err = events.NewEventManager()
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
		server := NewApiServer(backend, eventMgr, "http://localhost")
		strict := NewApiHandler(server)

		r := mux.NewRouter()

		// get an `http.Handler` that we can use
		handler = HandlerFromMuxWithBaseURL(strict, r, "")
	})

	AfterEach(func() {
	})

	ctx := context.Background()

	It("should call GET on /events/query/{sequenceId} for sequenceId 0", func() {
		eventMgr.IncrementSequenceId(ctx) // make sure sequenceId 0 does not exist

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

		var instanceArr InstanceList
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

		var instance ApiInstance
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

		var instanceArr InstanceList
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

		var instance API
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

		var instanceArr InstanceList
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

		var instance ComponentInstance
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

		var instanceArr InstanceList
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

		var instance Component
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

		var instanceArr InstanceList
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

		var instance SystemInstance
		err = json.Unmarshal(body, &instance)
		Expect(err).NotTo(HaveOccurred())
		Expect(instance.SystemInstanceId).To(Equal(systemInstanceId))
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

		var instanceArr InstanceList
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

		var instance System
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

		var findingArr InstanceList
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

		var finding Finding
		err = json.Unmarshal(body, &finding)
		Expect(err).NotTo(HaveOccurred())
		Expect(finding.FindingId).To(Equal(findingId))

		Expect(len(finding.Resources)).To(Equal(2))
	})

	/*


		It("PostEventsRegister should not panic or error", func() {
			Expect(func() {
				_, err := a.PostEventsRegister(ctx, PostEventsRegisterRequestObject{})
				Expect(err).To(BeNil())
			}).NotTo(Panic())
		})
	*/
})
