/*
Copyright © 2025 Lutz Behnke

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const resourceVersion = "emeland.io/v1"

// Resource is the top-level YAML document written by the create subcommands.
type Resource struct {
	Version string         `yaml:"version"`
	Kind    string         `yaml:"kind"`
	Spec    map[string]any `yaml:"spec"`
}

var outputDir string

func init() {
	createCmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "o", "data", "Directory to save the generated YAML file")
}

// writeResource marshals a Resource to YAML and writes it into outputDir.
func writeResource(r Resource, id uuid.UUID) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	data, err := yaml.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshalling YAML: %w", err)
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("%s-%s.yaml", strings.ToLower(r.Kind), id.String()))
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	fmt.Printf("Created %s resource: %s\n", r.Kind, filename)
	return nil
}

// parseAnnotations converts a slice of "key=value" strings into a map.
func parseAnnotations(raw []string) map[string]string {
	result := make(map[string]string, len(raw))
	for _, entry := range raw {
		k, v, _ := strings.Cut(entry, "=")
		result[k] = v
	}
	return result
}

// resolveDisplayName returns the display name from the positional arg or the -n/--name flag.
func resolveDisplayName(cmd *cobra.Command, args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	name, _ := cmd.Flags().GetString("name")
	if name != "" {
		return name, nil
	}
	return "", fmt.Errorf("display name is required (positional argument or --name/-n flag)")
}
