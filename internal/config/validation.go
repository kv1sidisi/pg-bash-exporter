package config

import (
	"fmt"
	"regexp"
	"strings"
)

var metricRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Validate checks config for correctness.
// returns all validation errors
func (c *Config) Validate() error {
	var allErrors []string

	if serverErrors := c.Server.validate(); len(serverErrors) > 0 {
		allErrors = append(allErrors, serverErrors...)
	}

	if loggingErrors := c.Logging.validate(); len(loggingErrors) > 0 {
		allErrors = append(allErrors, loggingErrors...)
	}

	if globalErrors := c.Global.validate(); len(globalErrors) > 0 {
		allErrors = append(allErrors, globalErrors...)
	}

	if len(c.Metrics) == 0 {
		allErrors = append(allErrors, "at least one metric must be defined")
	} else {
		for _, metric := range c.Metrics {
			if metricErrors := metric.validate(); len(metricErrors) > 0 {
				for _, err := range metricErrors {
					allErrors = append(allErrors, fmt.Sprintf("metric '%s': %s", metric.Name, err))
				}
			}
		}
	}

	if len(allErrors) > 0 {
		return fmt.Errorf(strings.Join(allErrors, "; "))
	}

	return nil
}

func (s *Server) validate() []string {
	var errs []string

	if s.ListenAddress == "" {
		errs = append(errs, "server.listen_address is required")
	}

	if s.MetricsPath == "" {
		errs = append(errs, "server.metrics_path is required")
	} else if s.MetricsPath[0] != '/' {
		errs = append(errs, "server.metrics_path must start with '/'")
	}

	return errs
}

func (l *Logging) validate() []string {
	var errs []string

	validLevels := map[string]bool{
		"info":       true,
		"debug":      true,
		"production": true,
		"error":      true,
	}

	if !validLevels[l.Level] {
		errs = append(errs, fmt.Sprintf("logging.level: %s is not valid. Valid levels: info, debug, production, error", l.Level))
	}
	return errs
}

func (g *Global) validate() []string {
	var errs []string

	if g.Timeout <= 0 {
		errs = append(errs, "global.timeout must be > 0 (10s)")
	}

	if g.CacheTTL <= 0 {
		errs = append(errs, "global.cache_ttl must be > 0 (5m)")
	}

	if g.MaxConcurrent <= 0 {
		errs = append(errs, "global.max_concurrent must be > 0")
	}

	return errs
}

func (m *Metric) validate() []string {
	var errs []string

	if m.Name == "" {
		errs = append(errs, "name is required")
	}

	if m.Help == "" {
		errs = append(errs, "help string is required")
	}

	validTypes := map[string]bool{
		"gauge": true,
		"count": true,
	}

	if !validTypes[m.Type] {
		errs = append(errs, "type is invalid. valid: gauge, count")
	}

	if m.Command == "" {
		errs = append(errs, "command is required")
	}

	if labelErrors := validateLabels(m.Labels); len(labelErrors) > 0 {
		errs = append(errs, labelErrors...)
	}

	for _, subMetric := range m.SubMetrics {
		if subMetricErrors := subMetric.validate(); len(subMetricErrors) > 0 {
			for _, err := range subMetricErrors {
				errs = append(errs, fmt.Sprintf("sub-metric '%s': %s", subMetric.Name, err))
			}
		}
	}

	return errs
}

func (sm *SubMetric) validate() []string {
	var errs []string

	if sm.Name == "" {
		errs = append(errs, "name is required")
	}

	if sm.Help == "" {
		errs = append(errs, "help string is required")
	}

	validTypes := map[string]bool{
		"gauge": true,
		"count": true,
	}

	if !validTypes[sm.Type] {
		errs = append(errs, "type is invalid. valid: gauge, count")
	}

	if sm.Field < 0 {
		errs = append(errs, "field must be >= 0")
	}

	if labelErrors := validateLabels(sm.Labels); len(labelErrors) > 0 {
		errs = append(errs, labelErrors...)
	}

	for _, dynLbl := range sm.DynamicLabels {
		if dynLbl.Name == "" {
			errs = append(errs, "dynamic_label name is required")
		}
		if !metricRegex.MatchString(dynLbl.Name) {
			errs = append(errs, fmt.Sprintf("dynamic_label name: %s is not valid", dynLbl.Name))
		}
		if dynLbl.Field < 0 {
			errs = append(errs, fmt.Sprintf("dynamic_label name: %s ,field must be >= 0", dynLbl.Name))
		}
	}

	return errs
}

func validateLabels(labels map[string]string) []string {
	var errs []string

	for name, str := range labels {
		if !metricRegex.MatchString(name) {
			errs = append(errs, fmt.Sprintf("label name %s id not valid", name))
		}
		if str == "" {
			errs = append(errs, fmt.Sprintf("label %s requires value", name))
		}
	}

	return errs
}
