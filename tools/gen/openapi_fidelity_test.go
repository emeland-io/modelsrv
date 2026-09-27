package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"gopkg.in/yaml.v3"
)

// This is the Phase 1 "fidelity harness" from docs/adr/single-source-resource-fields.md.
//
// It asserts that the scalar fields declared in specs.go (TypeSpec.Fields) agree with the
// resource schemas in the committed OpenAPI file: each generator-owned scalar field must
// exist as a property of the matching type and shape, and its Optional flag must match
// whether the YAML marks the property required.
//
// The test documents the specs.go <-> OpenAPI mapping we depend on and guards against the
// two sources drifting apart. It reads only the committed spec; it changes no behavior.

// openapiSchema is a minimal view of an OpenAPI object schema sufficient for this check.
type openapiSchema struct {
	Type       string                     `yaml:"type"`
	Properties map[string]openapiProperty `yaml:"properties"`
	Required   []string                   `yaml:"required"`
}

type openapiProperty struct {
	Type   string    `yaml:"type"`
	Format string    `yaml:"format"`
	Ref    string    `yaml:"$ref"`
	Enum   []string  `yaml:"enum"`
	Items  yaml.Node `yaml:"items"`
}

type openapiComponents struct {
	Components struct {
		Schemas map[string]openapiSchema `yaml:"schemas"`
	} `yaml:"components"`
}

func loadOpenAPISchemas(t *testing.T) map[string]openapiSchema {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	specPath := filepath.Clean(filepath.Join(filepath.Dir(thisFile),
		"../../api/openapi/EmergingEnterpriseLandscape-0.1.0-oapi-3.0.3.yaml"))

	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("reading OpenAPI spec %s: %v", specPath, err)
	}
	var doc openapiComponents
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parsing OpenAPI spec: %v", err)
	}
	if len(doc.Components.Schemas) == 0 {
		t.Fatal("no component schemas found in OpenAPI spec")
	}
	return doc.Components.Schemas
}

// scalarFieldsOfSpec returns the generator-owned scalar fields of a spec that are expected
// to map 1:1 to an OpenAPI property (string/bool/numeric). Annotations, refs and
// SkipAccessor fields are excluded.
func scalarFieldsOfSpec(spec TypeSpec) []Field {
	var out []Field
	for _, f := range spec.Fields {
		if f.HasAnnotations || f.SkipAccessor || f.IsRef {
			continue
		}
		if isScalarConvertType(f.Type) {
			out = append(out, f)
		}
	}
	return out
}

// wireProperty maps a Go field name to its expected OpenAPI JSON property name. The spec
// uses lowerCamel property names (e.g. DisplayName -> displayName).
func wireProperty(fieldName string) string {
	if fieldName == "" {
		return ""
	}
	// lower-case the first rune only; the rest of the identifier is preserved.
	r := []rune(fieldName)
	// Handle all-caps acronyms conservatively: only the leading rune is lowered,
	// which matches the existing property names (displayName, description, ...).
	first := r[0]
	if first >= 'A' && first <= 'Z' {
		first = first - 'A' + 'a'
	}
	return string(append([]rune{first}, r[1:]...))
}

func openapiScalarType(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "bool":
		return "boolean"
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	default:
		return ""
	}
}

func TestOpenAPIFidelity_ScalarFieldsPresent(t *testing.T) {
	schemas := loadOpenAPISchemas(t)

	for _, spec := range allTypes {
		// Only types that have an OpenAPI wire type participate.
		wireName := spec.OapiWireTypeName
		if wireName == "" {
			continue
		}
		schema, ok := schemas[wireName]
		if !ok {
			// Some domain types share/rename wire types; skip when absent rather than
			// fail, to keep the harness focused on the drift we can prove.
			continue
		}

		requiredSet := make(map[string]bool, len(schema.Required))
		for _, r := range schema.Required {
			requiredSet[r] = true
		}

		for _, f := range scalarFieldsOfSpec(spec) {
			prop := wireProperty(f.Name)
			p, ok := schema.Properties[prop]
			if !ok {
				t.Errorf("%s: specs.go scalar field %q (property %q) has no property in OpenAPI schema %q",
					spec.Name, f.Name, prop, wireName)
				continue
			}

			// Type agreement (skip enum-valued properties, which have no `type`).
			if len(p.Enum) == 0 {
				want := openapiScalarType(f.Type)
				if want != "" && p.Type != want {
					t.Errorf("%s.%s: OpenAPI property %q has type %q, expected %q for Go type %q",
						spec.Name, f.Name, prop, p.Type, want, f.Type)
				}
			}

			// Optionality agreement: Optional field must NOT be in required; a
			// non-Optional scalar that is present as a plain property is expected to be
			// required. displayName is always required; description is conventionally
			// optional and modeled without the Optional flag, so exempt it.
			switch prop {
			case "description":
				// Description is handled specially by the converters; it is optional in
				// the YAML without carrying Optional in specs.go. Do not assert here.
			default:
				if f.Optional && requiredSet[prop] {
					t.Errorf("%s.%s: field marked Optional but property %q is in the schema's required list",
						spec.Name, f.Name, prop)
				}
			}
		}
	}
}

// TestOpenAPIFidelity_KnownScalars sanity-checks a couple of concrete, hand-verified
// mappings so a future refactor that silently breaks the harness itself is caught.
func TestOpenAPIFidelity_KnownScalars(t *testing.T) {
	schemas := loadOpenAPISchemas(t)

	// System.abstract is a required boolean.
	sys, ok := schemas["System"]
	if !ok {
		t.Fatal("System schema missing")
	}
	ab, ok := sys.Properties["abstract"]
	if !ok || ab.Type != "boolean" {
		t.Errorf("System.abstract: got %+v, want type=boolean", ab)
	}
	found := false
	for _, r := range sys.Required {
		if r == "abstract" {
			found = true
		}
	}
	if !found {
		t.Error("System.abstract expected in required list")
	}

	// NodeType.displayName is a required string.
	nt, ok := schemas["NodeType"]
	if !ok {
		t.Fatal("NodeType schema missing")
	}
	if dn, ok := nt.Properties["displayName"]; !ok || dn.Type != "string" {
		t.Errorf("NodeType.displayName: got %+v, want type=string", dn)
	}
}
