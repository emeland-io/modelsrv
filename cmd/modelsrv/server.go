package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.emeland.io/modelsrv/pkg/endpoint"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

var serviceAddr string

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "minimal model server for the Emerging Enterprise Landscape (EmELand).",
	Long:  `minimal model server instance that serves the model via REST API and provides a minimal web UI.`,

	Run: func(cmd *cobra.Command, args []string) {
		eventMgr, err := events.NewEventManager()
		if err != nil {
			fmt.Println("Error creating event manager: ", err)
			return
		}

		sink, err := eventMgr.GetSink()
		if err != nil {
			fmt.Println("Error creating event sink: ", err)
			return
		}

		model, err := model.NewModel(sink)
		if err != nil {
			fmt.Println("Error creating model: ", err)
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
