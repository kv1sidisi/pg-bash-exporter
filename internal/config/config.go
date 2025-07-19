package config

import "time"

type Config struct {
	Server  Server   `yaml:"server"`
	Logging Logging  `yaml:"logging"`
	Global  Global   `yaml:"global"`
	Metrics []Metric `yaml:"metrics"`
}

type Server struct {
	ListenAddress string `yaml:"listen_address"`
	MetricsPath   string `yaml:"metrics_path"`
}

type Logging struct {
	Level string `yaml:"level"`
	Path  string `yaml:"path"`
}

type Global struct {
	Timeout          time.Duration `yaml:"timeout,omitempty"`
	CacheTTL         time.Duration `yaml:"cache_ttl,omitempty"`
	MaxConcurrent    int           `yaml:"max_concurrent,omitempty"`
	CommandBlacklist []string      `yaml:"command_blacklist,omitempty"`
	Shell            string        `yaml:"shell,omitempty"`
}

type Metric struct {
	Name            string            `yaml:"name"`
	Help            string            `yaml:"help"`
	Type            string            `yaml:"type"`
	Command         string            `yaml:"command"`
	Timeout         time.Duration     `yaml:"timeout,omitempty"`
	CacheTTL        time.Duration     `yaml:"cache_ttl,omitempty"`
	Labels          map[string]string `yaml:"labels,omitempty"`
	PostfixMetrics  []PostfixMetric   `yaml:"postfix_metrics,omitempty"`
	IgnoreBlacklist bool              `yaml:"ignore_blacklist,omitempty"`
	Field           int               `yaml:"field,omitempty"`
	DynamicLabels   []DynamicLabel    `yaml:"dynamic_labels,omitempty"`
	Shell           string            `yaml:"shell,omitempty"`
}

type PostfixMetric struct {
	Name          string            `yaml:"name"`
	Help          string            `yaml:"help"`
	Type          string            `yaml:"type"`
	Field         int               `yaml:"field"`
	Match         string            `yaml:"match,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty"`
	DynamicLabels []DynamicLabel    `yaml:"dynamic_labels,omitempty"`
}

type DynamicLabel struct {
	Name  string `yaml:"name"`
	Field int    `yaml:"field"`
}
