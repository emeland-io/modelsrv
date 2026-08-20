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
	"go.emeland.io/modelsrv/pkg/endpointprobe"
	"go.emeland.io/modelsrv/pkg/eventfilter"
	"go.emeland.io/modelsrv/pkg/filesensor"
	"go.uber.org/zap"
)

const otelConfigShutdownTimeout = 30 * time.Second

var serviceAddr string
var dataDir string
var sensorConfig string
var logLevel string
var logEncoding string
var metricsAddr string
var trustAuthHeaders bool
var auditorIdentity string
var auditorGroup string
var publicResourceTypes string
var subscribersFlag string
var eventHistoryLimit int
var otelConfigOut string
var otelConfigDebounce time.Duration
var otelCollectionInterval time.Duration
var otelListenAddr string
var otelExpiryThreshold time.Duration
var otelSubscribers []string

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "minimal model server for the Emerging Enterprise Landscape (EmELand).",
	Long:  `minimal model server instance that serves the model via REST API and provides a minimal web UI.`,

	RunE: runServer,
}

func runServer(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	cfg := zap.NewDevelopmentConfig()
	cfg.DisableStacktrace = true
	if logLevel != "" {
		var level zap.AtomicLevel
		if err := level.UnmarshalText([]byte(logLevel)); err != nil {
			return fmt.Errorf("invalid log level %q: %w", logLevel, err)
		}
		cfg.Level = level
	}
	switch logEncoding {
	case "", "console":
		cfg.Encoding = "console"
	case "json":
		cfg.Encoding = "json"
	default:
		return fmt.Errorf("invalid log encoding %q: must be console or json", logEncoding)
	}
	log, err := cfg.Build()
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	defer log.Sync() //nolint:errcheck

	logger := log.Sugar()

	b, err := backend.New(
		backend.WithEventHistoryLimit(eventHistoryLimit),
		backend.WithLogger(logger),
	)
	if err != nil {
		return fmt.Errorf("creating backend: %w", err)
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
		"sensorConfig", sensorConfig,
		"otelConfigOut", otelConfigOut,
	)
	logger.Infof("REST API: http://%s/api", serviceAddr)
	logger.Infof("Swagger UI: http://%s/swagger/", serviceAddr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var configWriter *endpointprobe.ConfigWriter
	var configSyncFilterID eventfilter.FilterID
	if otelConfigOut != "" {
		configWriter, err = endpointprobe.NewConfigWriter(endpointprobe.ConfigWriterConfig{
			Path:     otelConfigOut,
			Debounce: otelConfigDebounce,
			Logger:   logger,
			Opts: endpointprobe.CollectorConfigOptions{
				CollectionInterval: otelCollectionInterval,
				ListenAddr:         otelListenAddr,
				ExpiryThreshold:    otelExpiryThreshold,
				Subscribers:        otelSubscribers,
			},
		})
		if err != nil {
			return fmt.Errorf("invalid otel config writer configuration: %w", err)
		}
		go configWriter.Run(ctx)
		configSyncFilterID = b.GetChain().RegisterFilter(endpointprobe.NewConfigSyncFilter(configWriter))
		logger.Infow("otel config sync started",
			"path", otelConfigOut,
			"debounce", otelConfigDebounce,
		)
	}

	if sensorConfig != "" {
		cfg, err := filesensor.LoadConfigFile(sensorConfig)
		if err != nil {
			return fmt.Errorf("filesensor: could not load sensor config %s: %w", sensorConfig, err)
		}
		logger.Infow("file sensor: multi-source config", "path", sensorConfig, "sources", len(cfg.Sources))
		sources, err := filesensor.OpenSources(ctx, cfg)
		if err != nil {
			return fmt.Errorf("filesensor: could not open sources from %s: %w", sensorConfig, err)
		}
		summary := filesensor.ApplySources(ctx, sources, b.GetModel(), logger)
		if err := summary.Err(); err != nil {
			return fmt.Errorf("filesensor: %w", err)
		}
		registerStartupSubscribers(b.GetEventManager(), parseCommaSeparatedList(subscribersFlag), logger)
		filesensor.StartSources(ctx, sources, b.GetModel(), logger)
	} else {
		logger.Info("file sensor: watching for YAML/JSON/CSV in data directory")
		filesensor.ApplyExisting(dataPath, b.GetModel(), logger)
		registerStartupSubscribers(b.GetEventManager(), parseCommaSeparatedList(subscribersFlag), logger)
		filesensor.StartWatch(ctx, dataPath, b.GetModel(), logger)
	}

	webOpts := endpoint.WebListenerOptions{
		TrustAuthHeaders: trustAuthHeaders,
		AuthzConfig: authz.Config{
			AuditorIdentity: auditorIdentity,
			AuditorGroup:    auditorGroup,
			PublicTypes:     authz.ParsePublicResourceTypes(publicResourceTypes),
		},
		Logger: logger,
	}
	if err := endpoint.StartWebListener(b.GetModel(), b.GetEventManager(), serviceAddr, webOpts); err != nil {
		return fmt.Errorf("starting web listener: %w", err)
	}

	if metricsAddr != "" {
		if err := endpoint.StartMetricsListener(metricsAddr); err != nil {
			return fmt.Errorf("starting metrics listener: %w", err)
		}
	}

	logger.Info("server is running (Ctrl+C to stop)")

	sigCh := make(chan os.Signal, 1)
	notifyShutdownSignals(sigCh)
	sig := <-sigCh
	logger.Infow("shutdown signal received", "signal", sig.String())

	cancel()

	if configWriter != nil {
		b.GetChain().Unregister(configSyncFilterID)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), otelConfigShutdownTimeout)
		done := make(chan struct{})
		go func() {
			configWriter.Wait()
			close(done)
		}()

		select {
		case <-done:
			logger.Info("otel config sync stopped")
		case <-shutdownCtx.Done():
			logger.Warn("shutdown timeout waiting for otel config writer")
		}
		shutdownCancel()
	}

	endpoint.StopWebListener()
	logger.Info("goodbye")
	return nil
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVarP(&serviceAddr, "service-addr", "a", envOrDefault("SERVICE_ADDR", ":8080"), "The address the service listens on")
	serverCmd.Flags().StringVar(&dataDir, "data-dir", envOrDefault("DATA_DIR", "data"), "Directory to watch for landscape documents (.yaml/.yml/.json/.csv); relative paths are resolved from the process working directory")
	serverCmd.Flags().StringVar(&sensorConfig, "sensor-config", envOrDefault("SENSOR_CONFIG", ""), "Optional YAML file listing Sources (file://, https://, s3://) and per-glob parser options; when set, overrides --data-dir")
	serverCmd.Flags().StringVar(&logLevel, "log-level", envOrDefault("LOG_LEVEL", ""), "Log level: debug, info, warn, error (default: debug in dev mode)")
	serverCmd.Flags().StringVar(&logEncoding, "log-encoding", envOrDefault("LOG_ENCODING", ""), "Log encoding: console, json (default: console)")
	serverCmd.Flags().StringVar(&metricsAddr, "metrics-addr", envOrDefault("METRICS_ADDR", ""), "If set, serve /metrics on a separate port (e.g. :9090); otherwise metrics are on the main port")
	serverCmd.Flags().BoolVar(&trustAuthHeaders, "trust-auth-headers", envOrDefault("TRUST_AUTH_HEADERS", "") == "true", "Trust X-Auth-* identity headers from the BFF and enforce ownership visibility")
	serverCmd.Flags().StringVar(&auditorIdentity, "auditor-identity", envOrDefault("AUDITOR_IDENTITY", ""), "OIDC subject treated as auditor when matching X-Auth-Subject")
	serverCmd.Flags().StringVar(&auditorGroup, "auditor-group", envOrDefault("AUDITOR_GROUP", ""), "Group id treated as auditor when present in X-Auth-Groups")
	serverCmd.Flags().StringVar(&publicResourceTypes, "public-resource-types", envOrDefault("PUBLIC_RESOURCE_TYPES", ""), "Comma-separated resource types always visible (e.g. ContextType,FindingType)")
	serverCmd.Flags().StringVar(&subscribersFlag, "subscribers", envOrDefault("SUBSCRIBERS", ""), "Comma-separated downstream modelsrv base API URLs to pre-register (e.g. http://host:8080/api)")
	serverCmd.Flags().IntVar(&eventHistoryLimit, "event-history-limit", envIntOrDefault("EVENT_HISTORY_LIMIT", eventmgr.DefaultHistoryLimit), "Number of recent events the /events history API can serve exactly; older queries return synthesized current-state entries instead of an error")
	serverCmd.Flags().StringVar(&otelConfigOut, "otel-config-out", envOrDefault("OTEL_CONFIG_OUT", ""), "If set, keep an OTel collector config at this path in sync with ApiInstance endpoint annotations")
	serverCmd.Flags().DurationVar(&otelConfigDebounce, "otel-config-debounce", envDurationOrDefault("OTEL_CONFIG_DEBOUNCE", 2*time.Second), "Debounce window before rewriting --otel-config-out")
	serverCmd.Flags().DurationVar(&otelCollectionInterval, "otel-collection-interval", 5*time.Minute, "collection_interval for the http_check receiver in --otel-config-out")
	serverCmd.Flags().StringVar(&otelListenAddr, "otel-listen-addr", "0.0.0.0:24200", "listen_addr for the emeland exporter in --otel-config-out")
	serverCmd.Flags().DurationVar(&otelExpiryThreshold, "otel-expiry-threshold", 30*24*time.Hour, "Expiry threshold for the emeland exporter in --otel-config-out")
	serverCmd.Flags().StringArrayVar(&otelSubscribers, "otel-subscriber", nil, "Downstream modelsrv URL for the emeland exporter in --otel-config-out (repeatable)")
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
