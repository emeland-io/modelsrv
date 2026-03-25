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

var componentInstanceCmd = &cobra.Command{
	Use:   "component-instance [displayName]",
	Short: "Create a ComponentInstance resource",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		displayName, err := resolveDisplayName(cmd, args)
		if err != nil {
			return err
		}

		id := uuid.New()
		spec := map[string]any{
			"instanceId":  id.String(),
			"displayName": displayName,
		}

		if v, _ := cmd.Flags().GetString("component"); v != "" {
			spec["component"] = v
		}
		if v, _ := cmd.Flags().GetString("system-instance"); v != "" {
			spec["systemInstance"] = v
		}
		if ann, _ := cmd.Flags().GetStringSlice("annotation"); len(ann) > 0 {
			spec["annotations"] = parseAnnotations(ann)
		}

		r := Resource{Version: resourceVersion, Kind: "ComponentInstance", Spec: spec}
		if err := writeResource(r, id); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	createCmd.AddCommand(componentInstanceCmd)
	componentInstanceCmd.Flags().StringP("name", "n", "", "Display name of the resource")
	componentInstanceCmd.Flags().String("component", "", "Component UUID this instance refers to")
	componentInstanceCmd.Flags().String("system-instance", "", "SystemInstance UUID")
	componentInstanceCmd.Flags().StringSlice("annotation", nil, "Annotation in key=value format (repeatable)")
}
