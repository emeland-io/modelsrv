package c4injector

// Landscape is the hardcoded level-1 "system" — the enterprise landscape itself,
// not an EmELand System resource.
type Landscape struct {
	Name        string
	Description string
}

// DefaultLandscape returns the default landscape identity used when flags are unset.
func DefaultLandscape() Landscape {
	return Landscape{
		Name:        "Enterprise Landscape",
		Description: "EmELand-managed enterprise IT landscape: cooperating systems, APIs, and components.",
	}
}
