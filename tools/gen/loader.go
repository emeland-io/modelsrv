package main

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// resourceStruct is a parsed annotated resource definition.
type resourceStruct struct {
	Name    string
	Doc     string
	Struct  *ast.StructType
	Markers map[string]string
}

// forEachResourceStruct parses all Go source files in dir and returns the annotated
// resource structs (those carrying a +emeland:resource marker).
func forEachResourceStruct(dir string) ([]resourceStruct, error) {
	fset := token.NewFileSet()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	// Group parsed files by package name so go/doc can associate comments.
	filesByPkg := map[string][]*ast.File{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		filesByPkg[f.Name.Name] = append(filesByPkg[f.Name.Name], f)
	}

	var out []resourceStruct
	for _, files := range filesByPkg {
		// NewFromFiles is the non-deprecated entry point (ast.Package / parser.ParseDir
		// are deprecated).
		dpkg, err := doc.NewFromFiles(fset, files, ".")
		if err != nil {
			return nil, fmt.Errorf("building docs: %w", err)
		}
		for _, dt := range dpkg.Types {
			st, ok := structType(dt)
			if !ok {
				continue
			}
			markers := parseTypeMarkers(dt.Doc)
			if _, isResource := markers["resource"]; !isResource {
				continue
			}
			out = append(out, resourceStruct{Name: dt.Name, Doc: dt.Doc, Struct: st, Markers: markers})
		}
	}
	return out, nil
}

// loadTypeSpecs parses annotated resource-definition structs from the Go source files in
// dir (via go/ast + go/doc) and returns their TypeSpec *core* values: the fields that are
// set explicitly on the hand-written allTypes entries, before enrichWireMeta /
// enrichDomainMeta run. Callers apply enrichment separately, exactly as init() does today.
//
// This is the ADR step-2 loader (docs/adr/single-source-resource-fields.md): it lets a
// resource be declared once as an annotated Go struct instead of a TypeSpec literal.
func loadTypeSpecs(dir string) ([]TypeSpec, error) {
	structs, err := forEachResourceStruct(dir)
	if err != nil {
		return nil, err
	}
	var specs []TypeSpec
	for _, rsr := range structs {
		spec, err := buildTypeSpec(rsr.Name, rsr.Doc, rsr.Struct, rsr.Markers)
		if err != nil {
			return nil, fmt.Errorf("type %s: %w", rsr.Name, err)
		}
		applyWiring(&spec)
		specs = append(specs, spec)
	}
	return specs, nil
}

// loadResourceSchemas parses the same annotated structs and returns their OpenAPI-schema
// view (step 3). The OpenAPI schema fragments are rendered by emitSchema.
func loadResourceSchemas(dir string) ([]resourceSchema, error) {
	structs, err := forEachResourceStruct(dir)
	if err != nil {
		return nil, err
	}
	var out []resourceSchema
	for _, rsr := range structs {
		out = append(out, buildResourceSchema(rsr.Name, rsr.Doc, rsr.Struct, rsr.Markers))
	}
	return out, nil
}

// structType returns the *ast.StructType for a documented type, if it is a struct.
func structType(dt *doc.Type) (*ast.StructType, bool) {
	if dt.Decl == nil {
		return nil, false
	}
	for _, spec := range dt.Decl.Specs {
		tsp, ok := spec.(*ast.TypeSpec)
		if !ok || tsp.Name.Name != dt.Name {
			continue
		}
		st, ok := tsp.Type.(*ast.StructType)
		if ok {
			return st, true
		}
	}
	return nil, false
}

// parseTypeMarkers extracts "+emeland:key" / "+emeland:key=value" markers from a type's
// doc comment. A bare "+emeland:handler" yields key "handler" with an empty value.
func parseTypeMarkers(docText string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(docText, "\n") {
		line = strings.TrimSpace(line)
		const prefix = "+emeland:"
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		body := strings.TrimPrefix(line, prefix)
		key, val, _ := strings.Cut(body, "=")
		out[strings.TrimSpace(key)] = strings.TrimSpace(val)
	}
	return out
}

// buildTypeSpec assembles the pre-enrichment TypeSpec core from a parsed struct.
func buildTypeSpec(name, docText string, st *ast.StructType, markers map[string]string) (TypeSpec, error) {
	spec := TypeSpec{Name: name}

	// Dir: explicit marker or lower-cased name.
	if d := markers["dir"]; d != "" {
		spec.Dir = d
	} else {
		spec.Dir = strings.ToLower(name)
	}

	// Convention-derived identity fields (match the hand-written entries).
	spec.EventType = name
	if et := markers["event"]; et != "" {
		spec.EventType = et // e.g. ApiInstance -> APIInstance
	}
	spec.WireKind = name
	spec.OapiTypeName = name
	spec.NotFoundErr = "Err" + name + "NotFound"
	spec.NotFoundSentinel = "common.Err" + name + "NotFound"

	if a := markers["handleralias"]; a != "" {
		spec.HandlerPkgAlias = a
	}
	if d := markers["handlerdelete"]; d != "" {
		spec.HandlerDeleteName = d
	}

	_, spec.HasHandler = markers["handler"]

	if listPath := markers["list"]; listPath != "" {
		spec.RestListPath = listPath
	}

	// Fields, id and name.
	var extraImports []string
	seenImports := map[string]bool{}
	for _, astField := range st.Fields.List {
		if len(astField.Names) == 0 {
			continue // skip embedded fields
		}
		fieldName := astField.Names[0].Name
		goType := exprString(astField.Type)
		tag := fieldTag(astField)

		role := tag.Get("emeland")
		roles := strings.Split(role, ",")
		roleSet := map[string]bool{}
		for _, r := range roles {
			roleSet[strings.TrimSpace(r)] = true
		}

		// The id field sets IDField and is not emitted as a Field accessor. When a wire
		// name override is present (e.g. acronym ids like APIID -> apiId), the IDField
		// spelling is the PascalCase of that wire name (ApiId).
		if roleSet["id"] {
			if wire := roleValue(roleSet, "wire="); wire != "" {
				spec.IDField = upperFirst(wire)
			} else {
				spec.IDField = normalizeIDName(fieldName)
			}
			continue
		}

		f := Field{Name: fieldName, Type: goType}
		if roleSet["optional"] {
			f.Optional = true
		}
		if roleSet["annotations"] {
			f.HasAnnotations = true
		}
		if roleSet["name"] {
			spec.NameField = fieldName
		}
		spec.Fields = append(spec.Fields, f)

		// Record non-builtin package imports needed by field types.
		if imp := importForType(goType); imp != "" && !seenImports[imp] {
			seenImports[imp] = true
			extraImports = append(extraImports, imp)
		}
	}
	spec.ExtraImports = extraImports

	if _, ok := markers["client"]; ok {
		spec.HasClientTest = true
		spec.GenClientMethods = true
		// Client wrapper method names use the domain type name with an irregular-aware
		// plural (e.g. API -> GetAPIs, Identity -> GetIdentities).
		spec.ClientListMethod = "Get" + clientPlural(name)
		spec.ClientGetByIdMethod = "Get" + name + "ById"
		// oapi-codegen method names derive from the REST path's last segment (e.g.
		// /landscape/apis -> GetLandscapeApis). Requires the list= marker.
		seg := pascalFromPath(spec.RestListPath)
		spec.ClientListOapiMethod = "GetLandscape" + seg
		spec.ClientGetByIdOapiMethod = "GetLandscape" + seg + spec.IDField
	}

	if spec.NameField == "" {
		spec.NameField = "DisplayName"
	}
	if spec.IDField == "" {
		return spec, fmt.Errorf("no field tagged emeland:\"id\"")
	}

	// Test scaffolding (matches vocabulary-type entries).
	spec.TestDisplayName = "Test " + name
	spec.TestIDAssertExpr = "uuid.UUID(got." + spec.IDField + ")"
	spec.TestNameAssertExpr = "got.Get" + spec.NameField + "()"
	if spec.NameField == "DisplayName" {
		spec.TestNameAssertExpr = "got.DisplayName"
	}
	spec.TestSetup = buildVocabTestSetup(name, spec.Dir)

	return spec, nil
}

// buildVocabTestSetup renders the standard create/add TestSetup for a simple vocabulary
// type (display-name only), matching the hand-written NodeType/ContextType entries.
func buildVocabTestSetup(name, dir string) string {
	varName := "obj"
	return fmt.Sprintf(`%s := %s.New%s(testIDs["%s"])
			%s.SetDisplayName("Test %s")
			require.NoError(t, m.Add%s(%s))`,
		varName, dir, name, name, varName, name, name, varName)
}

// roleValue returns the value of the first role in the set with the given prefix
// (e.g. "wire="), or "".
func roleValue(roles map[string]bool, prefix string) string {
	for r := range roles {
		if strings.HasPrefix(r, prefix) {
			return strings.TrimPrefix(r, prefix)
		}
	}
	return ""
}

// upperFirst upper-cases the first rune (apiId -> ApiId).
func upperFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - 'a' + 'A'
	}
	return string(r)
}

// normalizeIDName maps a Go id field name to the TypeSpec IDField spelling: a trailing
// "ID" acronym becomes "Id" (NodeTypeID -> NodeTypeId), matching oapi/domain naming.
func normalizeIDName(fieldName string) string {
	if strings.HasSuffix(fieldName, "ID") {
		return strings.TrimSuffix(fieldName, "ID") + "Id"
	}
	return fieldName
}

// pluralize returns the regular English plural. Irregulars (e.g. "Identity" ->
// "Identities") will be handled by an explicit marker when those types are migrated.
func pluralize(name string) string {
	if strings.HasSuffix(name, "y") &&
		!strings.HasSuffix(name, "ay") && !strings.HasSuffix(name, "ey") &&
		!strings.HasSuffix(name, "oy") && !strings.HasSuffix(name, "uy") {
		return strings.TrimSuffix(name, "y") + "ies"
	}
	return name + "s"
}

// clientPlural pluralizes a domain type name for client wrapper method names, preserving
// acronym casing (API -> APIs, not Apis).
func clientPlural(name string) string {
	return pluralize(name)
}

// pascalFromPath converts the last segment of a REST list path into the PascalCase token
// oapi-codegen uses in its method names, e.g. "/landscape/apis" -> "Apis",
// "/landscape/system-instances" -> "SystemInstances", "/landscape/nodeTypes" -> "NodeTypes".
func pascalFromPath(path string) string {
	if path == "" {
		return ""
	}
	seg := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		seg = path[i+1:]
	}
	// Split on '-' (kebab) and upper-case the first rune of each part; other runes are
	// preserved (so camelCase segments like "nodeTypes" become "NodeTypes").
	parts := strings.Split(seg, "-")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		r := []rune(p)
		if r[0] >= 'a' && r[0] <= 'z' {
			r[0] = r[0] - 'a' + 'A'
		}
		b.WriteString(string(r))
	}
	return b.String()
}

// fieldTag returns the reflect.StructTag for an ast field (empty when absent).
func fieldTag(f *ast.Field) reflect.StructTag {
	if f.Tag == nil {
		return ""
	}
	// f.Tag.Value includes surrounding backticks; strip them.
	return reflect.StructTag(strings.Trim(f.Tag.Value, "`"))
}

// exprString renders a (possibly qualified) type expression as source, e.g.
// "string", "uuid.UUID", "annotations.Annotations".
func exprString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.ArrayType:
		return "[]" + exprString(t.Elt)
	default:
		return fmt.Sprintf("%T", e)
	}
}

// importForType maps a qualified field type to the model package import path it needs.
// Only the packages used by resource field types are mapped; builtins return "".
func importForType(goType string) string {
	pkg, _, ok := strings.Cut(goType, ".")
	if !ok {
		return "" // builtin like string/bool/uuid handled elsewhere
	}
	pkg = strings.TrimPrefix(pkg, "*")
	pkg = strings.TrimPrefix(pkg, "[]")
	switch pkg {
	case "annotations":
		return "go.emeland.io/modelsrv/pkg/model/annotations"
	default:
		return ""
	}
}
