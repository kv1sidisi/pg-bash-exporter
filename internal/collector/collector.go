package collector

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"pg-bash-exporter/internal/config"
	"pg-bash-exporter/internal/executor"
	"strconv"
	"strings"
)

type Collector struct {
	config   *config.Config
	logger   *slog.Logger
	executor executor.Executor
}

func NewCollector(cfg *config.Config, logger *slog.Logger, exec executor.Executor) *Collector {
	return &Collector{
		config:   cfg,
		logger:   logger,
		executor: exec,
	}
}

// mergeLabels creates a new map containing labels from parent and child metric.
//
// If label exists in both maps, value from child map is used.
//
// Allows sub-metrics to have own and parent labels, and override parent labels.
func mergeLabels(parent, child map[string]string) map[string]string {
	if parent == nil && child == nil {
		return nil
	}

	merged := make(map[string]string)

	for key, val := range parent {
		merged[key] = val
	}

	for key, val := range child {
		merged[key] = val
	}

	return merged
}

// getLabelNames gets names from DynamicLabel struct slice.
func getLabelNames(labels []config.DynamicLabel) []string {
	if len(labels) == 0 {
		return nil
	}

	names := make([]string, len(labels))

	for i, l := range labels {
		names[i] = l.Name
	}

	return names
}

// getLabelValues gets names from DynamicLabel struct slice and matches it with field number.
func getLabelValues(fields []string, labels []config.DynamicLabel) []string {
	if len(labels) == 0 {
		return nil
	}

	values := make([]string, len(labels))

	for i, l := range labels {
		if l.Field >= len(fields) {
			values[i] = ""
			continue
		}

		values[i] = fields[l.Field]
	}

	return values
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	for _, metricConfig := range c.config.Metrics {
		if len(metricConfig.SubMetrics) == 0 {
			desc := prometheus.NewDesc(
				metricConfig.Name,
				metricConfig.Help,
				nil,
				metricConfig.Labels)

			ch <- desc
			c.logger.Debug("simple metric description added", "metric", metricConfig.Name)
			continue
		}
		for _, subMetric := range metricConfig.SubMetrics {
			labels := mergeLabels(metricConfig.Labels, subMetric.Labels)

			dynLblNames := getLabelNames(subMetric.DynamicLabels)

			desc := prometheus.NewDesc(
				subMetric.Name,
				subMetric.Help,
				dynLblNames,
				labels,
			)
			ch <- desc
			c.logger.Debug("sub-metric description added", "sub-metric", subMetric.Name)
		}
	}

}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.logger.Info("Metrics collection started")

	for _, metricConfig := range c.config.Metrics {
		if len(metricConfig.SubMetrics) == 0 {
			c.collectSimpleMetric(ch, metricConfig)
		} else {
			c.collectComplicatedMetric(ch, metricConfig)
		}
	}

	c.logger.Info("Metrics collection finished")
}

func toPrometheusValueType(metricType string) prometheus.ValueType {
	switch metricType {
	case "gauge":
		return prometheus.GaugeValue
	case "counter":
		return prometheus.CounterValue
	default:
		return 0
	}
}

// collectSimpleMetric handles metric that are defined by single command with single output value and without sub-metrics
func (c *Collector) collectSimpleMetric(ch chan<- prometheus.Metric, metricConfig config.Metric) {
	out, err := executor.ExecuteCommand(context.Background(), metricConfig.Command)
	if err != nil {
		c.logger.Error("failed to execute command for metric", "metric", metricConfig.Name, "error", err)
	}

	result, err := strconv.ParseFloat(out, 64)
	if err != nil {
		c.logger.Error("failed to parse command output", "metric", metricConfig.Name, "error", err)
	}

	valueType := toPrometheusValueType(metricConfig.Type)
	if valueType == 0 {
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

	c.logger.Debug("metric collected successfully", "metric", metricConfig.Name)
}

// collectComplicatedMetric handles metric group defined with sub-metrics section.
// It runs one command and parses each line of the output to sub-metrics metrics.
func (c *Collector) collectComplicatedMetric(ch chan<- prometheus.Metric, metricConfig config.Metric) {
	out, err := executor.ExecuteCommand(context.Background(), metricConfig.Command)
	if err != nil {
		c.logger.Error("failed to execute command for metric", "metric", metricConfig.Name, "error", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		for _, subMetric := range metricConfig.SubMetrics {
			if subMetric.Field >= len(fields) {
				c.logger.Error("sub-metric`s field index out of range of command output fields", "sub-metric", subMetric.Name, "field_index", subMetric.Field, "line", line)
				continue
			}
			val, err := strconv.ParseFloat(fields[subMetric.Field], 64)
			if err != nil {
				c.logger.Error("failed to parse field for sub-metric", "sub-metric", subMetric.Name, "value", fields[subMetric.Field], "error", err)
				continue
			}

			valueType := toPrometheusValueType(subMetric.Type)
			if valueType == 0 {
				c.logger.Error("unsupported sub-metric type", "sub-metric", subMetric.Name, "type", subMetric.Type)
			}
			labels := mergeLabels(metricConfig.Labels, subMetric.Labels)

			dynLblNames := getLabelNames(subMetric.DynamicLabels)
			dynLblValues := getLabelValues(fields, subMetric.DynamicLabels)

			metric, err := prometheus.NewConstMetric(
				prometheus.NewDesc(subMetric.Name, subMetric.Help, dynLblNames, labels),
				valueType,
				val,
				dynLblValues...,
			)
			if err != nil {
				c.logger.Error("failed to create sub-metric", "sub-metric", subMetric.Name, "error", err)
			}
			ch <- metric
			c.logger.Debug("sub-metric collected successfully", "sub-metric", subMetric.Name)
		}
	}
}
