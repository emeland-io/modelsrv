package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"go.emeland.io/modelsrv/pkg/backend"
	"go.emeland.io/modelsrv/pkg/endpointprobe"
	"go.emeland.io/modelsrv/pkg/filesensor"
	"go.uber.org/zap"
)

var (
	certprobeDataDir            string
	certprobeOtelConfigOut      string
	certprobeCollectionInterval time.Duration
)

// certprobeCmd renders OpenTelemetry Collector http_check config from ApiInstance endpoints.
var certprobeCmd = &cobra.Command{
	Use:   "certprobe",
	Short: "Export OpenTelemetry Collector http_check config from ApiInstance endpoints",
	Long: `Load ApiInstances from --data-dir (same YAML as modelsrv server), discover probe
URLs using the same annotation rules as the in-process certprobe daemon, and write an
OpenTelemetry Collector receivers fragment for the http_check receiver.

The output enables httpcheck.tls.cert_remaining and lists one GET target per unique
host:port derived from emeland.io/endpoint.* annotations.`,
	RunE: runCertprobe,
}

func init() {
	rootCmd.AddCommand(certprobeCmd)

	certprobeCmd.Flags().StringVar(&certprobeOtelConfigOut, "otel-config-out", "", "Path to write the http_check receivers YAML fragment (required)")
	certprobeCmd.Flags().StringVar(&certprobeDataDir, "data-dir", envOrDefault("DATA_DIR", "data"), "Directory of YAML model definitions (.yaml/.yml)")
	certprobeCmd.Flags().DurationVar(&certprobeCollectionInterval, "collection-interval", 5*time.Minute, "collection_interval written into the http_check receiver config")
	_ = certprobeCmd.MarkFlagRequired("otel-config-out")
}

func runCertprobe(_ *cobra.Command, _ []string) error {
	cfg := zap.NewDevelopmentConfig()
	cfg.DisableStacktrace = true
	log, err := cfg.Build()
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	defer log.Sync() //nolint:errcheck
	logger := log.Sugar()

	dataPath := certprobeDataDir
	if !filepath.IsAbs(dataPath) {
		if abs, err := filepath.Abs(dataPath); err == nil {
			dataPath = abs
		}
	}

	b, err := backend.New()
	if err != nil {
		return fmt.Errorf("create backend: %w", err)
	}

	if err := filesensor.LoadDir(dataPath, b.GetModel(), logger); err != nil {
		return err
	}

	targets, err := endpointprobe.CollectTargets(context.Background(), endpointprobe.NewModelClient(b.GetModel()), logger)
	if err != nil {
		return fmt.Errorf("collect targets: %w", err)
	}

	out, err := endpointprobe.RenderHTTPCheckConfig(targets, certprobeCollectionInterval)
	if err != nil {
		return err
	}

	if err := os.WriteFile(certprobeOtelConfigOut, out, 0644); err != nil {
		return fmt.Errorf("write %s: %w", certprobeOtelConfigOut, err)
	}

	logger.Infow("wrote http_check config",
		"path", certprobeOtelConfigOut,
		"targets", len(targets),
	)
	return nil
}
