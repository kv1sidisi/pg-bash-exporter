package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"pg-bash-exporter/internal/cache"
	"pg-bash-exporter/internal/collector"
	"pg-bash-exporter/internal/config"
	"pg-bash-exporter/internal/executor"
	"syscall"
	"time"
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

	metricsCollector := collector.NewCollector(&cfg, slog.Default(), exec, cache, configPath)

	registry := prometheus.NewRegistry()
	registry.MustRegister(metricsCollector)

	registry.MustRegister(collector.Checks)
	registry.MustRegister(collector.CheckDuration)
	registry.MustRegister(collector.CommandErrors)
	registry.MustRegister(collector.CacheHits)
	registry.MustRegister(collector.CacheMisses)

	mux := http.NewServeMux()
	mux.Handle(cfg.Server.MetricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:    cfg.Server.ListenAddress,
		Handler: mux,
	}

	go func() {
		slog.Info("Starting pg-bash-exporter server",
			"listen_address", cfg.Server.ListenAddress,
			"metrics_path", cfg.Server.MetricsPath,
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	hReload := make(chan os.Signal, 1)
	signal.Notify(hReload, syscall.SIGHUP)

	go func() {
		for {
			<-hReload
			slog.Info("received SIGHUP, attempting to reload config")
			if err := metricsCollector.ReloadConfig(); err != nil {
				slog.Error("config reload failed", "error", err)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown:", "error", err)
		os.Exit(1)
	}

	slog.Info("server exiting")
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
