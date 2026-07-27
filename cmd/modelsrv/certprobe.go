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
)

// certprobeCmd renders OpenTelemetry Collector http_check config from ApiInstance endpoints.
var certprobeCmd = &cobra.Command{
	Use:   "certprobe",
	Short: "Export OpenTelemetry Collector http_check config from ApiInstance endpoints",
	Long: `Query ApiInstances from a running modelsrv (--server), discover probe URLs using
the same annotation rules as the in-process certprobe daemon, and write an
OpenTelemetry Collector receivers fragment for the http_check receiver.

The landscape is read over HTTP so resources from upstream subscribers and other
sensors are included — not only local file-sensor YAML.

The output enables httpcheck.tls.cert_remaining and lists one GET target per unique
host:port derived from emeland.io/endpoint.* annotations.`,
	RunE: runCertprobe,
}

func init() {
	rootCmd.AddCommand(certprobeCmd)

	certprobeCmd.Flags().StringVar(&certprobeOtelConfigOut, "otel-config-out", "", "Path to write the http_check receivers YAML fragment (required)")
	certprobeCmd.Flags().StringVarP(&certprobeServerURL, "server", "s", envOrDefault("MODELSRV_URL", "http://localhost:8080/api/"), "Running modelsrv API base URL (e.g. http://localhost:8080/api/)")
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

	out, err := endpointprobe.RenderHTTPCheckConfig(targets, certprobeCollectionInterval)
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

	logger.Infow("wrote http_check config",
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
