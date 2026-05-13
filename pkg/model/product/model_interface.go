package product

import "github.com/google/uuid"

// ProductModel provides CRUD operations for [Product] resources.
type ProductModel interface {
	AddProduct(Product) error
	DeleteProductById(uuid.UUID) error
	GetProducts() ([]Product, error)
	GetProductById(uuid.UUID) Product
}
