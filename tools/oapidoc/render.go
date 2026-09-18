package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// HTTP methods in a fixed, deterministic order.
var methodOrder = []string{"GET", "PUT", "POST", "DELETE", "PATCH", "HEAD", "OPTIONS", "TRACE"}

func render(specPath, outDir, version string) error {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		return fmt.Errorf("load spec: %w", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return fmt.Errorf("validate spec: %w", err)
	}

	if err := os.RemoveAll(outDir); err != nil {
		return fmt.Errorf("clear output dir: %w", err)
	}
	for _, sub := range []string{"", "endpoints", "schemas"} {
		if err := os.MkdirAll(filepath.Join(outDir, sub), 0o755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
	}

	if err := writeFile(filepath.Join(outDir, "README.md"), renderIndex(doc, version)); err != nil {
		return err
	}

	byTag := collectOperationsByTag(doc)
	tagNames := sortedKeys(byTag)
	for _, tag := range tagNames {
		content := renderTagPage(tag, tagDescription(doc, tag), byTag[tag])
		if err := writeFile(filepath.Join(outDir, "endpoints", tag+".md"), content); err != nil {
			return err
		}
	}

	schemas := schemaMap(doc)
	names := sortedKeys(schemas)
	for _, name := range names {
		content := renderSchemaPage(name, schemas[name])
		if err := writeFile(filepath.Join(outDir, "schemas", name+".md"), content); err != nil {
			return err
		}
	}

	return nil
}

func writeFile(path, content string) error {
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func renderIndex(doc *openapi3.T, version string) string {
	var b strings.Builder
	info := doc.Info
	title := "API"
	if info != nil && info.Title != "" {
		title = info.Title
	}
	fmt.Fprintf(&b, "# %s\n\n", title)
	if info != nil && info.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", strings.TrimSpace(info.Description))
	}
	if version != "" {
		fmt.Fprintf(&b, "- **Release:** `%s`\n", version)
	}
	if info != nil && info.Version != "" {
		fmt.Fprintf(&b, "- **API version:** `%s`\n", info.Version)
	}
	if version != "" || (info != nil && info.Version != "") {
		b.WriteString("\n")
	}

	if len(doc.Servers) > 0 {
		b.WriteString("## Servers\n\n")
		for _, s := range doc.Servers {
			line := fmt.Sprintf("- `%s`", s.URL)
			if s.Description != "" {
				line += " — " + strings.TrimSpace(s.Description)
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Endpoints\n\n")
	byTag := collectOperationsByTag(doc)
	listed := map[string]bool{}
	if len(doc.Tags) > 0 {
		for _, tag := range doc.Tags {
			if _, ok := byTag[tag.Name]; !ok {
				continue
			}
			listed[tag.Name] = true
			desc := strings.TrimSpace(tag.Description)
			line := fmt.Sprintf("- [%s](endpoints/%s.md)", tag.Name, tag.Name)
			if desc != "" {
				line += " — " + desc
			}
			b.WriteString(line + "\n")
		}
	}
	for _, name := range sortedKeys(byTag) {
		if listed[name] {
			continue
		}
		fmt.Fprintf(&b, "- [%s](endpoints/%s.md)\n", name, name)
	}
	b.WriteString("\n")

	b.WriteString("## Schemas\n\n")
	schemas := schemaMap(doc)
	for _, name := range sortedKeys(schemas) {
		ref := schemas[name]
		line := fmt.Sprintf("- [%s](schemas/%s.md)", name, name)
		if ref != nil && ref.Value != nil {
			if d := strings.TrimSpace(ref.Value.Description); d != "" {
				line += " — " + firstSentence(d)
			}
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}

type taggedOp struct {
	Method string
	Path   string
	Op     *openapi3.Operation
}

func collectOperationsByTag(doc *openapi3.T) map[string][]taggedOp {
	out := map[string][]taggedOp{}
	if doc.Paths == nil {
		return out
	}
	paths := sortedKeys(doc.Paths.Map())
	for _, path := range paths {
		item := doc.Paths.Find(path)
		if item == nil {
			continue
		}
		for _, method := range methodOrder {
			op := item.GetOperation(method)
			if op == nil {
				continue
			}
			tags := op.Tags
			if len(tags) == 0 {
				tags = []string{"untagged"}
			}
			for _, tag := range tags {
				out[tag] = append(out[tag], taggedOp{Method: method, Path: path, Op: op})
			}
		}
	}
	return out
}

func tagDescription(doc *openapi3.T, name string) string {
	for _, tag := range doc.Tags {
		if tag.Name == name {
			return strings.TrimSpace(tag.Description)
		}
	}
	return ""
}

func renderTagPage(tag, description string, ops []taggedOp) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", tag)
	if description != "" {
		fmt.Fprintf(&b, "%s\n\n", description)
	}
	for _, o := range ops {
		fmt.Fprintf(&b, "### %s `%s`\n\n", o.Method, o.Path)
		desc := strings.TrimSpace(o.Op.Description)
		if desc == "" {
			desc = strings.TrimSpace(o.Op.Summary)
		}
		if desc != "" {
			fmt.Fprintf(&b, "%s\n\n", desc)
		}

		params := o.Op.Parameters
		if len(params) > 0 {
			b.WriteString("**Parameters**\n\n")
			for _, pref := range params {
				if pref == nil || pref.Value == nil {
					continue
				}
				p := pref.Value
				typeStr := schemaTypeLink(p.Schema, "../schemas")
				req := ""
				if p.Required {
					req = ", required"
				}
				line := fmt.Sprintf("- `%s` (%s%s", p.Name, p.In, req)
				if typeStr != "" {
					line += ", " + typeStr
				}
				line += ")"
				if d := strings.TrimSpace(p.Description); d != "" {
					line += " — " + d
				}
				b.WriteString(line + "\n")
			}
			b.WriteString("\n")
		}

		if o.Op.RequestBody != nil && o.Op.RequestBody.Value != nil {
			rb := o.Op.RequestBody.Value
			b.WriteString("**Request body**\n\n")
			req := ""
			if rb.Required {
				req = " (required)"
			}
			if d := strings.TrimSpace(rb.Description); d != "" {
				fmt.Fprintf(&b, "%s%s\n\n", d, req)
			} else if req != "" {
				fmt.Fprintf(&b, "Required.\n\n")
			}
			for _, ct := range sortedContentTypes(rb.Content) {
				mt := rb.Content[ct]
				link := ""
				if mt != nil {
					link = schemaTypeLink(mt.Schema, "../schemas")
				}
				line := fmt.Sprintf("- `%s`", ct)
				if link != "" {
					line += ": " + link
				}
				b.WriteString(line + "\n")
			}
			b.WriteString("\n")
		}

		if o.Op.Responses != nil {
			codes := sortedKeys(o.Op.Responses.Map())
			if len(codes) > 0 {
				b.WriteString("**Responses**\n\n")
				for _, code := range codes {
					rref := o.Op.Responses.Value(code)
					if rref == nil {
						continue
					}
					desc := ""
					var content openapi3.Content
					if rref.Value != nil {
						if rref.Value.Description != nil {
							desc = strings.TrimSpace(*rref.Value.Description)
						}
						content = rref.Value.Content
					}
					line := fmt.Sprintf("- `%s`", code)
					if desc != "" {
						line += " — " + desc
					}
					b.WriteString(line + "\n")
					for _, ct := range sortedContentTypes(content) {
						mt := content[ct]
						link := ""
						if mt != nil {
							link = schemaTypeLink(mt.Schema, "../schemas")
						}
						sub := fmt.Sprintf("  - `%s`", ct)
						if link != "" {
							sub += ": " + link
						}
						b.WriteString(sub + "\n")
					}
				}
				b.WriteString("\n")
			}
		}
	}
	return b.String()
}

func renderSchemaPage(name string, ref *openapi3.SchemaRef) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", name)
	if ref == nil || ref.Value == nil {
		b.WriteString("_Schema definition unavailable._\n")
		return b.String()
	}
	s := ref.Value
	if d := strings.TrimSpace(s.Description); d != "" {
		fmt.Fprintf(&b, "%s\n\n", d)
	}

	typeStr := inlineType(s, ".")
	if typeStr != "" {
		fmt.Fprintf(&b, "- **Type:** %s\n", typeStr)
	}
	if len(s.Enum) > 0 {
		fmt.Fprintf(&b, "- **Enum:** %s\n", formatEnum(s.Enum))
	}
	if typeStr != "" || len(s.Enum) > 0 {
		b.WriteString("\n")
	}

	if len(s.Properties) > 0 {
		b.WriteString("## Properties\n\n")
		required := make(map[string]bool, len(s.Required))
		for _, r := range s.Required {
			required[r] = true
		}
		for _, propName := range sortedKeys(s.Properties) {
			pref := s.Properties[propName]
			line := fmt.Sprintf("- `%s`", propName)
			parts := []string{}
			if t := schemaTypeLink(pref, "."); t != "" {
				parts = append(parts, t)
			}
			if required[propName] {
				parts = append(parts, "required")
			}
			if len(parts) > 0 {
				line += " (" + strings.Join(parts, ", ") + ")"
			}
			if pref != nil && pref.Value != nil {
				if d := strings.TrimSpace(pref.Value.Description); d != "" {
					line += " — " + d
				}
				if len(pref.Value.Enum) > 0 && pref.Ref == "" {
					line += fmt.Sprintf(" Enum: %s.", formatEnum(pref.Value.Enum))
				}
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

func schemaMap(doc *openapi3.T) map[string]*openapi3.SchemaRef {
	if doc.Components == nil || doc.Components.Schemas == nil {
		return map[string]*openapi3.SchemaRef{}
	}
	return doc.Components.Schemas
}

// schemaTypeLink returns a human-readable type string, linking component refs
// when schemaDir is the relative directory containing schema Markdown files
// (e.g. "../schemas" from endpoints, "." from schemas).
func schemaTypeLink(ref *openapi3.SchemaRef, schemaDir string) string {
	if ref == nil {
		return ""
	}
	if name := componentSchemaName(ref.Ref); name != "" {
		return fmt.Sprintf("[%s](%s/%s.md)", name, schemaDir, name)
	}
	if ref.Value == nil {
		return ""
	}
	return inlineType(ref.Value, schemaDir)
}

func inlineType(s *openapi3.Schema, schemaDir string) string {
	if s == nil {
		return ""
	}
	if len(s.OneOf) > 0 {
		return "oneOf(" + joinSchemaRefs(s.OneOf, schemaDir) + ")"
	}
	if len(s.AnyOf) > 0 {
		return "anyOf(" + joinSchemaRefs(s.AnyOf, schemaDir) + ")"
	}
	if len(s.AllOf) > 0 {
		return "allOf(" + joinSchemaRefs(s.AllOf, schemaDir) + ")"
	}

	base := ""
	if s.Type != nil && len(s.Type.Slice()) > 0 {
		base = strings.Join(s.Type.Slice(), "|")
	}
	if s.Format != "" {
		if base == "" {
			base = "format:" + s.Format
		} else {
			base += " (" + s.Format + ")"
		}
	}
	if s.Items != nil {
		item := schemaTypeLink(s.Items, schemaDir)
		if item == "" {
			item = "object"
		}
		if base == "" || strings.Contains(base, "array") {
			return "array of " + item
		}
		return base + " of " + item
	}
	if base == "" && len(s.Properties) > 0 {
		return "object"
	}
	return base
}

func joinSchemaRefs(refs openapi3.SchemaRefs, schemaDir string) string {
	parts := make([]string, 0, len(refs))
	for _, r := range refs {
		t := schemaTypeLink(r, schemaDir)
		if t == "" {
			t = "object"
		}
		parts = append(parts, t)
	}
	return strings.Join(parts, ", ")
}

func componentSchemaName(ref string) string {
	const prefix = "#/components/schemas/"
	if strings.HasPrefix(ref, prefix) {
		return strings.TrimPrefix(ref, prefix)
	}
	return ""
}

func formatEnum(values []any) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, fmt.Sprintf("`%v`", v))
	}
	return strings.Join(parts, ", ")
}

func sortedContentTypes(c openapi3.Content) []string {
	if c == nil {
		return nil
	}
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// firstSentence returns the first sentence of s. A sentence ends at ". "
// (or ".\n") only when the following non-space character is uppercase or
// end-of-string, so abbreviations like "e.g." are preserved.
func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	for i := 0; i < len(s); i++ {
		if s[i] != '.' {
			continue
		}
		if i+1 == len(s) {
			return s
		}
		j := i + 1
		for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
			j++
		}
		if j == i+1 {
			// No whitespace after the period (e.g. "e.g.") — keep going.
			continue
		}
		if j == len(s) || (s[j] >= 'A' && s[j] <= 'Z') {
			return strings.TrimSpace(s[:i+1])
		}
	}
	return s
}
