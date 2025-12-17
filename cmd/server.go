package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/emeland/modelsrv/pkg/endpoint"
	"gitlab.com/emeland/modelsrv/pkg/events"
	"gitlab.com/emeland/modelsrv/pkg/model"
)

var serviceAddr string

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "start a model server instance",
	Long:  `start a model server instance that serves the model via REST API and provides a minimal web UI.`,

	Run: func(cmd *cobra.Command, args []string) {
		model, err := model.NewModel()
		if err != nil {
			fmt.Println("Error creating model:", err)
			return
		}

		eventMgr, err := events.NewEventManager()
		if err != nil {
			fmt.Println("Error creating event manager:", err)
			return
		}
		fmt.Println("Starting server...")
		endpoint.StarWebListener(model, eventMgr, serviceAddr)
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringP("service-addr", "a", ":8080", "The address the service listens on")
}
