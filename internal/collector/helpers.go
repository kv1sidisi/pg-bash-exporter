package collector

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"pg-bash-exporter/internal/config"
	"regexp"
	"strings"
)

// mergeLabels creates a new map containing labels from parent and child metric.
// If label exists in both maps, value from child map is used.
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

// toPrometheusValueType converts metric type string into a Prometheus ValueType.
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

// matchPattern checks if line matches provided regex pattern.
func (c *Collector) matchPattern(line, match string) (bool, error) {
	if match == "" {
		return true, nil
	}
	matched, err := regexp.MatchString(match, line)
	if err != nil {
		return false, err
	}

	return matched, nil
}

// isCommandBlacklisted checks if command is restricted by blacklist.
// metric can skip check by setting `ignore_blacklist: true` in config.
func isCommandBlacklisted(metric config.Metric, globalConfig config.Global) bool {
	if metric.IgnoreBlacklist {
		return false
	}

	fields := strings.Fields(metric.Command)
	if len(fields) == 0 {
		return false
	}
	executable := fields[0]

	for _, blacklistedCmd := range globalConfig.CommandBlacklist {
		if executable == blacklistedCmd {
			return true
		}
	}

	return false
}

// generateCacheKey creates a unique key for caching.
func generateCacheKey(metricName, command string) string {
	return fmt.Sprintf("%s::%s", metricName, command)
}
