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
	Props       []schemaProp
	Required    []string // wire property names, in declaration order
}

type schemaProp struct {
	WireName    string // JSON property name, e.g. "nodeTypeId"
	Description string // prose (field doc comment)
	Type        string // OpenAPI scalar type: string/boolean/integer/number; "" for array
	Format      string // e.g. "uuid"; empty when none
	IsArray     bool   // annotations-style array of $ref
	ItemsRef    string // schema name referenced by array items, e.g. "Annotation"
}

// buildResourceSchema assembles a resourceSchema from a parsed struct, its type doc, and
// its markers. Property order follows struct field declaration order. The id and name
// fields are required; other scalar fields are required unless tagged optional.
func buildResourceSchema(name, typeDoc string, st *ast.StructType, markers map[string]string) resourceSchema {
	rs := resourceSchema{
		Name:        name,
		Description: stripMarkers(typeDoc),
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
		default:
			prop.Type = openapiScalarType(goTypeToScalar(goType))
			prop.Format = openapiFormat(goType)
		}
		rs.Props = append(rs.Props, prop)

		// Required set: id and name always; other non-optional, non-annotation scalars.
		if roles["id"] || roles["name"] {
			rs.Required = append(rs.Required, prop.WireName)
		}
	}
	return rs
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

// wirePropName returns the JSON property name for a field: the id field is normalized
// (NodeTypeID -> nodeTypeId), others are lowerCamel of the Go name.
func wirePropName(fieldName string, roles map[string]bool) string {
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
		fmt.Fprintf(&b, "%sdescription: %s\n", p1, rs.Description)
	}
	fmt.Fprintf(&b, "%sproperties:\n", p1)
	for _, prop := range rs.Props {
		fmt.Fprintf(&b, "%s%s:\n", p2, prop.WireName)
		if prop.IsArray {
			fmt.Fprintf(&b, "%stype: array\n", p3)
			if prop.Description != "" {
				fmt.Fprintf(&b, "%sdescription: %s\n", p3, prop.Description)
			}
			fmt.Fprintf(&b, "%sitems:\n", p3)
			fmt.Fprintf(&b, "%s$ref: '#/components/schemas/%s'\n", p4, prop.ItemsRef)
			continue
		}
		fmt.Fprintf(&b, "%stype: %s\n", p3, prop.Type)
		if prop.Description != "" {
			fmt.Fprintf(&b, "%sdescription: %s\n", p3, prop.Description)
		}
		if prop.Format != "" {
			fmt.Fprintf(&b, "%sformat: %s\n", p3, prop.Format)
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
