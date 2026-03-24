package main

import (
	"fmt"

	"github.com/spf13/cobra"
	eventmgr "go.emeland.io/modelsrv/internal/events"
	"go.emeland.io/modelsrv/pkg/endpoint"
	"go.emeland.io/modelsrv/pkg/model"
)

var serviceAddr string

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "minimal model server for the Emerging Enterprise Landscape (EmELand).",
	Long:  `minimal model server instance that serves the model via REST API and provides a minimal web UI.`,

	Run: func(cmd *cobra.Command, args []string) {
		eventMgr, err := eventmgr.NewEventManager()
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
		if err := endpoint.StarWebListener(model, eventMgr, serviceAddr); err != nil {
			fmt.Println("Error starting web listener: ", err)
			return
		}
		fmt.Println("Server started successfully")
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVarP(&serviceAddr, "service-addr", "a", ":8080", "The address the service listens on")
}
