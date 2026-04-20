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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig(cfgFile)
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.modelsrv.yaml)")
	rootCmd.PersistentFlags().String("server", "", "EmELand server base URL (e.g. http://localhost:8082)")
	_ = viper.BindPFlag("server.url", rootCmd.PersistentFlags().Lookup("server"))
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	rootCmd.AddCommand(newCreateCmd())
	rootCmd.AddCommand(newGetCmd())

	return rootCmd
}

func initConfig(cfgFile string) error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("finding home directory: %w", err)
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".modelsrv")
		viper.SetConfigType("yaml")
	}
	viper.SetEnvPrefix("EMELAND")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only check for path error if it's not already a ConfigFileNotFoundError
			if _, ok2 := err.(*os.PathError); !ok2 {
				return err
			}
		}
	}
	return nil
}

// serverURL returns the configured server URL or an error.
func serverURL() (string, error) {
	u := viper.GetString("server.url")
	if u == "" {
		home, _ := os.UserHomeDir()
		return "", fmt.Errorf("server URL required: use --server flag or set server.url in %s",
			filepath.Join(home, ".modelsrv.yaml"))
	}
	return strings.TrimRight(u, "/") + "/api", nil
}

// Execute builds the command tree and runs it. Called by main.
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
