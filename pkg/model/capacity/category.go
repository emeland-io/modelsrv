package capacity

import "fmt"

// Category classifies how much of a capacity resource type exists in a context.
type Category string

const (
	CategoryRequested Category = "requested"
	CategoryProvided  Category = "provided"
	CategoryConsumed  Category = "consumed"
)

// ParseCategory validates and returns a Category.
func ParseCategory(s string) (Category, error) {
	switch Category(s) {
	case CategoryRequested, CategoryProvided, CategoryConsumed:
		return Category(s), nil
	default:
		return "", fmt.Errorf("invalid capacity category %q (expected requested, provided, or consumed)", s)
	}
}
