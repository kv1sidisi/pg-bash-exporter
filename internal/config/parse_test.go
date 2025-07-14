package config_test

import (
	"os"
	"pg-bash-exporter/internal/config"
	"strings"
	"testing"
)

func TestLoadSuccess(t *testing.T) {
	good_yaml := `
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: "1s"
  cache_ttl: "1s"
  max_concurrent: 1
metrics:
  - name: "my_metric"
    help: "help"
    type: "gauge"
    command: "echo 1"
`
	f, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal("couldnt make temp file")
	}
	defer os.Remove(f.Name())

	if _, err := f.Write([]byte(good_yaml)); err != nil {
		t.Fatal("couldnt write to temp file")
	}
	f.Close()

	var cfg config.Config
	err = config.Load(f.Name(), &cfg)

	if err != nil {
		t.Errorf("load failed but should have passed. error: %v", err)
	}
}

func TestLoadBadPath(t *testing.T) {
	var cfg config.Config
	err := config.Load("/tmp/this/file/does/not/exist.yaml", &cfg)
	if err == nil {
		t.Error("load passed but should have failed")
	}
}

func TestLoadBadYAML(t *testing.T) {
	bad_yaml := "server: 'bad"

	f, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal("couldnt make temp file")
	}
	defer os.Remove(f.Name())

	if _, err := f.Write([]byte(bad_yaml)); err != nil {
		t.Fatal("couldnt write to temp file")
	}
	f.Close()

	var cfg config.Config
	err = config.Load(f.Name(), &cfg)
	if err == nil {
		t.Error("load passed with bad yaml but should have failed")
	}
}

func TestConfigValidation(t *testing.T) {
	testCases := []struct {
		name          string
		yaml          string
		expectedError string
	}{
		{
			name: "missing server address",
			yaml: `
server:
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: 1s
  cache_ttl: 1s
  max_concurrent: 1
metrics:
  - name: "my_metric"
    help: "some help"
    type: "gauge"
    command: "echo 1"
`,
			expectedError: "server.listen_address is required",
		},
		{
			name: "metric without help",
			yaml: `
server:
  listen_address: ":8080"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: 1s
  cache_ttl: 1s
  max_concurrent: 1
metrics:
  - name: "my_metric"
    # help: "some help"
    type: "gauge"
    command: "echo 1"
`,
			expectedError: "help string is required",
		},
		{
			name: "metric with bad type",
			yaml: `
server:
  listen_address: ":8080"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: 1s
  cache_ttl: 1s
  max_concurrent: 1
metrics:
  - name: "my_metric"
    help: "some help"
    type: "bad type"
    command: "echo 1"
`,
			expectedError: "type is invalid. valid: gauge, count",
		},
		{
			name: "no metrics defined",
			yaml: `
server:
  listen_address: ":8080"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: 1s
  cache_ttl: 1s
  max_concurrent: 1
metrics: []
`,
			expectedError: "at least one metric must be defined",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "test-*.yaml")
			if err != nil {
				t.Fatalf("could not create temp file: %v", err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tc.yaml)); err != nil {
				t.Fatalf("could not write to temp file: %v", err)
			}
			tmpfile.Close()

			var cfg config.Config
			err = config.Load(tmpfile.Name(), &cfg)

			if err == nil {
				t.Fatal("Load() passed, but it should have failed")
			}

			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("error message should contain '%s', but it was: '%s'", tc.expectedError, err.Error())
			}
		})
	}
}
