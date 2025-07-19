package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

var (
	ValidationFlag bool
	configPath     string
)

func init() {
	flag.BoolVar(&ValidationFlag, "validate-config", false, "Validate the configuration file.")
	flag.StringVar(&configPath, "config", "", "Path to the configuration file.")
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage of pg-bash-exporter:

pg-bash-exporter [flags]

Flags:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Environment variables:
  CONFIG_PATH: Path to the configuration file. (e.g., "/etc/pg-bash-exporter/config.yaml")
  LISTEN_ADDRESS: Server listen address. (e.g., "0.0.0.0:9876")
  METRICS_PATH: Metrics path. (e.g., "/metrics")
  BLACKLIST_FILE_PATH: Path to a YAML file with blacklisted commands.
`)
	}

	flag.Parse()

	configPath := config.GetPath(configPath)

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

	config.SetupLogger(cfg.Logging)

	slog.Info("Configuration loaded and logger initialized successfully")

	cache := cache.New()

	exec := &executor.CommandExecutor{}

	metricsCollector := collector.NewCollector(&cfg, slog.Default(), exec, cache, configPath)

	registry := prometheus.NewRegistry()
	registry.MustRegister(metricsCollector)

	registry.MustRegister(collector.Checks)
	registry.MustRegister(collector.CheckDuration)
	registry.MustRegister(collector.CommandErrors)
	registry.MustRegister(collector.CacheHits)
	registry.MustRegister(collector.CacheMisses)
	registry.MustRegister(collector.ConfigReloads)
	registry.MustRegister(collector.ConfigReloadErrors)
	registry.MustRegister(collector.CommandDuration)
	registry.MustRegister(collector.ConcurrentCommands)

	mux := http.NewServeMux()
	mux.Handle(cfg.Server.MetricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	mux.HandleFunc("/-/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "This endpoint requires a POST request.\n")
			return
		}

		if err := metricsCollector.ReloadConfig(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to reload config: %s\n", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Config reloaded successfully.\n")
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
<head><title>PG-Bash Exporter</title></head>
<body>
<h1>PG-Bash Exporter</h1>
<p><a href='` + cfg.Server.MetricsPath + `'>Metrics</a></p>
</body>
</html>`))
	})
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
