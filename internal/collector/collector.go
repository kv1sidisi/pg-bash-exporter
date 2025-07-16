package collector

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"pg-bash-exporter/internal/config"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Executor interface {
	ExecuteCommand(ctx context.Context, command string, timeout time.Duration) (string, error)
}

type Collector struct {
	config   *config.Config
	logger   *slog.Logger
	executor Executor
}

func NewCollector(cfg *config.Config, logger *slog.Logger, exec Executor) *Collector {
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
			dynLblNames := getLabelNames(metricConfig.DynamicLabels)

			desc := prometheus.NewDesc(
				metricConfig.Name,
				metricConfig.Help,
				dynLblNames,
				metricConfig.Labels)

			ch <- desc
			c.logger.Debug("simple metric description added", "metric", metricConfig.Name)
			continue
		}
		for _, subMetric := range metricConfig.SubMetrics {
			labels := mergeLabels(metricConfig.Labels, subMetric.Labels)

			dynLblNames := getLabelNames(subMetric.DynamicLabels)

			fullName := metricConfig.Name + "_" + subMetric.Name

			desc := prometheus.NewDesc(
				fullName,
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

func toPrometheusValueType(metricType string) (prometheus.ValueType, error) {
	switch metricType {
	case "gauge":
		return prometheus.GaugeValue, nil
	case "counter":
		return prometheus.CounterValue, nil
	default:
		return 0, fmt.Errorf("unsupported metric type: %s", metricType)
	}
}

// collectSimpleMetric handles metric that are defined by single command with single output value and without sub-metrics
func (c *Collector) collectSimpleMetric(ch chan<- prometheus.Metric, metricConfig config.Metric) {
	timeout := c.config.Global.Timeout

	if metricConfig.Timeout > 0 {
		timeout = metricConfig.Timeout
	}
	out, err := c.executor.ExecuteCommand(context.Background(), metricConfig.Command, timeout)
	if err != nil {
		c.logger.Error("failed to execute command for metric", "metric", metricConfig.Name, "error", err)
		return
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if metricConfig.Field >= len(fields) {
			c.logger.Error("metric`s field index out of range of command output fields", "metric", metricConfig.Name, "field_index", metricConfig.Field, "line", line)
			continue
		}

		val, err := strconv.ParseFloat(fields[metricConfig.Field], 64)
		if err != nil {
			c.logger.Error("failed to parse field for metric", "metric", metricConfig.Name, "value", fields[metricConfig.Field], "error", err)
			continue
		}

		valueType, err := toPrometheusValueType(metricConfig.Type)
		if err != nil {
			c.logger.Error(err.Error(), "metric", metricConfig.Name)
			return
		}

		dynLblNames := getLabelNames(metricConfig.DynamicLabels)
		dynLblValues := getLabelValues(fields, metricConfig.DynamicLabels)

		metric, err := prometheus.NewConstMetric(
			prometheus.NewDesc(metricConfig.Name, metricConfig.Help, dynLblNames, metricConfig.Labels),
			valueType,
			val,
			dynLblValues...,
		)
		if err != nil {
			c.logger.Error("failed to create sub-metric", "sub-metric", metricConfig.Name, "error", err)
			continue
		}
		ch <- metric
	}
	c.logger.Debug("metric collected successfully", "metric", metricConfig.Name)
}

func (c *Collector) matchPattern(line, match, metricName string) bool {
	if match == "" {
		return true
	}
	matched, err := regexp.MatchString(match, line)
	if err != nil {
		c.logger.Error("invalid regex patterin in sub-metric", "metric", metricName, "pattern", match, "error", err)
		return false
	}

	return matched
}

// collectComplicatedMetric handles metric group defined with sub-metrics section.
// It runs one command and parses each line of the output to sub-metrics metrics.
func (c *Collector) collectComplicatedMetric(ch chan<- prometheus.Metric, metricConfig config.Metric) {
	timeout := c.config.Global.Timeout

	if metricConfig.Timeout > 0 {
		timeout = metricConfig.Timeout
	}

	out, err := c.executor.ExecuteCommand(context.Background(), metricConfig.Command, timeout)
	if err != nil {
		c.logger.Error("failed to execute command for metric", "metric", metricConfig.Name, "error", err)
		return
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		for _, subMetric := range metricConfig.SubMetrics {

			if !c.matchPattern(line, subMetric.Match, subMetric.Name) {
				continue
			}

			if subMetric.Field >= len(fields) {
				c.logger.Error("sub-metric`s field index out of range of command output fields", "sub-metric", subMetric.Name, "field_index", subMetric.Field, "line", line)
				continue
			}
			val, err := strconv.ParseFloat(fields[subMetric.Field], 64)
			if err != nil {
				c.logger.Error("failed to parse field for sub-metric", "sub-metric", subMetric.Name, "value", fields[subMetric.Field], "error", err)
				continue
			}

			valueType, err := toPrometheusValueType(subMetric.Type)
			if err != nil {
				c.logger.Error(err.Error(), "sub-metric", subMetric.Name)
				return
			}
			labels := mergeLabels(metricConfig.Labels, subMetric.Labels)

			dynLblNames := getLabelNames(subMetric.DynamicLabels)
			dynLblValues := getLabelValues(fields, subMetric.DynamicLabels)

			fullName := metricConfig.Name + "_" + subMetric.Name

			metric, err := prometheus.NewConstMetric(
				prometheus.NewDesc(fullName, subMetric.Help, dynLblNames, labels),
				valueType,
				val,
				dynLblValues...,
			)
			if err != nil {
				c.logger.Error("failed to create sub-metric", "sub-metric", subMetric.Name, "error", err)
				continue
			}
			ch <- metric
			c.logger.Debug("sub-metric collected successfully", "sub-metric", subMetric.Name)
		}
	}
}
