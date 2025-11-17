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

	eventMgr, err = events.NewEventManager()
	Expect(err).NotTo(HaveOccurred())

	By("bootstrapping test environment")
})

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
		url := "http://localhost/landscape/api"
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

		Expect(*(instanceArr[0].Reference)).To(Equal(fmt.Sprintf("http://localhost/landscape/api/%s", apiId.String())))
	})

	/*
		It("GetLandscapeApisApiId should not panic or error", func() {
			Expect(func() {
				_, err := a.GetLandscapeApisApiId(ctx, GetLandscapeApisApiIdRequestObject{})
				Expect(err).To(BeNil())
			}).NotTo(Panic())
		})

		It("GetLandscapeComponentInstances should not panic or error", func() {
			Expect(func() {
				_, err := a.GetLandscapeComponentInstances(ctx, GetLandscapeComponentInstancesRequestObject{})
				Expect(err).To(BeNil())
			}).NotTo(Panic())
		})

		It("GetLandscapeComponentInstancesComponentInstanceId should not panic or error", func() {
			Expect(func() {
				_, err := a.GetLandscapeComponentInstancesComponentInstanceId(ctx, GetLandscapeComponentInstancesComponentInstanceIdRequestObject{})
				Expect(err).To(BeNil())
			}).NotTo(Panic())
		})

		It("GetLandscapeComponents should not panic or error", func() {
			Expect(func() {
				_, err := a.GetLandscapeComponents(ctx, GetLandscapeComponentsRequestObject{})
				Expect(err).To(BeNil())
			}).NotTo(Panic())
		})

		It("GetLandscapeComponentsComponentId should not panic or error", func() {
			Expect(func() {
				_, err := a.GetLandscapeComponentsComponentId(ctx, GetLandscapeComponentsComponentIdRequestObject{})
				Expect(err).To(BeNil())
			}).NotTo(Panic())
		})

		It("GetLandscapeFindings should not panic or error", func() {
			Expect(func() {
				_, err := a.GetLandscapeFindings(ctx, GetLandscapeFindingsRequestObject{})
				Expect(err).To(BeNil())
			}).NotTo(Panic())
		})

		It("GetLandscapeFindingsFindingId should not panic or error", func() {
			Expect(func() {
				_, err := a.GetLandscapeFindingsFindingId(ctx, GetLandscapeFindingsFindingIdRequestObject{})
				Expect(err).To(BeNil())
			}).NotTo(Panic())
		})

		It("GetLandscapeSystemInstances should not panic or error", func() {
			Expect(func() {
				_, err := a.GetLandscapeSystemInstances(ctx, GetLandscapeSystemInstancesRequestObject{})
				Expect(err).To(BeNil())
			}).NotTo(Panic())
		})

		It("GetLandscapeSystemInstancesSystemInstanceId should not panic or error", func() {
			Expect(func() {
				_, err := a.GetLandscapeSystemInstancesSystemInstanceId(ctx, GetLandscapeSystemInstancesSystemInstanceIdRequestObject{})
				Expect(err).To(BeNil())
			}).NotTo(Panic())
		})

		It("GetLandscapeSystems should not panic or error", func() {
			Expect(func() {
				_, err := a.GetLandscapeSystems(ctx, GetLandscapeSystemsRequestObject{})
				Expect(err).To(BeNil())
			}).NotTo(Panic())
		})

		It("GetLandscapeSystemsSystemId should not panic or error", func() {
			Expect(func() {
				_, err := a.GetLandscapeSystemsSystemId(ctx, GetLandscapeSystemsSystemIdRequestObject{})
				Expect(err).To(BeNil())
			}).NotTo(Panic())
		})

		It("GetTest should not panic or error", func() {
			Expect(func() {
				_, err := a.GetTest(ctx, GetTestRequestObject{})
				Expect(err).To(BeNil())
			}).NotTo(Panic())
		})

		It("PostEventsRegister should not panic or error", func() {
			Expect(func() {
				_, err := a.PostEventsRegister(ctx, PostEventsRegisterRequestObject{})
				Expect(err).To(BeNil())
			}).NotTo(Panic())
		})
	*/
})
