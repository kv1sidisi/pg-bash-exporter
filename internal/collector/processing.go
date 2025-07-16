package collector

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"pg-bash-exporter/internal/config"
	"strconv"
	"strings"
)

// getCommandOutput executes command from metric config.
// returns command output split into lines.
// returns error if command fails to execute.
func (c *Collector) getCommandOutput(metricConfig config.Metric) ([]string, error) {
	timeout := c.config.Global.Timeout

	if metricConfig.Timeout > 0 {
		timeout = metricConfig.Timeout
	}

	out, err := c.executor.ExecuteCommand(context.Background(), metricConfig.Command, timeout)
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(out), "\n"), nil
}

// collectSimpleMetric handles metric that are defined by single command and without sub-metrics
func (c *Collector) collectSimpleMetric(ch chan<- prometheus.Metric, metricConfig config.Metric) {
	lines, err := c.getCommandOutput(metricConfig)
	if err != nil {
		c.logger.Error("failed to execute command for metric", "metric", metricConfig.Name, "error", err)
		return
	}
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

// collectComplicatedMetric handles metric group defined with sub-metrics section.
// It runs one command and parses each line of the output to sub-metrics metrics.
func (c *Collector) collectComplicatedMetric(ch chan<- prometheus.Metric, metricConfig config.Metric) {
	lines, err := c.getCommandOutput(metricConfig)
	if err != nil {
		c.logger.Error("failed to execute command for metric", "metric", metricConfig.Name, "error", err)
		return
	}

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		for _, subMetric := range metricConfig.SubMetrics {
			if matched, err := c.matchPattern(line, subMetric.Match); !matched {
				c.logger.Error("invalid regex patterin in sub-metric", "sub-metric", subMetric.Name, "pattern", subMetric.Match, "error", err)
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
