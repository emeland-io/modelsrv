package oapi_test

import (
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.emeland.io/modelsrv/internal/oapi"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/iam"
)

var _ = Describe("Group members on the wire", func() {
	var m model.Model

	BeforeEach(func() {
		var err error
		m, err = model.NewModel(events.NewListSink())
		Expect(err).NotTo(HaveOccurred())
	})

	It("round-trips member ids through the DTO and JSON", func() {
		alice := iam.NewIdentity(uuid.New())
		alice.SetDisplayName("alice")
		Expect(m.AddIdentity(alice)).To(Succeed())
		ghost := uuid.New() // not (yet) known to the model

		g := iam.NewGroup(uuid.New())
		g.SetDisplayName("admins")
		g.SetMembers([]*iam.IdentityRef{
			{Identity: alice},
			{IdentityId: ghost},
		})

		raw, err := json.Marshal(oapi.GroupToDto(g))
		Expect(err).NotTo(HaveOccurred())

		var dto oapi.Group
		Expect(json.Unmarshal(raw, &dto)).To(Succeed())
		back, err := oapi.GroupFromDto(m, &dto)
		Expect(err).NotTo(HaveOccurred())

		members := back.GetMembers()
		Expect(members).To(HaveLen(2))
		Expect(members[0].EffectiveIdentityID()).To(Equal(alice.GetIdentityId()))
		Expect(members[0].ResolvedIdentity()).NotTo(BeNil(), "known identities are resolved")
		Expect(members[1].EffectiveIdentityID()).To(Equal(ghost))
		Expect(members[1].ResolvedIdentity()).To(BeNil(), "unknown identities stay as plain ids")
	})

	It("skips nil and empty member references", func() {
		keep := uuid.New()
		g := iam.NewGroup(uuid.New())
		g.SetDisplayName("sparse")
		g.SetMembers([]*iam.IdentityRef{nil, {}, {IdentityId: keep}})

		dto := oapi.GroupToDto(g)
		Expect(dto.Members).NotTo(BeNil())
		Expect(*dto.Members).To(HaveLen(1))
		Expect(uuid.UUID((*dto.Members)[0])).To(Equal(keep))
	})

	It("omits the members field for a group without members", func() {
		g := iam.NewGroup(uuid.New())
		g.SetDisplayName("empty")

		raw, err := json.Marshal(oapi.GroupToDto(g))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(raw)).NotTo(ContainSubstring("members"))

		var dto oapi.Group
		Expect(json.Unmarshal(raw, &dto)).To(Succeed())
		back, err := oapi.GroupFromDto(m, &dto)
		Expect(err).NotTo(HaveOccurred())
		Expect(back.GetMembers()).To(BeEmpty())
	})
})
