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

func fetchResourceList(baseURL, path string) ([]common.InstanceListItem, error) {
	resp, err := http.Get(baseURL + path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 but received %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		InstanceId  *string `json:"instanceId"`
		FindingId   *string `json:"findingId"`
		NodeId      *string `json:"nodeId"`
		Id          *string `json:"id"`
		DisplayName *string `json:"displayName"`
		Reference   *string `json:"reference"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	items := make([]common.InstanceListItem, 0, len(raw))
	for _, r := range raw {
		var item common.InstanceListItem
		switch {
		case r.InstanceId != nil:
			item.Id, _ = uuid.Parse(*r.InstanceId)
		case r.FindingId != nil:
			item.Id, _ = uuid.Parse(*r.FindingId)
		case r.NodeId != nil:
			item.Id, _ = uuid.Parse(*r.NodeId)
		case r.Id != nil:
			item.Id, _ = uuid.Parse(*r.Id)
		}
		if r.DisplayName != nil {
			item.Name = *r.DisplayName
		}
		if r.Reference != nil {
			item.Reference = *r.Reference
		}
		items = append(items, item)
	}
	return items, nil
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
				items, err := fetchResourceList(url, def.listPath)
				if err != nil {
					return fmt.Errorf("fetching %s: %w", plural, err)
				}
				return renderInstanceList(cmd, outputFormat, items)
			},
		}
		cmd.Flags().StringP("output", "o", "table", "Output format: table or json")
		getCmd.AddCommand(cmd)
	}

	return getCmd
}
