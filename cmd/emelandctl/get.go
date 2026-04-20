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
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"go.emeland.io/modelsrv/pkg/client"
)

func newGetCmd() *cobra.Command {
	var outputFormat string

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Query resources from an EmELand server",
	}

	findingsCmd := &cobra.Command{
		Use:   "findings",
		Short: "List findings from the EmELand server",
		RunE: func(cmd *cobra.Command, args []string) error {
			url, err := serverURL()
			if err != nil {
				return err
			}
			c, err := client.NewModelSrvClient(url)
			if err != nil {
				return fmt.Errorf("creating client: %w", err)
			}
			findings, err := c.GetFindings()
			if err != nil {
				return fmt.Errorf("fetching findings: %w", err)
			}
			if outputFormat == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(findings)
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			if _, err := fmt.Fprintln(w, "ID\tNAME\tREFERENCE"); err != nil {
				return err
			}
			for _, f := range *findings {
				id, name, ref := "", "", ""
				if f.InstanceId != nil {
					id = f.InstanceId.String()
				}
				if f.DisplayName != nil {
					name = *f.DisplayName
				}
				if f.Reference != nil {
					ref = *f.Reference
				}
				if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", id, name, ref); err != nil {
					return err
				}
			}
			return w.Flush()
		},
	}

	findingsCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table or json")
	getCmd.AddCommand(findingsCmd)

	return getCmd
}
