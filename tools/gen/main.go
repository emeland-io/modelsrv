package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

//go:embed handler.tmpl
var handlerTemplate string

//go:embed type.tmpl
var typeTemplate string

//go:embed client.tmpl
var clientTemplate string

//go:embed client_test.tmpl
var clientTestTemplate string

//go:embed model_store_test.tmpl
var modelStoreTestTemplate string

//go:embed replication_decode.tmpl
var replicationDecodeTemplate string

//go:embed replication_encode.tmpl
var replicationEncodeTemplate string

//go:embed server_handler.tmpl
var serverHandlerTemplate string

//go:embed convert_from.tmpl
var convertFromTemplate string

//go:embed convert_to.tmpl
var convertToTemplate string

type convertGenData struct {
	Imports []string
	Specs   []convertGenSpec
}

type convertGenSpec struct {
	TypeSpec
	HasDisplayName      bool
	HasSummary          bool
	HasDescription      bool
	HasHash             bool
	HasAnnotations      bool
	IamStyleDescription bool
	ExtraStringFields   []string
}

func main() {
	funcMap := buildTemplateFuncMap()

	tmpl := template.Must(template.New("type").Funcs(funcMap).Parse(typeTemplate))
	handlerTmpl := template.Must(template.New("handler").Funcs(funcMap).Parse(handlerTemplate))
	clientTmpl := template.Must(template.New("client").Funcs(funcMap).Parse(clientTemplate))
	clientTestTmpl := template.Must(template.New("client_test").Funcs(funcMap).Parse(clientTestTemplate))
	modelStoreTestTmpl := template.Must(template.New("model_store_test").Funcs(funcMap).Parse(modelStoreTestTemplate))

	_, genFile, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "runtime.Caller failed")
		os.Exit(1)
	}
	baseDir := filepath.Clean(filepath.Join(filepath.Dir(genFile), "../../pkg/model"))

	for _, spec := range allTypes {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, spec); err != nil {
			fmt.Fprintf(os.Stderr, "Error executing template for %s: %v\n", spec.Name, err)
			os.Exit(1)
		}

		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting code for %s: %v\n", spec.Name, err)
			os.Exit(1)
		}

		outDir := filepath.Join(baseDir, spec.Dir)
		if err := os.MkdirAll(outDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "mkdir %s: %v\n", outDir, err)
			os.Exit(1)
		}
		filename := filepath.Join(outDir, fmt.Sprintf("%s_gen.go", strings.ToLower(spec.Name)))
		if err := os.WriteFile(filename, formatted, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file %s: %v\n", filename, err)
			os.Exit(1)
		}

		fmt.Printf("Generated %s\n", filename)
	}

	for _, spec := range allTypes {
		if !spec.HasHandler {
			continue
		}

		var buf bytes.Buffer
		if err := handlerTmpl.Execute(&buf, spec); err != nil {
			fmt.Fprintf(os.Stderr, "Error executing handler template for %s: %v\n", spec.Name, err)
			os.Exit(1)
		}

		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting handler code for %s: %v\n", spec.Name, err)
			os.Exit(1)
		}

		filename := filepath.Join(baseDir, fmt.Sprintf("handlers_%s_gen.go", strings.ToLower(spec.Name)))
		if err := os.WriteFile(filename, formatted, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing handler file %s: %v\n", filename, err)
			os.Exit(1)
		}

		fmt.Printf("Generated %s\n", filename)
	}

	// --- Generate client wrapper methods ---
	clientDir := filepath.Clean(filepath.Join(filepath.Dir(genFile), "../../pkg/client"))
	{
		var buf bytes.Buffer
		if err := clientTmpl.Execute(&buf, allTypes); err != nil {
			fmt.Fprintf(os.Stderr, "Error executing client template: %v\n", err)
			os.Exit(1)
		}
		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting client code: %v\n%s\n", err, buf.String())
			os.Exit(1)
		}
		filename := filepath.Join(clientDir, "client_gen.go")
		if err := os.WriteFile(filename, formatted, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", filename, err)
			os.Exit(1)
		}
		fmt.Printf("Generated %s\n", filename)
	}

	// --- Generate client integration test ---
	{
		var buf bytes.Buffer
		if err := clientTestTmpl.Execute(&buf, allTypes); err != nil {
			fmt.Fprintf(os.Stderr, "Error executing client_test template: %v\n", err)
			os.Exit(1)
		}
		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting client_test code: %v\n%s\n", err, buf.String())
			os.Exit(1)
		}
		filename := filepath.Join(clientDir, "client_gen_test.go")
		if err := os.WriteFile(filename, formatted, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", filename, err)
			os.Exit(1)
		}
		fmt.Printf("Generated %s\n", filename)
	}

	// --- Generate model store integration tests ---
	{
		var buf bytes.Buffer
		if err := modelStoreTestTmpl.Execute(&buf, allTypes); err != nil {
			fmt.Fprintf(os.Stderr, "Error executing model_store_test template: %v\n", err)
			os.Exit(1)
		}
		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting model_store_test code: %v\n%s\n", err, buf.String())
			os.Exit(1)
		}
		filename := filepath.Join(baseDir, "model_store_gen_test.go")
		if err := os.WriteFile(filename, formatted, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", filename, err)
			os.Exit(1)
		}
		fmt.Printf("Generated %s\n", filename)
	}

	oapiDir := filepath.Clean(filepath.Join(filepath.Dir(genFile), "../../internal/oapi"))
	replicationTmpl := template.Must(template.New("replication_decode").Parse(replicationDecodeTemplate))
	replicationEncodeTmpl := template.Must(template.New("replication_encode").Parse(replicationEncodeTemplate))
	serverHandlerTmpl := template.Must(template.New("server_handler").Parse(serverHandlerTemplate))
	convertFromTmpl := template.Must(template.New("convert_from").Parse(convertFromTemplate))
	convertToTmpl := template.Must(template.New("convert_to").Parse(convertToTemplate))

	writeOapi := func(name string, tmpl *template.Template, data any) {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			fmt.Fprintf(os.Stderr, "Error executing %s template: %v\n", name, err)
			os.Exit(1)
		}
		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting %s: %v\n%s\n", name, err, buf.String())
			os.Exit(1)
		}
		filename := filepath.Join(oapiDir, name)
		if err := os.WriteFile(filename, formatted, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", filename, err)
			os.Exit(1)
		}
		fmt.Printf("Generated %s\n", filename)
	}

	writeOapi("replication_decode_gen.go", replicationTmpl, allTypes)
	writeOapi("replication_encode_gen.go", replicationEncodeTmpl, buildReplicationEncodeGenData())
	writeOapi("server_handlers_gen.go", serverHandlerTmpl, allTypes)
	writeOapi("convert_from_wire_gen.go", convertFromTmpl, buildConvertFromGenData())
	writeOapi("convert_dto_encode_gen.go", convertToTmpl, buildConvertToGenData())
}

type replicationEncodeGenData struct {
	Imports []string
	Specs   []TypeSpec
}

// buildReplicationEncodeGenData collects the types that flow through replication
// (i.e. have an events.ResourceType and a domain package) for replication_encode.tmpl.
func buildReplicationEncodeGenData() replicationEncodeGenData {
	var specs []TypeSpec
	base := map[string]string{}
	for _, spec := range allTypes {
		if spec.EventsResource == "" || spec.DomainPkgImport == "" {
			continue
		}
		specs = append(specs, spec)
		base[spec.DomainPkgImport] = spec.DomainPkgAlias
	}
	return replicationEncodeGenData{
		Imports: formatImportLines(base, nil),
		Specs:   specs,
	}
}

func buildConvertFromGenData() convertGenData {
	data := buildConvertSpecs()
	data.Imports = formatImportLines(map[string]string{
		"fmt":                              "",
		"github.com/google/uuid":           "uuid",
		"go.emeland.io/modelsrv/pkg/model": "model",
	}, data.Specs)
	return data
}

func buildConvertToGenData() convertGenData {
	data := buildConvertSpecs()
	base := map[string]string{}
	for _, spec := range data.Specs {
		base[spec.DomainPkgImport] = spec.DomainPkgAlias
	}
	data.Imports = formatImportLines(base, data.Specs)
	return data
}

func buildConvertSpecs() convertGenData {
	var specs []convertGenSpec
	for _, spec := range allTypes {
		if spec.SkipConvert || spec.DomainPkgImport == "" {
			continue
		}
		cgs := convertGenSpec{TypeSpec: spec}
		handledStringFields := map[string]bool{
			"DisplayName": true,
			"Summary":     true,
			"Description": true,
			"Hash":        true,
		}
		for _, f := range spec.Fields {
			switch f.Name {
			case "DisplayName":
				cgs.HasDisplayName = true
			case "Summary":
				cgs.HasSummary = true
			case "Description":
				cgs.HasDescription = true
			case "Hash":
				cgs.HasHash = true
			case "Annotations":
				cgs.HasAnnotations = true
			default:
				if f.Type == "string" && !handledStringFields[f.Name] {
					cgs.ExtraStringFields = append(cgs.ExtraStringFields, f.Name)
				}
			}
		}
		if spec.Dir == "iam" {
			cgs.IamStyleDescription = true
		}
		specs = append(specs, cgs)
	}
	return convertGenData{Specs: specs}
}

func formatImportLines(base map[string]string, specs []convertGenSpec) []string {
	aliasByImport := make(map[string]string, len(base)+len(specs))
	for path, alias := range base {
		aliasByImport[path] = alias
	}
	for _, spec := range specs {
		aliasByImport[spec.DomainPkgImport] = spec.DomainPkgAlias
	}
	imports := make([]string, 0, len(aliasByImport))
	for path, alias := range aliasByImport {
		if alias == "" {
			imports = append(imports, `"`+path+`"`)
		} else {
			imports = append(imports, alias+` "`+path+`"`)
		}
	}
	for i := 0; i < len(imports); i++ {
		for j := i + 1; j < len(imports); j++ {
			if imports[j] < imports[i] {
				imports[i], imports[j] = imports[j], imports[i]
			}
		}
	}
	return imports
}
