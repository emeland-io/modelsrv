package authz_test

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.emeland.io/modelsrv/pkg/authz"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model/annotations"
)

type stubOwnable struct {
	id   uuid.UUID
	ann  annotations.Annotations
	sink events.EventSink
}

func (s *stubOwnable) GetResourceId() uuid.UUID                { return s.id }
func (s *stubOwnable) GetAnnotations() annotations.Annotations { return s.ann }
func (s *stubOwnable) Receive(resType events.ResourceType, op events.Operation, resourceId uuid.UUID, object ...any) error {
	return nil
}

func newOwnable(identityOwners, groupOwners string) *stubOwnable {
	s := &stubOwnable{id: uuid.New(), sink: &stubOwnable{}}
	s.ann = annotations.NewAnnotations(s)
	if identityOwners != "" {
		s.ann.Add(authz.OwnerIdentitiesKey, identityOwners)
	}
	if groupOwners != "" {
		s.ann.Add(authz.OwnerGroupsKey, groupOwners)
	}
	return s
}

var _ = Describe("HasOwner", func() {
	It("reports false when no owner annotations are set", func() {
		unowned := newOwnable("", "")
		Expect(authz.HasOwner(unowned.GetAnnotations())).To(BeFalse())
	})

	It("reports true when an identity owner is set", func() {
		owned := newOwnable("user-1", "")
		Expect(authz.HasOwner(owned.GetAnnotations())).To(BeTrue())
	})
})

var _ = Describe("Evaluator.CanSee", func() {
	Describe("auditor access", func() {
		It("grants access when the BFF auditor header is set", func() {
			e := authz.NewEvaluator(authz.Config{})
			p := authz.Principal{Subject: "other", AuditorHeader: true}
			r := newOwnable("", "")
			Expect(e.CanSee(p, events.SystemResource, r)).To(BeTrue())
		})

		It("grants access when the principal belongs to the configured auditor group", func() {
			e := authz.NewEvaluator(authz.Config{AuditorGroup: "audit-group"})
			p := authz.Principal{Subject: "user", Groups: []string{"audit-group"}}
			r := newOwnable("", "")
			Expect(e.CanSee(p, events.SystemResource, r)).To(BeTrue())
		})

		It("grants access when the principal matches the configured auditor identity", func() {
			e := authz.NewEvaluator(authz.Config{AuditorIdentity: "auditor-sub"})
			p := authz.Principal{Subject: "auditor-sub"}
			Expect(e.CanSee(p, events.SystemResource, newOwnable("", ""))).To(BeTrue())
		})
	})

	Describe("ownership", func() {
		It("grants access to the identity owner", func() {
			e := authz.NewEvaluator(authz.Config{})
			p := authz.Principal{Subject: "owner-user"}
			r := newOwnable("owner-user", "")
			Expect(e.CanSee(p, events.SystemResource, r)).To(BeTrue())
		})

		It("grants access to a member of an owning group", func() {
			e := authz.NewEvaluator(authz.Config{})
			p := authz.Principal{Subject: "member", Groups: []string{"team-a"}}
			r := newOwnable("", "team-a")
			Expect(e.CanSee(p, events.SystemResource, r)).To(BeTrue())
		})

		It("denies access to unowned resources for non-auditors", func() {
			e := authz.NewEvaluator(authz.Config{})
			p := authz.Principal{Subject: "random-user"}
			r := newOwnable("", "")
			Expect(e.CanSee(p, events.SystemResource, r)).To(BeFalse())
		})
	})

	Describe("public resource types", func() {
		var (
			e authz.Evaluator
			p authz.Principal
			r *stubOwnable
		)

		BeforeEach(func() {
			e = *authz.NewEvaluator(authz.Config{
				PublicTypes: map[events.ResourceType]bool{events.ContextTypeResource: true},
			})
			p = authz.Principal{Subject: "random-user"}
			r = newOwnable("", "")
		})

		It("allows any principal to see public types", func() {
			Expect(e.CanSee(p, events.ContextTypeResource, r)).To(BeTrue())
		})

		It("still hides non-public unowned types", func() {
			Expect(e.CanSee(p, events.SystemResource, r)).To(BeFalse())
		})
	})

	Describe("anonymous principals", func() {
		var (
			e         authz.Evaluator
			anonymous authz.Principal
		)

		BeforeEach(func() {
			e = *authz.NewEvaluator(authz.Config{
				PublicTypes: map[events.ResourceType]bool{events.ContextTypeResource: true},
			})
			anonymous = authz.Principal{}
		})

		It("sees only public resource types", func() {
			unowned := newOwnable("", "")
			Expect(e.CanSee(anonymous, events.ContextTypeResource, unowned)).To(BeTrue())
		})

		It("does not see owned non-public resources", func() {
			owned := newOwnable("some-user", "team-a")
			Expect(e.CanSee(anonymous, events.SystemResource, owned)).To(BeFalse())
		})

		It("does not see unowned non-public resources", func() {
			unowned := newOwnable("", "")
			Expect(e.CanSee(anonymous, events.SystemResource, unowned)).To(BeFalse())
		})
	})
})

var _ = Describe("FilterVisible", func() {
	It("returns only resources visible to the principal", func() {
		e := authz.NewEvaluator(authz.Config{})
		p := authz.Principal{Subject: "user-1", Groups: []string{"team-a"}}
		mine := newOwnable("user-1", "")
		team := newOwnable("", "team-a")
		foreign := newOwnable("someone-else", "team-b")
		unowned := newOwnable("", "")

		out := authz.FilterVisible(e, p, events.SystemResource, []*stubOwnable{mine, team, foreign, unowned})
		Expect(out).To(HaveLen(2))
		Expect(out).To(ConsistOf(mine, team))
	})

	It("returns all items when the evaluator is nil", func() {
		items := []*stubOwnable{newOwnable("a", ""), newOwnable("b", "")}
		out := authz.FilterVisible[*stubOwnable](nil, authz.Principal{}, events.SystemResource, items)
		Expect(out).To(HaveLen(2))
	})
})

var _ = Describe("PrincipalFromCtx", func() {
	It("retrieves the principal stored in the context", func() {
		ctx := authz.WithPrincipal(context.Background(), authz.Principal{Subject: "sub-1"})
		p := authz.PrincipalFromCtx(ctx)
		Expect(p.Subject).To(Equal("sub-1"))
	})
})

var _ = Describe("ParseGroups", func() {
	It("splits comma-separated group ids and trims whitespace", func() {
		groups := authz.ParseGroups("a, b ,c")
		Expect(groups).To(Equal([]string{"a", "b", "c"}))
	})
})
