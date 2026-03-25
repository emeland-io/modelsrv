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

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context [displayName]",
	Short: "Create a Context resource",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		displayName, err := resolveDisplayName(cmd, args)
		if err != nil {
			return err
		}

		id := uuid.New()
		spec := map[string]any{
			"contextId":   id.String(),
			"displayName": displayName,
		}

		if v, _ := cmd.Flags().GetString("desc"); v != "" {
			spec["description"] = v
		}
		if v, _ := cmd.Flags().GetString("parent"); v != "" {
			spec["parent"] = v
		}
		if v, _ := cmd.Flags().GetString("type"); v != "" {
			spec["type"] = v
		}
		if ann, _ := cmd.Flags().GetStringSlice("annotation"); len(ann) > 0 {
			spec["annotations"] = parseAnnotations(ann)
		}

		r := Resource{Version: resourceVersion, Kind: "Context", Spec: spec}
		if err := writeResource(r, id); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	createCmd.AddCommand(contextCmd)
	contextCmd.Flags().StringP("name", "n", "", "Display name of the resource")
	contextCmd.Flags().String("desc", "", "Description of the context")
	contextCmd.Flags().String("parent", "", "Parent context UUID")
	contextCmd.Flags().String("type", "", "Context type UUID")
	contextCmd.Flags().StringSlice("annotation", nil, "Annotation in key=value format (repeatable)")
}
