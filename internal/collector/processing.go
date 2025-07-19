package collector

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"pg-bash-exporter/internal/config"
	"strconv"
	"strings"
	"time"
)

// getCommandOutput executes command from metric config.
// returns command output split into lines.
// returns error if command fails to execute.
func (c *Collector) getCommandOutput(metricConfig config.Metric) ([]string, error) {
	if isCommandBlacklisted(metricConfig, c.config.Global) {
		return nil, fmt.Errorf("command '%s' for metric '%s' is in black list", metricConfig.Command, metricConfig.Name)
	}

	cacheKey := generateCacheKey(metricConfig.Name, metricConfig.Command)
	val, err, ok := c.cache.Get(cacheKey)

	ttl := c.config.Global.CacheTTL
	if metricConfig.CacheTTL > config.DefaultCacheTTL {
		ttl = metricConfig.CacheTTL
	}

	if ok {
		CacheHits.Inc()
		c.logger.Debug("cache taken", "command", metricConfig.Command)
		return strings.Split(strings.TrimSpace(val), "\n"), err
	}
	CacheMisses.Inc()

	timeout := c.config.Global.Timeout

	if metricConfig.Timeout > 0 {
		timeout = metricConfig.Timeout
	}

	shell := c.config.Global.Shell
	if metricConfig.Shell != "" {
		shell = metricConfig.Shell
	}
	if shell == "" {
		shell = "bash"
	}

	start := time.Now()
	out, err := c.executor.ExecuteCommand(context.Background(), shell, metricConfig.Command, timeout)
	duration := time.Since(start).Seconds()
	CommandDuration.WithLabelValues(metricConfig.Name).Observe(duration)

	c.cache.Set(cacheKey, out, err, ttl)

	if err != nil {
		CommandErrors.WithLabelValues(metricConfig.Name).Inc()
		return nil, err
	}

	return strings.Split(strings.TrimSpace(out), "\n"), nil
}

// collectSimpleMetric handles metric that are defined by single command and without postfix-metrics
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
			c.logger.Error("failed to create postfix-metric", "postfix-metric", metricConfig.Name, "error", err)
			continue
		}
		ch <- metric
	}
}

// collectComplicatedMetric handles metric group defined with postfix-metrics section.
// It runs one command and parses each line of the output to postfix-metrics metrics.
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

		for _, postfixMetric := range metricConfig.PostfixMetrics {
			if matched, err := c.matchPattern(line, postfixMetric.Match); !matched || err != nil {
				if err != nil {
					c.logger.Error("invalid regex patterin in postfix-metric", "postfix-metric", postfixMetric.Name, "pattern", postfixMetric.Match, "error", err)
				}
				continue
			}

			if postfixMetric.Field >= len(fields) {
				c.logger.Error("postfix-metric`s field index out of range of command output fields", "postfix-metric", postfixMetric.Name, "field_index", postfixMetric.Field, "line", line)
				continue
			}
			val, err := strconv.ParseFloat(fields[postfixMetric.Field], 64)
			if err != nil {
				c.logger.Error("failed to parse field for postfix-metric", "postfix-metric", postfixMetric.Name, "value", fields[postfixMetric.Field], "error", err)
				continue
			}

			valueType, err := toPrometheusValueType(postfixMetric.Type)
			if err != nil {
				c.logger.Error(err.Error(), "postfix-metric", postfixMetric.Name)
				return
			}
			labels := mergeLabels(metricConfig.Labels, postfixMetric.Labels)

			dynLblNames := getLabelNames(postfixMetric.DynamicLabels)
			dynLblValues := getLabelValues(fields, postfixMetric.DynamicLabels)

			fullName := metricConfig.Name + "_" + postfixMetric.Name

			metric, err := prometheus.NewConstMetric(
				prometheus.NewDesc(fullName, postfixMetric.Help, dynLblNames, labels),
				valueType,
				val,
				dynLblValues...,
			)
			if err != nil {
				c.logger.Error("failed to create postfix-metric", "postfix-metric", postfixMetric.Name, "error", err)
				continue
			}
			ch <- metric
		}
	}
}
