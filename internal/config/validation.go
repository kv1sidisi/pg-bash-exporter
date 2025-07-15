package config

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	metricRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	validTypes = map[string]bool{
		"gauge":   true,
		"counter": true,
	}
)

// Validate checks config for correctness.
// returns all validation errors
func (c *Config) Validate() error {
	var allErrors []error

	if err := c.Server.validate(); err != nil {
		allErrors = append(allErrors, err)
	}

	if err := c.Logging.validate(); err != nil {
		allErrors = append(allErrors, err)
	}

	if err := c.Global.validate(); err != nil {
		allErrors = append(allErrors, err)
	}

	if len(c.Metrics) == 0 {
		allErrors = append(allErrors, errors.New("at least one metric must be defined"))
	} else {
		for _, metric := range c.Metrics {
			if err := metric.validate(); err != nil {
				allErrors = append(allErrors, fmt.Errorf("metric '%s': %w", metric.Name, err))
			}
		}
	}

	return errors.Join(allErrors...)
}

func (s *Server) validate() error {
	var errs []error

	if s.ListenAddress == "" {
		errs = append(errs, errors.New("server.listen_address is required"))
	}

	if s.MetricsPath == "" {
		errs = append(errs, errors.New("server.metrics_path is required"))
	} else if s.MetricsPath[0] != '/' {
		errs = append(errs, errors.New("server.metrics_path must start with '/'"))
	}

	return errors.Join(errs...)
}

func (l *Logging) validate() error {
	validLevels := map[string]bool{
		"info":  true,
		"debug": true,
		"error": true,
	}

	if !validLevels[l.Level] {
		return fmt.Errorf("logging.level: %s is not valid. Valid levels: info, debug, error", l.Level)
	}
	return nil
}

func (g *Global) validate() error {
	var errs []error

	if g.Timeout <= 0 {
		errs = append(errs, errors.New("global.timeout must be > 0 (10s)"))
	}

	if g.CacheTTL <= 0 {
		errs = append(errs, errors.New("global.cache_ttl must be > 0 (5m)"))
	}

	if g.MaxConcurrent <= 0 {
		errs = append(errs, errors.New("global.max_concurrent must be > 0"))
	}

	return errors.Join(errs...)
}

func (m *Metric) validate() error {
	var errs []error

	if m.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if m.Help == "" {
		errs = append(errs, errors.New("help string is required"))
	}

	if !validTypes[m.Type] {
		errs = append(errs, errors.New("type is invalid. valid: gauge, counter"))
	}

	if m.Command == "" {
		errs = append(errs, errors.New("command is required"))
	}

	if err := validateLabels(m.Labels); err != nil {
		errs = append(errs, err)
	}

	for _, subMetric := range m.SubMetrics {
		if err := subMetric.validate(); err != nil {
			errs = append(errs, fmt.Errorf("sub-metric '%s': %w", subMetric.Name, err))
		}
	}

	return errors.Join(errs...)
}

func (sm *SubMetric) validate() error {
	var errs []error

	if sm.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if sm.Help == "" {
		errs = append(errs, errors.New("help string is required"))
	}

	if !validTypes[sm.Type] {
		errs = append(errs, errors.New("type is invalid. valid: gauge, counter"))
	}

	if sm.Field < 0 {
		errs = append(errs, errors.New("field must be >= 0"))
	}

	if err := validateLabels(sm.Labels); err != nil {
		errs = append(errs, err)
	}

	for _, dynLbl := range sm.DynamicLabels {
		if dynLbl.Name == "" {
			errs = append(errs, errors.New("dynamic_label name is required"))
		}
		if !metricRegex.MatchString(dynLbl.Name) {
			errs = append(errs, fmt.Errorf("dynamic_label name: %s is not valid", dynLbl.Name))
		}
		if dynLbl.Field < 0 {
			errs = append(errs, fmt.Errorf("dynamic_label name: %s, field must be >= 0", dynLbl.Name))
		}
	}

	return errors.Join(errs...)
}

func validateLabels(labels map[string]string) error {
	var errs []error

	for name, str := range labels {
		if !metricRegex.MatchString(name) {
			errs = append(errs, fmt.Errorf("label name %s is not valid", name))
		}
		if str == "" {
			errs = append(errs, fmt.Errorf("label %s requires value", name))
		}
	}

	return errors.Join(errs...)
}
