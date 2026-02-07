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

package oapi_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/internal/oapi"
)

var _ = Describe("calling the modelsrv API functions for phase 0", func() {
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

	It("should call GET on /landscape/contexts", func() {
		url := "http://localhost/landscape/contexts"
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
		Expect(len(instanceArr)).To(Equal(2))

		Expect(*(instanceArr[0].InstanceId)).To(Equal(contextId))
		Expect(*(instanceArr[0].Reference)).To(Equal(fmt.Sprintf("http://localhost/landscape/contexts/%s", contextId.String())))

	})

	It("should call GET on /landscape/contexts/{contextId}", func() {
		url := fmt.Sprintf("http://localhost/landscape/contexts/%s", contextId.String())
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var context oapi.Context
		err = json.Unmarshal(body, &context)
		Expect(err).NotTo(HaveOccurred())
		Expect(context.ContextId).To(Equal(contextId))
		Expect(context.DisplayName).To(Equal("the real test context"))
		Expect(*(context.Parent)).To(Equal(parentContextId))
		// Expect(context.Type).To(Equal(contextTypeId))
	})

	It("should call GET on /landscape/contextTypes", func() {
		url := "http://localhost/landscape/contextTypes"
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

		Expect(*(instanceArr[0].InstanceId)).To(Equal(contextTypeId))
		Expect(*(instanceArr[0].Reference)).To(Equal(fmt.Sprintf("http://localhost/landscape/contextTypes/%s", contextTypeId.String())))

	})

	It("should call GET on /landscape/contextTypes/{contextTypeId}", func() {
		url := fmt.Sprintf("http://localhost/landscape/contextTypes/%s", contextTypeId.String())
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var contextType oapi.ContextType
		err = json.Unmarshal(body, &contextType)
		Expect(err).NotTo(HaveOccurred())
		Expect(contextType.ContextTypeId).To(Equal(contextTypeId))
	})

	It("should call GET on /landscape/nodes", func() {
		url := "http://localhost/landscape/nodes"
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

		Expect(*(instanceArr[0].InstanceId)).To(Equal(nodeId))
		Expect(*(instanceArr[0].Reference)).To(Equal(fmt.Sprintf("http://localhost/landscape/nodes/%s", nodeId.String())))

	})

	It("should call GET on /landscape/nodes/{nodeId}", func() {
		url := fmt.Sprintf("http://localhost/landscape/nodes/%s", nodeId.String())
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).NotTo(Equal(0))

		var node oapi.Node
		err = json.Unmarshal(body, &node)
		Expect(err).NotTo(HaveOccurred())
		Expect(node.NodeId).To(Equal(nodeId))
	})

})
