// Command oapidoc renders an OpenAPI 3.x spec into Markdown files suitable
// for browsing in the GitHub web UI.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	specPath := flag.String("spec", "", "path to the OpenAPI YAML/JSON spec (required)")
	outDir := flag.String("out", "", "output directory for rendered Markdown (required)")
	version := flag.String("version", "", "modelsrv release tag to embed in the docs (e.g. v0.11.0)")
	flag.Parse()

	if *specPath == "" || *outDir == "" {
		fmt.Fprintln(os.Stderr, "usage: oapidoc -spec <path> -out <dir> [-version <tag>]")
		flag.PrintDefaults()
		os.Exit(2)
	}

	if err := render(*specPath, *outDir, *version); err != nil {
		fmt.Fprintf(os.Stderr, "oapidoc: %v\n", err)
		os.Exit(1)
	}

	abs, _ := filepath.Abs(*outDir)
	fmt.Printf("wrote API docs to %s\n", abs)
}
