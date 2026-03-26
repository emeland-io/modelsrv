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

// flagDef describes an optional flag for a resource subcommand.
type flagDef struct {
	name     string // flag name (e.g. "system")
	short    string // single-char shorthand, or ""
	specKey  string // key in the YAML spec (e.g. "system")
	usage    string
	isBool   bool
}

// resourceDef declares everything needed to generate a create subcommand.
type resourceDef struct {
	use        string // cobra Use (subcommand name)
	short      string // short description
	kind       string // YAML Kind value
	idField    string // spec key for the generated UUID (e.g. "systemId")
	nameField  string // spec key for the display name (default "displayName")
	flags      []flagDef
}

var (
	outputDir  string
	outputFile string
)

func init() {
	createCmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "d", "data", "Directory to save the generated YAML file (auto-generated filename)")
	createCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "Write to this exact file path")
	createCmd.MarkFlagsMutuallyExclusive("output-dir", "output")
}

// registerResourceCmd creates and registers a cobra subcommand from a resourceDef.
func registerResourceCmd(def resourceDef) {
	if def.nameField == "" {
		def.nameField = "displayName"
	}

	cmd := &cobra.Command{
		Use:   def.use + " [displayName]",
		Short: def.short,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			displayName, err := resolveDisplayName(cmd, args)
			if err != nil {
				return err
			}

			id := uuid.New()
			spec := map[string]any{
				def.idField:   id.String(),
				def.nameField: displayName,
			}

			for _, f := range def.flags {
				if f.isBool {
					if v, _ := cmd.Flags().GetBool(f.name); v {
						spec[f.specKey] = true
					}
				} else {
					if v, _ := cmd.Flags().GetString(f.name); v != "" {
						spec[f.specKey] = v
					}
				}
			}

			if ann, _ := cmd.Flags().GetStringSlice("annotation"); len(ann) > 0 {
				spec["annotations"] = parseAnnotations(ann)
			}

			r := Resource{Version: resourceVersion, Kind: def.kind, Spec: spec}
			return writeResource(r, id)
		},
	}

	cmd.Flags().StringP("name", "n", "", "Display name of the resource")
	cmd.Flags().StringSlice("annotation", nil, "Annotation in key=value format (repeatable)")
	for _, f := range def.flags {
		if f.isBool {
			cmd.Flags().Bool(f.name, false, f.usage)
		} else if f.short != "" {
			cmd.Flags().StringP(f.name, f.short, "", f.usage)
		} else {
			cmd.Flags().String(f.name, "", f.usage)
		}
	}

	createCmd.AddCommand(cmd)
}

// writeResource marshals a Resource to YAML and writes it to the path
// determined by --output or --output-dir.
func writeResource(r Resource, id uuid.UUID) error {
	data, err := yaml.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshalling YAML: %w", err)
	}

	filename := outputFile
	if filename == "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
		filename = filepath.Join(outputDir, fmt.Sprintf("%s-%s.yaml", strings.ToLower(r.Kind), id.String()))
	} else {
		if dir := filepath.Dir(filename); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("creating output directory: %w", err)
			}
		}
	}

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
