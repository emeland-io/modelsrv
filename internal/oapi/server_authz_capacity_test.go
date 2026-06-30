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
	mdlcap "go.emeland.io/modelsrv/pkg/model/capacity"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
)

var _ = Describe("ownership visibility for Capacity", func() {
	var (
		handler      http.Handler
		ownedCapID   = uuid.New()
		foreignCapID = uuid.New()
		crtID        = uuid.New()
		ctxID        = uuid.New()
	)

	BeforeEach(func() {
		sink := events.NewListSink()
		m, err := model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())

		crt := mdlcap.NewCapacityResourceType(crtID)
		crt.SetDisplayName("CPU")
		crt.SetUnit("cores")
		Expect(m.AddCapacityResourceType(crt)).To(Succeed())

		ct := mdlctx.NewContextType(uuid.New())
		ct.SetDisplayName("Environment")
		Expect(m.AddContextType(ct)).To(Succeed())

		ctx := mdlctx.NewContext(ctxID)
		ctx.SetDisplayName("Production")
		ctx.SetContextTypeById(ct.GetContextTypeId())
		Expect(m.AddContext(ctx)).To(Succeed())

		addCap := func(id uuid.UUID, name, owners string, category mdlcap.Category) {
			c := mdlcap.NewCapacity(id)
			c.SetDisplayName(name)
			c.SetCapacityResourceTypeById(crtID)
			c.SetContextById(ctxID)
			c.SetCategory(category)
			c.SetAmount(mdlcap.Amount("8"))
			if owners != "" {
				c.GetAnnotations().Add(authz.OwnerGroupsKey, owners)
			}
			Expect(m.AddCapacity(c)).To(Succeed())
		}

		addCap(ownedCapID, "owned capacity", "team-a", mdlcap.CategoryProvided)
		addCap(foreignCapID, "foreign capacity", "team-b", mdlcap.CategoryRequested)

		em, err := eventmgr.NewEventManager()
		Expect(err).NotTo(HaveOccurred())
		eval := authz.NewEvaluator(authz.Config{
			AuditorGroup: "audit-group",
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

	listCapacityIds := func(subject, groups string) []string {
		resp := get("http://localhost/landscape/capacities", subject, groups)
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

	It("shows owned capacity to the owner group", func() {
		ids := listCapacityIds("user-1", "team-a")
		Expect(ids).To(ConsistOf(ownedCapID.String()))
	})

	It("omits foreign capacity from list for non-owners", func() {
		ids := listCapacityIds("user-2", "team-a")
		Expect(ids).To(ConsistOf(ownedCapID.String()))
		Expect(ids).NotTo(ContainElement(foreignCapID.String()))
	})

	It("returns 404 for foreign capacity get-by-id", func() {
		resp := get(fmt.Sprintf("http://localhost/landscape/capacities/%s", foreignCapID), "user-1", "team-a")
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	})

	It("shows all capacity entries to auditors", func() {
		ids := listCapacityIds("auditor", "audit-group")
		Expect(ids).To(ConsistOf(ownedCapID.String(), foreignCapID.String()))
	})
})
