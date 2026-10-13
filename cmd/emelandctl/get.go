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
	"io"
	"net/http"
	"text/tabwriter"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"go.emeland.io/modelsrv/pkg/model/common"
)

func renderInstanceList(cmd *cobra.Command, format string, items []common.InstanceListItem) error {
	if format == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "ID\tNAME\tREFERENCE"); err != nil {
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", item.Id, item.Name, item.Reference); err != nil {
			return err
		}
	}
	return w.Flush()
}

func fetchResourceList(baseURL, path, idField string) ([]common.InstanceListItem, []byte, error) {
	resp, err := http.Get(baseURL + path)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	var raw []map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, fmt.Errorf("decoding response: %w", err)
	}
	items := make([]common.InstanceListItem, 0, len(raw))
	for _, r := range raw {
		var item common.InstanceListItem
		for _, key := range []string{idField, "instanceId", "findingId", "nodeId", "id"} {
			if key == "" {
				continue
			}
			if rawID, ok := r[key]; ok {
				var s string
				if err := json.Unmarshal(rawID, &s); err == nil {
					if id, err := uuid.Parse(s); err == nil {
						item.Id = id
						break
					}
				}
			}
		}
		if rawName, ok := r["displayName"]; ok {
			_ = json.Unmarshal(rawName, &item.Name)
		}
		if rawRef, ok := r["reference"]; ok {
			_ = json.Unmarshal(rawRef, &item.Reference)
		}
		items = append(items, item)
	}
	return items, body, nil
}

func newGetCmd() *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Query resources from an EmELand server",
	}

	for _, def := range resourceTypes {
		if def.listPath == "" {
			continue
		}
		def := def // capture loop variable
		plural := def.use + "s"
		if def.use == "identity" {
			plural = "identities"
		}
		cmd := &cobra.Command{
			Use:   plural,
			Short: fmt.Sprintf("List %s from the EmELand server", plural),
			RunE: func(cmd *cobra.Command, args []string) error {
				outputFormat, _ := cmd.Flags().GetString("output")
				url, err := serverURL()
				if err != nil {
					return err
				}
				items, body, err := fetchResourceList(url, def.listPath, def.idField)
				if err != nil {
					return fmt.Errorf("fetching %s: %w", plural, err)
				}
				if outputFormat == "json" {
					// Preserve the server payload so full-resource list endpoints
					// expose every field, not just InstanceList summaries.
					var pretty any
					if err := json.Unmarshal(body, &pretty); err != nil {
						return err
					}
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					return enc.Encode(pretty)
				}
				return renderInstanceList(cmd, outputFormat, items)
			},
		}
		cmd.Flags().StringP("output", "o", "table", "Output format: table or json")
		getCmd.AddCommand(cmd)
	}

	return getCmd
}
