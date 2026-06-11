//nolint:errcheck
package oapi_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	eventmgr "go.emeland.io/modelsrv/internal/events"
	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/authz"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/system"
)

var _ = Describe("ownership visibility on the API", func() {
	var (
		handler http.Handler

		ownedSystemId   = uuid.New()
		groupSystemId   = uuid.New()
		foreignSystemId = uuid.New()
		unownedSystemId = uuid.New()
		contextTypeId2  = uuid.New()
	)

	BeforeEach(func() {
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		addSystem := func(id uuid.UUID, name, ownerIdentities, ownerGroups string) {
			s := system.NewSystem(id)
			s.SetDisplayName(name)
			if ownerIdentities != "" {
				s.GetAnnotations().Add(authz.OwnerIdentitiesKey, ownerIdentities)
			}
			if ownerGroups != "" {
				s.GetAnnotations().Add(authz.OwnerGroupsKey, ownerGroups)
			}
			Expect(m.AddSystem(s)).To(Succeed())
		}

		addSystem(ownedSystemId, "owned by user-1", "user-1", "")
		addSystem(groupSystemId, "owned by team-a", "", "team-a")
		addSystem(foreignSystemId, "owned by someone else", "user-2", "team-b")
		addSystem(unownedSystemId, "unowned", "", "")

		ct := mdlctx.NewContextType(contextTypeId2)
		ct.SetDisplayName("public context type")
		Expect(m.AddContextType(ct)).To(Succeed())

		em, err := eventmgr.NewEventManager()
		Expect(err).NotTo(HaveOccurred())

		eval := authz.NewEvaluator(authz.Config{
			AuditorGroup: "audit-group",
			PublicTypes:  authz.ParsePublicResourceTypes("ContextType"),
		})
		server := oapi.NewApiServer(m, em, "http://localhost", eval)
		strict := oapi.NewApiHandler(server, oapi.ApiHandlerOptions{TrustAuthHeaders: true})
		handler = oapi.HandlerFromMuxWithBaseURL(strict, mux.NewRouter(), "")
	})

	get := func(url, subject, groups string) *http.Response {
		req := httptest.NewRequest("GET", url, nil)
		if subject != "" {
			req.Header.Set("X-Auth-Subject", subject)
		}
		if groups != "" {
			req.Header.Set("X-Auth-Groups", groups)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w.Result()
	}

	listSystemIds := func(subject, groups string) []string {
		resp := get("http://localhost/landscape/systems", subject, groups)
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		var items []struct {
			InstanceId string `json:"instanceId"`
		}
		Expect(json.Unmarshal(body, &items)).To(Succeed())
		ids := make([]string, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.InstanceId)
		}
		return ids
	}

	It("shows an owner only their identity- and group-owned resources", func() {
		ids := listSystemIds("user-1", "team-a")
		Expect(ids).To(ConsistOf(ownedSystemId.String(), groupSystemId.String()))
	})

	It("hides everything not owned from a non-auditor", func() {
		ids := listSystemIds("user-3", "team-c")
		Expect(ids).To(BeEmpty())
	})

	It("shows the auditor all resources via the auditor group", func() {
		ids := listSystemIds("auditor-user", "audit-group")
		Expect(ids).To(HaveLen(4))
	})

	It("shows all resources when the BFF flags the caller as auditor", func() {
		req := httptest.NewRequest("GET", "http://localhost/landscape/systems", nil)
		req.Header.Set("X-Auth-Subject", "any-user")
		req.Header.Set("X-Auth-Auditor", "true")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, _ := io.ReadAll(resp.Body)
		var items []map[string]any
		Expect(json.Unmarshal(body, &items)).To(Succeed())
		Expect(items).To(HaveLen(4))
	})

	It("returns the resource by id to its owner", func() {
		resp := get(fmt.Sprintf("http://localhost/landscape/systems/%s", ownedSystemId), "user-1", "")
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("returns 404 (not 403) for a foreign-owned resource", func() {
		resp := get(fmt.Sprintf("http://localhost/landscape/systems/%s", foreignSystemId), "user-1", "team-a")
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	})

	It("returns 404 for an unowned resource to a non-auditor", func() {
		resp := get(fmt.Sprintf("http://localhost/landscape/systems/%s", unownedSystemId), "user-1", "team-a")
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	})

	It("shows public resource types to anonymous callers", func() {
		resp := get("http://localhost/landscape/contextTypes", "", "")
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, _ := io.ReadAll(resp.Body)
		var items []map[string]any
		Expect(json.Unmarshal(body, &items)).To(Succeed())
		Expect(items).To(HaveLen(1))
	})

	It("hides non-public resources from anonymous callers", func() {
		ids := listSystemIds("", "")
		Expect(ids).To(BeEmpty())
	})
})
