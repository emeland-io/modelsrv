package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.emeland.io/modelsrv/pkg/client"
	"go.emeland.io/modelsrv/pkg/endpointprobe"
	"go.uber.org/zap"
)

var (
	certprobeServerURL          string
	certprobeOtelConfigOut      string
	certprobeCollectionInterval time.Duration
	certprobeListenAddr         string
	certprobeExpiryThreshold    time.Duration
	certprobeSubscribers        []string
)

// certprobeCmd generates a ready-to-run OTel Collector config for modelsrv-otel-exporter.
// Despite the subcommand name, this CLI does not probe endpoints — it only writes YAML.
var certprobeCmd = &cobra.Command{
	Use:   "certprobe",
	Short: "Generate OTel Collector config (does not probe endpoints)",
	Long: `Query ApiInstances from a running modelsrv (--server), discover probe URLs
from emeland.io/endpoint.* annotations, and write a complete OTel Collector
config for the modelsrv-otel-exporter binary.

This command is a config generator only. It does not dial annotated hosts.
Actual probing is done by modelsrv-otel-exporter --config <file>
(collector http_check receiver).

The generated config includes:
- http_check receiver with targets derived from ApiInstance annotations
- emeland exporter with endpoint_mapping (URL -> ApiInstance UUID)
- service pipeline wiring them together

For continuous updates while the server runs, prefer:
  modelsrv server --otel-config-out collector.yaml

Usage:
  modelsrv certprobe --otel-config-out collector.yaml --subscriber http://modelsrv:8080
  modelsrv-otel-exporter --config collector.yaml`,
	RunE: runCertprobe,
}

func init() {
	rootCmd.AddCommand(certprobeCmd)

	certprobeCmd.Flags().StringVar(&certprobeOtelConfigOut, "otel-config-out", "", "Path to write the collector config (required)")
	certprobeCmd.Flags().StringVarP(&certprobeServerURL, "server", "s", envOrDefault("MODELSRV_URL", "http://localhost:8080/api/"), "Running modelsrv API base URL")
	certprobeCmd.Flags().DurationVar(&certprobeCollectionInterval, "collection-interval", 5*time.Minute, "collection_interval for the http_check receiver")
	certprobeCmd.Flags().StringVar(&certprobeListenAddr, "listen-addr", "0.0.0.0:24200", "listen_addr for the emeland exporter")
	certprobeCmd.Flags().DurationVar(&certprobeExpiryThreshold, "expiry-threshold", 30*24*time.Hour, "Expiry threshold for certificate findings")
	certprobeCmd.Flags().StringArrayVar(&certprobeSubscribers, "subscriber", nil, "Downstream modelsrv URL to push events to (repeatable)")
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

	baseURL := normalizeAPIBaseURL(certprobeServerURL)
	c, err := client.NewModelSrvClient(baseURL)
	if err != nil {
		return fmt.Errorf("create modelsrv client: %w", err)
	}

	if err := c.GetTest(); err != nil {
		return fmt.Errorf("connect to modelsrv at %s: %w", baseURL, err)
	}

	targets, err := endpointprobe.CollectTargets(context.Background(), c, logger)
	if err != nil {
		return fmt.Errorf("collect targets: %w", err)
	}

	out, err := endpointprobe.RenderCollectorConfig(targets, endpointprobe.CollectorConfigOptions{
		CollectionInterval: certprobeCollectionInterval,
		ListenAddr:         certprobeListenAddr,
		ExpiryThreshold:    certprobeExpiryThreshold,
		Subscribers:        certprobeSubscribers,
	})
	if err != nil {
		return err
	}

	if dir := filepath.Dir(certprobeOtelConfigOut); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create output directory %s: %w", dir, err)
		}
	}
	if err := os.WriteFile(certprobeOtelConfigOut, out, 0644); err != nil {
		return fmt.Errorf("write %s: %w", certprobeOtelConfigOut, err)
	}

	logger.Infow("wrote collector config",
		"path", certprobeOtelConfigOut,
		"server", baseURL,
		"targets", len(targets),
	)
	return nil
}

// normalizeAPIBaseURL ensures the URL ends with a single trailing slash so the
// OpenAPI client joins paths correctly.
func normalizeAPIBaseURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return "http://localhost:8080/api/"
	}
	return strings.TrimRight(u, "/") + "/"
}
