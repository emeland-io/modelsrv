package product_test

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/mocks"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/iam"
	mdlproduct "go.emeland.io/modelsrv/pkg/model/product"
	"go.uber.org/mock/gomock"
)

func TestProduct(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Product Suite")
}

var _ = Describe("Product functionalities", func() {
	var (
		productID uuid.UUID
		sinkMock  *mocks.MockEventSink
		product   mdlproduct.Product
	)

	BeforeEach(func() {
		productID = uuid.New()
		sinkMock = mocks.NewMockEventSink(gomock.NewController(GinkgoT()))
		product = mdlproduct.NewProduct(sinkMock, productID)
	})

	When("Product is created", func() {
		It("must not be nil", func() {
			Expect(product).NotTo(BeNil())
		})

		It("has the provided UUID", func() {
			Expect(product.GetProductId()).To(Equal(productID))
		})

		It("has annotations set", func() {
			Expect(product.GetAnnotations()).NotTo(BeNil())
		})
	})

	When("Product is updated", func() {
		Context("Product is not registered", func() {
			When("Display name gets updated", func() {
				It("updates the display name", func() {
					product.SetDisplayName("Test Product")

					Expect(product.GetDisplayName()).To(Equal("Test Product"))
				})
			})

			When("Description gets updated", func() {
				It("updates the description", func() {
					product.SetDescription("A test product")

					Expect(product.GetDescription()).To(Equal("A test product"))
				})
			})

			When("Vendor gets updated", func() {
				It("updates the vendor ref", func() {
					vendor := iam.NewOrgUnit(sinkMock, uuid.New())
					product.SetVendor(&iam.OrgUnitRef{
						OrgUnit:   vendor,
						OrgUnitId: vendor.GetOrgUnitId(),
					})

					Expect(product.GetVendor()).NotTo(BeNil())
					Expect(product.GetVendor().OrgUnit).To(Equal(vendor))
					Expect(product.GetVendor().OrgUnitId).To(Equal(vendor.GetOrgUnitId()))
				})
			})

			When("Versions get updated", func() {
				It("updates the versions", func() {
					versions := []mdlproduct.ProductionVersion{
						{Artefacts: []uuid.UUID{uuid.New()}},
					}

					product.SetVersions(versions)

					Expect(product.GetVersions()).To(Equal(versions))
				})
			})
		})
	})

	When("Product is updated", func() {
		Context("Product is registered", func() {
			BeforeEach(func() {
				product.Register()
			})

			When("DisplayName gets updated", func() {
				It("emits an event and calls Receive", func() {
					sinkMock.EXPECT().Receive(events.ProductResource, events.UpdateOperation, product.GetProductId(), gomock.Any())

					product.SetDisplayName("FooBar")
				})
			})
		})
	})
})

var _ = Describe("Product operations with model", func() {
	var (
		sink      *events.ListSink
		testModel model.Model
	)

	BeforeEach(func() {
		var err error
		sink = events.NewListSink()
		testModel, err = model.NewModel(sink)
		Expect(err).NotTo(HaveOccurred())
	})

	It("supports full CRUD lifecycle with correct event sequence", func() {
		productID := uuid.New()
		p1 := mdlproduct.NewProduct(testModel.GetSink(), productID)

		p1.SetDisplayName("Test Product")
		p1.SetDescription("a test product")
		Expect(testModel.GetProductById(productID)).To(BeNil())

		err := testModel.AddProduct(p1)
		Expect(err).NotTo(HaveOccurred())
		Expect(testModel.GetProductById(productID)).To(Equal(p1))

		products, err := testModel.GetProducts()
		Expect(err).NotTo(HaveOccurred())
		Expect(products).To(HaveLen(1))

		p1.SetDisplayName("the real test product")
		p1.SetDescription("a test product, but with more bla bla")

		p2 := mdlproduct.NewProduct(testModel.GetSink(), productID)
		p2.SetDisplayName("The other Test Product")
		p2.SetDescription("a different test product, but same Id")
		err = testModel.AddProduct(p2)
		Expect(err).NotTo(HaveOccurred())

		err = testModel.DeleteProductById(productID)
		Expect(err).NotTo(HaveOccurred())
		Expect(testModel.GetProductById(productID)).To(BeNil())

		err = testModel.DeleteProductById(productID)
		Expect(err).To(MatchError(common.ErrProductNotFound))

		expectedEvents := []struct {
			resourceType events.ResourceType
			operation    events.Operation
			resourceID   uuid.UUID
		}{
			{events.ProductResource, events.CreateOperation, productID},
			{events.ProductResource, events.UpdateOperation, productID},
			{events.ProductResource, events.UpdateOperation, productID},
			{events.ProductResource, events.UpdateOperation, productID},
			{events.ProductResource, events.DeleteOperation, productID},
		}

		actualEvents := sink.GetEvents()
		Expect(actualEvents).To(HaveLen(len(expectedEvents)))
		for i, expected := range expectedEvents {
			Expect(actualEvents[i].ResourceType).To(Equal(expected.resourceType))
			Expect(actualEvents[i].Operation).To(Equal(expected.operation))
			Expect(actualEvents[i].ResourceId).To(Equal(expected.resourceID))
		}
	})
})
