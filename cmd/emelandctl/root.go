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
	"os"

	"github.com/spf13/cobra"
)

// newRootCmd builds a fresh command tree with no shared state.
func newRootCmd() *cobra.Command {
	var cfgFile string

	rootCmd := &cobra.Command{
		Use:   "emelandctl",
		Short: "The Emerging Enterprise Landscape (EmELand) CLI tool",
		Long: `This contains multiple tools to work with the Emerging Enterprise Landscape (EmELand) example mapping,
	including the minimal model server for the EmELand example mapping.
	It will allow you to query resource objects of your landscape. Furthermore, it will receive daa from other 
	EmELand model servers, which are adapters to special data sources like a Kubernetes cluster or Enterprise
	Portal.`,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.modelsrv.yaml)")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	rootCmd.AddCommand(newCreateCmd())

	return rootCmd
}

// Execute builds the command tree and runs it. Called by main.
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
