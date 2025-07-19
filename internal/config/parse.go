package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultTimeout       = 30 * time.Second
	DefaultCacheTTL      = 3 * time.Second
	DefaultMaxConcurrent = 10
)

// GetPath returns config file path with priority: flag > env > default
func GetPath(configPath string) string {
	if configPath != "" {
		return configPath
	}

	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		return envPath
	}

	return "configs/config.example.yaml"
}

func (c *Config) applyDefaults() {
	if c.Global.Timeout == 0 {
		c.Global.Timeout = DefaultTimeout
	}
	if c.Global.CacheTTL == 0 {
		c.Global.CacheTTL = DefaultCacheTTL
	}
	if c.Global.MaxConcurrent == 0 {
		c.Global.MaxConcurrent = DefaultMaxConcurrent
	}
}

// Load reads and parses a YAML configuration file into Config struct.
//
// Returns an error if the file can`t be read or if the YAML is invalid. Panics if cfg is nil.
func Load(path string, cfg *Config) error {
	if cfg == nil {
		panic("provided Config is nil")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse YAML from file %s: %w", path, err)
	}

	cfg.applyDefaults()

	err = cfg.Validate()
	if err != nil {
		return fmt.Errorf("configuration is invalid: %s", err)
	}

	return nil
}
