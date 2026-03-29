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

	"github.com/spf13/cobra"
)

// newCreateCmd builds the "create" subcommand with all resource type subcommands.
func newCreateCmd() *cobra.Command {
	var outputDir string
	var outputFile string

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new resource in the landscape",
		Long:  `Create a new resource in the Emerging Enterprise Landscape (EmELand).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return fmt.Errorf("unknown resource type %q; see 'emelandctl create --help' for available types", args[0])
		},
	}

	createCmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "d", "data", "Directory to save the generated YAML file (auto-generated filename)")
	createCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "Write to this exact file path")
	createCmd.MarkFlagsMutuallyExclusive("output-dir", "output")

	for _, def := range resourceTypes {
		registerResourceCmd(createCmd, def, &outputDir, &outputFile)
	}

	return createCmd
}
