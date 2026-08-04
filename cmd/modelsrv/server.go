package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	eventmgr "go.emeland.io/modelsrv/internal/events"
	"go.emeland.io/modelsrv/pkg/authz"
	"go.emeland.io/modelsrv/pkg/backend"
	"go.emeland.io/modelsrv/pkg/endpoint"
	"go.emeland.io/modelsrv/pkg/filesensor"
	"go.uber.org/zap"
)

var serviceAddr string
var dataDir string
var metricsAddr string
var trustAuthHeaders bool
var auditorIdentity string
var auditorGroup string
var publicResourceTypes string
var enableCertprobe bool
var certprobeInterval time.Duration
var certprobeDebounce time.Duration
var certprobeTimeout time.Duration
var certprobeExpiryWarning time.Duration
var maxConcurrentProbes int
var eventHistoryLimit int

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

		b, err := backend.New(backend.WithEventHistoryLimit(eventHistoryLimit))
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

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		filesensor.Start(ctx, dataPath, b.GetModel(), logger)

		webOpts := endpoint.WebListenerOptions{
			TrustAuthHeaders: trustAuthHeaders,
			AuthzConfig: authz.Config{
				AuditorIdentity: auditorIdentity,
				AuditorGroup:    auditorGroup,
				PublicTypes:     authz.ParsePublicResourceTypes(publicResourceTypes),
			},
		}
		if err := endpoint.StartWebListener(b.GetModel(), b.GetEventManager(), serviceAddr, webOpts); err != nil {
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

		cancel()

		endpoint.StopWebListener()
		logger.Info("goodbye")
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVarP(&serviceAddr, "service-addr", "a", envOrDefault("SERVICE_ADDR", ":8080"), "The address the service listens on")
	serverCmd.Flags().StringVar(&dataDir, "data-dir", envOrDefault("DATA_DIR", "data"), "Directory to watch for YAML model definitions (.yaml/.yml); relative paths are resolved from the process working directory")
	serverCmd.Flags().StringVar(&metricsAddr, "metrics-addr", envOrDefault("METRICS_ADDR", ""), "If set, serve /metrics on a separate port (e.g. :9090); otherwise metrics are on the main port")
	serverCmd.Flags().BoolVar(&trustAuthHeaders, "trust-auth-headers", envOrDefault("TRUST_AUTH_HEADERS", "") == "true", "Trust X-Auth-* identity headers from the BFF and enforce ownership visibility")
	serverCmd.Flags().StringVar(&auditorIdentity, "auditor-identity", envOrDefault("AUDITOR_IDENTITY", ""), "OIDC subject treated as auditor when matching X-Auth-Subject")
	serverCmd.Flags().StringVar(&auditorGroup, "auditor-group", envOrDefault("AUDITOR_GROUP", ""), "Group id treated as auditor when present in X-Auth-Groups")
	serverCmd.Flags().StringVar(&publicResourceTypes, "public-resource-types", envOrDefault("PUBLIC_RESOURCE_TYPES", ""), "Comma-separated resource types always visible (e.g. ContextType,FindingType)")
	serverCmd.Flags().BoolVar(&enableCertprobe, "enable-certprobe", envOrDefault("ENABLE_CERTPROBE", "true") == "true", "Run certificate probing as a background process inside modelsrv")
	serverCmd.Flags().DurationVar(&certprobeInterval, "certprobe-interval", 5*time.Minute, "Certprobe background scan interval")
	serverCmd.Flags().DurationVar(&certprobeDebounce, "certprobe-debounce", envDurationOrDefault("CERTPROBE_DEBOUNCE", 5*time.Second), "Debounce window before event-triggered certprobe rescans")
	serverCmd.Flags().DurationVar(&certprobeTimeout, "certprobe-timeout", 10*time.Second, "Per-probe HTTP/TLS timeout")
	serverCmd.Flags().DurationVar(&certprobeExpiryWarning, "expiry-warning", 720*time.Hour, "Raise CertificateExpiringSoon when remaining cert lifetime is at or below this duration")
	serverCmd.Flags().IntVar(&maxConcurrentProbes, "max-concurrent-probes", 10, "Certprobe worker pool size")
	serverCmd.Flags().IntVar(&eventHistoryLimit, "event-history-limit", envIntOrDefault("EVENT_HISTORY_LIMIT", eventmgr.DefaultHistoryLimit), "Number of recent events the /events history API can serve exactly; older queries return synthesized current-state entries instead of an error")
}

func envOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func notifyShutdownSignals(ch chan os.Signal) {
	if runtime.GOOS == "windows" {
		signal.Notify(ch, os.Interrupt)
		return
	}
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
}
