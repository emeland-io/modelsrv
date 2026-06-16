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

// flagToListPath maps flag names that take a resource UUID to their API list path.
var flagToListPath = map[string]string{
	"system":          "/landscape/systems",
	"parent":          "", // ambiguous, resolved per resource type
	"context":         "/landscape/contexts",
	"type":            "/landscape/contextTypes",
	"node-type":       "/landscape/nodeTypes",
	"component":       "/landscape/components",
	"api":             "/landscape/apis",
	"system-instance": "/landscape/system-instances",
	"artifact":        "/landscape/artifacts",
	"vendor":          "/landscape/orgUnits",
}

// completionForPath returns a cobra completion function that fetches resources from the given API path.
func completionForPath(listPath string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		url, err := serverURL()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		items, err := fetchResourceList(url, listPath)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var completions []string
		for _, item := range items {
			completions = append(completions, fmt.Sprintf("%s\t%s", item.Id, item.Name))
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

// registerFlagCompletions registers shell completion for UUID-reference flags on a command.
func registerFlagCompletions(cmd *cobra.Command, def resourceDef) {
	for _, f := range def.flags {
		if f.isBool {
			continue
		}
		listPath := resolveListPathForFlag(f.name, def)
		if listPath == "" {
			continue
		}
		_ = cmd.RegisterFlagCompletionFunc(f.name, completionForPath(listPath))
	}
}

// resolveListPathForFlag returns the API list path for a given flag name, handling
// the ambiguous "parent" flag by looking at the resource type.
func resolveListPathForFlag(flagName string, def resourceDef) string {
	if flagName == "parent" {
		// "parent" refers to the same resource type
		return def.listPath
	}
	return flagToListPath[flagName]
}
