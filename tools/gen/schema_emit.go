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
	WireName    string   // JSON property name, e.g. "nodeTypeId"
	Description string   // prose (field doc comment)
	Type        string   // OpenAPI scalar type: string/boolean/integer/number; "" for array/ref
	Format      string   // e.g. "uuid"; empty when none
	IsArray     bool     // array property (items are either a $ref or a scalar)
	ItemsRef    string   // schema name referenced by array items (array-of-ref), e.g. "Annotation"
	ItemsType   string   // scalar type of array items (array-of-scalar), e.g. "string"
	ItemsFormat string   // format of array items (array-of-scalar), e.g. "uuid"
	Ref         string   // bare $ref to another schema (e.g. "Version"); no type/description emitted
	Enum        []string // enum values; when set, emitted before description and no `type` key
	Pattern     string   // OpenAPI `pattern` (regex)
	Example     string   // OpenAPI `example`
}

// buildResourceSchema assembles a resourceSchema from a parsed struct, its type doc, and
// its markers. Property order follows struct field declaration order. The `required` list
// is emitted in declaration order for fields tagged emeland:"required" (fully explicit).
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
			prop.ItemsRef = "Annotation"
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

// quote renders a free-text string as a YAML single-quoted scalar (embedded single quotes
// doubled). Applied uniformly to descriptions, patterns, examples and enum values so the
// emitter needs no value-aware quoting or block-scalar styles. Newlines are folded to
// spaces (the content is prose; semantics are unaffected).
func quote(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
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

// emitSchema renders a resourceSchema to YAML. All free-text values are single-quoted and
// single-line (see quote); there are no block scalars or value-aware quoting. baseIndent is
// the number of spaces before the schema name (4 in the components block).
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
		fmt.Fprintf(&b, "%sdescription: %s\n", p1, quote(rs.Description))
	}
	fmt.Fprintf(&b, "%sproperties:\n", p1)
	for _, prop := range rs.Props {
		fmt.Fprintf(&b, "%s%s:\n", p2, prop.WireName)
		if prop.Ref != "" {
			// Bare $ref property (no type/description).
			fmt.Fprintf(&b, "%s$ref: '#/components/schemas/%s'\n", p3, prop.Ref)
			continue
		}
		if prop.IsArray {
			fmt.Fprintf(&b, "%stype: array\n", p3)
			if prop.Description != "" {
				fmt.Fprintf(&b, "%sdescription: %s\n", p3, quote(prop.Description))
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
			// Enum property: enum list then description, no `type` key.
			fmt.Fprintf(&b, "%senum:\n", p3)
			for _, v := range prop.Enum {
				fmt.Fprintf(&b, "%s- %s\n", p4, quote(v))
			}
			if prop.Description != "" {
				fmt.Fprintf(&b, "%sdescription: %s\n", p3, quote(prop.Description))
			}
			continue
		}
		fmt.Fprintf(&b, "%stype: %s\n", p3, prop.Type)
		if prop.Description != "" {
			fmt.Fprintf(&b, "%sdescription: %s\n", p3, quote(prop.Description))
		}
		if prop.Format != "" {
			fmt.Fprintf(&b, "%sformat: %s\n", p3, prop.Format)
		}
		if prop.Pattern != "" {
			fmt.Fprintf(&b, "%spattern: %s\n", p3, quote(prop.Pattern))
		}
		if prop.Example != "" {
			fmt.Fprintf(&b, "%sexample: %s\n", p3, quote(prop.Example))
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
