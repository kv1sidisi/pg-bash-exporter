package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"pg-bash-exporter/internal/cache"
	"pg-bash-exporter/internal/collector"
	"pg-bash-exporter/internal/config"
	"pg-bash-exporter/internal/executor"
	"testing"
)

func setupCollector(cfg *config.Config, configPath string) *collector.Collector {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	exec := &executor.CommandExecutor{}
	cache := cache.New()
	return collector.NewCollector(cfg, logger, exec, cache, configPath)
}

func TestReloadEndpoint(t *testing.T) {
	configV1 := `
logging:
  level: "info"
metrics:
  - name: "metric_v1"
    help: "help v1"
    type: "gauge"
    command: "echo 1"
`
	configV2 := `
logging:
  level: "info"
metrics:
  - name: "metric_v2"
    help: "help v2"
    type: "counter"
    command: "echo 2"
`

	tmpfile, err := os.CreateTemp(t.TempDir(), "test-config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configV1)); err != nil {
		t.Fatalf("failed to write v1 config: %v", err)
	}

	// Initial load
	var cfg config.Config
	if err := config.Load(tmpfile.Name(), &cfg); err != nil {
		t.Fatalf("failed to load v1 config: %v", err)
	}

	collector := setupCollector(&cfg, tmpfile.Name())

	if collector.GetConfig().Metrics[0].Name != "metric_v1" {
		t.Fatalf("expected initial metric to be metric_v1, got %s", collector.GetConfig().Metrics[0].Name)
	}

	// Setup server
	mux := newRouter(collector, prometheus.NewRegistry(), "/metrics")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Update config file
	if err := os.WriteFile(tmpfile.Name(), []byte(configV2), 0644); err != nil {
		t.Fatalf("failed to write v2 config: %v", err)
	}

	// Send reload request
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/reload", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	// Check if config was reloaded
	if len(collector.GetConfig().Metrics) != 1 {
		t.Fatalf("expected 1 metric after reload, got %d", len(collector.GetConfig().Metrics))
	}
	if collector.GetConfig().Metrics[0].Name != "metric_v2" {
		t.Errorf("expected metric_v2 after reload, got %s", collector.GetConfig().Metrics[0].Name)
	}
	if collector.GetConfig().Metrics[0].Type != "counter" {
		t.Errorf("expected counter type after reload, got %s", collector.GetConfig().Metrics[0].Type)
	}

	// Send reload request with wrong method
	req, err = http.NewRequest(http.MethodPost, srv.URL+"/reload", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", res.StatusCode)
	}
}
