package collector

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"pg-bash-exporter/internal/config"
	"pg-bash-exporter/internal/executor"
	"strconv"
)

type Collector struct {
	config *config.Config
	logger *slog.Logger
}

func NewCollector(cfg *config.Config, logger *slog.Logger) *Collector {
	return &Collector{
		config: cfg,
		logger: logger,
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	for _, metricConfig := range c.config.Metrics {
		if len(metricConfig.SubMetrics) > 0 {
			c.logger.Info("DESC skipping metric ")
			continue
		}
		desc := prometheus.NewDesc(
			metricConfig.Name,
			metricConfig.Help,
			nil,
			metricConfig.Labels)
		ch <- desc
	}
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.logger.Info("Metrics collection started")

	for _, metricConfig := range c.config.Metrics {
		if len(metricConfig.SubMetrics) > 0 {
			c.logger.Info("COLLECT skipping metric ")
			continue
		}
		out, err := executor.ExecuteCommand(context.Background(), metricConfig.Command)
		if err != nil {
			c.logger.Error("failed to execute command for metric", "metric", metricConfig.Name, "error", err)
		}

		result, err := strconv.ParseFloat(out, 64)
		if err != nil {
			c.logger.Error("failed to parse command output", "metric", metricConfig.Name, "error", err)
		}

		var valueType prometheus.ValueType
		switch metricConfig.Type {
		case "gauge":
			valueType = prometheus.GaugeValue
		case "counter":
			valueType = prometheus.CounterValue
		default:
			c.logger.Error("unsupported metric type", "metric", metricConfig.Name, "type", metricConfig.Type)
		}

		metric, err := prometheus.NewConstMetric(
			prometheus.NewDesc(metricConfig.Name, metricConfig.Help, nil, metricConfig.Labels),
			valueType,
			result)
		if err != nil {
			c.logger.Error("failed to create metric", "metric", metricConfig.Name, "error", err)
		}
		ch <- metric
	}

	c.logger.Info("Metrics collection finished")
}
