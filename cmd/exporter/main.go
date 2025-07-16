package main

import (
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"pg-bash-exporter/internal/cache"
	"pg-bash-exporter/internal/collector"
	"pg-bash-exporter/internal/config"
	"pg-bash-exporter/internal/executor"
)

var ValidationFlag bool

func init() {
	flag.BoolVar(&ValidationFlag, "validate-config", false, "Validate the configuration file.")
}

func main() {
	flag.Parse()

	configPath := config.GetPath()

	if ValidationFlag {
		var cfg config.Config

		fmt.Println("Validating configuration file:", configPath)

		if err := config.Load(configPath, &cfg); err != nil {
			log.Fatalf("configuration is invalid: %v", err)
		}

		fmt.Println("Configuration is valid.")

		os.Exit(0)
	}

	var cfg config.Config

	if err := config.Load(configPath, &cfg); err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	setupLogger(cfg.Logging)

	slog.Info("Configuration loaded and logger initialized successfully")

	cache := cache.New()

	exec := &executor.BashExecutor{}

	collector := collector.NewCollector(&cfg, slog.Default(), exec, cache)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)
	metricsHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	http.Handle("/metrics", metricsHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><head><title>PG Bash Exporter</title></head><body>PG Bash Exporter<p><a href="/metrics">Metrics</a></p></body></html>`))
	})

	slog.Info("Starting pg-bash-exporter server",
		"listen_address", cfg.Server.ListenAddress,
		"metrics_path", cfg.Server.MetricsPath,
	)
	if err := http.ListenAndServe(cfg.Server.ListenAddress, nil); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func setupLogger(cfg config.Logging) {
	var logOutput io.Writer

	if cfg.Path == "" {
		logOutput = os.Stdout
	} else {
		file, err := os.OpenFile(cfg.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
		if err != nil {
			log.Fatalf("failed to open log file: %v", err)
		}
		logOutput = file
	}

	var logLevel slog.Level

	switch cfg.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(logOutput, &slog.HandlerOptions{Level: logLevel})

	logger := slog.New(handler)

	slog.SetDefault(logger)
}
