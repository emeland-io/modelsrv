package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
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

func loadNodeTypeSchema(t *testing.T) resourceSchema {
	t.Helper()
	return loadSchemaByName(t, "NodeType")
}

func loadSchemaByName(t *testing.T, name string) resourceSchema {
	t.Helper()
	schemas, err := loadResourceSchemas("testdata")
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

func TestSchemaEmitter_NodeTypeGolden(t *testing.T) {
	nt := loadNodeTypeSchema(t)
	got := emitSchema(nt, 4)

	want, ok := schemaBlock(readCommittedSpec(t), "NodeType", 4)
	if !ok {
		t.Fatal("NodeType block not found in committed spec")
	}

	if got != want {
		t.Errorf("emitted NodeType schema does not match committed spec.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// TestSchemaEmitter_Golden proves each migrated type's emitted schema block matches the
// committed spec byte-for-byte. System exercises $ref (version), a required bool
// (abstract), a non-id/name uuid ref (parent), a no-description field (displayName), and
// an id that is not in the required list.
func TestSchemaEmitter_Golden(t *testing.T) {
	spec := readCommittedSpec(t)
	for _, name := range []string{
		"NodeType", "System", "API", "Component",
		"ContextType", "Node", "SystemInstance", "ApiInstance",
		"OrgUnit", "Group", "Identity", "Parameter",
		"FindingType", "FilterRule", "MergeRule", "Product",
		"ArtifactInstance", "Capability", "ComponentInstance", "CapacityResourceType",
		"Finding", "PermissionSpec", "RoleSpec", "Permission", "Role", "Binding", "Capacity",
		"Metric", "Threshold", "MetricInstance", "MetricValue",
		"Artifact", "Context",
	} {
		t.Run(name, func(t *testing.T) {
			got := emitSchema(loadSchemaByName(t, name), 4)
			want, ok := schemaBlock(spec, name, 4)
			if !ok {
				t.Fatalf("%s block not found in committed spec", name)
			}
			if got != want {
				t.Errorf("emitted %s schema does not match committed spec.\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
			}
		})
	}
}

// TestMergeSchemas_ReproducesSpec proves the step-4 pipeline is a no-op today: running the
// generate-and-merge pass over the committed spec, regenerating the migrated types'
// blocks, yields a file byte-identical to the committed spec.
func TestMergeSchemas_ReproducesSpec(t *testing.T) {
	specText := readCommittedSpec(t)

	schemas, err := loadResourceSchemas("testdata")
	if err != nil {
		t.Fatalf("loadResourceSchemas: %v", err)
	}

	merged, err := mergeSchemas(specText, schemas, 4)
	if err != nil {
		t.Fatalf("mergeSchemas: %v", err)
	}
	if merged != specText {
		t.Errorf("merged spec differs from committed spec (migrated types: %d)", len(schemas))
	}
}

// TestSchemaBlock_Boundaries sanity-checks the block extractor on a known schema so a
// future change to the extractor that silently grabs too much/little is caught.
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
