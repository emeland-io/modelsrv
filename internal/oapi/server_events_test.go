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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ievents "go.emeland.io/modelsrv/internal/events"
	"go.emeland.io/modelsrv/internal/oapi"
)

var _ = Describe("calling the event handling functions", func() {
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

	It("should register a consumer using POST /landscape/register", func() {
		url := "http://localhost/events/register"
		callbackUrl := "http://localhost:8080/callback"

		subscriber := ievents.NewSubscriber(callbackUrl)
		Expect(subscriber.GetURL()).To(Equal(callbackUrl))

		postBody := `{"callbackUrl": "` + callbackUrl + `", "eventTypes": ["FindingResource"]}`
		req := httptest.NewRequest("POST", url, io.NopCloser(strings.NewReader(postBody)))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusCreated))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(body)).To(Equal(0))

		Expect(len(eventMgr.GetSubscribers())).NotTo(Equal(0))
		Expect(eventMgr.GetSubscribers()[0]).To(Equal(subscriber))
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
		initialSubscribers := eventMgr.GetSubscribers()
		initialSubscriberCount := len(initialSubscribers)
		eventMgr.AddSubscriber("http://remote-server.example.com/emeland/")

		postData = []byte(`{"callbackUrl":"http://remote-server.example.com/emeland/"}`)
		req = httptest.NewRequest("POST", url, bytes.NewBuffer(postData))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp = w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		Expect(len(eventMgr.GetSubscribers())).To(Equal(initialSubscriberCount))

		defer resp.Body.Close()
	})

})
