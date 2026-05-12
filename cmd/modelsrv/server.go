package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"
	"go.emeland.io/modelsrv/pkg/backend"
	"go.emeland.io/modelsrv/pkg/endpoint"
	"go.emeland.io/modelsrv/pkg/filesensor"
	"go.uber.org/zap"
)

var serviceAddr string
var dataDir string
var metricsAddr string

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "minimal model server for the Emerging Enterprise Landscape (EmELand).",
	Long:  `minimal model server instance that serves the model via REST API and provides a minimal web UI.`,

	Run: func(cmd *cobra.Command, args []string) {
		cfg := zap.NewDevelopmentConfig()
		cfg.DisableStacktrace = true
		log, err := cfg.Build()
		if err != nil {
			fmt.Fprintf(os.Stderr, "logger: %v\n", err)
			os.Exit(1)
		}
		defer log.Sync() //nolint:errcheck

		logger := log.Sugar()

		b, err := backend.New()
		if err != nil {
			logger.Errorw("error creating backend", "error", err)
			return
		}

		dataPath := dataDir
		if !filepath.IsAbs(dataPath) {
			if abs, err := filepath.Abs(dataPath); err == nil {
				dataPath = abs
			}
		}
		logger.Infow("starting modelsrv",
			"listen", serviceAddr,
			"dataDir", dataPath,
		)
		logger.Infof("REST API: http://%s/api", serviceAddr)
		logger.Infof("Swagger UI: http://%s/swagger/", serviceAddr)
		logger.Info("file sensor: watching for YAML in data directory")

		filesensor.Start(context.Background(), dataPath, b.GetModel(), logger)

		if err := endpoint.StarWebListener(b.GetModel(), b.GetEventManager(), serviceAddr); err != nil {
			logger.Errorw("error starting web listener", "error", err)
			return
		}

		if metricsAddr != "" {
			if err := endpoint.StartMetricsListener(metricsAddr); err != nil {
				logger.Errorw("error starting metrics listener", "error", err)
				return
			}
		}

		logger.Info("server is running (Ctrl+C to stop)")

		sigCh := make(chan os.Signal, 1)
		notifyShutdownSignals(sigCh)
		sig := <-sigCh
		logger.Infow("shutdown signal received", "signal", sig.String())
		endpoint.StopWebListener()
		logger.Info("goodbye")
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVarP(&serviceAddr, "service-addr", "a", ":8080", "The address the service listens on")
	serverCmd.Flags().StringVar(&dataDir, "data-dir", "data", "Directory to watch for YAML model definitions (.yaml/.yml); relative paths are resolved from the process working directory")
	serverCmd.Flags().StringVar(&metricsAddr, "metrics-addr", "", "If set, serve /metrics on a separate port (e.g. :9090); otherwise metrics are on the main port")
}

func notifyShutdownSignals(ch chan os.Signal) {
	if runtime.GOOS == "windows" {
		signal.Notify(ch, os.Interrupt)
		return
	}
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
}
