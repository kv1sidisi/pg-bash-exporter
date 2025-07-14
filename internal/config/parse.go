package config

import (
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var configPathFlag string

// init sets flags with package initialization
func init() {
	flag.StringVar(&configPathFlag, "config", "", "Path to the configuration file. Priority: flag > CONFIG_PATH env var > default value.")
}

// GetPath returns config file path with priority: flag > env > default
func GetPath() string {
	if configPathFlag != "" {
		return configPathFlag
	}

	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		return envPath
	}

	return "configs/config.yaml"
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

	return nil
}
