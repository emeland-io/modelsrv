package main

import (
	"fmt"
	"go/ast"
	"strings"
)

// resourceSchema is the OpenAPI-schema view of a resource type, built from an annotated
// struct. It is deliberately separate from TypeSpec/Field (which drive Go generation) so
// schema concerns stay isolated. emitSchema renders it to the exact YAML the committed
// spec uses. See docs/adr/single-source-resource-fields.md (step 3).
type resourceSchema struct {
	Name        string // schema name, e.g. "NodeType"
	Description string // type-level prose (struct doc comment minus +emeland markers)
	DescStyle   string // block-scalar indicator for multi-line descriptions: "|" (default) or ">"
	Props       []schemaProp
	Required    []string // wire property names, in declaration order
}

type schemaProp struct {
	WireName    string // JSON property name, e.g. "nodeTypeId"
	Description string // prose (field doc comment)
	Type        string // OpenAPI scalar type: string/boolean/integer/number; "" for array/ref
	Format      string // e.g. "uuid"; empty when none
	IsArray     bool   // array property (items are either a $ref or a scalar)
	ItemsRef    string // schema name referenced by array items (array-of-ref), e.g. "Annotation"
	ItemsType   string // scalar type of array items (array-of-scalar), e.g. "string"
	ItemsFormat string // format of array items (array-of-scalar), e.g. "uuid"
	Ref         string // bare $ref to another schema (e.g. "Version"); no type/description emitted
	Enum        []string // enum values; when set, emitted before description and no `type` key
	Pattern     string   // OpenAPI `pattern` (regex), emitted after description
	Example     string   // OpenAPI `example`, emitted after pattern
	QuoteDesc   bool     // single-quote the description (value contains YAML-significant chars)
}

// buildResourceSchema assembles a resourceSchema from a parsed struct, its type doc, and
// its markers. Property order follows struct field declaration order. The `required` list
// is emitted in declaration order for fields tagged emeland:"required" (fully explicit).
func buildResourceSchema(name, typeDoc string, st *ast.StructType, markers map[string]string) resourceSchema {
	rs := resourceSchema{
		Name:        name,
		Description: stripMarkers(typeDoc),
	}
	if markers["descstyle"] == "folded" {
		rs.DescStyle = ">"
	} else {
		rs.DescStyle = "|"
	}

	for _, astField := range st.Fields.List {
		if len(astField.Names) == 0 {
			continue
		}
		fieldName := astField.Names[0].Name
		goType := exprString(astField.Type)
		roles := tagRoles(fieldTag(astField))
		desc := strings.TrimSpace(astField.Doc.Text())

		prop := schemaProp{
			WireName:    wirePropName(fieldName, roles),
			Description: desc,
		}

		switch {
		case roles["annotations"]:
			prop.IsArray = true
			prop.ItemsRef = annotationsItemRef(goType)
		case arrayRefTarget(roles) != "":
			// Array of $ref, e.g. Product.versions -> ProductionVersion.
			prop.IsArray = true
			prop.ItemsRef = arrayRefTarget(roles)
		case goType == "[]uuid.UUID":
			// Array of UUIDs, e.g. Component.consumes / provides.
			prop.IsArray = true
			prop.ItemsType = "string"
			prop.ItemsFormat = "uuid"
		case goType == "[]string":
			// Array of plain strings, e.g. Parameter.values.
			prop.IsArray = true
			prop.ItemsType = "string"
		case refTarget(roles) != "":
			prop.Ref = refTarget(roles)
		case enumValues(roles) != nil:
			prop.Enum = enumValues(roles)
		default:
			prop.Type = openapiScalarType(goTypeToScalar(goType))
			prop.Format = openapiFormat(goType)
			prop.Pattern = tagValue(roles, "pattern=")
			prop.Example = tagValue(roles, "example=")
			prop.QuoteDesc = roles["quotedesc"]
		}
		rs.Props = append(rs.Props, prop)

		// Required set, in declaration order: any field explicitly tagged emeland:"required".
		// Required-ness is fully explicit (no name/id default) because the committed spec
		// varies (e.g. FindingType lists no required properties at all).
		if roles["required"] {
			rs.Required = append(rs.Required, prop.WireName)
		}
	}
	return rs
}

// refTarget returns the schema name from an emeland:"ref=Schema" tag, or "".
func refTarget(roles map[string]bool) string {
	for r := range roles {
		if strings.HasPrefix(r, "ref=") {
			return strings.TrimPrefix(r, "ref=")
		}
	}
	return ""
}

// enumValues returns the enum values from an emeland:"enum=A|B|C" tag, or nil. Values are
// pipe-separated (comma is the role separator).
func enumValues(roles map[string]bool) []string {
	for r := range roles {
		if strings.HasPrefix(r, "enum=") {
			return strings.Split(strings.TrimPrefix(r, "enum="), "|")
		}
	}
	return nil
}

// arrayRefTarget returns the schema name from an emeland:"arrayref=Schema" tag (an array
// whose items are a $ref), or "".
func arrayRefTarget(roles map[string]bool) string {
	for r := range roles {
		if strings.HasPrefix(r, "arrayref=") {
			return strings.TrimPrefix(r, "arrayref=")
		}
	}
	return ""
}

// tagValue returns the value of the first role with the given prefix (e.g. "pattern="), or "".
func tagValue(roles map[string]bool, prefix string) string {
	for r := range roles {
		if strings.HasPrefix(r, prefix) {
			return strings.TrimPrefix(r, prefix)
		}
	}
	return ""
}

// yamlScalar renders a string value as the committed spec does: single-quoted when the
// value contains a colon or a double quote (which would otherwise require quoting), plain
// otherwise. Matches the Artifact.hash description/pattern/example formatting.
func yamlScalar(s string) string {
	if strings.Contains(s, ":") || strings.Contains(s, "\"") {
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
	return s
}

// stripMarkers returns doc text with "+emeland:" marker lines removed and surrounding
// blank lines trimmed, collapsed to a single line (the committed spec uses one-line
// descriptions).
func stripMarkers(docText string) string {
	var kept []string
	for _, line := range strings.Split(docText, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "+emeland:") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

// tagRoles parses an emeland struct tag into a set of role keywords.
func tagRoles(tag interface{ Get(string) string }) map[string]bool {
	set := map[string]bool{}
	for _, r := range strings.Split(tag.Get("emeland"), ",") {
		if r = strings.TrimSpace(r); r != "" {
			set[r] = true
		}
	}
	return set
}

// wirePropName returns the JSON property name for a field. An explicit emeland:"wire=x"
// tag wins (needed for acronym ids like APIID -> apiId); otherwise the id field is
// normalized (NodeTypeID -> nodeTypeId) and other fields are lowerCamel of the Go name.
func wirePropName(fieldName string, roles map[string]bool) string {
	for r := range roles {
		if strings.HasPrefix(r, "wire=") {
			return strings.TrimPrefix(r, "wire=")
		}
	}
	if roles["id"] {
		return lowerFirst(normalizeIDName(fieldName))
	}
	return lowerFirst(fieldName)
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'A' && r[0] <= 'Z' {
		r[0] = r[0] - 'A' + 'a'
	}
	return string(r)
}

// goTypeToScalar maps a Go field type to the scalar key understood by openapiScalarType.
// uuid.UUID is a string in the wire schema.
func goTypeToScalar(goType string) string {
	switch goType {
	case "uuid.UUID":
		return "string"
	default:
		return goType
	}
}

// openapiFormat returns the OpenAPI `format` for a Go type, or "" when none applies.
func openapiFormat(goType string) string {
	if goType == "uuid.UUID" {
		return "uuid"
	}
	return ""
}

// annotationsItemRef returns the schema name an annotations array references.
func annotationsItemRef(goType string) string {
	// annotations.Annotations -> items $ref Annotation.
	return "Annotation"
}

// openapiScalarType maps a scalar Go type name to its OpenAPI `type`. Returns "" for
// types that have no direct scalar mapping (e.g. arrays, refs).
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

// writeDescription writes a `description:` line, using a block scalar (indicator "|" or
// ">") when the text contains newlines (multi-line prose, e.g. Product/Metric). keyIndent
// is the indent of the `description:` key; contentIndent is the indent of block-scalar
// content lines.
func writeDescription(b *strings.Builder, keyIndent string, contentIndent int, indicator, text string) {
	if !strings.Contains(text, "\n") {
		fmt.Fprintf(b, "%sdescription: %s\n", keyIndent, text)
		return
	}
	if indicator == "" {
		indicator = "|"
	}
	ci := strings.Repeat(" ", contentIndent)
	fmt.Fprintf(b, "%sdescription: %s\n", keyIndent, indicator)
	for _, line := range strings.Split(text, "\n") {
		fmt.Fprintf(b, "%s%s\n", ci, line)
	}
}

// emitSchema renders a resourceSchema to YAML matching the committed spec's formatting.
// baseIndent is the number of spaces before the schema name (4 in the components block).
func emitSchema(rs resourceSchema, baseIndent int) string {
	ind := strings.Repeat(" ", baseIndent)
	p1 := strings.Repeat(" ", baseIndent+2) // schema body
	p2 := strings.Repeat(" ", baseIndent+4) // property name
	p3 := strings.Repeat(" ", baseIndent+6) // property body
	p4 := strings.Repeat(" ", baseIndent+8) // array items body

	var b strings.Builder
	fmt.Fprintf(&b, "%s%s:\n", ind, rs.Name)
	fmt.Fprintf(&b, "%stype: object\n", p1)
	if rs.Description != "" {
		writeDescription(&b, p1, baseIndent+4, rs.DescStyle, rs.Description)
	}
	fmt.Fprintf(&b, "%sproperties:\n", p1)
	for _, prop := range rs.Props {
		fmt.Fprintf(&b, "%s%s:\n", p2, prop.WireName)
		if prop.Ref != "" {
			// Bare $ref property: no type or description (matches the committed spec's
			// version -> Version references).
			fmt.Fprintf(&b, "%s$ref: '#/components/schemas/%s'\n", p3, prop.Ref)
			continue
		}
		if prop.IsArray {
			fmt.Fprintf(&b, "%stype: array\n", p3)
			if prop.Description != "" {
				fmt.Fprintf(&b, "%sdescription: %s\n", p3, prop.Description)
			}
			fmt.Fprintf(&b, "%sitems:\n", p3)
			if prop.ItemsRef != "" {
				fmt.Fprintf(&b, "%s$ref: '#/components/schemas/%s'\n", p4, prop.ItemsRef)
			} else {
				fmt.Fprintf(&b, "%stype: %s\n", p4, prop.ItemsType)
				if prop.ItemsFormat != "" {
					fmt.Fprintf(&b, "%sformat: %s\n", p4, prop.ItemsFormat)
				}
			}
			continue
		}
		if len(prop.Enum) > 0 {
			// Enum property: enum list then description, no `type` key (matches API.type).
			fmt.Fprintf(&b, "%senum:\n", p3)
			for _, v := range prop.Enum {
				fmt.Fprintf(&b, "%s- %s\n", p4, v)
			}
			if prop.Description != "" {
				fmt.Fprintf(&b, "%sdescription: %s\n", p3, prop.Description)
			}
			continue
		}
		fmt.Fprintf(&b, "%stype: %s\n", p3, prop.Type)
		if prop.Description != "" {
			desc := prop.Description
			if prop.QuoteDesc {
				desc = yamlScalar(desc)
			}
			fmt.Fprintf(&b, "%sdescription: %s\n", p3, desc)
		}
		if prop.Format != "" {
			fmt.Fprintf(&b, "%sformat: %s\n", p3, prop.Format)
		}
		if prop.Pattern != "" {
			fmt.Fprintf(&b, "%spattern: %s\n", p3, yamlScalar(prop.Pattern))
		}
		if prop.Example != "" {
			fmt.Fprintf(&b, "%sexample: %s\n", p3, yamlScalar(prop.Example))
		}
	}
	if len(rs.Required) > 0 {
		fmt.Fprintf(&b, "%srequired:\n", p1)
		for _, r := range rs.Required {
			fmt.Fprintf(&b, "%s- %s\n", p2, r)
		}
	}
	return b.String()
}
