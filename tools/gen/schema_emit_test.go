package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"gopkg.in/yaml.v3"
)

// committedSpecPath returns the path to the committed OpenAPI spec.
func committedSpecPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile),
		"../../api/openapi/EmergingEnterpriseLandscape-0.1.0-oapi-3.0.3.yaml"))
}

func readCommittedSpec(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(committedSpecPath(t))
	if err != nil {
		t.Fatalf("reading spec: %v", err)
	}
	return string(data)
}

func loadSchemaByName(t *testing.T, name string) resourceSchema {
	t.Helper()
	schemas, err := loadResourceSchemas("modeldefs")
	if err != nil {
		t.Fatalf("loadResourceSchemas: %v", err)
	}
	for _, s := range schemas {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("no %s schema loaded", name)
	return resourceSchema{}
}

// parseSchemaBlock parses a `    <name>:` schema block into a normalized map[string]any by
// unmarshalling its YAML. The block text is dedented so it is a valid standalone document.
func parseSchemaBlock(t *testing.T, block string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := yaml.Unmarshal([]byte(block), &m); err != nil {
		t.Fatalf("parsing schema block: %v\n%s", err, block)
	}
	return m
}

// migratedTypes are all resource types generated from annotated structs.
var migratedTypes = []string{
	"NodeType", "System", "API", "Component",
	"ContextType", "Node", "SystemInstance", "ApiInstance",
	"OrgUnit", "Group", "Identity", "Parameter",
	"FindingType", "FilterRule", "MergeRule", "Product",
	"ArtifactInstance", "Capability", "ComponentInstance", "CapacityResourceType",
	"Finding", "PermissionSpec", "RoleSpec", "Permission", "Role", "Binding", "Capacity",
	"Metric", "Threshold", "MetricInstance", "MetricValue",
	"Artifact", "Context",
}

// TestSchemaEmitter_Semantic proves each migrated type's emitted schema is *semantically*
// equal (parsed OpenAPI model) to the committed spec's block. Formatting (quoting, block
// style, one-line vs folded descriptions) is intentionally not compared; only the parsed
// content is, which is what oapi-codegen consumes.
func TestSchemaEmitter_Semantic(t *testing.T) {
	spec := readCommittedSpec(t)
	for _, name := range migratedTypes {
		t.Run(name, func(t *testing.T) {
			emitted := emitSchema(loadSchemaByName(t, name), 4)
			committed, ok := schemaBlock(spec, name, 4)
			if !ok {
				t.Fatalf("%s block not found in committed spec", name)
			}
			got := parseSchemaBlock(t, emitted)
			want := parseSchemaBlock(t, committed)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("emitted %s schema not semantically equal to committed spec.\n--- emitted (parsed) ---\n%#v\n--- committed (parsed) ---\n%#v", name, got, want)
			}
		})
	}
}

// TestMergeSchemas_Semantic proves the in-place merge produces a spec that parses to the
// same OpenAPI model as the committed spec (formatting of migrated blocks may differ).
func TestMergeSchemas_Semantic(t *testing.T) {
	specText := readCommittedSpec(t)

	schemas, err := loadResourceSchemas("modeldefs")
	if err != nil {
		t.Fatalf("loadResourceSchemas: %v", err)
	}
	merged, err := mergeSchemas(specText, schemas, 4)
	if err != nil {
		t.Fatalf("mergeSchemas: %v", err)
	}

	var mergedDoc, committedDoc map[string]any
	if err := yaml.Unmarshal([]byte(merged), &mergedDoc); err != nil {
		t.Fatalf("parsing merged spec: %v", err)
	}
	if err := yaml.Unmarshal([]byte(specText), &committedDoc); err != nil {
		t.Fatalf("parsing committed spec: %v", err)
	}
	if !reflect.DeepEqual(mergedDoc, committedDoc) {
		t.Error("merged spec is not semantically equal to the committed spec")
	}
}

// TestSchemaEmitter_KnownValues pins a couple of hand-verified property shapes so a future
// change that silently alters them is caught (folded in from the removed fidelity harness).
func TestSchemaEmitter_KnownValues(t *testing.T) {
	sys := parseSchemaBlock(t, emitSchema(loadSchemaByName(t, "System"), 4))["System"].(map[string]any)
	props := sys["properties"].(map[string]any)
	abstract := props["abstract"].(map[string]any)
	if abstract["type"] != "boolean" {
		t.Errorf("System.abstract type = %v, want boolean", abstract["type"])
	}
	if !containsStr(toStrs(sys["required"]), "abstract") {
		t.Error("System.abstract expected in required list")
	}

	nt := parseSchemaBlock(t, emitSchema(loadSchemaByName(t, "NodeType"), 4))["NodeType"].(map[string]any)
	ntProps := nt["properties"].(map[string]any)
	dn := ntProps["displayName"].(map[string]any)
	if dn["type"] != "string" {
		t.Errorf("NodeType.displayName type = %v, want string", dn["type"])
	}
}

func toStrs(v any) []string {
	arr, _ := v.([]any)
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func containsStr(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// TestSchemaBlock_Boundaries sanity-checks the block extractor on a known schema.
func TestSchemaBlock_Boundaries(t *testing.T) {
	spec := readCommittedSpec(t)
	block, ok := schemaBlock(spec, "NodeType", 4)
	if !ok {
		t.Fatal("NodeType block not found")
	}
	if !hasPrefix(block, "    NodeType:\n") {
		t.Errorf("block should start with the schema header, got:\n%s", block)
	}
	if containsLine(block, "    Node:") {
		t.Error("block leaked into the sibling Node schema")
	}
}

func hasPrefix(s, prefix string) bool { return len(s) >= len(prefix) && s[:len(prefix)] == prefix }

func containsLine(block, line string) bool {
	for _, l := range splitLines(block) {
		if l == line {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
