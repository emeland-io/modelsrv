package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// specSchemaBaseIndent is the indentation (spaces) of a schema name under
// components/schemas in the OpenAPI file.
const specSchemaBaseIndent = 4

// modelDefsDir is the directory (relative to this file) holding annotated resource
// definition structs parsed by the loader. During the migration it points at the pilot
// definitions; it becomes the canonical location as types are migrated.
const modelDefsDir = "modeldefs"

// openAPISpecRelPath is the OpenAPI file path relative to this source file.
const openAPISpecRelPath = "../../api/openapi/EmergingEnterpriseLandscape-0.1.0-oapi-3.0.3.yaml"

// runSpecMerge regenerates the migrated resource types' schema blocks in the OpenAPI file
// in place, passing all other content through verbatim, and writes the result back. It is
// the step-4 pipeline entry point (invoked as `go run ./tools/gen -mode=spec` before
// oapi-codegen). While migrated schemas are byte-identical to the committed ones, this
// rewrites the file with identical bytes (a no-op diff).
func runSpecMerge() error {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("runtime.Caller failed")
	}
	baseDir := filepath.Dir(thisFile)
	specPath := filepath.Clean(filepath.Join(baseDir, openAPISpecRelPath))
	defsDir := filepath.Join(baseDir, modelDefsDir)

	schemas, err := loadResourceSchemas(defsDir)
	if err != nil {
		return fmt.Errorf("loading resource schemas from %s: %w", defsDir, err)
	}
	if len(schemas) == 0 {
		// Nothing migrated yet; leave the spec untouched.
		return nil
	}

	data, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", specPath, err)
	}

	merged, err := mergeSchemas(string(data), schemas, specSchemaBaseIndent)
	if err != nil {
		return err
	}

	if merged == string(data) {
		// No change; avoid touching the file's mtime.
		fmt.Printf("spec merge: %s already up to date (%d migrated schema(s))\n", filepath.Base(specPath), len(schemas))
		return nil
	}
	if err := os.WriteFile(specPath, []byte(merged), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", specPath, err)
	}
	fmt.Printf("spec merge: regenerated %d schema block(s) in %s\n", len(schemas), filepath.Base(specPath))
	return nil
}

// mergeSchemas returns specText with each migrated resource type's component-schema block
// replaced by the freshly emitted block, and every other byte passed through unchanged.
//
// This is the ADR step-4 pipeline: the OpenAPI file oapi-codegen consumes is produced by
// regenerating only the schema blocks of types that have been migrated to annotated
// structs (loadResourceSchemas), leaving all other schemas, paths, and prose verbatim.
// While the emitted blocks are byte-identical to the committed ones, the merged output is
// byte-identical to the committed spec, so flipping the pipeline is a no-op.
//
// baseIndent is the indentation of a schema name under components/schemas (4 in this spec).
func mergeSchemas(specText string, schemas []resourceSchema, baseIndent int) (string, error) {
	out := specText
	for _, rs := range schemas {
		block, ok := schemaBlock(out, rs.Name, baseIndent)
		if !ok {
			return "", fmt.Errorf("schema block for %q not found in spec", rs.Name)
		}
		emitted := emitSchema(rs, baseIndent)
		out = strings.Replace(out, block, emitted, 1)
	}
	return out, nil
}

// schemaBlock returns the exact text of the `<indent><name>:` block within
// components/schemas: from the schema-name line up to (but not including) the next line at
// the same or lesser indentation. The returned text ends with a trailing newline, matching
// emitSchema's output. ok is false when the schema is not present.
func schemaBlock(specText, name string, baseIndent int) (string, bool) {
	indent := strings.Repeat(" ", baseIndent)
	header := indent + name + ":"

	lines := strings.Split(specText, "\n")
	start := -1
	for i, l := range lines {
		if l == header {
			start = i
			break
		}
	}
	if start < 0 {
		return "", false
	}

	deeper := indent + " " // strictly deeper than baseIndent
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		l := lines[i]
		if l == "" {
			continue // blank lines belong to the block
		}
		// A line that is indented but not deeper than baseIndent, or not indented at all,
		// starts the next sibling/section and ends this block.
		if strings.HasPrefix(l, indent) && !strings.HasPrefix(l, deeper) {
			end = i
			break
		}
		if !strings.HasPrefix(l, indent) {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n") + "\n", true
}
