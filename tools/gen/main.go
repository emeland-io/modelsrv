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

func main() {
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
		"refParamType": func(r RefByRefSpec) string {
			if r.ParamGoType != "" {
				return r.ParamGoType
			}
			return r.ResourceTypeName
		},
		"handlerQualifier": func(s TypeSpec) string {
			if s.HandlerPkgAlias != "" {
				return s.HandlerPkgAlias
			}
			return s.Dir
		},
		"handlerMethod": func(s TypeSpec) string {
			if s.HandlerMethodSuffix != "" {
				return s.HandlerMethodSuffix
			}
			return s.Name
		},
	}

	tmpl := template.Must(template.New("type").Funcs(funcMap).Parse(typeTemplate))
	handlerTmpl := template.Must(template.New("handler").Funcs(funcMap).Parse(handlerTemplate))
	clientTmpl := template.Must(template.New("client").Funcs(funcMap).Parse(clientTemplate))
	clientTestTmpl := template.Must(template.New("client_test").Funcs(funcMap).Parse(clientTestTemplate))

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
}
